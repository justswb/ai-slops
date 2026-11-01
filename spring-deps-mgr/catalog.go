package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type library struct {
	id, name, group, disabled string
	deps                      []dep
}
type catalogMetadata struct {
	BootVersion  struct{ Default string }
	Dependencies struct {
		Values []struct {
			Name   string
			Values []struct{ ID, Name string }
		}
	}
}
type catalogDependency struct{ GroupID, ArtifactID, Version, Scope, Bom, Repository string }

func loadCatalog(client *http.Client, o options, src string) ([]library, string, error) {
	target := strings.TrimRight(o.target, "/")
	raw, err := get(client, target, "application/vnd.initializr.v2.2+json")
	if err != nil {
		return nil, "", err
	}
	var metadata catalogMetadata
	if err = json.Unmarshal(raw, &metadata); err != nil {
		return nil, "", err
	}
	boot := o.boot
	if boot == "" {
		matches := bootPattern.FindAllStringSubmatch(src, -1)
		if len(matches) == 1 {
			boot = matches[0][1]
		} else {
			ts, err := tokenize(src)
			if err != nil {
				return nil, "", err
			}
			for _, t := range ts {
				if t.quoted && t.value[1:len(t.value)-1] == "org.springframework.boot" {
					return nil, "", fmt.Errorf("cannot detect Spring Boot plugin version; use --boot-version")
				}
			}
			boot = metadata.BootVersion.Default
		}
	}
	if boot == "" {
		return nil, "", fmt.Errorf("Initializr returned no default Boot version; use --boot-version")
	}
	raw, err = get(client, target+"/dependencies?"+url.Values{"bootVersion": {boot}}.Encode(), "application/json")
	if err != nil {
		return nil, "", err
	}
	var resolved struct{ Dependencies map[string]catalogDependency }
	if err = json.Unmarshal(raw, &resolved); err != nil {
		return nil, "", err
	}
	if len(resolved.Dependencies) == 0 {
		return nil, "", fmt.Errorf("Initializr returned an empty dependency catalog")
	}
	var result []library
	for _, group := range metadata.Dependencies.Values {
		for _, item := range group.Values {
			d, ok := resolved.Dependencies[item.ID]
			if !ok {
				continue
			}
			if d.GroupID == "" || d.ArtifactID == "" {
				continue
			}
			coordinate := d.GroupID + ":" + d.ArtifactID
			if d.Version != "" {
				coordinate += ":" + d.Version
			}
			l := library{id: item.ID, name: item.Name, group: group.Name}
			configurations := []string{"implementation"}
			switch d.Scope {
			case "runtime":
				configurations = []string{"runtimeOnly"}
			case "provided":
				configurations = []string{"compileOnly"}
			case "test":
				configurations = []string{"testImplementation"}
			case "annotationProcessor":
				configurations = []string{"compileOnly", "annotationProcessor", "testCompileOnly", "testAnnotationProcessor"}
			case "compile", "":
			default:
				l.disabled = "unsupported scope: " + d.Scope
			}
			for _, c := range configurations {
				l.deps = append(l.deps, dep{c, coordinate})
			}
			if d.Bom != "" || d.Repository != "" {
				l.disabled = "requires BOM / repository setup"
			}
			if item.ID == "native" || item.ID == "dgs-codegen" {
				l.disabled = "requires Gradle plugin setup"
			}
			if !coordPattern.MatchString(coordinate) {
				l.disabled = "non-literal version"
			}
			result = append(result, l)
		}
	}
	if len(result) == 0 {
		return nil, "", fmt.Errorf("no matching library IDs in Initializr catalog")
	}
	return result, boot, nil
}

func depKey(d dep) string { return d.Config + ":" + module(d.Coordinate) }

