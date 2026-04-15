package restore_test

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"rebackup/internal/restore"
	"rebackup/pkg/logger"
)

// createTestArchive writes a .tar.gz containing the given entries to dst.
// entries: map of archive path → content ("" for directories).
func createTestArchive(t *testing.T, dst string, entries map[string]string) {
	t.Helper()

	f, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	for name, content := range entries {
		if content == "" {
			// directory entry
			if err := tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     name + "/",
				Mode:     0o755,
			}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0o644,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}

// createTraversalArchive writes an archive with a path-traversal entry.
func createTraversalArchive(t *testing.T, dst string) {
	t.Helper()
	f, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	// Dangerous entry — would escape the target directory.
	malicious := "../../evil.txt"
	body := "pwned"
	_ = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     malicious,
		Size:     int64(len(body)),
		Mode:     0o644,
	})
	_, _ = tw.Write([]byte(body))

	// Safe entry alongside the dangerous one.
	safe := "safe.txt"
	_ = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     safe,
		Size:     5,
		Mode:     0o644,
	})
	_, _ = tw.Write([]byte("hello"))

	_ = tw.Close()
	_ = gz.Close()
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestRestore_BasicExtraction(t *testing.T) {
	log := logger.New()
	r := restore.New(log)

	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	createTestArchive(t, archivePath, map[string]string{
		"data/file1.txt":        "content one",
		"data/sub/file2.txt":    "content two",
		"data/sub/deep/file.go": "package main",
	})

	target := t.TempDir()
	if err := r.Restore(archivePath, target); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	// Verify extracted files exist with correct content.
	check := map[string]string{
		"data/file1.txt":        "content one",
		"data/sub/file2.txt":    "content two",
		"data/sub/deep/file.go": "package main",
	}
	for rel, want := range check {
		got, err := os.ReadFile(filepath.Join(target, rel))
		if err != nil {
			t.Errorf("missing extracted file %s: %v", rel, err)
			continue
		}
		if string(got) != want {
			t.Errorf("file %s: got %q, want %q", rel, got, want)
		}
	}
}

func TestRestore_PathTraversalBlocked(t *testing.T) {
	log := logger.New()
	r := restore.New(log)

	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "evil.tar.gz")
	createTraversalArchive(t, archivePath)

	target := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	// Restore must NOT error out — it skips dangerous entries gracefully.
	if err := r.Restore(archivePath, target); err != nil {
		t.Fatalf("Restore() unexpected fatal error: %v", err)
	}

	// The evil file must NOT exist outside the target.
	evilPath := filepath.Join(tmpDir, "evil.txt")
	if _, err := os.Stat(evilPath); err == nil {
		t.Fatal("path traversal not prevented — evil.txt exists outside target!")
	}

	// The safe file MUST have been extracted.
	safePath := filepath.Join(target, "safe.txt")
	if _, err := os.Stat(safePath); err != nil {
		t.Errorf("safe.txt should have been extracted: %v", err)
	}
}

func TestRestore_MissingArchive(t *testing.T) {
	log := logger.New()
	r := restore.New(log)

	err := r.Restore("/nonexistent/backup.tar.gz", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing archive, got nil")
	}
}

func TestRestore_CorruptedArchive(t *testing.T) {
	log := logger.New()
	r := restore.New(log)

	// Write garbage bytes as the "archive".
	archivePath := filepath.Join(t.TempDir(), "corrupt.tar.gz")
	if err := os.WriteFile(archivePath, []byte("this is not gzip data!!!"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := r.Restore(archivePath, t.TempDir())
	if err == nil {
		t.Fatal("expected error for corrupted archive, got nil")
	}
}

func TestListContents_Valid(t *testing.T) {
	log := logger.New()
	r := restore.New(log)

	archivePath := filepath.Join(t.TempDir(), "list.tar.gz")
	createTestArchive(t, archivePath, map[string]string{
		"data/hello.txt": "hi",
		"data/world.txt": "there",
	})

	// ListContents must not return an error on a valid archive.
	if err := r.ListContents(archivePath); err != nil {
		t.Fatalf("ListContents() error: %v", err)
	}
}

func TestListContents_Corrupted(t *testing.T) {
	log := logger.New()
	r := restore.New(log)

	archivePath := filepath.Join(t.TempDir(), "bad.tar.gz")
	_ = os.WriteFile(archivePath, []byte("garbage"), 0o644)

	if err := r.ListContents(archivePath); err == nil {
		t.Fatal("expected error for corrupted archive")
	}
}

func TestRestore_DirectoriesCreated(t *testing.T) {
	log := logger.New()
	r := restore.New(log)

	archivePath := filepath.Join(t.TempDir(), "dirs.tar.gz")
	createTestArchive(t, archivePath, map[string]string{
		"a/b/c/": "",            // explicit directory
		"a/b/c/file.txt": "ok", // file inside nested dir
	})

	target := t.TempDir()
	if err := r.Restore(archivePath, target); err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	dirPath := filepath.Join(target, "a", "b", "c")
	fi, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("nested directory not created: %v", err)
	}
	if !fi.IsDir() {
		t.Fatal("expected directory, got file")
	}
}
