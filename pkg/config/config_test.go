package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestLoad_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "adoctl")
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configContent := `azure:
  organization: test-org
  project: test-project
  personal_access_token: test-token
  api_version: "7.1"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalOrg := os.Getenv("AZURE_ORGANIZATION")
	originalProj := os.Getenv("AZURE_PROJECT")
	originalToken := os.Getenv("AZURE_PAT")
	originalAPIVer := os.Getenv("AZURE_API_VERSION")
	os.Unsetenv("AZURE_ORGANIZATION")
	os.Unsetenv("AZURE_PROJECT")
	os.Unsetenv("AZURE_PAT")
	os.Unsetenv("AZURE_API_VERSION")
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
		os.Setenv("AZURE_ORGANIZATION", originalOrg)
		os.Setenv("AZURE_PROJECT", originalProj)
		os.Setenv("AZURE_PAT", originalToken)
		os.Setenv("AZURE_API_VERSION", originalAPIVer)
	}()

	actualConfigPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() failed: %v", err)
	}

	cfg, err := loadFromPath(actualConfigPath)
	if err != nil {
		t.Fatalf("loadFromPath() returned error: %v", err)
	}

	if cfg.Azure.Organization != "test-org" {
		t.Errorf("Expected organization 'test-org', got '%s'", cfg.Azure.Organization)
	}
	if cfg.Azure.Project != "test-project" {
		t.Errorf("Expected project 'test-project', got '%s'", cfg.Azure.Project)
	}
	if cfg.Azure.PersonalAccessToken != "test-token" {
		t.Errorf("Expected personal_access_token 'test-token', got '%s'", cfg.Azure.PersonalAccessToken)
	}
	if cfg.Azure.APIVersion != "7.1" {
		t.Errorf("Expected api_version '7.1', got '%s'", cfg.Azure.APIVersion)
	}
}

func TestLoad_MinimalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "adoctl")
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	configContent := `azure:
  organization: minimal-org
  project: minimal-project
  personal_access_token: minimal-token
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalOrg := os.Getenv("AZURE_ORGANIZATION")
	originalProj := os.Getenv("AZURE_PROJECT")
	originalToken := os.Getenv("AZURE_PAT")
	originalAPIVer := os.Getenv("AZURE_API_VERSION")
	os.Unsetenv("AZURE_ORGANIZATION")
	os.Unsetenv("AZURE_PROJECT")
	os.Unsetenv("AZURE_PAT")
	os.Unsetenv("AZURE_API_VERSION")
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
		os.Setenv("AZURE_ORGANIZATION", originalOrg)
		os.Setenv("AZURE_PROJECT", originalProj)
		os.Setenv("AZURE_PAT", originalToken)
		os.Setenv("AZURE_API_VERSION", originalAPIVer)
	}()

	actualConfigPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() failed: %v", err)
	}

	cfg, err := loadFromPath(actualConfigPath)
	if err != nil {
		t.Fatalf("loadFromPath() returned error: %v", err)
	}

	if cfg.Azure.APIVersion != "7.1" {
		t.Errorf("Expected default api_version '7.1', got '%s'", cfg.Azure.APIVersion)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "adoctl")
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	invalidContent := `azure:
  organization: test-org
  project: test-project
  - invalid yaml
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalOrg := os.Getenv("AZURE_ORGANIZATION")
	originalProj := os.Getenv("AZURE_PROJECT")
	originalToken := os.Getenv("AZURE_PAT")
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
		os.Setenv("AZURE_ORGANIZATION", originalOrg)
		os.Setenv("AZURE_PROJECT", originalProj)
		os.Setenv("AZURE_PAT", originalToken)
	}()

	_, err := loadFromPath(configPath)
	if err == nil {
		t.Error("loadFromPath() expected error for invalid YAML, got nil")
	}
}

func TestLoad_MissingOrganization(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `azure:
  project: test-project
  personal_access_token: test-token
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err := loadFromPath(configPath)
	if err == nil {
		t.Error("loadFromPath() expected error for missing organization, got nil")
	}
	if err != nil && !contains(err.Error(), "azure organization not configured") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLoad_MissingProject(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `azure:
  organization: test-org
  personal_access_token: test-token
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err := loadFromPath(configPath)
	if err == nil {
		t.Error("loadFromPath() expected error for missing project, got nil")
	}
	if err != nil && !contains(err.Error(), "azure project not configured") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `azure:
  organization: test-org
  project: test-project
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err := loadFromPath(configPath)
	if err == nil {
		t.Error("loadFromPath() expected error for missing token, got nil")
	}
	if err != nil && !contains(err.Error(), "azure personal access token not configured") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yaml")

	_, err := loadFromPath(configPath)
	if err == nil {
		t.Error("loadFromPath() expected error for nonexistent file, got nil")
	}
}

