package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"golang.org/x/term"
)

type choice struct {
	id, label, disabled string
	deps                []dep
	checked, initial    bool
}
type selection struct {
	rows      []choice
	query     string
	cursor    int
	cancelled bool
}

func (m *selection) visible() []int {
	var result []int
	for i, c := range m.rows {
		if strings.Contains(strings.ToLower(c.label), strings.ToLower(m.query)) {
			result = append(result, i)
		}
	}
	return result
}
func (m *selection) key(k string) bool {
	visible := m.visible()
	switch k {
	case "ctrl-c":
		m.cancelled = true
		return true
	case "enter":
		return true
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down":
		if m.cursor+1 < len(visible) {
			m.cursor++
		}
	case "space":
		if len(visible) > 0 {
			c := &m.rows[visible[m.cursor]]
			if c.disabled == "" {
				c.checked = !c.checked
			}
		}
	case "backspace":
		r := []rune(m.query)
		if len(r) > 0 {
			m.query = string(r[:len(r)-1])
			m.cursor = 0
		}
	default:
		if len([]rune(k)) == 1 && unicode.IsPrint([]rune(k)[0]) {
			m.query += k
			m.cursor = 0
		}
	}
	return false
}

type keyReader struct{ in *bufio.Reader }

func newKeyReader(in io.Reader) *keyReader { return &keyReader{bufio.NewReader(in)} }
func (r *keyReader) next() (string, error) {
	for {
		c, _, err := r.in.ReadRune()
		if err != nil {
			return "", err
		}
		switch c {
		case 3:
			return "ctrl-c", nil
		case '\r', '\n':
			return "enter", nil
		case ' ':
			return "space", nil
		case 8, 127:
			return "backspace", nil
		case 27:
			next, _, err := r.in.ReadRune()
			if err != nil {
				return "", err
			}
			if next == 3 {
				return "ctrl-c", nil
			}
			if next != '[' && next != 'O' {
				r.in.UnreadRune()
				continue
			}
			for {
				last, _, err := r.in.ReadRune()
				if err != nil {
					return "", err
				}
				if last >= 0x40 && last <= 0x7e {
					if last == 'A' {
						return "up", nil
					}
					if last == 'B' {
						return "down", nil
					}
					break
				}
			}
		default:
			if unicode.IsPrint(c) {
				return string(c), nil
			}
		}
	}
}

// Escape/control characters from remote catalog data must not become terminal instructions.
func safeText(s string, width int) string {
	var out []rune
	for _, r := range s {
		if unicode.IsControl(r) {
			r = ' '
		}
		out = append(out, r)
		if len(out) >= width {
			break
		}
	}
	return string(out)
}

func terminalChecklist(rows []choice, title string) ([]choice, bool, error) {
	inFD := int(os.Stdin.Fd())
	outFD := int(os.Stderr.Fd())
	restoreOutput, err := enableVirtualTerminal()
	if err != nil {
		return nil, false, err
	}
	defer restoreOutput()
	state, err := term.MakeRaw(inFD)
	if err != nil {
		return nil, false, err
	}
	defer term.Restore(inFD, state)
	fmt.Fprint(os.Stderr, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(os.Stderr, "\x1b[?25h\x1b[?1049l")
	m := selection{rows: rows}
	keys := newKeyReader(os.Stdin)
	for {
		width, height, err := term.GetSize(outFD)
		if err != nil {
			width = 100
			height = 24
		}
		width = max(20, width-1)
		page := max(1, height-8)
		visible := m.visible()
		start := max(0, min(m.cursor-page/2, len(visible)-page))
		end := min(len(visible), start+page)
		lines := []string{safeText(title, width), "Up/Down: move  Space: toggle  Enter: apply  Ctrl-C: cancel", safeText("Filter: "+m.query+"  (type to search, Backspace to clear)", width), ""}
		for n := start; n < end; n++ {
			c := m.rows[visible[n]]
			mark := "[ ]"
			if c.checked {
				mark = "[x]"
			}
			pointer := "  "
			if n == m.cursor {
				pointer = "> "
			}
			label := c.label
			if c.disabled != "" {
				label += " [locked: " + c.disabled + "]"
			}
			line := safeText(pointer+mark+" "+label, width)
			if n == m.cursor {
				line = "\x1b[7m" + line + "\x1b[0m"
			}
			lines = append(lines, line)
		}
		if len(visible) == 0 {
			lines = append(lines, "No matches. Backspace to clear your filter.")
		}
		changed := 0
		for _, c := range m.rows {
			if c.checked != c.initial {
				changed++
			}
		}
		lines = append(lines, fmt.Sprintf("%d shown / %d total | %d changes", len(visible), len(m.rows), changed))
		if len(visible) > 0 && m.rows[visible[m.cursor]].disabled != "" {
			lines = append(lines, safeText("Locked: "+m.rows[visible[m.cursor]].disabled, width))
		}
		fmt.Fprint(os.Stderr, "\x1b[H\x1b[2J"+strings.Join(lines, "\r\n"))
		k, err := keys.next()
		if err != nil {
			return nil, false, err
		}
		if m.key(k) {
			return m.rows, m.cancelled, nil
		}
	}
}
