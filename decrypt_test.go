package dotenvx

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// reset clears cached state for testing
func reset(env string) {
	if env == "dev" {
		os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	} else {
		os.Unsetenv("DOTENV_PRIVATE_KEY")
	}
	if env == "prod" {
		os.Setenv("DOTENV_PRIVATE_KEY_PRODUCTION", "7d797417f477635f8753c5325d5a68552ab7048f46c518be7f0ae3bc245d3ab8")
	} else {
		os.Unsetenv("DOTENV_PRIVATE_KEY_PRODUCTION")
	}
	Mu.Lock()
	defer Mu.Unlock()
	Once = sync.Once{}
	EnvMap = nil
}

func TestDevelopment(t *testing.T) {
	reset("dev")

	if got := Getenv("GREETING"); got != "hello" {
		t.Errorf("dev: got %q, want hello", got)
	}

	found := false
	for _, env := range Environ() {
		if env == "GREETING=hello" {
			found = true
			break
		}
	}
	if !found {
		t.Error("dev: GREETING=hello not found")
	}
}

func TestProduction(t *testing.T) {
	reset("prod")

	if got := Getenv("GREETING"); got != "world" {
		t.Errorf("prod: got %q, want world", got)
	}
}

func TestNoKeys(t *testing.T) {
	reset("")

	if len(GetenvMap()) != 0 {
		t.Error("no keys: expected empty map")
	}

	if len(Environ()) != 0 {
		t.Error("no keys: expected empty environ")
	}

	if got := Getenv("ANYTHING"); got != "" {
		t.Errorf("no keys: expected empty, got %q", got)
	}
}

func TestFileNotFound(t *testing.T) {
	reset("dev")

	// Change to a directory where .env doesn't exist
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir := "/tmp"
	os.Chdir(tempDir)
	reset("dev")

	if len(GetenvMap()) != 0 {
		t.Error("file not found: expected empty map")
	}

	if len(Environ()) != 0 {
		t.Error("file not found: expected empty environ")
	}

	if got := Getenv("ANYTHING"); got != "" {
		t.Errorf("file not found: expected empty, got %q", got)
	}
}

