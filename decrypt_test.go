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
