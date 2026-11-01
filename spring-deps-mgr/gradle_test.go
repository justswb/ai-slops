package main

import (
	"strings"
	"testing"
)

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
