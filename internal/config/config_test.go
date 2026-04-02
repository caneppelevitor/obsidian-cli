package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Logging.Tasks.LogFile != "tasks-log.md" {
		t.Errorf("Tasks logFile = %q, want %q", cfg.Logging.Tasks.LogFile, "tasks-log.md")
	}
	if cfg.Logging.Ideas.LogFile != "ideas-log.md" {
		t.Errorf("Ideas logFile = %q, want %q", cfg.Logging.Ideas.LogFile, "ideas-log.md")
	}
	if cfg.Logging.Questions.LogFile != "questions-log.md" {
		t.Errorf("Questions logFile = %q, want %q", cfg.Logging.Questions.LogFile, "questions-log.md")
	}
	if cfg.Logging.Insights.LogFile != "insights-log.md" {
		t.Errorf("Insights logFile = %q, want %q", cfg.Logging.Insights.LogFile, "insights-log.md")
	}
	if !cfg.Interface.AutoScroll {
		t.Error("AutoScroll should be true by default")
	}
	if !cfg.Interface.ShowLineNumbers {
		t.Error("ShowLineNumbers should be true by default")
	}
}

func TestEisenhowerTagDefaults(t *testing.T) {
	cfg := DefaultConfig()
	tags := cfg.Interface.EisenhowerTags

	expected := map[string]string{
		"#do":        "131",
		"#delegate":  "180",
		"#schedule":  "66",
		"#eliminate": "145",
	}

	for k, v := range expected {
		if tags[k] != v {
			t.Errorf("EisenhowerTags[%q] = %q, want %q", k, tags[k], v)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp directory for config
	tmpDir := t.TempDir()
	origConfigDir := ConfigDir
	origConfigFile := ConfigFile
	ConfigDir = tmpDir
	ConfigFile = filepath.Join(tmpDir, "config.yaml")
	defer func() {
		ConfigDir = origConfigDir
		ConfigFile = origConfigFile
	}()

	cfg := DefaultConfig()
	cfg.Vault.DefaultPath = "/test/vault"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Vault.DefaultPath != "/test/vault" {
		t.Errorf("Vault path = %q, want %q", loaded.Vault.DefaultPath, "/test/vault")
	}
}

func TestMergeConfig(t *testing.T) {
	defaults := DefaultConfig()
	user := AppConfig{
		Vault: VaultConfig{DefaultPath: "/my/vault"},
		Logging: LoggingConfig{
			Tasks: LogFileConfig{LogFile: "custom-tasks.md"},
		},
	}

	merged := mergeConfig(defaults, user)

	if merged.Vault.DefaultPath != "/my/vault" {
		t.Errorf("Vault path = %q, want %q", merged.Vault.DefaultPath, "/my/vault")
	}
	if merged.Logging.Tasks.LogFile != "custom-tasks.md" {
		t.Errorf("Tasks logFile = %q, want %q", merged.Logging.Tasks.LogFile, "custom-tasks.md")
	}
	// Other defaults should be preserved
	if merged.Logging.Ideas.LogFile != "ideas-log.md" {
		t.Errorf("Ideas logFile = %q, want %q", merged.Logging.Ideas.LogFile, "ideas-log.md")
	}
	if !merged.Interface.AutoScroll {
		t.Error("AutoScroll should still be true after merge")
	}
}

func TestLoadFromPartialYAML(t *testing.T) {
	tmpDir := t.TempDir()
	origConfigDir := ConfigDir
	origConfigFile := ConfigFile
	ConfigDir = tmpDir
	ConfigFile = filepath.Join(tmpDir, "config.yaml")
	defer func() {
		ConfigDir = origConfigDir
		ConfigFile = origConfigFile
	}()

	// Write a partial YAML config
	partial := map[string]interface{}{
		"vault": map[string]interface{}{
			"defaultPath": "/partial/vault",
		},
	}
	data, _ := yaml.Marshal(partial)
	os.WriteFile(ConfigFile, data, 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Vault.DefaultPath != "/partial/vault" {
		t.Errorf("Vault path = %q, want %q", cfg.Vault.DefaultPath, "/partial/vault")
	}
	// Defaults should fill in
	if cfg.Logging.Tasks.LogFile != "tasks-log.md" {
		t.Errorf("Tasks logFile = %q, want %q", cfg.Logging.Tasks.LogFile, "tasks-log.md")
	}
}
