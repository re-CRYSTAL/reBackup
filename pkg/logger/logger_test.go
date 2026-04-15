package logger_test

import (
	"testing"

	"rebackup/pkg/logger"
)

func TestNew(t *testing.T) {
	l := logger.New()
	if l == nil {
		t.Fatal("New() returned nil")
	}
}

func TestLogger_InfoDoesNotPanic(t *testing.T) {
	l := logger.New()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Info panicked: %v", r)
		}
	}()
	l.Info("test message")
	l.Infof("formatted %s", "message")
}

func TestLogger_ErrorDoesNotPanic(t *testing.T) {
	l := logger.New()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Error panicked: %v", r)
		}
	}()
	l.Error("error message")
	l.Errorf("formatted error %d", 42)
}

func TestLogger_ProgressDoesNotPanic(t *testing.T) {
	l := logger.New()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Progress panicked: %v", r)
		}
	}()
	// Boundary values
	l.Progress(0, 0, "Test")    // zero total — must be a no-op
	l.Progress(0, 100, "Test")  // start
	l.Progress(50, 100, "Test") // midpoint
	l.Progress(100, 100, "Test") // completion → should print newline
	l.Progress(200, 100, "Test") // overflow — should clamp to 100 %
}
