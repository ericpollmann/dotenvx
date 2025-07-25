package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func inTempDir(t *testing.T) string {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	return originalDir
}

func capturedStdout() (*os.File, *os.File, *os.File) {
	newStdin, newStdout, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = newStdout
	return newStdin, newStdout, oldStdout
}

func restoreStdout(newStdout, oldStdout *os.File) {
	newStdout.Close()
	os.Stdout = oldStdout
}

func captureStdout(fn func()) string {
	newStdin, newStdout, oldStdout := capturedStdout()
	fn()
	restoreStdout(newStdout, oldStdout)
	var buf bytes.Buffer
	io.Copy(&buf, newStdin)
	return buf.String()
}

func TestMainWithEnv(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	// Create test .env file with both quoted and unquoted values
	envContent := `DOTENV_PUBLIC_KEY="020c5f23e6e02f087af380212814755c22f3d742b218666642d1dec184b7c6ae69"
GREETING="encrypted:BL8cvfR8496FAJV3dbdSZj/D6qlhOc3lAhuAB24AGp4WASPH8BBoe21T+T9jlO/M0GY03RZ94Etk7VPWIP21vh+YLGu0fWe2usFdTFs+/BnlsT8K8+V9Xte/yXA2NhrRxy3T7ygL"
PLAIN_VALUE=hello
QUOTED_PLAIN="quoted value"
`
	if err := os.WriteFile(".env", []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Set environment variable
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	output := captureStdout(func() { main() })

	// Check that output contains expected patterns
	if !strings.Contains(output, "DOTENV_PUBLIC_KEY=") {
		t.Error("Expected DOTENV_PUBLIC_KEY in output")
	}
	if !strings.Contains(output, "GREETING=hello") {
		t.Error("Expected GREETING=hello in output")
	}
	if !strings.Contains(output, "PLAIN_VALUE=hello") {
		t.Error("Expected PLAIN_VALUE=hello in output")
	}
	if !strings.Contains(output, "QUOTED_PLAIN=quoted value") {
		t.Error("Expected QUOTED_PLAIN=quoted value in output")
	}
}

func TestMainNoEnv(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	os.Unsetenv("DOTENV_PRIVATE_KEY")
	os.Unsetenv("DOTENV_PRIVATE_KEY_PRODUCTION")
	output := captureStdout(func() { main() })

	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected empty output, got: %s", output)
	}
}