func TestGetConfigPath(t *testing.T) {
	homeDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("HOME", homeDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
	}()

	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() returned error: %v", err)
	}

	expectedPath := filepath.Join(homeDir, ".config", "adoctl", "config.yaml")
	if path != expectedPath {
		t.Errorf("Expected config path '%s', got '%s'", expectedPath, path)
	}
}

func TestGetConfigPath_WithXDG(t *testing.T) {
	homeDir := t.TempDir()
	xdgDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("HOME", homeDir)
	os.Setenv("XDG_CONFIG_HOME", xdgDir)
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
	}()

	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() returned error: %v", err)
	}

	expectedPath := filepath.Join(xdgDir, "adoctl", "config.yaml")
	if path != expectedPath {
		t.Errorf("Expected config path '%s', got '%s'", expectedPath, path)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnv("TEST_VAR", "default")
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}

	result = getEnv("NONEXISTENT_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

func TestLoad_WithEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	partialConfig := `azure:
  organization: file-org
`
	if err := os.WriteFile(configPath, []byte(partialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	originalHome := os.Getenv("HOME")
	originalOrg := os.Getenv("AZURE_ORGANIZATION")
	originalProj := os.Getenv("AZURE_PROJECT")
	originalToken := os.Getenv("AZURE_PAT")
	originalAPIVer := os.Getenv("AZURE_API_VERSION")
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	os.Setenv("AZURE_ORGANIZATION", "env-org")
	os.Setenv("AZURE_PROJECT", "env-project")
	os.Setenv("AZURE_PAT", "env-token")
	os.Setenv("AZURE_API_VERSION", "7.2")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("AZURE_ORGANIZATION", originalOrg)
		os.Setenv("AZURE_PROJECT", originalProj)
		os.Setenv("AZURE_PAT", originalToken)
		os.Setenv("AZURE_API_VERSION", originalAPIVer)
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Azure.Organization != "env-org" {
		t.Errorf("Expected organization 'env-org', got '%s'", cfg.Azure.Organization)
	}
	if cfg.Azure.Project != "env-project" {
		t.Errorf("Expected project 'env-project', got '%s'", cfg.Azure.Project)
	}
	if cfg.Azure.PersonalAccessToken != "env-token" {
		t.Errorf("Expected personal_access_token 'env-token', got '%s'", cfg.Azure.PersonalAccessToken)
	}
	if cfg.Azure.APIVersion != "7.2" {
		t.Errorf("Expected api_version '7.2', got '%s'", cfg.Azure.APIVersion)
	}
}

func TestLoad_FileNotFound_UsesEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")

	originalHome := os.Getenv("HOME")
	originalOrg := os.Getenv("AZURE_ORGANIZATION")
	originalProj := os.Getenv("AZURE_PROJECT")
	originalToken := os.Getenv("AZURE_PAT")
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	os.Setenv("AZURE_ORGANIZATION", "env-org")
	os.Setenv("AZURE_PROJECT", "env-project")
	os.Setenv("AZURE_PAT", "env-token")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("AZURE_ORGANIZATION", originalOrg)
		os.Setenv("AZURE_PROJECT", originalProj)
		os.Setenv("AZURE_PAT", originalToken)
	}()

	cfg, err := loadFromPath(nonExistentPath)
	if err != nil {
		t.Fatalf("loadFromPath() returned error: %v", err)
	}

	if cfg.Azure.Organization != "env-org" {
		t.Errorf("Expected organization 'env-org', got '%s'", cfg.Azure.Organization)
	}
}

func TestLoad_FileNotFound_MissingOrg(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	os.Setenv("AZURE_ORGANIZATION", "")
	os.Setenv("AZURE_PROJECT", "")
	os.Setenv("AZURE_PAT", "")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("AZURE_ORGANIZATION")
		os.Unsetenv("AZURE_PROJECT")
		os.Unsetenv("AZURE_PAT")
	}()

	_, err := loadFromPath(nonExistentPath)
	if err == nil {
		t.Error("loadFromPath() expected error when file not found and org not set, got nil")
	}
}

