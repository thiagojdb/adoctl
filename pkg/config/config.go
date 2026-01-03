package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"adoctl/pkg/errors"

	"gopkg.in/yaml.v3"
)

const DefaultParallelProcesses = 32

// Profile represents a named Azure DevOps configuration
type Profile struct {
	Name    string `yaml:"name"`
	Azure   AzureConfig `yaml:"azure"`
	Default bool   `yaml:"default,omitempty"`
}

// Config holds the complete configuration including profiles
type Config struct {
	Azure         AzureConfig      `yaml:"azure"`
	ThreadPool    ThreadPoolConfig `yaml:"threadpool"`
	Profiles      []Profile        `yaml:"profiles,omitempty"`
	ActiveProfile string           `yaml:"active_profile,omitempty"`
}

type ThreadPoolConfig struct {
	ParallelProcesses int `yaml:"parallel_processes"`
}

type AzureConfig struct {
	Organization        string `yaml:"organization"`
	Project             string `yaml:"project"`
	PersonalAccessToken string `yaml:"personal_access_token"`
	APIVersion          string `yaml:"api_version"`
}

// Load loads the configuration, optionally with a specific profile
func Load(profileName ...string) (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, errors.NewWithError(errors.ExitCodeConfig, "failed to get config path", err)
	}
	return loadFromPath(configPath, profileName...)
}

// LoadWithProfile loads configuration with a specific profile
func LoadWithProfile(profileName string) (*Config, error) {
	return Load(profileName)
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "adoctl", "config.yaml"), nil
}

// GetProfilesDir returns the directory for profile-specific configs
func GetProfilesDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	profilesDir := filepath.Join(configDir, "adoctl", "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return "", err
	}
	return profilesDir, nil
}

// Save saves the configuration to file
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return errors.NewWithError(errors.ExitCodeFileOperation, "failed to create config directory", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.NewWithError(errors.ExitCodeConfig, "failed to marshal config", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return errors.NewWithError(errors.ExitCodeFileOperation, "failed to write config file", err)
	}

	return nil
}

// GetProfile returns a profile by name
func (c *Config) GetProfile(name string) (*Profile, error) {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i], nil
		}
	}
	return nil, fmt.Errorf("profile '%s' not found", name)
}

// SetProfile sets the active profile
func (c *Config) SetProfile(name string) error {
	if name == "" {
		c.ActiveProfile = ""
		return nil
	}

	if _, err := c.GetProfile(name); err != nil {
		return err
	}

	c.ActiveProfile = name
	return nil
}

// AddProfile adds a new profile
func (c *Config) AddProfile(profile Profile) error {
	if _, err := c.GetProfile(profile.Name); err == nil {
		return fmt.Errorf("profile '%s' already exists", profile.Name)
	}

	c.Profiles = append(c.Profiles, profile)
	return nil
}

// RemoveProfile removes a profile
func (c *Config) RemoveProfile(name string) error {
	if c.ActiveProfile == name {
		return fmt.Errorf("cannot remove active profile '%s'", name)
	}

	for i, p := range c.Profiles {
		if p.Name == name {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile '%s' not found", name)
}

// ListProfiles returns a list of profile names
func (c *Config) ListProfiles() []string {
	names := make([]string, 0, len(c.Profiles))
	for _, p := range c.Profiles {
		names = append(names, p.Name)
	}
	return names
}

// IsProfileActive returns true if the given profile is active
func (c *Config) IsProfileActive(name string) bool {
	return c.ActiveProfile == name
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func loadFromPath(configPath string, profileName ...string) (*Config, error) {
	cfg := &Config{}

	if err := loadConfigFile(configPath, cfg); err != nil {
		return nil, err
	}

	applyEnvironmentOverrides(cfg)

	// Apply profile if specified or if there's an active profile
	targetProfile := ""
	if len(profileName) > 0 && profileName[0] != "" {
		targetProfile = profileName[0]
	} else if cfg.ActiveProfile != "" {
		targetProfile = cfg.ActiveProfile
	}

	if targetProfile != "" {
		if profile, err := cfg.GetProfile(targetProfile); err == nil {
			// Apply profile settings over base config
			applyProfileConfig(cfg, profile)
		}
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyProfileConfig(cfg *Config, profile *Profile) {
	if profile.Azure.Organization != "" {
		cfg.Azure.Organization = profile.Azure.Organization
	}
	if profile.Azure.Project != "" {
		cfg.Azure.Project = profile.Azure.Project
	}
	if profile.Azure.PersonalAccessToken != "" {
		cfg.Azure.PersonalAccessToken = profile.Azure.PersonalAccessToken
	}
	if profile.Azure.APIVersion != "" {
		cfg.Azure.APIVersion = profile.Azure.APIVersion
	}
}

// loadConfigFile reads and parses the config file from the given path
func loadConfigFile(path string, cfg *Config) error {
	if _, err := os.Stat(path); err != nil {
		// File doesn't exist, that's okay - we'll use env vars
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return errors.NewWithError(errors.ExitCodeFileOperation, "failed to read config file", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return errors.NewWithError(errors.ExitCodeConfig, "failed to parse config file", err)
	}

	return nil
}

// applyEnvironmentOverrides applies environment variable overrides to the config
func applyEnvironmentOverrides(cfg *Config) {
	if cfg.Azure.Organization == "" {
		cfg.Azure.Organization = getEnv("AZURE_ORGANIZATION", "")
	}
	if cfg.Azure.Project == "" {
		cfg.Azure.Project = getEnv("AZURE_PROJECT", "")
	}
	if cfg.Azure.PersonalAccessToken == "" {
		cfg.Azure.PersonalAccessToken = getEnv("AZURE_PAT", "")
	}
	if cfg.Azure.APIVersion == "" {
		cfg.Azure.APIVersion = getEnv("AZURE_API_VERSION", "7.1")
	}
	if cfg.ThreadPool.ParallelProcesses == 0 {
		cfg.ThreadPool.ParallelProcesses = getEnvInt("ADOCTL_THREADPOOL_SIZE", DefaultParallelProcesses)
	}

	// Profile can be overridden via environment
	if profileEnv := os.Getenv("ADOCTL_PROFILE"); profileEnv != "" {
		cfg.ActiveProfile = profileEnv
	}
}

// validateConfig ensures all required configuration fields are set
func validateConfig(cfg *Config) error {
	if cfg.Azure.Organization == "" {
		return errors.ConfigError("azure organization not configured. Set it in config file, use --profile, or set AZURE_ORGANIZATION environment variable")
	}
	if cfg.Azure.Project == "" {
		return errors.ConfigError("azure project not configured. Set it in config file, use --profile, or set AZURE_PROJECT environment variable")
	}
	if cfg.Azure.PersonalAccessToken == "" {
		return errors.ConfigError("azure personal access token not configured. Set it in config file, use --profile, or set AZURE_PAT environment variable")
	}
	return nil
}