func TestQuotedValues(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "dotenvx_quoted_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create .env file with quoted values
	envContent := `# Test file with quoted values
DOTENV_PUBLIC_KEY="020c5f23e6e02f087af380212814755c22f3d742b218666642d1dec184b7c6ae69"
export GREETING="encrypted:BL8cvfR8496FAJV3dbdSZj/D6qlhOc3lAhuAB24AGp4WASPH8BBoe21T+T9jlO/M0GY03RZ94Etk7VPWIP21vh+YLGu0fWe2usFdTFs+/BnlsT8K8+V9Xte/yXA2NhrRxy3T7ygL"
PLAIN_DOUBLE="hello world"
PLAIN_SINGLE='single quoted'
PLAIN_UNQUOTED=unquoted
ENCRYPTED_UNQUOTED=encrypted:BL8cvfR8496FAJV3dbdSZj/D6qlhOc3lAhuAB24AGp4WASPH8BBoe21T+T9jlO/M0GY03RZ94Etk7VPWIP21vh+YLGu0fWe2usFdTFs+/BnlsT8K8+V9Xte/yXA2NhrRxy3T7ygL
`
	if err := os.WriteFile(".env", []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}

	reset("dev")

	// Test that all values are parsed correctly
	tests := []struct {
		key      string
		expected string
	}{
		{"DOTENV_PUBLIC_KEY", "020c5f23e6e02f087af380212814755c22f3d742b218666642d1dec184b7c6ae69"},
		{"GREETING", "hello"},
		{"PLAIN_DOUBLE", "hello world"},
		{"PLAIN_SINGLE", "single quoted"},
		{"PLAIN_UNQUOTED", "unquoted"},
		{"ENCRYPTED_UNQUOTED", "hello"},
	}

	for _, tt := range tests {
		if got := Getenv(tt.key); got != tt.expected {
			t.Errorf("Getenv(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestCustomEnvironments(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "dotenvx_custom_env_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create multiple env files
	files := map[string]string{
		".env.staging":   `ENVIRONMENT=staging`,
		".env.qa.test":   `ENVIRONMENT=qa_test`,
		".env.dev.local": `ENVIRONMENT=dev_local`,
	}

	for file, content := range files {
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		envVar   string
		expected string
		file     string
	}{
		{"DOTENV_PRIVATE_KEY_STAGING", "staging", ".env.staging"},
		{"DOTENV_PRIVATE_KEY_QA_TEST", "qa_test", ".env.qa.test"},
		{"DOTENV_PRIVATE_KEY_DEV_LOCAL", "dev_local", ".env.dev.local"},
	}

	for _, tt := range tests {
		// Reset and clear all DOTENV_PRIVATE_KEY* variables
		reset("")
		for _, env := range os.Environ() {
			if strings.HasPrefix(env, "DOTENV_PRIVATE_KEY") {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) > 0 {
					os.Unsetenv(parts[0])
				}
			}
		}

		// Set only the specific key we want to test
		os.Setenv(tt.envVar, "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
		defer os.Unsetenv(tt.envVar)

		if got := Getenv("ENVIRONMENT"); got != tt.expected {
			t.Errorf("With %s, Getenv(ENVIRONMENT) = %q, want %q", tt.envVar, got, tt.expected)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "dotenvx_edge_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Test empty env var value
	reset("")
	os.Setenv("DOTENV_PRIVATE_KEY_EMPTY", "")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY_EMPTY")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	// Create .env file
	if err := os.WriteFile(".env", []byte(`TEST=value`), 0644); err != nil {
		t.Fatal(err)
	}

	// Should skip empty key and use DOTENV_PRIVATE_KEY
	if got := Getenv("TEST"); got != "value" {
		t.Errorf("Expected to skip empty key, got %q", got)
	}

	// Test with debug mode to cover debug statements
	reset("")
	Debug = true
	defer func() { Debug = false }()

	// Set a key for a file that doesn't exist first
	os.Setenv("DOTENV_PRIVATE_KEY_MISSING", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY_MISSING")

	// Should try MISSING first, fail to find .env.missing, then use DOTENV_PRIVATE_KEY
	if got := Getenv("TEST"); got != "value" {
		t.Errorf("Expected to fallback to .env, got %q", got)
	}

	// Test debug output for missing key
	if got := Getenv("NONEXISTENT"); got != "" {
		t.Errorf("Expected empty for nonexistent key, got %q", got)
	}

	// Test with no keys at all
	reset("")
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "DOTENV_PRIVATE_KEY") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) > 0 {
				os.Unsetenv(parts[0])
			}
		}
	}

	// Should log "No key found" in debug mode
	if got := Getenv("ANYTHING"); got != "" {
		t.Errorf("Expected empty with no keys, got %q", got)
	}

	// Test empty value debug logging
	reset("")
	if err := os.WriteFile(".env", []byte(`EMPTY_VALUE=`), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")

	// Should log "Empty string value for $EMPTY_VALUE"
	if got := Getenv("EMPTY_VALUE"); got != "" {
		t.Errorf("Expected empty value, got %q", got)
	}
}

func TestInvalidKeyFormat(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "dotenvx_invalid_key_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Enable debug mode
	Debug = true
	defer func() { Debug = false }()

	// Create a file
	if err := os.WriteFile(".env", []byte(`TEST=value`), 0644); err != nil {
		t.Fatal(err)
	}

	// Set an invalid key (not valid hex)
	reset("")
	os.Setenv("DOTENV_PRIVATE_KEY", "invalid-key-format")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	// Should log "Invalid key format: DOTENV_PRIVATE_KEY"
	if got := Getenv("TEST"); got != "" {
		t.Errorf("Expected empty with invalid key, got %q", got)
	}

	// Test with multiple keys where first is invalid
	reset("")
	os.Setenv("DOTENV_PRIVATE_KEY_BAD", "not-a-hex-key")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY_BAD")

	// Create .env.bad file for the bad key
	if err := os.WriteFile(".env.bad", []byte(`BAD=value`), 0644); err != nil {
		t.Fatal(err)
	}

	// Should skip bad key and use good key
	if got := Getenv("TEST"); got != "value" {
		t.Errorf("Expected to skip invalid key and use valid one, got %q", got)
	}
}

func TestFilePermissionError(t *testing.T) {
	// Skip on Windows as permission handling is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "dotenvx_perm_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Enable debug mode
	Debug = true
	defer func() { Debug = false }()

	// Create a file with no read permissions
	if err := os.WriteFile(".env", []byte(`TEST=value`), 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(".env", 0644) // Restore permissions for cleanup

	// Set the key
	reset("")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	// Should log "Unable to open: .env"
	if got := Getenv("TEST"); got != "" {
		t.Errorf("Expected empty when file cannot be read, got %q", got)
	}
}
