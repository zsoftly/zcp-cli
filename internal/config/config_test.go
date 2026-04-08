package config_test

import (
	"os"
	"path/filepath"
	"runtime"
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
	}{
		{"active profile", "", false},
		{"explicit profile", "prod", false},
		{"missing profile", "dev", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := config.ResolveProfile(cfg, tt.profileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && p == nil {
				t.Error("expected non-nil Profile")
			}
		})
	}
}

func TestResolveProfileNoActive(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]config.Profile{},
	}
	_, err := config.ResolveProfile(cfg, "")
	if err == nil {
		t.Error("expected error when no active profile, got nil")
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
