package main

import (
	"strings"
	"testing"
)

func TestIndentationTabsToFourSpaces(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		src := strings.ReplaceAll("plugins {\n\tid 'java'\n}\njava {\n\ttoolchain {\n\t\tlanguageVersion = JavaLanguageVersion.of(21)\n\t}\n}\n// comment\n\t/* note */\nvalue = 'a\tb'\ntext = \"\"\"first\n\tkeep literal tab\nlast\"\"\"\n", "\n", nl)
		want := strings.ReplaceAll("plugins {\n    id 'java'\n}\njava {\n    toolchain {\n        languageVersion = JavaLanguageVersion.of(21)\n    }\n}\n// comment\n    /* note */\nvalue = 'a\tb'\ntext = \"\"\"first\n\tkeep literal tab\nlast\"\"\"\n", "\n", nl)
		got, err := normalizeIndentation(src)
		if err != nil || got != want {
			t.Fatalf("got %q %v; want %q", got, err, want)
		}
		again, err := normalizeIndentation(got)
		if err != nil || again != got {
			t.Fatal("not idempotent", err)
		}
	}
}
