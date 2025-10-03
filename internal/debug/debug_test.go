package debug

import (
	"bytes"
	"os"
	"testing"
)

func TestSetVerbose(t *testing.T) {
	// Test setting verbose to true
	SetVerbose(true)
	if !IsVerbose() {
		t.Error("Expected verbose to be true")
	}

	// Test setting verbose to false
	SetVerbose(false)
	if IsVerbose() {
		t.Error("Expected verbose to be false")
	}
}

func TestPrintfVerboseMode(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with verbose enabled
	SetVerbose(true)
	Printf("test message: %s", "hello")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "test message: hello" {
		t.Errorf("Expected 'test message: hello', got '%s'", output)
	}
}

func TestPrintfSilentMode(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with verbose disabled
	SetVerbose(false)
	Printf("test message: %s", "hello")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("Expected empty output, got '%s'", output)
	}
}