func makeChoices(b block, catalog []library, libMode bool) []choice {
	if libMode {
		var rows []choice
		for _, l := range catalog {
			c := choice{id: l.id, label: l.name + " [" + l.id + "] — " + l.group, deps: l.deps, disabled: l.disabled}
			for _, e := range b.entries {
				for _, d := range l.deps {
					if module(e.Coordinate) == module(d.Coordinate) {
						c.checked = true
						if e.complex {
							c.disabled = "complex existing declaration"
						}
					}
				}
			}
			c.initial = c.checked
			rows = append(rows, c)
		}
		return rows
	}
	var rows []choice
	seen := map[string]bool{}
	// Existing versions/configurations are authoritative and remain unchanged unless unchecked.
	for _, e := range b.entries {
		key := depKey(e.dep)
		if seen[key] {
			continue
		}
		seen[key] = true
		c := choice{label: e.Coordinate + "  (" + e.Config + ")", deps: []dep{e.dep}, checked: true, initial: true}
		for _, other := range b.entries {
			if depKey(other.dep) == key && other.complex {
				c.disabled = "complex declaration; read-only"
			}
		}
		rows = append(rows, c)
	}
	existingCount := len(rows)
	for _, l := range catalog {
		for _, d := range l.deps {
			key := depKey(d)
			if seen[key] {
				continue
			}
			seen[key] = true
			rows = append(rows, choice{label: d.Coordinate + "  (" + d.Config + ") — " + l.name, deps: []dep{d}, disabled: l.disabled})
		}
	}
	sort.SliceStable(rows[existingCount:], func(i, j int) bool { return rows[existingCount+i].label < rows[existingCount+j].label })
	return rows
}

func applyIndividual(src string, rows []choice, kotlin bool) (string, error) {
	var add, remove []dep
	for _, c := range rows {
		if c.checked == c.initial || c.disabled != "" {
			continue
		}
		if c.checked {
			add = append(add, c.deps...)
		} else {
			// A row represents a module/configuration, including duplicate version declarations.
			for _, d := range c.deps {
				d.Coordinate = module(d.Coordinate)
				remove = append(remove, d)
			}
		}
	}
	result, err := edit(src, "remove", remove, kotlin)
	if err != nil {
		return "", err
	}
	return edit(result, "add", add, kotlin)
}

func applyLibraries(client *http.Client, o options, src string, rows []choice, kotlin bool) (string, error) {
	var selectedIDs, removeIDs []string
	changed := false
	for _, c := range rows {
		if c.disabled != "" {
			continue
		}
		if c.checked {
			selectedIDs = append(selectedIDs, c.id)
		}
		if c.checked != c.initial {
			changed = true
		}
		if c.initial && !c.checked {
			removeIDs = append(removeIDs, c.id)
		}
	}
	if !changed {
		return src, nil
	}
	var add, remove []dep
	var err error
	if len(selectedIDs) > 0 {
		o.items = selectedIDs
		add, err = resolveLibraries(client, o, src)
		if err != nil {
			return "", err
		}
	}
	if len(removeIDs) > 0 {
		o.items = removeIDs
		remove, err = resolveLibraries(client, o, src)
		if err != nil {
			return "", err
		}
	}
	// Resolve the entire selected set so removing one bundle cannot remove a shared dependency.
	keep := map[string]bool{}
	for _, c := range rows {
		if c.checked {
			for _, d := range c.deps {
				keep[depKey(d)] = true
			}
		}
	}
	for _, d := range add {
		keep[depKey(d)] = true
	}
	filtered := remove[:0]
	for _, d := range remove {
		if !keep[depKey(d)] {
			d.Coordinate = module(d.Coordinate)
			filtered = append(filtered, d)
		}
	}
	result, err := edit(src, "remove", filtered, kotlin)
	if err != nil {
		return "", err
	}
	// Preserve explicit versions of dependencies that are already present.
	b, err := parse(result)
	if err != nil {
		return "", err
	}
	present := map[string]bool{}
	for _, e := range b.entries {
		present[depKey(e.dep)] = true
	}
	var missing []dep
	for _, d := range add {
		if !present[depKey(d)] {
			missing = append(missing, d)
		}
	}
	return edit(result, "add", missing, kotlin)
}
