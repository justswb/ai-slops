package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

type dep struct{ Config, Coordinate string }
type token struct {
	value             string
	start, end, depth int
	quoted            bool
}
type declaration struct {
	dep
	start, end, coordStart, coordEnd int
	complex                          bool
}
type block struct {
	open, close int
	entries     []declaration
}

// tokenize keeps source offsets and ignores comments; braces in strings are inert.
func tokenize(s string) ([]token, error) {
	var out []token
	depth := 0
	for i := 0; i < len(s); {
		if unicode.IsSpace(rune(s[i])) {
			i++
			continue
		}
		if strings.HasPrefix(s[i:], "//") {
			if n := strings.IndexByte(s[i:], '\n'); n >= 0 {
				i += n
			} else {
				i = len(s)
			}
			continue
		}
		if strings.HasPrefix(s[i:], "/*") {
			n := strings.Index(s[i+2:], "*/")
			if n < 0 {
				return nil, fmt.Errorf("unterminated comment")
			}
			i += n + 4
			continue
		}
		start := i
		q := s[i] == '\'' || s[i] == '"'
		if q {
			quote := s[i]
			i++
			closed := false
			for i < len(s) {
				if s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == quote {
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated string")
			}
		} else if unicode.IsLetter(rune(s[i])) || s[i] == '_' {
			for i < len(s) && (unicode.IsLetter(rune(s[i])) || unicode.IsDigit(rune(s[i])) || s[i] == '_') {
				i++
			}
		} else {
			i++
		}
		value := s[start:i]
		if value == "}" {
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced braces")
			}
		}
		out = append(out, token{value, start, i, depth, q})
		if value == "{" {
			depth++
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced braces")
	}
	return out, nil
}

var configPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var coordPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+(?::[A-Za-z0-9_.+@-]+)?(?::[A-Za-z0-9_.-]+)?$`)

func parse(s string) (block, error) {
	ts, err := tokenize(s)
	if err != nil {
		return block{}, err
	}
	b := block{open: -1, close: -1}
	found := 0
	for i, t := range ts {
		if t.value != "dependencies" || t.quoted || i+1 >= len(ts) || ts[i+1].value != "{" {
			continue
		}
		if t.depth != 0 {
			continue
		}
		found++
		b.open = ts[i+1].start
		j := i + 2
		for ; j < len(ts); j++ {
			if ts[j].value == "}" && ts[j].depth == 0 {
				b.close = ts[j].start
				break
			}
		}
		for k := i + 2; k < j; k++ {
			c := ts[k]
			if c.depth != 1 || !configPattern.MatchString(c.value) {
				continue
			}
			n := k + 1
			paren := n < j && ts[n].value == "("
			if paren {
				n++
			}
			if n >= j || !ts[n].quoted {
				continue
			}
			v := ts[n]
			coord := v.value[1 : len(v.value)-1]
			if !coordPattern.MatchString(coord) {
				continue
			}
			end := v.end
			next := n + 1
			complex := false
			if paren {
				if next >= j || ts[next].value != ")" {
					complex = true
				} else {
					end = ts[next].end
					next++
				}
			}
			if next < j && ts[next].value == ";" {
				end = ts[next].end
				next++
			}
			if next < j && (ts[next].value == "{" || !strings.Contains(s[end:ts[next].start], "\n")) {
				complex = true
			}
			b.entries = append(b.entries, declaration{dep{c.value, coord}, c.start, end, v.start + 1, v.end - 1, complex})
			k = n
		}
	}
	if found > 1 {
		return b, fmt.Errorf("multiple top-level dependencies blocks; select a simpler build file")
	}
	if found == 0 {
		for _, t := range ts {
			if t.value == "dependencies" && !t.quoted {
				return b, fmt.Errorf("only nested or dynamic dependencies found; select a module build file")
			}
		}
	}
	return b, nil
}

func module(s string) string {
	p := strings.Split(s, ":")
	if len(p) >= 2 {
		return p[0] + ":" + p[1]
	}
	return s
}

func edit(src, action string, wanted []dep, kotlin bool) (string, error) {
	// Validate the whole batch before returning any changes to the caller.
	for _, d := range wanted {
		if !coordPattern.MatchString(d.Coordinate) || d.Config != "" && !configPattern.MatchString(d.Config) {
			return "", fmt.Errorf("invalid dependency: %s %s", d.Config, d.Coordinate)
		}
	}
	result := src
	for _, d := range wanted {
		b, err := parse(result)
		if err != nil {
			return "", err
		}
		type replacement struct {
			start, end int
			text       string
		}
		var changes []replacement
		matched := false
		for _, e := range b.entries {
			if module(e.Coordinate) != module(d.Coordinate) || (d.Config != "" && e.Config != d.Config) {
				continue
			}
			if action == "remove" && strings.Count(d.Coordinate, ":") > 1 && e.Coordinate != d.Coordinate {
				continue
			}
			if e.complex {
				return "", fmt.Errorf("%s has a complex declaration; edit it manually", e.Coordinate)
			}
			matched = true
			if action == "remove" {
				start, end := removalRange(result, e)
				changes = append(changes, replacement{start, end, ""})
			} else {
				changes = append(changes, replacement{e.coordStart, e.coordEnd, d.Coordinate})
			}
		}
		for i := len(changes) - 1; i >= 0; i-- {
			c := changes[i]
			result = result[:c.start] + c.text + result[c.end:]
		}
		if action == "remove" || matched {
			continue
		}
		nl := "\n"
		if strings.Contains(result, "\r\n") {
			nl = "\r\n"
		}
		line := d.Config + " '" + d.Coordinate + "'"
		if kotlin {
			line = d.Config + "(\"" + d.Coordinate + "\")"
		}
		if b.open < 0 {
			if result != "" && !strings.HasSuffix(result, "\n") {
				result += nl
			}
			result += nl + "dependencies {" + nl + "    " + line + nl + "}" + nl
			continue
		}
		insert := b.close
		indent := "    "
		prefix := nl + indent
		last := strings.LastIndex(result[:insert], "\n") + 1
		if strings.TrimSpace(result[last:insert]) == "" {
			insert = last
			prefix = indent
		}
		result = result[:insert] + prefix + line + nl + result[insert:]
	}
	if result != src {
		return cleanDependencyGaps(result)
	}
	return result, nil
}

// Normalize only whitespace-only gaps between parsed declarations and the block
// boundaries. A gap containing a comment, expression or string stays untouched.
func cleanDependencyGaps(src string) (string, error) {
	b, err := parse(src)
	if err != nil {
		return "", err
	}
	if b.open < 0 {
		return src, nil
	}
	type gap struct{ start, end int }
	var gaps []gap
	start := b.open + 1
	for _, e := range b.entries {
		gaps = append(gaps, gap{start, e.start})
		start = e.end
	}
	gaps = append(gaps, gap{start, b.close})
	nl := "\n"
	if strings.Contains(src, "\r\n") {
		nl = "\r\n"
	}
	result := src
	for i := len(gaps) - 1; i >= 0; i-- {
		g := gaps[i]
		space := src[g.start:g.end]
		if strings.TrimSpace(space) != "" || !strings.Contains(space, "\n") {
			continue
		}
		// Keep the indentation of the following declaration or closing brace.
		indent := space[strings.LastIndexByte(space, '\n')+1:]
		result = result[:g.start] + nl + indent + result[g.end:]
	}
	// Reparse after gap cleanup to retain correct source offsets, then normalize
	// only simple standalone declarations; nested closures keep their formatting.
	b, err = parse(result)
	if err != nil {
		return "", err
	}
	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if e.complex {
			continue
		}
		lineStart := strings.LastIndex(result[:e.start], "\n") + 1
		if strings.TrimSpace(result[lineStart:e.start]) == "" {
			result = result[:lineStart] + "    " + result[e.start:]
		}
	}
	return result, nil
}

// A standalone declaration owns its indentation and newline, but not comments
// or neighboring statements. Removing the entire owned line prevents blank-line growth.
func removalRange(src string, e declaration) (int, int) {
	start, end := e.start, e.end
	lineStart := strings.LastIndex(src[:start], "\n") + 1
	lineEnd := len(src)
	if next := strings.IndexByte(src[end:], '\n'); next >= 0 {
		lineEnd = end + next
	}
	if strings.TrimSpace(src[lineStart:start]) == "" {
		if strings.TrimSpace(src[end:lineEnd]) == "" {
			start = lineStart
			end = lineEnd
			if end < len(src) {
				end++
			}
		} else {
			// Leave the existing indentation in front of a trailing comment.
			for end < lineEnd && (src[end] == ' ' || src[end] == '\t') {
				end++
			}
		}
	}
	return start, end
}
