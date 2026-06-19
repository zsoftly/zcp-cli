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
	DefaultAPIURL = "https://api.zcp.zsoftly.ca/api"
	// DefaultTimeout is the default HTTP request timeout in seconds.
	DefaultTimeout = 30
)

// Profile holds credentials and settings for a named profile.
type Profile struct {
	Name        string `yaml:"name"`
	BearerToken string `yaml:"bearer_token"`
	APIURL      string `yaml:"api_url,omitempty"`
	DefaultZone string `yaml:"default_zone,omitempty"`
	// Region and Project are the profile's default region/project slugs, set at
	// `profile add` time (like `aws configure`). They satisfy the mandatory
	// region/project requirement when --region/--project and ZCP_REGION/
	// ZCP_PROJECT are not given.
	Region  string `yaml:"region,omitempty"`
	Project string `yaml:"project,omitempty"`
	// CloudProvider is the account's brand cloud-provider slug, auto-detected at
	// `auth validate` / `profile add` time so create commands need not ask for it.
	CloudProvider string `yaml:"cloud_provider,omitempty"`
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
// It prefers profileName if provided, else ZCP_PROFILE env var, else cfg.ActiveProfile.
// It also applies ZCP_BEARER_TOKEN and ZCP_API_URL environment variable overrides.
// Returns an error if no profile is configured or credentials are missing (unless overridden by env).
func ResolveProfile(cfg *Config, profileName string) (*Profile, error) {
	name := profileName
	if name == "" {
		name = os.Getenv("ZCP_PROFILE")
	}
	if name == "" {
		name = cfg.ActiveProfile
	}

	// Look up the named profile if one was resolved
	var p Profile
	var profileFound bool
	if name != "" {
		if prof, ok := cfg.Profiles[name]; ok {
			p = prof
			profileFound = true
		}
	}

	// Override with environment variables
	if envToken := os.Getenv("ZCP_BEARER_TOKEN"); envToken != "" {
		p.BearerToken = envToken
	}
	if envURL := os.Getenv("ZCP_API_URL"); envURL != "" {
		p.APIURL = envURL
	}

	// Validate: profile not found (and no env override to save us)
	if name != "" && !profileFound && p.BearerToken == "" {
		return nil, fmt.Errorf("profile %q not found — run: zcp profile list", name)
	}

	// Validate: credentials missing
	if p.BearerToken == "" {
		if name == "" {
			return nil, errors.New("no active profile configured and ZCP_BEARER_TOKEN not set — run: zcp profile add")
		}
		return nil, fmt.Errorf("profile %q is missing credentials and ZCP_BEARER_TOKEN not set — run: zcp profile add", name)
	}

	return &p, nil
}

// ActiveAPIURL returns the resolved API URL for the given profile, applying overrides.
// Order of precedence: flagURL > ZCP_API_URL env > profile APIURL > DefaultAPIURL
// ScopeDefaults returns the active profile's default region and project slugs.
// The profile name is resolved the same way ResolveProfile does (profileName >
// ZCP_PROFILE > the active profile). It returns empty strings — never an error —
// when config cannot be loaded or no matching profile exists, so callers can
// layer it underneath flag/env precedence without failing on an unconfigured
// account. This is the single source of truth for the profile fallback used by
// both the root scope gate and the per-command region/project resolvers.
func ScopeDefaults(profileName string) (region, project string) {
	cfg, err := Load()
	if err != nil {
		return "", ""
	}
	name := profileName
	if name == "" {
		name = os.Getenv("ZCP_PROFILE")
	}
	if name == "" {
		name = cfg.ActiveProfile
	}
	if name == "" {
		return "", ""
	}
	if p, ok := cfg.Profiles[name]; ok {
		return p.Region, p.Project
	}
	return "", ""
}

func ActiveAPIURL(profile *Profile, flagURL string) string {
	if flagURL != "" {
		return flagURL
	}
	if envURL := os.Getenv("ZCP_API_URL"); envURL != "" {
		return envURL
	}
	if profile != nil && profile.APIURL != "" {
		return profile.APIURL
	}
	return DefaultAPIURL
}
