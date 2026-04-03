package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ConfigDir  = filepath.Join(homeDir(), ".obsidian-cli")
	ConfigFile = filepath.Join(ConfigDir, "config.yaml")
)

// AppConfig mirrors the YAML configuration schema exactly.
type AppConfig struct {
	Vault        VaultConfig        `yaml:"vault"`
	Logging      LoggingConfig      `yaml:"logging"`
	DailyNotes   DailyNotesConfig   `yaml:"dailyNotes"`
	Interface    InterfaceConfig    `yaml:"interface"`
	Organization OrganizationConfig `yaml:"organization"`
	Advanced     AdvancedConfig     `yaml:"advanced"`
}

type VaultConfig struct {
	DefaultPath string `yaml:"defaultPath"`
	RootPath    string `yaml:"rootPath"` // Root for file browser; defaults to defaultPath if empty
}

type LoggingConfig struct {
	Tasks           LogFileConfig `yaml:"tasks"`
	Ideas           LogFileConfig `yaml:"ideas"`
	Questions       LogFileConfig `yaml:"questions"`
	Insights        LogFileConfig `yaml:"insights"`
	TimestampFormat string        `yaml:"timestampFormat"`
}

type LogFileConfig struct {
	LogFile string `yaml:"logFile"`
	AutoLog bool   `yaml:"autoLog"`
}

type DailyNotesConfig struct {
	Sections    []string `yaml:"sections"`
	Tags        []string `yaml:"tags"`
	TitleFormat string   `yaml:"titleFormat"`
}

type InterfaceConfig struct {
	Theme           ThemeConfig       `yaml:"theme"`
	AutoScroll      bool              `yaml:"autoScroll"`
	ShowLineNumbers bool              `yaml:"showLineNumbers"`
	EisenhowerTags  map[string]string `yaml:"eisenhowerTags"`
}

type ThemeConfig struct {
	Border    string `yaml:"border"`
	Title     string `yaml:"title"`
	Content   string `yaml:"content"`
	Input     string `yaml:"input"`
	Highlight string `yaml:"highlight"`
}

type OrganizationConfig struct {
	SectionPrefixes map[string]string `yaml:"sectionPrefixes"`
}

type AdvancedConfig struct {
	Backup      BackupConfig      `yaml:"backup"`
	Performance PerformanceConfig `yaml:"performance"`
}

type BackupConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Directory  string `yaml:"directory"`
	MaxBackups int    `yaml:"maxBackups"`
}

type PerformanceConfig struct {
	MaxFileSize int  `yaml:"maxFileSize"`
	WatchFiles  bool `yaml:"watchFiles"`
}

