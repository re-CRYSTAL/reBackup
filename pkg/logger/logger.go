// Package logger provides a lightweight structured logger with level prefixes
// and an ASCII progress bar rendered via carriage-return overwrites.
package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Logger wraps the standard log package with INFO / ERROR / DEBUG levels.
type Logger struct {
	info  *log.Logger
	error *log.Logger
	debug *log.Logger
}

// New returns a ready-to-use Logger that writes to stdout (INFO/DEBUG) and
// stderr (ERROR).
func New() *Logger {
	flags := log.LstdFlags
	return &Logger{
		info:  log.New(os.Stdout, "[INFO]  ", flags),
		error: log.New(os.Stderr, "[ERROR] ", flags),
		debug: log.New(os.Stdout, "[DEBUG] ", flags),
	}
}

// ── Info ────────────────────────────────────────────────────────────────────

func (l *Logger) Info(msg string) {
	l.info.Println(msg)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.info.Printf(format+"\n", args...)
}

// ── Error ───────────────────────────────────────────────────────────────────

func (l *Logger) Error(msg string) {
	l.error.Println(msg)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.error.Printf(format+"\n", args...)
}

// ── Debug ───────────────────────────────────────────────────────────────────

func (l *Logger) Debug(msg string) {
	l.debug.Println(msg)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.debug.Printf(format+"\n", args...)
}

// ── Progress bar ─────────────────────────────────────────────────────────────

const barWidth = 40

// Progress renders a single-line ASCII progress bar that overwrites itself
// on each call. A newline is printed once current >= total.
//
//	Backup   [████████████████████░░░░░░░░░░░░░░░░░░░░]  50.0% (50/100)
func (l *Logger) Progress(current, total int64, label string) {
	if total <= 0 {
		return
	}

	pct := float64(current) / float64(total) * 100.0
	if pct > 100.0 {
		pct = 100.0
	}

	filled := int(pct / 100.0 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("\r%-10s [%s] %5.1f%% (%d/%d)", label, bar, pct, current, total)

	if current >= total {
		fmt.Println()
	}
}
