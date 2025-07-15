package dotenvx

import (
	"os"
	"testing"
)

func TestDevelopment(t *testing.T) {
	os.Unsetenv("DOTENV_PRIVATE_KEY_PRODUCTION")
	os.Setenv("DOTENV_PRIVATE_KEY", "2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d")
	reset()

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
	os.Setenv("DOTENV_PRIVATE_KEY_PRODUCTION", "7d797417f477635f8753c5325d5a68552ab7048f46c518be7f0ae3bc245d3ab8")
	reset()

	if got := Getenv("GREETING"); got != "world" {
		t.Errorf("prod: got %q, want world", got)
	}
}

func TestNoKeys(t *testing.T) {
	os.Unsetenv("DOTENV_PRIVATE_KEY")
	os.Unsetenv("DOTENV_PRIVATE_KEY_PRODUCTION")
	reset()

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
