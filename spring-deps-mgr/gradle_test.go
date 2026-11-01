package main

import (
	"strings"
	"testing"
)

func TestOtherBlocksRemainByteIdentical(t *testing.T) {
	prefix := "plugins {\n\tid 'java'\n}\njava {\n\ttoolchain {\n\t\tlanguageVersion = JavaLanguageVersion.of(21)\n\t}\n}\nrepositories {\n        mavenCentral()\n}\n\n"
	suffix := "\ntasks.named('test') {\n\tuseJUnitPlatform()\n}\n"
	src := prefix + "dependencies {\n    implementation 'keep:module'\n}\n" + suffix
	for _, action := range []string{"add", "remove"} {
		got, err := edit(src, action, []dep{{"runtimeOnly", "new:module"}}, false)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(got, prefix) || !strings.HasSuffix(got, suffix) {
			t.Fatalf("unrelated indentation changed: %q", got)
		}
		src = got
	}
}

func TestEdit(t *testing.T) {
	for _, kotlin := range []bool{false, true} {
		src := "plugins { id(\"java\") }\r\n// dependencies { fake }\r\ndependencies {\r\n    implementation(\"a:b:1\") // keep\r\n    testImplementation(\"c:d\")\r\n}\r\n"
		got, err := edit(src, "add", []dep{{"implementation", "a:b:2"}}, kotlin)
		if err != nil || !strings.Contains(got, "a:b:2") || strings.Contains(got, "a:b:1") || !strings.Contains(got, "// keep\r\n") {
			t.Fatalf("%s %v", got, err)
		}
		again, err := edit(got, "add", []dep{{"implementation", "a:b:2"}}, kotlin)
		if err != nil || again != got {
			t.Fatal("not idempotent", err)
		}
		got, err = edit(got, "remove", []dep{{"", "a:b"}}, kotlin)
		if err != nil || strings.Contains(got, "a:b") || !strings.Contains(got, "c:d") {
			t.Fatalf("%s %v", got, err)
		}
	}
}

func TestSafety(t *testing.T) {
	for _, src := range []string{"dependencies {", "dependencies {}\ndependencies {}", "subprojects { dependencies {} }", "buildscript { dependencies {} }"} {
		if _, err := edit(src, "add", []dep{{"implementation", "a:b"}}, false); err == nil {
			t.Fatalf("accepted ambiguous input: %s", src)
		}
	}
	src := "dependencies {\n implementation(\"a:b\") { exclude(group = \"x\") }\n // implementation 'x:y'\n}\n"
	if _, err := edit(src, "remove", []dep{{"", "a:b"}}, false); err == nil {
		t.Fatal("must refuse closure removal")
	}
}

func TestCreateBlock(t *testing.T) {
	got, err := edit("plugins { id 'java' }\n", "add", []dep{{"implementation", "a:b"}}, false)
	if err != nil || !strings.Contains(got, "implementation 'a:b'") {
		t.Fatal(got, err)
	}
}

func TestRepeatedEditsPreserveWhitespace(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		for _, indent := range []string{"    ", "\t", "  ", "        "} {
			original := "dependencies {" + nl + indent + "implementation 'keep:module'" + nl + "}" + nl
			want := "dependencies {" + nl + "    implementation 'keep:module'" + nl + "}" + nl
			got := original
			for i := 0; i < 20; i++ {
				var err error
				got, err = edit(got, "add", []dep{{"runtimeOnly", "new:module"}}, false)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(got, nl+"    runtimeOnly") {
					t.Fatalf("wrong indentation: %q", got)
				}
				got, err = edit(got, "remove", []dep{{"runtimeOnly", "new:module"}}, false)
				if err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Fatalf("cycle %d accumulated whitespace:\n%q\nwant %q", i, got, want)
				}
			}
		}
	}
}

func TestRemovalPreservesCommentsAndUnrelatedSpacing(t *testing.T) {
	src := "// before\ndependencies {\n    implementation 'a:b' // keep this comment\n\n    runtimeOnly 'c:d'   \n}\n// after\n"
	got, err := edit(src, "remove", []dep{{"", "a:b"}, {"", "c:d"}}, false)
	want := "// before\ndependencies {\n    // keep this comment\n\n}\n// after\n"
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

func TestLegacyBlankLinesAreCleanedOnDeselection(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		src := strings.ReplaceAll("// before\n\ndependencies {\n\t\n\t\n    implementation 'keep:module'\n \t\n\n    runtimeOnly 'remove:module'\n\t\n}\n\n// after\n", "\n", nl)
		want := strings.ReplaceAll("// before\n\ndependencies {\n    implementation 'keep:module'\n}\n\n// after\n", "\n", nl)
		got, err := applyIndividual(src, []choice{{initial: true, checked: false, deps: []dep{{"runtimeOnly", "remove:module"}}}}, false)
		if err != nil || got != want {
			t.Fatalf("got %q %v; want %q", got, err, want)
		}
		for i := 0; i < 10; i++ {
			got, err = edit(got, "add", []dep{{"runtimeOnly", "remove:module"}}, false)
			if err != nil {
				t.Fatal(err)
			}
			got, err = edit(got, "remove", []dep{{"runtimeOnly", "remove:module"}}, false)
			if err != nil || got != want {
				t.Fatalf("cycle %d: %q %v", i, got, err)
			}
		}
	}
}

func TestBlankLineCleanupProtectsMultilineContent(t *testing.T) {
	src := "dependencies {\n    /* notes\n\n       preserved\n    */\n    implementation(\"\"\"multi\n\nline\"\"\")\n    runtimeOnly 'remove:module'\n}\n"
	want := strings.Replace(src, "    runtimeOnly 'remove:module'\n", "", 1)
	got, err := edit(src, "remove", []dep{{"runtimeOnly", "remove:module"}}, false)
	if err != nil || got != want {
		t.Fatalf("got %q %v; want %q", got, err, want)
	}
}
