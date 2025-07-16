package dotenvx

import (
	"os"
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
