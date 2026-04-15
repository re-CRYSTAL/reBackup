package backup_test

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rebackup/internal/backup"
	"rebackup/pkg/logger"
)

// makeTestDir creates a temporary directory with several files and sub-dirs.
func makeTestDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{
		"file1.txt":        "hello world",
		"sub/file2.txt":    "nested content",
		"sub/deep/file.go": "package main",
	}

	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCreate_ProducesValidArchive(t *testing.T) {
	log := logger.New()
	b := backup.New(log)

	src := makeTestDir(t)
	outDir := t.TempDir()

	archivePath, err := b.Create(src, outDir)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Archive file must exist.
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive not found: %v", err)
	}

	// Archive must be non-empty.
	fi, _ := os.Stat(archivePath)
	if fi.Size() == 0 {
		t.Fatal("archive is empty")
	}

	// Verify the gzip + tar structure is readable and contains expected entries.
	entries := listTarGz(t, archivePath)
	mustContain(t, entries, "file1.txt")
	mustContain(t, entries, filepath.Join("sub", "file2.txt"))
	mustContain(t, entries, filepath.Join("sub", "deep", "file.go"))
}

func TestCreate_ArchiveNameHasTimestamp(t *testing.T) {
	name := backup.ArchiveName()
	if !strings.HasPrefix(name, "backup_") {
		t.Errorf("expected prefix 'backup_', got %q", name)
	}
	if !strings.HasSuffix(name, ".tar.gz") {
		t.Errorf("expected suffix '.tar.gz', got %q", name)
	}
}

func TestCreate_NonexistentSource(t *testing.T) {
	log := logger.New()
	b := backup.New(log)

	_, err := b.Create("/nonexistent/path/xyz", t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent source, got nil")
	}
}

func TestCreate_EmptyDirectory(t *testing.T) {
	log := logger.New()
	b := backup.New(log)

	emptyDir := t.TempDir()
	outDir := t.TempDir()

	archivePath, err := b.Create(emptyDir, outDir)
	if err != nil {
		t.Fatalf("Create() on empty dir error: %v", err)
	}

	fi, _ := os.Stat(archivePath)
	if fi == nil || fi.Size() == 0 {
		t.Fatal("archive for empty dir should still be non-zero (gzip header)")
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func listTarGz(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		names = append(names, hdr.Name)
	}
	return names
}

func mustContain(t *testing.T, entries []string, sub string) {
	t.Helper()
	for _, e := range entries {
		if strings.Contains(e, sub) {
			return
		}
	}
	t.Errorf("archive entries do not contain %q\nGot: %v", sub, entries)
}
