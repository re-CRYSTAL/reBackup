// Package restore safely extracts .tar.gz archives with path traversal protection.
package restore

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"rebackup/internal/security"
	"rebackup/pkg/logger"
)

// maxFileSize guards against decompression-bomb attacks (10 GiB per entry).
const maxFileSize = 10 * 1024 * 1024 * 1024

// Restore handles archive extraction.
type Restore struct {
	log *logger.Logger
}

// New returns a Restore instance backed by the given logger.
func New(log *logger.Logger) *Restore {
	return &Restore{log: log}
}

// ─────────────────────────────────────────────────────────────────────────────
// ListContents
// ─────────────────────────────────────────────────────────────────────────────

// ListContents prints a table of all entries in a .tar.gz archive without
// extracting anything.
func (r *Restore) ListContents(archivePath string) error {
	if err := security.ValidateArchivePath(archivePath); err != nil {
		return err
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("cannot open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("not a valid gzip stream (corrupted?): %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	const colFmt = "%-10s  %-9s  %-20s  %s\n"
	fmt.Printf(colFmt, "TYPE", "SIZE", "MODIFIED", "NAME")
	fmt.Println(strings.Repeat("─", 70))

	var totalEntries int
	var totalSize int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read error (archive may be corrupted): %w", err)
		}

		kind := typeLabel(hdr.Typeflag)
		name := hdr.Name
		if hdr.Typeflag == tar.TypeSymlink {
			name = fmt.Sprintf("%s -> %s", hdr.Name, hdr.Linkname)
		}

		fmt.Printf(colFmt,
			kind,
			formatSize(hdr.Size),
			hdr.ModTime.Format("2006-01-02 15:04:05"),
			name,
		)

		totalEntries++
		totalSize += hdr.Size
	}

	fmt.Printf("%s\n", strings.Repeat("─", 70))
	fmt.Printf("Total: %d entries  |  %s\n", totalEntries, formatSize(totalSize))
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Restore
// ─────────────────────────────────────────────────────────────────────────────

// Restore extracts archivePath into targetPath.
//
// Security guarantees:
//   - Every entry path is validated by security.SafePath before any write.
//   - Entries that would escape targetPath are skipped and logged.
//   - Regular file writes are capped at maxFileSize to prevent bombs.
//   - A corrupted archive mid-stream is surfaced as a fatal error.
func (r *Restore) Restore(archivePath, targetPath string) error {
	if err := security.ValidateArchivePath(archivePath); err != nil {
		return err
	}

	r.log.Infof("Restore start  archive=%q  target=%q", archivePath, targetPath)

	fi, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("archive not found: %w", err)
	}
	if fi.IsDir() {
		return fmt.Errorf("%q is a directory, expected a .tar.gz file", archivePath)
	}

	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return fmt.Errorf("cannot create target directory: %w", err)
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("cannot resolve target path: %w", err)
	}

	// Pre-count entries so the progress bar has an accurate denominator.
	total, countErr := countArchiveEntries(archivePath)
	if countErr != nil {
		r.log.Errorf("Pre-count failed (%v); progress bar disabled", countErr)
	} else {
		r.log.Infof("Archive entries: %d", total)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("cannot open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("not a valid gzip archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	var extracted, skipped int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("archive read error (corrupted?): %w", err)
		}

		// ── SECURITY CHECK ───────────────────────────────────────────────────
		safeDest, err := security.SafePath(absTarget, hdr.Name)
		if err != nil {
			r.log.Errorf("SECURITY SKIP %q: %v", hdr.Name, err)
			skipped++
			continue
		}
		// ─────────────────────────────────────────────────────────────────────

		if err := r.extractEntry(tr, hdr, safeDest); err != nil {
			r.log.Errorf("Extract error %q: %v (skipping)", hdr.Name, err)
			skipped++
			continue
		}

		extracted++
		if total > 0 {
			r.log.Progress(extracted, total, "Restore")
		} else {
			r.log.Infof("Extracted: %s", hdr.Name)
		}
	}

	r.log.Infof("Restore done  extracted=%d  skipped=%d  target=%s",
		extracted, skipped, targetPath)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Extraction helpers
// ─────────────────────────────────────────────────────────────────────────────

// extractEntry dispatches to the appropriate handler for each tar entry type.
func (r *Restore) extractEntry(tr *tar.Reader, hdr *tar.Header, dest string) error {
	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(dest, hdr.FileInfo().Mode().Perm())

	case tar.TypeReg, tar.TypeRegA:
		return r.writeRegularFile(tr, hdr, dest)

	case tar.TypeSymlink:
		// Remove a stale symlink that may already exist at dest.
		_ = os.Remove(dest)
		return os.Symlink(hdr.Linkname, dest)

	case tar.TypeLink:
		_ = os.Remove(dest)
		return os.Link(hdr.Linkname, dest)

	default:
		r.log.Debugf("Unsupported entry type %d (%s) — skipped", hdr.Typeflag, hdr.Name)
		return nil
	}
}

// writeRegularFile creates a file at dest and streams content from tr into it.
// Extraction is limited to maxFileSize bytes to prevent decompression bombs.
func (r *Restore) writeRegularFile(tr *tar.Reader, hdr *tar.Header, dest string) error {
	// Ensure all parent directories exist.
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create parent dirs: %w", err)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	limited := io.LimitReader(tr, maxFileSize)
	if _, err := io.Copy(f, limited); err != nil {
		// Remove the partially-written file on error.
		_ = os.Remove(dest)
		return fmt.Errorf("write content: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Archive inspection helpers
// ─────────────────────────────────────────────────────────────────────────────

// countArchiveEntries opens the archive and counts entries for the progress bar.
// It is a second pass over the file — acceptable for UX improvement.
func countArchiveEntries(archivePath string) (int64, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var n int64
	for {
		if _, err := tr.Next(); err == io.EOF {
			return n, nil
		} else if err != nil {
			return n, err
		}
		n++
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Formatting helpers
// ─────────────────────────────────────────────────────────────────────────────

func typeLabel(flag byte) string {
	switch flag {
	case tar.TypeDir:
		return "dir"
	case tar.TypeSymlink:
		return "symlink"
	case tar.TypeLink:
		return "hardlink"
	default:
		return "file"
	}
}

func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
