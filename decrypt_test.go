package dotenvx

import (
	"os"
	"runtime"
	"strings"
	"testing"

	ecies "github.com/ecies/go/v2"
)

func init() {
	Debug = true
}

// reset sets up environment variables for testing
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

	if len(Environ()) != 0 {
		t.Error("no keys: expected empty environ")
	}

	if got := Getenv("ANYTHING"); got != "" {
		t.Errorf("no keys: expected empty, got %q", got)
	}
}

func inTempDir(t *testing.T) string {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	return originalDir
}

func TestFileNotFound(t *testing.T) {
	defer os.Chdir(inTempDir(t))
	reset("dev")

	if len(Environ()) != 0 {
		t.Error("file not found: expected empty environ")
	}

	if got := Getenv("ANYTHING"); got != "" {
		t.Errorf("file not found: expected empty, got %q", got)
	}
}

func TestQuotedValues(t *testing.T) {
	defer os.Chdir(inTempDir(t))

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
	defer os.Chdir(inTempDir(t))

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
	defer os.Chdir(inTempDir(t))

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
	defer os.Chdir(inTempDir(t))

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
	defer os.Chdir(inTempDir(t))

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

func TestEnvironDebugCoverage(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	// Test 1: No keys - should trigger debug in Environ
	reset("")
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "DOTENV_PRIVATE_KEY") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) > 0 {
				os.Unsetenv(parts[0])
			}
		}
	}

	// Should log "Error finding envFile" and return empty
	if got := Environ(); len(got) != 0 {
		t.Errorf("Expected empty environ with no keys, got %d items", len(got))
	}

	// Test 2: Valid key but no file - should trigger getEnvVars debug
	reset("")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	// No .env file exists, should log "File not found: .env" and "Error retrieving all values"
	if got := Environ(); len(got) != 0 {
		t.Errorf("Expected empty environ when file not found, got %d items", len(got))
	}

	// Test 3: Invalid key format in getEnvVars
	if err := os.WriteFile(".env", []byte(`TEST=encrypted:invalid`), 0644); err != nil {
		t.Fatal(err)
	}

	// Should still work for non-encrypted values but log about invalid format
	if err := os.WriteFile(".env", []byte(`PLAIN=value`), 0644); err != nil {
		t.Fatal(err)
	}

	env := Environ()
	found := false
	for _, e := range env {
		if e == "PLAIN=value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find PLAIN=value in environ")
	}
}

func TestGetEnvVarsDebugCoverage(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	// Test 1: File not found debug logging in getEnvVars
	reset("")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	// Direct call to trigger specific debug paths
	privateKey, _ := ecies.NewPrivateKeyFromHex("2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	envFile := EnvFile{Path: ".env.missing", Key: privateKey}
	vars, err := getEnvVars(&envFile, "")
	if err == nil || len(vars) != 0 {
		t.Error("Expected error and empty vars for missing file")
	}

	// Test 2: Invalid key format debug logging
	if err := os.WriteFile(".env", []byte(`TEST=value`), 0644); err != nil {
		t.Fatal(err)
	}

	// For invalid key test, the file should still be readable but won't decrypt
	envFile = EnvFile{Path: ".env", Key: nil}
	vars, err = getEnvVars(&envFile, "")
	// With nil key, non-encrypted values can still be read
	if err != nil {
		t.Errorf("Expected no error for reading plain values, got: %v", err)
	}
	// Should have read TEST=value
	if len(vars) != 1 || vars[0].Name != "TEST" || vars[0].Value != "value" {
		t.Errorf("Expected to read TEST=value, got: %+v", vars)
	}

	// Test 3: Permission error debug logging
	if runtime.GOOS != "windows" {
		if err := os.Chmod(".env", 0000); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(".env", 0644)

		privateKey2, _ := ecies.NewPrivateKeyFromHex("2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
		envFile = EnvFile{Path: ".env", Key: privateKey2}
		vars, err = getEnvVars(&envFile, "")
		if err == nil || len(vars) != 0 {
			t.Error("Expected error and empty vars for permission denied")
		}
	}
}

func TestEnvironGetEnvVarsError(t *testing.T) {
	defer os.Chdir(inTempDir(t))

	// Create a file with bad permissions
	if err := os.WriteFile(".env", []byte(`TEST=value`), 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(".env", 0644)

	reset("")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	defer os.Unsetenv("DOTENV_PRIVATE_KEY")

	// Should trigger debug logging and error path in Environ
	if runtime.GOOS != "windows" {
		result := Environ()
		if len(result) != 0 {
			t.Errorf("Expected empty environ when file can't be read, got %d items", len(result))
		}
	}
}
