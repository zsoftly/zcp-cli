package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/internal/config"
)

func TestLoadEmpty(t *testing.T) {
	// Point config to a temp dir with no file
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ActiveProfile != "" {
		t.Errorf("expected empty ActiveProfile, got %q", cfg.ActiveProfile)
	}
	if cfg.Profiles == nil {
		t.Error("expected non-nil Profiles map")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", dir)
	} else {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}

	cfg := &config.Config{
		ActiveProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				Name:        "default",
				BearerToken: "test-bearer-token",
				APIURL:      "",
			},
		},
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created with restricted permissions (Unix only; Windows has no chmod)
	path, _ := config.ConfigFilePath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("config file permissions = %o, want 0600", info.Mode().Perm())
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}
	if loaded.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want %q", loaded.ActiveProfile, "default")
	}
	p, ok := loaded.Profiles["default"]
	if !ok {
		t.Fatal("profile 'default' not found after load")
	}
	if p.BearerToken != "test-bearer-token" {
		t.Errorf("BearerToken = %q, want %q", p.BearerToken, "test-bearer-token")
	}
}

func TestResolveProfile(t *testing.T) {
	cfg := &config.Config{
		ActiveProfile: "prod",
		Profiles: map[string]config.Profile{
			"prod": {Name: "prod", BearerToken: "token"},
		},
	}

	tests := []struct {
		name        string
		profileName string
		wantErr     bool
		errContains string
	}{
		{"active profile", "", false, ""},
		{"explicit profile", "prod", false, ""},
		{"missing profile", "dev", true, "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ZCP_BEARER_TOKEN", "") // clear env so profile token is used
			t.Setenv("ZCP_PROFILE", "")      // prevent ambient leak
			p, err := config.ResolveProfile(cfg, tt.profileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err, tt.errContains)
				}
			}
			if !tt.wantErr && p == nil {
				t.Error("expected non-nil Profile")
			}
		})
	}
}

func TestResolveProfileEnvToken(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "env-token")
	cfg := &config.Config{
		Profiles: map[string]config.Profile{},
	}
	// No profile configured, but ZCP_BEARER_TOKEN is set — should succeed
	p, err := config.ResolveProfile(cfg, "")
	if err != nil {
		t.Fatalf("expected success with ZCP_BEARER_TOKEN set, got: %v", err)
	}
	if p.BearerToken != "env-token" {
		t.Errorf("BearerToken = %q, want %q", p.BearerToken, "env-token")
	}
}

func TestResolveProfileEnvTokenOverridesProfile(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "env-token")
	t.Setenv("ZCP_PROFILE", "") // prevent ambient leak
	cfg := &config.Config{
		ActiveProfile: "prod",
		Profiles: map[string]config.Profile{
			"prod": {Name: "prod", BearerToken: "profile-token"},
		},
	}
	p, err := config.ResolveProfile(cfg, "")
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if p.Name != "prod" {
		t.Errorf("Name = %q, want %q (should resolve prod profile)", p.Name, "prod")
	}
	if p.BearerToken != "env-token" {
		t.Errorf("BearerToken = %q, want env override %q", p.BearerToken, "env-token")
	}
}

func TestResolveProfileEnvProfile(t *testing.T) {
	t.Setenv("ZCP_PROFILE", "staging")
	t.Setenv("ZCP_BEARER_TOKEN", "") // clear so profile token is used
	cfg := &config.Config{
		ActiveProfile: "prod",
		Profiles: map[string]config.Profile{
			"prod":    {Name: "prod", BearerToken: "prod-token"},
			"staging": {Name: "staging", BearerToken: "staging-token"},
		},
	}
	p, err := config.ResolveProfile(cfg, "")
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if p.BearerToken != "staging-token" {
		t.Errorf("BearerToken = %q, want %q (ZCP_PROFILE should select staging)", p.BearerToken, "staging-token")
	}
}

func TestResolveProfileEnvAPIURL(t *testing.T) {
	t.Setenv("ZCP_API_URL", "https://env.example.com")
	cfg := &config.Config{
		ActiveProfile: "prod",
		Profiles: map[string]config.Profile{
			"prod": {Name: "prod", BearerToken: "token", APIURL: "https://profile.example.com"},
		},
	}
	p, err := config.ResolveProfile(cfg, "")
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if p.APIURL != "https://env.example.com" {
		t.Errorf("APIURL = %q, want env override %q", p.APIURL, "https://env.example.com")
	}
}

func TestActiveAPIURLEnvOverride(t *testing.T) {
	t.Setenv("ZCP_API_URL", "https://env.example.com")
	p := &config.Profile{APIURL: "https://profile.example.com"}

	got := config.ActiveAPIURL(p, "")
	if got != "https://env.example.com" {
		t.Errorf("ActiveAPIURL = %q, want env override %q", got, "https://env.example.com")
	}

	// Flag still takes precedence over env
	got = config.ActiveAPIURL(p, "https://flag.example.com")
	if got != "https://flag.example.com" {
		t.Errorf("ActiveAPIURL = %q, want flag override %q", got, "https://flag.example.com")
	}
}

func TestResolveProfileNoActive(t *testing.T) {
	// Clear env vars that could interfere
	t.Setenv("ZCP_BEARER_TOKEN", "")
	t.Setenv("ZCP_PROFILE", "")
	cfg := &config.Config{
		Profiles: map[string]config.Profile{},
	}
	_, err := config.ResolveProfile(cfg, "")
	if err == nil {
		t.Error("expected error when no active profile and no env vars, got nil")
	}
}

func TestResolveProfileMissingCredentials(t *testing.T) {
	cfg := &config.Config{
		ActiveProfile: "dev",
		Profiles: map[string]config.Profile{
			"dev": {Name: "dev", BearerToken: ""},
		},
	}
	_, err := config.ResolveProfile(cfg, "dev")
	if err == nil {
		t.Error("ResolveProfile() expected error for missing bearer token, got nil")
	}
}

func TestActiveAPIURL(t *testing.T) {
	p := &config.Profile{APIURL: "https://custom.example.com"}

	tests := []struct {
		flagURL string
		want    string
	}{
		{"", "https://custom.example.com"},
		{"https://override.example.com", "https://override.example.com"},
	}
	for _, tt := range tests {
		got := config.ActiveAPIURL(p, tt.flagURL)
		if got != tt.want {
			t.Errorf("ActiveAPIURL(%q) = %q, want %q", tt.flagURL, got, tt.want)
		}
	}

	// Nil profile, no flag -> DefaultAPIURL
	got := config.ActiveAPIURL(nil, "")
	if got != config.DefaultAPIURL {
		t.Errorf("ActiveAPIURL(nil, \"\") = %q, want DefaultAPIURL", got)
	}
}

func TestConfigFilePath(t *testing.T) {
	dir := t.TempDir()
	// ConfigFilePath uses APPDATA on Windows, XDG_CONFIG_HOME on Unix
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", dir)
	} else {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}

	path, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath() error = %v", err)
	}
	expected := filepath.Join(dir, "zcp", "config.yaml")
	if path != expected {
		t.Errorf("ConfigFilePath() = %q, want %q", path, expected)
	}
}
