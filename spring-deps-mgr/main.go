package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

var version = "0.0.0"

const usage = `spring-deps-mgr: interactive Gradle dependency checklist

Usage: spring-deps-mgr [--lib] [options]

Run in your Gradle project. Existing dependencies are checked.
Up/Down: move | Space: toggle | Enter: apply | Ctrl-C: cancel
Type to filter the list; Backspace clears the filter.

  --lib                 select Initializr library bundles instead of individual dependencies
  --file PATH           build.gradle or build.gradle.kts (default: auto-detect)
  --boot-version VER    override detected Spring Boot version
  --target URL          Initializr URL (default: https://start.spring.io)
  --dry-run             select interactively, then print without writing
  -h, --help            show help
  -v, --version         show version
`

type options struct {
	file, boot, target string
	lib, dry           bool
	items              []string
}

func arguments(args []string) (options, error) {
	o := options{target: "https://start.spring.io"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--lib":
			o.lib = true
		case "--dry-run":
			o.dry = true
		case "--file", "--boot-version", "--target":
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				return o, fmt.Errorf("%s requires a value", a)
			}
			switch a {
			case "--file":
				o.file = args[i]
			case "--boot-version":
				o.boot = args[i]
			case "--target":
				o.target = args[i]
			}
		default:
			return o, fmt.Errorf("unexpected argument %q; run spring-deps-mgr without a command to open the checklist", a)
		}
	}
	return o, nil
}

func main() {
	args := os.Args[1:]
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Print(usage)
			return
		}
		if a == "--version" || a == "-v" {
			fmt.Println(version)
			return
		}
	}
	if err := run(args, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "spring-deps-mgr:", err)
		os.Exit(1)
	}
}

type chooser func([]choice, string) ([]choice, bool, error)

func run(args []string, out io.Writer) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) {
		return fmt.Errorf("an interactive terminal is required; use --help for usage")
	}
	return runWithChooser(args, out, terminalChecklist)
}

func runWithChooser(args []string, out io.Writer, choose chooser) error {
	o, err := arguments(args)
	if err != nil {
		return err
	}
	if o.file == "" {
		for _, name := range []string{"build.gradle", "build.gradle.kts"} {
			if _, err := os.Stat(name); err == nil {
				if o.file != "" {
					return fmt.Errorf("both Gradle files exist; use --file")
				}
				o.file = name
			}
		}
		if o.file == "" {
			return fmt.Errorf("no build.gradle or build.gradle.kts; use --file")
		}
	}
	name := filepath.Base(o.file)
	if name != "build.gradle" && name != "build.gradle.kts" {
		return fmt.Errorf("only build.gradle and build.gradle.kts are supported")
	}
	info, err := os.Lstat(o.file)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("build file must be a regular file (no symlinks)")
	}
	raw, err := os.ReadFile(o.file)
	if err != nil {
		return err
	}
	src := string(raw)
	b, err := parse(src)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	catalog, boot, err := loadCatalog(client, o, src)
	if err != nil {
		return err
	}
	o.boot = boot
	rows := makeChoices(b, catalog, o.lib)
	title := "Dependencies"
	if o.lib {
		title = "Libraries"
	}
	title += " | " + o.file + " | Boot " + boot
	selected, cancelled, err := choose(rows, title)
	if err != nil {
		return err
	}
	if cancelled {
		fmt.Fprintln(out, "Cancelled. No changes.")
		return nil
	}
	var updated string
	if o.lib {
		updated, err = applyLibraries(client, o, src, selected, strings.HasSuffix(o.file, ".kts"))
	} else {
		updated, err = applyIndividual(src, selected, strings.HasSuffix(o.file, ".kts"))
	}
	if err != nil {
		return err
	}
	if o.dry {
		_, err = io.WriteString(out, updated)
		return err
	}
	if updated == src {
		fmt.Fprintln(out, "No changes.")
		return nil
	}
	current, err := os.ReadFile(o.file)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, raw) {
		return fmt.Errorf("build file changed while selecting dependencies; retry")
	}
	if err = atomicWrite(o.file, []byte(updated), info.Mode()); err != nil {
		return err
	}
	fmt.Fprintln(out, "Updated", o.file)
	return nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".spring-deps-mgr-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
