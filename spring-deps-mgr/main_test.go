package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibraryRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fmt.Fprint(w, `{"bootVersion":{"default":"3.5.0"},"dependencies":{"values":[{"name":"Tools","values":[{"id":"lombok","name":"Lombok"}]}]}}`)
			return
		}
		if r.URL.Path == "/dependencies" {
			fmt.Fprint(w, `{"dependencies":{"lombok":{"groupId":"org.projectlombok","artifactId":"lombok","scope":"annotationProcessor"}}}`)
			return
		}
		if r.URL.Query().Get("dependencies") != "lombok" || r.URL.Query().Get("bootVersion") != "3.5.0" {
			t.Errorf("unexpected query %s", r.URL.RawQuery)
		}
		z := zip.NewWriter(w)
		f, _ := z.Create("build.gradle")
		fmt.Fprint(f, "dependencies {\n implementation 'org.springframework.boot:spring-boot-starter'\n compileOnly 'org.projectlombok:lombok'\n annotationProcessor 'org.projectlombok:lombok'\n testImplementation 'org.springframework.boot:spring-boot-starter-test'\n}\n")
		z.Close()
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "build.gradle")
	original := "plugins { id 'org.springframework.boot' version '3.5.0' }\ndependencies {\n implementation 'org.springframework.boot:spring-boot-starter'\n}\n"
	os.WriteFile(path, []byte(original), 0644)
	var out bytes.Buffer
	args := []string{"--lib", "--file", path, "--target", server.URL}
	choose := func(rows []choice, title string) ([]choice, bool, error) {
		if len(rows) != 1 {
			t.Fatalf("unexpected rows: %+v", rows)
		}
		rows[0].checked = !rows[0].checked
		return rows, false, nil
	}
	if err := runWithChooser(append(args, "--dry-run"), &out, choose); err != nil {
		t.Fatal(err)
	}
	unchanged, _ := os.ReadFile(path)
	if string(unchanged) != original {
		t.Fatal("dry run wrote file")
	}
	if !strings.Contains(out.String(), "annotationProcessor") || strings.Contains(out.String(), "starter-test") {
		t.Fatal(out.String())
	}
	if err := runWithChooser(args, &out, choose); err != nil {
		t.Fatal(err)
	}
	if err := runWithChooser(args, &out, choose); err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(path)
	if strings.Contains(string(actual), "lombok") {
		t.Fatal(string(actual))
	}
	if !strings.Contains(string(actual), "implementation 'org.springframework.boot:spring-boot-starter'") {
		t.Fatal("removed baseline starter", string(actual))
	}
}

func TestFailurePreservesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "build.gradle.kts")
	src := "dependencies {}\n"
	os.WriteFile(path, []byte(src), 0644)
	err := runWithChooser([]string{"add", "a:b:1", "invalid", "--file", path}, &bytes.Buffer{}, func(rows []choice, title string) ([]choice, bool, error) {
		t.Fatal("unexpected selector")
		return nil, false, nil
	})
	if err == nil {
		t.Fatal("invalid batch accepted")
	}
	actual, _ := os.ReadFile(path)
	if string(actual) != src {
		t.Fatal("partial write")
	}
}

func TestIndividualInteractiveFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fmt.Fprint(w, `{"bootVersion":{"default":"4.1.2"},"dependencies":{"values":[{"name":"SQL","values":[{"id":"h2","name":"H2 Database"}]}]}}`)
		} else {
			fmt.Fprint(w, `{"dependencies":{"h2":{"groupId":"com.h2database","artifactId":"h2","scope":"runtime"}}}`)
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "build.gradle.kts")
	original := "dependencies {\n implementation(\"a:b:1\")\n}\n"
	os.WriteFile(path, []byte(original), 0644)
	args := []string{"--file", path, "--target", server.URL}
	choose := func(rows []choice, title string) ([]choice, bool, error) {
		if len(rows) != 2 || !rows[0].checked || rows[1].checked {
			t.Fatalf("bad initial state: %+v", rows)
		}
		m := selection{rows: rows}
		for _, k := range []string{"space", "down", "space", "enter"} {
			m.key(k)
		}
		return m.rows, m.cancelled, nil
	}
	var out bytes.Buffer
	if err := runWithChooser(args, &out, choose); err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(path)
	if strings.Contains(string(actual), "a:b") || !strings.Contains(string(actual), `runtimeOnly("com.h2database:h2")`) {
		t.Fatal(string(actual))
	}
	// Ctrl-C discards pending checkbox changes.
	if err := runWithChooser(args, &out, func(rows []choice, title string) ([]choice, bool, error) {
		m := selection{rows: rows}
		m.key("space")
		m.key("ctrl-c")
		return m.rows, m.cancelled, nil
	}); err != nil {
		t.Fatal(err)
	}
	afterCancel, _ := os.ReadFile(path)
	if !bytes.Equal(actual, afterCancel) {
		t.Fatal("cancel changed file")
	}
	// An external edit during selection must not be overwritten.
	err := runWithChooser(args, &out, func(rows []choice, title string) ([]choice, bool, error) {
		rows[0].checked = false
		os.WriteFile(path, append(actual, []byte("// external edit\n")...), 0644)
		return rows, false, nil
	})
	if err == nil || !strings.Contains(err.Error(), "changed while selecting") {
		t.Fatal(err)
	}
}

func TestLibrarySharedDependencies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("dependencies")
		if id != "a" && id != "b" {
			t.Errorf("unexpected library: %s", id)
		}
		z := zip.NewWriter(w)
		f, _ := z.Create("build.gradle")
		fmt.Fprintf(f, "dependencies {\n implementation 'shared:module'\n implementation 'example:%s'\n}\n", id)
		z.Close()
	}))
	defer server.Close()
	src := "dependencies {\n implementation 'shared:module:custom'\n implementation 'example:a'\n implementation 'example:b'\n}\n"
	rows := []choice{{id: "a", initial: true, checked: false}, {id: "b", initial: true, checked: true}}
	got, err := applyLibraries(server.Client(), options{target: server.URL, boot: "4.1.2"}, src, rows, false)
	if err != nil || strings.Contains(got, "example:a") || !strings.Contains(got, "example:b") || !strings.Contains(got, "shared:module:custom") {
		t.Fatal(got, err)
	}
}

func TestLibraryErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		zip    bool
	}{
		{"http", 400, "unsupported Boot version", false},
		{"invalid zip", 200, "not zip", false},
		{"bom", 200, "dependencyManagement { imports {} }\ndependencies {}", true},
		{"repository", 200, "repositories { maven { url 'https://example.org' } }\ndependencies {}", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				if tc.zip {
					z := zip.NewWriter(w)
					f, _ := z.Create("build.gradle")
					fmt.Fprint(f, tc.body)
					z.Close()
				} else {
					fmt.Fprint(w, tc.body)
				}
			}))
			defer s.Close()
			if _, err := resolveLibraries(s.Client(), options{target: s.URL, boot: "3.5.0", items: []string{"web"}}, ""); err == nil {
				t.Fatal("accepted unsupported response")
			}
		})
	}
}
