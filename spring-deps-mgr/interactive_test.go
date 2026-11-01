package main

import (
	"io"
	"strings"
	"testing"
)

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
