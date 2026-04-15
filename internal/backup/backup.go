// Package backup creates compressed .tar.gz archives from a source directory.
package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rebackup/pkg/logger"
)

// TelegramMaxBytes is the Telegram Bot API document upload limit (~50 MB).
const TelegramMaxBytes = 50 * 1024 * 1024

// Backup handles archive creation.
type Backup struct {
	log *logger.Logger
}

// New returns a Backup instance backed by the given logger.
func New(log *logger.Logger) *Backup {
	return &Backup{log: log}
}

// ArchiveName returns a timestamped archive filename.
//
//	backup_2024-01-15_10-30.tar.gz
func ArchiveName() string {
	return fmt.Sprintf("backup_%s.tar.gz", time.Now().Format("2006-01-02_15-04"))
}

// ─────────────────────────────────────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────────────────────────────────────

// Create builds a .tar.gz archive of sourcePath inside outputDir.
// The archive name includes a timestamp; the full path is returned.
//
// Graceful error handling:
//   - Inaccessible files are logged and skipped (walk continues).
//   - A partial archive is deleted when a fatal error occurs.
func (b *Backup) Create(sourcePath, outputDir string) (string, error) {
	sourcePath = filepath.Clean(sourcePath)
	b.log.Infof("Backup start  source=%q  output=%q", sourcePath, outputDir)

	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("source inaccessible: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create output directory: %w", err)
	}

	// Pre-scan so the progress bar has an accurate total.
	totalEntries, totalBytes, _ := scanDirectory(sourcePath)
	if totalEntries > 0 {
		b.log.Infof("Pre-scan: %d entries | %s uncompressed", totalEntries, FormatSize(totalBytes))
	}

	archivePath := filepath.Join(outputDir, ArchiveName())
	b.log.Infof("Creating archive: %s", archivePath)

	outFile, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("cannot create archive file: %w", err)
	}

	// On any error path, remove the partial archive.
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(archivePath)
		}
	}()

	gzW := gzip.NewWriter(outFile)
	twW := tar.NewWriter(gzW)

	// Paths inside the archive are relative to the parent of sourcePath so that
	// the archive contains e.g.  data/file.txt  rather than  /home/user/data/file.txt
	baseDir := filepath.Dir(sourcePath)

	var doneEntries, doneBytes int64

	walkErr := filepath.Walk(sourcePath, func(path string, _ os.FileInfo, walkErr error) error {
		if walkErr != nil {
			b.log.Errorf("Skip inaccessible: %s (%v)", path, walkErr)
			return nil // best-effort: continue
		}

		// Always use Lstat so symlinks are represented correctly.
		linfo, err := os.Lstat(path)
		if err != nil {
			b.log.Errorf("Lstat failed: %s (%v)", path, err)
			return nil
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			b.log.Errorf("Rel-path failed: %s (%v)", path, err)
			return nil
		}

		// Read symlink target when needed.
		linkTarget := ""
		if linfo.Mode()&os.ModeSymlink != 0 {
			if linkTarget, err = os.Readlink(path); err != nil {
				b.log.Errorf("Readlink failed: %s (%v)", path, err)
				return nil
			}
		}

		// Build tar header.
		hdr, err := tar.FileInfoHeader(linfo, linkTarget)
		if err != nil {
			b.log.Errorf("Header build failed: %s (%v)", path, err)
			return nil
		}
		hdr.Name = relPath
		// Directories must end with "/" in a POSIX tar.
		if linfo.IsDir() && !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/"
		}

		if err := twW.WriteHeader(hdr); err != nil {
			// Fatal: the archive is in an inconsistent state.
			return fmt.Errorf("write tar header %s: %w", relPath, err)
		}

		// Copy file content (skip dirs, symlinks, etc.).
		if linfo.Mode().IsRegular() {
			n, err := copyFileContent(path, twW)
			if err != nil {
				b.log.Errorf("Copy failed: %s (%v) — skipping content", path, err)
				// Content skip: header already written, file will appear empty.
			} else {
				doneBytes += n
			}
		}

		doneEntries++
		if totalEntries > 0 {
			b.log.Progress(doneEntries, totalEntries, "Backup")
		}
		return nil
	})

	if walkErr != nil {
		// Close writers before returning so deferred outFile.Close is clean.
		_ = twW.Close()
		_ = gzW.Close()
		_ = outFile.Close()
		return "", fmt.Errorf("directory walk: %w", walkErr)
	}

	// Close writers in order: tar → gzip → file.
	if err := twW.Close(); err != nil {
		_ = gzW.Close()
		_ = outFile.Close()
		return "", fmt.Errorf("tar finalization: %w", err)
	}
	if err := gzW.Close(); err != nil {
		_ = outFile.Close()
		return "", fmt.Errorf("gzip finalization: %w", err)
	}
	if err := outFile.Close(); err != nil {
		return "", fmt.Errorf("archive file close: %w", err)
	}

	fi, _ := os.Stat(archivePath)
	var compressedSize int64
	if fi != nil {
		compressedSize = fi.Size()
	}
	b.log.Infof("Backup done   archive=%s  compressed=%s  original=%s",
		archivePath, FormatSize(compressedSize), FormatSize(doneBytes))

	ok = true
	return archivePath, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Telegram
// ─────────────────────────────────────────────────────────────────────────────

// SendToTelegram uploads the archive as a document to a Telegram chat.
//
// Required env vars:
//
//	TELEGRAM_TOKEN   – bot token from @BotFather
//	TELEGRAM_CHAT_ID – numeric chat / channel ID
func (b *Backup) SendToTelegram(archivePath string) error {
	token := os.Getenv("TELEGRAM_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return fmt.Errorf("TELEGRAM_TOKEN and TELEGRAM_CHAT_ID must be set")
	}

	fi, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("cannot stat archive: %w", err)
	}
	if fi.Size() > TelegramMaxBytes {
		return fmt.Errorf("archive is too large for Telegram: %s (limit 50 MiB)",
			FormatSize(fi.Size()))
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("cannot open archive: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("chat_id", chatID)

	part, err := mw.CreateFormFile("document", filepath.Base(archivePath))
	if err != nil {
		return fmt.Errorf("multipart build: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("multipart copy: %w", err)
	}
	mw.Close()

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", token)
	resp, err := http.Post(apiURL, mw.FormDataContentType(), &body) //nolint:gosec
	if err != nil {
		return fmt.Errorf("HTTP request to Telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Telegram API returned %d: %s", resp.StatusCode, string(raw))
	}

	b.log.Infof("Archive uploaded to Telegram (chat_id=%s)", chatID)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// copyFileContent opens path and streams its bytes into dst.
func copyFileContent(path string, dst io.Writer) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(dst, f)
}

// scanDirectory counts entries and sums sizes for progress tracking.
func scanDirectory(root string) (entries, size int64, err error) {
	err = filepath.Walk(root, func(_ string, info os.FileInfo, werr error) error {
		if werr != nil {
			return nil
		}
		entries++
		if info.Mode().IsRegular() {
			size += info.Size()
		}
		return nil
	})
	return
}

// FormatSize converts bytes to a human-readable IEC string (KiB, MiB, …).
func FormatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