// Profile tests
func TestConfig_ProfileManagement(t *testing.T) {
	cfg := &Config{
		Azure: AzureConfig{
			Organization: "default-org",
			Project:      "default-project",
		},
	}

	// Test adding a profile
	profile := Profile{
		Name: "work",
		Azure: AzureConfig{
			Organization: "work-org",
			Project:      "work-project",
		},
	}

	if err := cfg.AddProfile(profile); err != nil {
		t.Fatalf("AddProfile() failed: %v", err)
	}

	// Test getting a profile
	retrieved, err := cfg.GetProfile("work")
	if err != nil {
		t.Fatalf("GetProfile() failed: %v", err)
	}
	if retrieved.Azure.Organization != "work-org" {
		t.Errorf("Expected organization 'work-org', got '%s'", retrieved.Azure.Organization)
	}

	// Test listing profiles
	profiles := cfg.ListProfiles()
	if len(profiles) != 1 || profiles[0] != "work" {
		t.Errorf("Expected profiles ['work'], got %v", profiles)
	}

	// Test setting active profile
	if err := cfg.SetProfile("work"); err != nil {
		t.Fatalf("SetProfile() failed: %v", err)
	}
	if cfg.ActiveProfile != "work" {
		t.Errorf("Expected active profile 'work', got '%s'", cfg.ActiveProfile)
	}

	// Test IsProfileActive
	if !cfg.IsProfileActive("work") {
		t.Error("Expected IsProfileActive('work') to be true")
	}

	// Test adding duplicate profile
	if err := cfg.AddProfile(profile); err == nil {
		t.Error("AddProfile() expected error for duplicate profile")
	}

	// Test removing profile (should fail because it's active)
	if err := cfg.RemoveProfile("work"); err == nil {
		t.Error("RemoveProfile() expected error when removing active profile")
	}

	// Deactivate and then remove
	cfg.SetProfile("")
	if err := cfg.RemoveProfile("work"); err != nil {
		t.Fatalf("RemoveProfile() failed: %v", err)
	}

	// Test getting non-existent profile
	if _, err := cfg.GetProfile("nonexistent"); err == nil {
		t.Error("GetProfile() expected error for non-existent profile")
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Azure: AzureConfig{
			Organization:        "test-org",
			Project:             "test-project",
			PersonalAccessToken: "test-token",
			APIVersion:          "7.1",
		},
		ThreadPool: ThreadPoolConfig{
			ParallelProcesses: 32,
		},
		Profiles: []Profile{
			{
				Name: "personal",
				Azure: AzureConfig{
					Organization: "personal-org",
					Project:      "personal-project",
				},
			},
		},
		ActiveProfile: "personal",
	}

	// Save config
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	data, _ := yaml.Marshal(cfg)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config - this will apply the active profile "personal"
	loaded, err := loadFromPath(configPath, "")
	if err != nil {
		t.Fatalf("loadFromPath() failed: %v", err)
	}

	// Since active_profile is "personal", the loaded config should have personal-org
	if loaded.Azure.Organization != "personal-org" {
		t.Errorf("Expected organization 'personal-org' (from active profile), got '%s'", loaded.Azure.Organization)
	}
	if len(loaded.Profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(loaded.Profiles))
	}
	if loaded.ActiveProfile != "personal" {
		t.Errorf("Expected active profile 'personal', got '%s'", loaded.ActiveProfile)
	}
}

func TestConfig_LoadWithProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `azure:
  organization: default-org
  project: default-project
  personal_access_token: default-token
profiles:
  - name: work
    azure:
      organization: work-org
      project: work-project
      personal_access_token: work-token
  - name: personal
    azure:
      organization: personal-org
      project: personal-project
      personal_access_token: personal-token
active_profile: work
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Load with active profile
	cfg, err := loadFromPath(configPath)
	if err != nil {
		t.Fatalf("loadFromPath() failed: %v", err)
	}

	// Should use work profile settings
	if cfg.Azure.Organization != "work-org" {
		t.Errorf("Expected organization 'work-org' from profile, got '%s'", cfg.Azure.Organization)
	}
	if cfg.Azure.Project != "work-project" {
		t.Errorf("Expected project 'work-project' from profile, got '%s'", cfg.Azure.Project)
	}

	// Load with specific profile
	cfg2, err := loadFromPath(configPath, "personal")
	if err != nil {
		t.Fatalf("loadFromPath() with profile failed: %v", err)
	}

	if cfg2.Azure.Organization != "personal-org" {
		t.Errorf("Expected organization 'personal-org', got '%s'", cfg2.Azure.Organization)
	}
}

func TestConfig_GetProfilesDir(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
	}()

	profilesDir, err := GetProfilesDir()
	if err != nil {
		t.Fatalf("GetProfilesDir() failed: %v", err)
	}

	expectedDir := filepath.Join(tmpDir, ".config", "adoctl", "profiles")
	if profilesDir != expectedDir {
		t.Errorf("Expected profiles dir '%s', got '%s'", expectedDir, profilesDir)
	}

	// Verify directory was created
	if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
		t.Error("Profiles directory was not created")
	}
}
