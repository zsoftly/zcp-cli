// Package config manages ZCP CLI configuration and profiles.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultAPIURL is the default ZCP API base URL.
	DefaultAPIURL = "https://portal.webberstop.com/backend/api"
	// DefaultTimeout is the default HTTP request timeout in seconds.
	DefaultTimeout = 30
)

// Profile holds credentials and settings for a named profile.
type Profile struct {
	Name        string `yaml:"name"`
	BearerToken string `yaml:"bearer_token"`
	APIURL      string `yaml:"api_url,omitempty"`
	DefaultZone string `yaml:"default_zone,omitempty"`
}

// Config is the top-level config structure stored on disk.
type Config struct {
	ActiveProfile string             `yaml:"active_profile"`
	Profiles      map[string]Profile `yaml:"profiles,omitempty"`
}

// ConfigFilePath returns the platform-appropriate config file path.
// Linux/macOS: ~/.config/zcp/config.yaml
// Windows:     %AppData%/zcp/config.yaml
func ConfigFilePath() (string, error) {
	var base string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", errors.New("APPDATA environment variable not set")
		}
		base = appData
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		// Respect XDG_CONFIG_HOME if set
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg != "" {
			base = xdg
		} else {
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, "zcp", "config.yaml"), nil
}

// Load reads the config file from disk, returning an empty config if it does not exist.
func Load() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				Profiles: make(map[string]Profile),
			}, nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	return &cfg, nil
}

// Save writes the config to disk, creating parent directories as needed.
func Save(cfg *Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}

	// Restrict permissions: config contains credentials
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}

	return nil
}

// ResolveProfile returns the Profile to use for a request.
// It prefers profileName if provided, else cfg.ActiveProfile.
// Returns an error if no profile is configured or credentials are missing.
func ResolveProfile(cfg *Config, profileName string) (*Profile, error) {
	name := profileName
	if name == "" {
		name = cfg.ActiveProfile
	}
	if name == "" {
		return nil, errors.New("no active profile configured — run: zcp profile add")
	}

	p, ok := cfg.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found — run: zcp profile list", name)
	}
	if p.BearerToken == "" {
		return nil, fmt.Errorf("profile %q is missing credentials — run: zcp profile add", name)
	}

	return &p, nil
}

// ActiveAPIURL returns the resolved API URL for the given profile, applying overrides.
// Order of precedence: flagURL > profile APIURL > DefaultAPIURL
func ActiveAPIURL(profile *Profile, flagURL string) string {
	if flagURL != "" {
		return flagURL
	}
	if profile != nil && profile.APIURL != "" {
		return profile.APIURL
	}
	return DefaultAPIURL
}
