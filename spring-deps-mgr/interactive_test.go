package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestInlineRenderingPreservesTerminalHistory(t *testing.T) {
	var out bytes.Buffer
	r := inlineRenderer{out: &out}
	r.draw([]string{"Dependencies", "[x] existing", "[ ] new"})
	r.draw([]string{"Dependencies", "[x] existing"})
	r.finish()
	got := out.String()
	for _, forbidden := range []string{"\x1b[?1049", "\x1b[2J", "\x1b[H"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("screen switch or history clearing: %q", got)
		}
	}
	if !strings.Contains(got, "\x1b[2A\r\x1b[0JDependencies") {
		t.Fatalf("must redraw only its own lines: %q", got)
	}
	if !strings.HasSuffix(got, "\x1b[0m\x1b[?25h\r\n") {
		t.Fatalf("must restore cursor below checklist: %q", got)
	}
}

func TestChecklistKeys(t *testing.T) {
	m := selection{rows: []choice{{label: "existing", checked: true, initial: true}, {label: "new"}, {label: "locked", disabled: "complex"}}}
	for _, k := range []string{"space", "down", "space", "down", "space"} {
		m.key(k)
	}
	if m.rows[0].checked || !m.rows[1].checked || m.rows[2].checked {
		t.Fatal(m.rows)
	}
	if !m.key("enter") || m.cancelled {
		t.Fatal("enter must apply")
	}
	m = selection{rows: []choice{{label: "x"}}}
	m.key("space")
	m.key("ctrl-c")
	if !m.cancelled {
		t.Fatal("must cancel")
	}
}

func TestFilterPreservesHiddenSelections(t *testing.T) {
	m := selection{rows: []choice{{label: "Lombok", checked: true, initial: true}, {label: "Web"}}}
	m.key("w")
	m.key("e")
	m.key("b")
	m.key("space")
	if !m.rows[0].checked || !m.rows[1].checked {
		t.Fatal(m.rows)
	}
	m.key("backspace")
	if m.query != "we" {
		t.Fatal(m.query)
	}
}

func TestIndividualChanges(t *testing.T) {
	src := "dependencies {\n implementation 'a:b:1'\n testImplementation 'x:y'\n}\n"
	rows := []choice{{label: "a", initial: true, checked: false, deps: []dep{{"implementation", "a:b:1"}}}, {label: "x", initial: true, checked: true, deps: []dep{{"testImplementation", "x:y"}}}, {label: "c", checked: true, deps: []dep{{"runtimeOnly", "c:d"}}}}
	got, err := applyIndividual(src, rows, false)
	if err != nil || strings.Contains(got, "a:b") || !strings.Contains(got, "x:y") || !strings.Contains(got, "runtimeOnly 'c:d'") {
		t.Fatal(got, err)
	}
}

// One byte per Read catches arrow sequences fragmented across terminal reads.
type byteReader struct{ io.Reader }

func (r byteReader) Read(p []byte) (int, error) { return r.Reader.Read(p[:1]) }
func TestFragmentedKeys(t *testing.T) {
	r := newKeyReader(byteReader{strings.NewReader("\x1b[A\x1b[B \r\x03")})
	for _, want := range []string{"up", "down", "space", "enter", "ctrl-c"} {
		got, err := r.next()
		if err != nil || got != want {
			t.Fatalf("got %q %v want %q", got, err, want)
		}
	}
}

func TestOldCommandsRejected(t *testing.T) {
	for _, args := range [][]string{{"add"}, {"remove", "a:b"}, {"--lib", "web"}} {
		if _, err := arguments(args); err == nil {
			t.Fatal("accepted old interface", args)
		}
	}
	if _, err := arguments(nil); err != nil {
		t.Fatal(err)
	}
}

func TestLibraryCheckedFirst(t *testing.T) {
	b, err := parse("dependencies {\n implementation 'g:b'\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	rows := makeChoices(b, []library{{id: "a", name: "A", deps: []dep{{"implementation", "g:a"}}}, {id: "b", name: "B", deps: []dep{{"implementation", "g:b"}}}, {id: "c", name: "C", deps: []dep{{"implementation", "g:c"}}}}, true)
	if rows[0].id != "b" || !rows[0].checked {
		t.Fatal(rows)
	}
	m := selection{rows: rows}
	m.key("down")
	m.key("space")
	v := m.visible()
	if m.rows[v[0]].id != "b" || m.rows[v[1]].id != "a" || m.rows[v[m.cursor]].id != "a" {
		t.Fatal(m.rows, v, m.cursor)
	}
	m.key("up")
	m.key("space")
	v = m.visible()
	if m.rows[v[0]].id != "b" || m.rows[v[0]].checked || m.rows[v[1]].id != "a" || m.rows[v[m.cursor]].id != "b" || m.cursor != 0 {
		t.Fatal(m.rows, v, m.cursor)
	}
	m.key("a")
	m.key("space")
	m.key("backspace")
	v = m.visible()
	if m.rows[v[0]].id != "b" || m.rows[v[1]].id != "a" || m.rows[v[2]].id != "c" {
		t.Fatal("filtering and toggling must preserve initial ordering", m.rows, v)
	}
}

func TestDependencyNamesWithoutCoordinates(t *testing.T) {
	b, err := parse("dependencies {\n implementation 'org.springframework.boot:spring-boot-starter-web:1'\n runtimeOnly 'my.company:custom-driver:2'\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	rows := makeChoices(b, []library{{name: "Spring Web", deps: []dep{{"implementation", "org.springframework.boot:spring-boot-starter-web"}}}, {name: "Validation", deps: []dep{{"implementation", "org.springframework.boot:spring-boot-starter-validation"}}}}, false)
	if rows[0].label != "Spring Web  (implementation)" || rows[1].label != "custom-driver  (runtimeOnly)" || rows[2].label != "Validation  (implementation)" {
		t.Fatal(rows)
	}
	for _, c := range rows {
		if strings.Contains(c.label, ":") || strings.Contains(c.label, "org.springframework") {
			t.Fatal(c.label)
		}
	}
	m := selection{rows: rows, query: "org.springframework"}
	if len(m.visible()) != 2 {
		t.Fatal("coordinate search must still work")
	}
}
