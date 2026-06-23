package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvLoadsValuesWithoutOverridingExistingEnv(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "already-exported")

	envPath := filepath.Join(t.TempDir(), ".env")
	content := []byte(`
# Local openHunt settings
OLLAMA_MODEL="gemma4:e4b"
OPENHUNT_TEST_API_URL=http://localhost:11434 # local default
OPENHUNT_TEST_OUTPUT_DIR='Market-Insights'
`)
	if err := os.WriteFile(envPath, content, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := LoadDotEnv(envPath); err != nil {
		t.Fatalf("LoadDotEnv returned error: %v", err)
	}

	if got := os.Getenv("OLLAMA_MODEL"); got != "already-exported" {
		t.Fatalf("expected existing OLLAMA_MODEL to win, got %q", got)
	}
	if got := os.Getenv("OPENHUNT_TEST_API_URL"); got != "http://localhost:11434" {
		t.Fatalf("expected OPENHUNT_TEST_API_URL from .env, got %q", got)
	}
	if got := os.Getenv("OPENHUNT_TEST_OUTPUT_DIR"); got != "Market-Insights" {
		t.Fatalf("expected OPENHUNT_TEST_OUTPUT_DIR from .env, got %q", got)
	}
}

func TestLoadDotEnvIgnoresMissingFile(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), ".env")); err != nil {
		t.Fatalf("missing .env should not error: %v", err)
	}
}
