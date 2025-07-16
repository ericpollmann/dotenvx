package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"io"
	"sync"
	
	"github.com/ericpollmann/dotenvx"
)

// reset clears cached state for testing
func reset() {
	dotenvx.Mu.Lock()
	defer dotenvx.Mu.Unlock()
	dotenvx.Once = sync.Once{}
	dotenvx.EnvMap = nil
}

func TestMainWithEnv(t *testing.T) {
	// Reset cache before test
	reset()
	
	// Save original working directory and stdout
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	oldStdout := os.Stdout

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "dotenvx_test_direct")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Change to temp directory
	os.Chdir(tempDir)

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

	// Capture stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run main function
	main()

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

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

func TestAMainNoEnv(t *testing.T) {
	// Reset cache before test
	reset()
	
	// Save original working directory and stdout
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	oldStdout := os.Stdout

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "dotenvx_test_empty")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Change to temp directory (no .env files)
	os.Chdir(tempDir)

	// Unset environment variables
	os.Unsetenv("DOTENV_PRIVATE_KEY")
	os.Unsetenv("DOTENV_PRIVATE_KEY_PRODUCTION")

	// Capture stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run main function
	main()

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected empty output, got: %s", output)
	}
}