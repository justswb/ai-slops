package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const maxResponse = 16 << 20

func get(client *http.Client, address, accept string) ([]byte, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("target must use http or https")
	}
	req, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, maxResponse+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponse {
		return nil, fmt.Errorf("Initializr response too large")
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Initializr HTTP %d: %.500s", res.StatusCode, data)
	}
	return data, nil
}

var bootPattern = regexp.MustCompile(`(?:id\s*\(?\s*['"]org\.springframework\.boot['"]\s*\)?\s*version\s*\(?\s*['"])([^'"]+)`)

func resolveLibraries(client *http.Client, o options, src string) ([]dep, error) {
	boot := o.boot
	if boot == "" {
		matches := bootPattern.FindAllStringSubmatch(src, -1)
		if len(matches) == 1 {
			boot = matches[0][1]
		} else {
			return nil, fmt.Errorf("cannot detect Spring Boot plugin version; use --boot-version")
		}
	}
	q := url.Values{"type": {"gradle-project"}, "language": {"java"}, "bootVersion": {boot}, "dependencies": {strings.Join(o.items, ",")}}
	data, err := get(client, strings.TrimRight(o.target, "/")+"/starter.zip?"+q.Encode(), "application/zip")
	if err != nil {
		return nil, err
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid Initializr archive: %w", err)
	}
	var build string
	for _, f := range archive.File {
		if f.Name != "build.gradle" {
			continue
		}
		if build != "" {
			return nil, fmt.Errorf("duplicate generated build.gradle")
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(r, maxResponse+1))
		r.Close()
		if err != nil {
			return nil, err
		}
		if len(b) > maxResponse {
			return nil, fmt.Errorf("generated build too large")
		}
		build = string(b)
	}
	if build == "" {
		return nil, fmt.Errorf("Initializr archive has no root build.gradle")
	}
	// Dependencies needing BOMs or custom repositories cannot safely be applied in isolation.
	ts, err := tokenize(build)
	if err != nil {
		return nil, err
	}
	for _, t := range ts {
		if !t.quoted && (t.value == "dependencyManagement" || t.value == "maven" || t.value == "mavenLocal" || t.value == "platform" || t.value == "enforcedPlatform") {
			return nil, fmt.Errorf("selected library requires BOM or repository configuration; configure it manually before adding individual coordinates")
		}
	}
	b, err := parse(build)
	if err != nil {
		return nil, err
	}
	var wanted []dep
	for _, d := range b.entries {
		// Baseline application/test dependencies do not belong to a selected library.
		if module(d.Coordinate) == "org.springframework.boot:spring-boot-starter" || module(d.Coordinate) == "org.springframework.boot:spring-boot-starter-test" || module(d.Coordinate) == "org.junit.platform:junit-platform-launcher" {
			continue
		}
		if d.complex {
			return nil, fmt.Errorf("Initializr generated a complex dependency: %s", d.Coordinate)
		}
		wanted = append(wanted, d.dep)
	}
	if len(wanted) == 0 {
		return nil, fmt.Errorf("selected libraries produced no editable dependencies")
	}
	return wanted, nil
}
