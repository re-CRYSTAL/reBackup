// Package security provides helpers that guard against common archive-extraction
// vulnerabilities, in particular path traversal (Zip Slip / tar slip).
package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SafePath resolves filePath relative to targetDir and verifies that the
// resulting absolute path is still contained within targetDir.
//
// Attack scenarios that are blocked:
//
//	../../etc/passwd              → escapes via parent traversal
//	/etc/passwd                   → absolute path inside archive
//	subdir/../../etc/passwd       → nested traversal
//	../targetDirSuffix/evil.txt   → prefix-match bypass
//
// Returns the safe absolute destination path, or an error.
func SafePath(targetDir, filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("archive entry has empty name")
	}

	// Block absolute paths immediately — filepath.Join("/target", "/etc/passwd")
	// produces "/target/etc/passwd" in Go, which would bypass the check below,
	// so we must reject absolute entries before joining.
	if filepath.IsAbs(filePath) {
		return "", fmt.Errorf(
			"path traversal detected: entry %q is an absolute path",
			filePath,
		)
	}

	// Resolve targetDir to an absolute, cleaned path.
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve target directory %q: %w", targetDir, err)
	}

	// Build the canonical destination path.
	// filepath.Join + filepath.Clean resolves all ".." segments.
	candidate := filepath.Clean(filepath.Join(absTarget, filePath))

	// Ensure the candidate is inside targetDir.
	// We append a separator to absTarget so that a directory named
	// "/tmp/restore" does not accidentally match "/tmp/restoreOther/...".
	separator := string(filepath.Separator)
	guardedTarget := absTarget + separator

	if candidate != absTarget && !strings.HasPrefix(candidate+separator, guardedTarget) {
		return "", fmt.Errorf(
			"path traversal detected: entry %q would be written outside %q",
			filePath, targetDir,
		)
	}

	return candidate, nil
}

// ValidateArchivePath performs a quick sanity-check on an archive path
// before any file operations are attempted.
func ValidateArchivePath(path string) error {
	if path == "" {
		return fmt.Errorf("archive path is empty")
	}

	clean := filepath.Clean(path)

	// Reject relative paths that start with ".." (e.g. "../sneaky.tar.gz").
	if !filepath.IsAbs(clean) {
		parts := strings.Split(clean, string(filepath.Separator))
		if len(parts) > 0 && parts[0] == ".." {
			return fmt.Errorf("invalid archive path %q: starts outside working directory", path)
		}
	}

	return nil
}