// DefaultConfig returns the hardcoded default configuration.
func DefaultConfig() AppConfig {
	return AppConfig{
		Vault: VaultConfig{
			DefaultPath: "",
		},
		Logging: LoggingConfig{
			Tasks:           LogFileConfig{LogFile: "tasks-log.md", AutoLog: true},
			Ideas:           LogFileConfig{LogFile: "ideas-log.md", AutoLog: true},
			Questions:       LogFileConfig{LogFile: "questions-log.md", AutoLog: true},
			Insights:        LogFileConfig{LogFile: "insights-log.md", AutoLog: true},
			TimestampFormat: "simple",
		},
		DailyNotes: DailyNotesConfig{
			Sections:    []string{"Daily Log", "Tasks", "Ideas", "Questions", "Insights", "Links to Expand"},
			Tags:        []string{"#daily", "#inbox"},
			TitleFormat: "YYYY-MM-DD",
		},
		Interface: InterfaceConfig{
			Theme: ThemeConfig{
				Border:    "cyan",
				Title:     "white",
				Content:   "white",
				Input:     "yellow",
				Highlight: "green",
			},
			AutoScroll:      true,
			ShowLineNumbers: true,
			EisenhowerTags: map[string]string{
				"#do":        "131",
				"#delegate":  "180",
				"#schedule":  "66",
				"#eliminate": "145",
			},
		},
		Organization: OrganizationConfig{
			SectionPrefixes: map[string]string{
				"[]": "Tasks",
				"-":  "Ideas",
				"?":  "Questions",
				"!":  "Insights",
			},
		},
		Advanced: AdvancedConfig{
			Backup: BackupConfig{
				Enabled:    false,
				Directory:  ".obsidian-cli-backups",
				MaxBackups: 5,
			},
			Performance: PerformanceConfig{
				MaxFileSize: 10,
				WatchFiles:  false,
			},
		},
	}
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() error {
	return os.MkdirAll(ConfigDir, 0o755)
}

// Load reads the config from disk and merges with defaults.
func Load() (AppConfig, error) {
	cfg := DefaultConfig()

	if err := EnsureConfigDir(); err != nil {
		return cfg, fmt.Errorf("creating config dir: %w", err)
	}

	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	var userCfg AppConfig
	if err := yaml.Unmarshal(data, &userCfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	// Merge user config over defaults
	merged := mergeConfig(cfg, userCfg)

	// Expand ~ and $HOME in paths
	merged.Vault.DefaultPath = expandPath(merged.Vault.DefaultPath)
	merged.Vault.RootPath = expandPath(merged.Vault.RootPath)

	return merged, nil
}

// Save writes the config to disk as YAML.
func Save(cfg AppConfig) error {
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(ConfigFile, data, 0o644)
}

// GetVaultPath returns the configured vault path.
func GetVaultPath() (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	return cfg.Vault.DefaultPath, nil
}

// GetVaultRootPath returns the vault root path for file browsing.
// Falls back to defaultPath if rootPath is not set.
func GetVaultRootPath() (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	if cfg.Vault.RootPath != "" {
		return cfg.Vault.RootPath, nil
	}
	return cfg.Vault.DefaultPath, nil
}

// GetLogFile returns the log filename for a given log type (task, idea, question, insight).
func GetLogFile(logType string) (string, error) {
	defaults := map[string]string{
		"task":     "tasks-log.md",
		"idea":     "ideas-log.md",
		"question": "questions-log.md",
		"insight":  "insights-log.md",
	}

	cfg, err := Load()
	if err != nil {
		return defaults[logType], err
	}

	files := map[string]string{
		"task":     cfg.Logging.Tasks.LogFile,
		"idea":     cfg.Logging.Ideas.LogFile,
		"question": cfg.Logging.Questions.LogFile,
		"insight":  cfg.Logging.Insights.LogFile,
	}

	if f := files[logType]; f != "" {
		return f, nil
	}
	return defaults[logType], nil
}

// GetTaskLogFile returns the task log filename. Convenience wrapper around GetLogFile.
func GetTaskLogFile() (string, error) {
	return GetLogFile("task")
}

// GetEisenhowerTags returns the Eisenhower tag color map.
func GetEisenhowerTags() (map[string]string, error) {
	cfg, err := Load()
	if err != nil {
		return DefaultConfig().Interface.EisenhowerTags, err
	}
	if len(cfg.Interface.EisenhowerTags) == 0 {
		return DefaultConfig().Interface.EisenhowerTags, nil
	}
	return cfg.Interface.EisenhowerTags, nil
}

// mergeConfig merges user config over defaults.
// Non-zero user values override defaults.
func mergeConfig(defaults, user AppConfig) AppConfig {
	result := defaults

	if user.Vault.DefaultPath != "" {
		result.Vault.DefaultPath = user.Vault.DefaultPath
	}
	if user.Vault.RootPath != "" {
		result.Vault.RootPath = user.Vault.RootPath
	}

	// Logging
	if user.Logging.Tasks.LogFile != "" {
		result.Logging.Tasks.LogFile = user.Logging.Tasks.LogFile
	}
	if user.Logging.Ideas.LogFile != "" {
		result.Logging.Ideas.LogFile = user.Logging.Ideas.LogFile
	}
	if user.Logging.Questions.LogFile != "" {
		result.Logging.Questions.LogFile = user.Logging.Questions.LogFile
	}
	if user.Logging.Insights.LogFile != "" {
		result.Logging.Insights.LogFile = user.Logging.Insights.LogFile
	}
	if user.Logging.TimestampFormat != "" {
		result.Logging.TimestampFormat = user.Logging.TimestampFormat
	}

	// Daily notes
	if len(user.DailyNotes.Sections) > 0 {
		result.DailyNotes.Sections = user.DailyNotes.Sections
	}
	if len(user.DailyNotes.Tags) > 0 {
		result.DailyNotes.Tags = user.DailyNotes.Tags
	}
	if user.DailyNotes.TitleFormat != "" {
		result.DailyNotes.TitleFormat = user.DailyNotes.TitleFormat
	}

	// Interface
	if user.Interface.Theme.Border != "" {
		result.Interface.Theme.Border = user.Interface.Theme.Border
	}
	if user.Interface.Theme.Title != "" {
		result.Interface.Theme.Title = user.Interface.Theme.Title
	}
	if user.Interface.Theme.Content != "" {
		result.Interface.Theme.Content = user.Interface.Theme.Content
	}
	if user.Interface.Theme.Input != "" {
		result.Interface.Theme.Input = user.Interface.Theme.Input
	}
	if user.Interface.Theme.Highlight != "" {
		result.Interface.Theme.Highlight = user.Interface.Theme.Highlight
	}
	if len(user.Interface.EisenhowerTags) > 0 {
		result.Interface.EisenhowerTags = user.Interface.EisenhowerTags
	}

	// Organization
	if len(user.Organization.SectionPrefixes) > 0 {
		result.Organization.SectionPrefixes = user.Organization.SectionPrefixes
	}

	// Advanced
	if user.Advanced.Backup.Directory != "" {
		result.Advanced.Backup.Directory = user.Advanced.Backup.Directory
	}
	if user.Advanced.Backup.MaxBackups != 0 {
		result.Advanced.Backup.MaxBackups = user.Advanced.Backup.MaxBackups
	}
	if user.Advanced.Performance.MaxFileSize != 0 {
		result.Advanced.Performance.MaxFileSize = user.Advanced.Performance.MaxFileSize
	}

	return result
}

func expandPath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir(), path[2:])
	}
	if path == "~" {
		return homeDir()
	}
	return os.ExpandEnv(path)
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}
