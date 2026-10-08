package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// This file contains generic helpers for editing INI files (such as
// ~/.aws/config) without disturbing content awsc does not own, and for
// writing files safely.

var sectionHeaderPattern = regexp.MustCompile(`^\[([^\]]*)\]\s*([#;].*)?$`)

// iniSection is a contiguous block of lines starting at a section header. The
// preamble before the first header is a section with an empty name.
type iniSection struct {
	name  string // header contents without brackets, whitespace-normalised
	lines []string
}

// parseSections splits content into sections, preserving every line.
func parseSections(content string) []*iniSection {
	sections := []*iniSection{{}}
	for _, line := range strings.Split(content, "\n") {
		if name, ok := sectionName(line); ok {
			sections = append(sections, &iniSection{name: name, lines: []string{line}})
			continue
		}
		cur := sections[len(sections)-1]
		cur.lines = append(cur.lines, line)
	}
	return sections
}

// joinSections reassembles sections into file content. Sections with nil
// lines are dropped.
func joinSections(sections []*iniSection) string {
	var lines []string
	for _, s := range sections {
		lines = append(lines, s.lines...)
	}
	return strings.Join(lines, "\n")
}

// appendSections appends new sections to content, separated by blank lines.
func appendSections(content string, blocks [][]string) string {
	if len(blocks) == 0 {
		return content
	}
	content = strings.TrimRight(content, "\n")
	for _, block := range blocks {
		if content != "" {
			content += "\n\n"
		}
		content += strings.Join(block, "\n")
	}
	return content + "\n"
}

// sectionName returns the normalised name of a section header line, accepting
// trailing comments and irregular inner whitespace like the AWS CLI parser.
func sectionName(line string) (string, bool) {
	m := sectionHeaderPattern.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return "", false
	}
	return strings.Join(strings.Fields(m[1]), " "), true
}

// sectionKeys returns the top-level keys set in a section and whether it
// contains anything outside allowed (unknown keys or nested values).
func sectionKeys(s *iniSection, allowed map[string]bool) (keys map[string]string, foreign bool) {
	keys = map[string]string{}
	for _, line := range s.lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			foreign = true // nested value, e.g. "s3 =\n  max_concurrent_requests = 10"
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		key = strings.TrimSpace(key)
		if !ok || !allowed[key] {
			foreign = true
			continue
		}
		keys[key] = strings.TrimSpace(value)
	}
	return keys, foreign
}

// setSectionKey replaces the value of key in a section.
func setSectionKey(s *iniSection, key, value string) {
	for i, line := range s.lines[1:] {
		if k, _, ok := strings.Cut(strings.TrimSpace(line), "="); ok && strings.TrimSpace(k) == key {
			s.lines[i+1] = key + " = " + value
		}
	}
}

// replaceSectionBody replaces a section's content with lines, keeping its
// trailing blank lines. An identical section is left untouched.
func replaceSectionBody(s *iniSection, lines []string) {
	end := len(s.lines)
	for end > 0 && strings.TrimSpace(s.lines[end-1]) == "" {
		end--
	}
	if strings.Join(s.lines[:end], "\n") == strings.Join(lines, "\n") {
		return
	}
	s.lines = append(append([]string{}, lines...), s.lines[end:]...)
}

// validateINIValue rejects values that would break INI structure (newlines,
// carriage returns, or section brackets).
func validateINIValue(label, value string) error {
	if strings.ContainsAny(value, "\r\n[]") {
		return fmt.Errorf("invalid %s %q: contains illegal characters", label, value)
	}
	return nil
}

// lockFile takes an exclusive advisory lock so concurrent awsc processes do
// not overwrite each other's changes. The returned function releases it.
func lockFile(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// WriteFileAtomic writes data to path via a temp file and rename, so a crash
// never leaves a truncated file. New files get perm. When preserveMode is true
// and the file already exists, its current permissions are kept; otherwise perm
// is enforced. Symlinked targets are resolved first so the link is preserved.
func WriteFileAtomic(path string, data []byte, perm os.FileMode, preserveMode bool) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if preserveMode {
		if info, err := os.Stat(path); err == nil {
			perm = info.Mode().Perm()
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
