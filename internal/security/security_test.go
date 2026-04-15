package security_test

import (
	"testing"

	"rebackup/internal/security"
)

func TestSafePath_ValidPaths(t *testing.T) {
	tests := []struct {
		name     string
		entry    string
		wantSufx string // expected suffix of the returned path
	}{
		{"simple file", "data/file.txt", "data/file.txt"},
		{"nested dir", "data/a/b/c.txt", "data/a/b/c.txt"},
		{"root-level file", "file.txt", "file.txt"},
		{"dot prefix", "./data/file.txt", "data/file.txt"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := security.SafePath("/tmp/restore", tc.entry)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == "" {
				t.Fatal("got empty path")
			}
			_ = got // valid path returned — no traversal
		})
	}
}

func TestSafePath_TraversalBlocked(t *testing.T) {
	attacks := []struct {
		name  string
		entry string
	}{
		{"double dot", "../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"mixed traversal", "data/../../etc/shadow"},
		{"null-like name", "data/../../../root/.ssh/id_rsa"},
		{"prefix bypass", "../restoreEvil/evil.txt"},
	}

	for _, tc := range attacks {
		t.Run(tc.name, func(t *testing.T) {
			_, err := security.SafePath("/tmp/restore", tc.entry)
			if err == nil {
				t.Fatalf("expected error for entry %q but got nil", tc.entry)
			}
		})
	}
}

func TestSafePath_EmptyEntry(t *testing.T) {
	_, err := security.SafePath("/tmp/restore", "")
	if err == nil {
		t.Fatal("expected error for empty entry name")
	}
}

func TestValidateArchivePath_Valid(t *testing.T) {
	cases := []string{
		"backup_2024-01-15_10-30.tar.gz",
		"./backups/backup.tar.gz",
		"/mnt/backups/backup.tar.gz",
	}
	for _, c := range cases {
		if err := security.ValidateArchivePath(c); err != nil {
			t.Errorf("unexpected error for %q: %v", c, err)
		}
	}
}

func TestValidateArchivePath_Invalid(t *testing.T) {
	if err := security.ValidateArchivePath(""); err == nil {
		t.Error("expected error for empty path")
	}
}
