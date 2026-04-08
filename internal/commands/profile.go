package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/config"
	"golang.org/x/term"
)

// NewProfileCmd returns the 'profile' cobra command.
func NewProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage configuration profiles",
		Long: `Profiles store named credential sets for different ZCP environments or accounts.

Each profile contains a bearer token and optionally a custom API URL.
One profile can be set as the active (default) profile.`,
	}
	cmd.AddCommand(newProfileAddCmd())
	cmd.AddCommand(newProfileListCmd())
	cmd.AddCommand(newProfileUseCmd())
	cmd.AddCommand(newProfileDeleteCmd())
	cmd.AddCommand(newProfileShowCmd())
	cmd.AddCommand(newProfileUpdateCmd())
	cmd.AddCommand(newProfileRenameCmd())
	return cmd
}

func newProfileAddCmd() *cobra.Command {
	var bearerToken, apiURL string
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add or update a profile",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp profile add default
  zcp profile add prod --bearer-token <token>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// If flags not provided, prompt interactively
			if !nonInteractive {
				if bearerToken == "" {
					bearerToken, err = prompt("Bearer Token: ", true)
					if err != nil {
						return err
					}
				}
			}

			if bearerToken == "" {
				return fmt.Errorf("bearer token is required")
			}

			if cfg.Profiles == nil {
				cfg.Profiles = make(map[string]config.Profile)
			}

			cfg.Profiles[name] = config.Profile{
				Name:        name,
				BearerToken: bearerToken,
				APIURL:      apiURL,
			}

			// Set as active if it's the first or only profile
			if cfg.ActiveProfile == "" {
				cfg.ActiveProfile = name
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Profile %q saved.\n", name)
			if cfg.ActiveProfile == name {
				fmt.Fprintln(os.Stdout, "Set as active profile.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&bearerToken, "bearer-token", "", "Bearer token (prompted if not provided)")
	cmd.Flags().StringVar(&apiURL, "api-url-override", "", "Custom API URL (optional)")
	cmd.Flags().BoolVar(&nonInteractive, "no-input", false, "Fail if credentials not provided via flags")
	return cmd
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Profiles) == 0 {
				fmt.Fprintln(os.Stdout, "No profiles configured. Run: zcp profile add")
				return nil
			}
			fmt.Fprintf(os.Stdout, "%-20s %-10s %s\n", "NAME", "ACTIVE", "API URL")
			fmt.Fprintf(os.Stdout, "%s\n", strings.Repeat("-", 60))
			for name, p := range cfg.Profiles {
				active := ""
				if name == cfg.ActiveProfile {
					active = "*"
				}
				apiURL := p.APIURL
				if apiURL == "" {
					apiURL = config.DefaultAPIURL
				}
				fmt.Fprintf(os.Stdout, "%-20s %-10s %s\n", name, active, apiURL)
			}
			return nil
		},
	}
}

func newProfileUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "use <name>",
		Short:             "Set the active profile",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("profile %q not found", name)
			}
			cfg.ActiveProfile = name
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Active profile set to %q\n", name)
			return nil
		},
	}
}

func newProfileDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:               "delete <name>",
		Short:             "Delete a profile",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("profile %q not found", name)
			}
			if !force {
				answer, _ := prompt(fmt.Sprintf("Delete profile %q? [y/N]: ", name), false)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Fprintln(os.Stdout, "Aborted.")
					return nil
				}
			}
			delete(cfg.Profiles, name)
			if cfg.ActiveProfile == name {
				cfg.ActiveProfile = ""
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Profile %q deleted.\n", name)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func newProfileShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "show [name]",
		Short:             "Show profile details (credentials are masked)",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeProfileNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := cfg.ActiveProfile
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("no active profile — run: zcp profile add")
			}
			p, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("profile %q not found", name)
			}
			apiURL := p.APIURL
			if apiURL == "" {
				apiURL = config.DefaultAPIURL
			}
			fmt.Fprintf(os.Stdout, "Profile: %s\n", name)
			fmt.Fprintf(os.Stdout, "API URL: %s\n", apiURL)
			fmt.Fprintf(os.Stdout, "Bearer Token: %s\n", maskSecret(p.BearerToken))
			if name == cfg.ActiveProfile {
				fmt.Fprintln(os.Stdout, "Status: active")
			}
			// Credential completeness hint
			if p.BearerToken == "" {
				fmt.Fprintln(os.Stdout, "Warning: profile is missing credentials — run: zcp profile update "+name)
			}
			return nil
		},
	}
}

func newProfileUpdateCmd() *cobra.Command {
	var bearerToken, apiURL string

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update fields of an existing profile",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp profile update prod --bearer-token <new-token>
  zcp profile update prod --api-url-override https://new.api.url`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			p, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("profile %q not found — run: zcp profile list", name)
			}
			changed := false
			if bearerToken != "" {
				p.BearerToken = bearerToken
				changed = true
			}
			if cmd.Flags().Changed("api-url-override") {
				p.APIURL = apiURL
				changed = true
			}
			if !changed {
				return fmt.Errorf("no fields to update — use --bearer-token or --api-url-override")
			}
			cfg.Profiles[name] = p
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Profile %q updated.\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&bearerToken, "bearer-token", "", "New bearer token")
	cmd.Flags().StringVar(&apiURL, "api-url-override", "", "New custom API URL (set to empty string to clear)")
	return cmd
}

func newProfileRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName, newName := args[0], args[1]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			p, ok := cfg.Profiles[oldName]
			if !ok {
				return fmt.Errorf("profile %q not found", oldName)
			}
			if _, exists := cfg.Profiles[newName]; exists {
				return fmt.Errorf("profile %q already exists", newName)
			}
			p.Name = newName
			cfg.Profiles[newName] = p
			delete(cfg.Profiles, oldName)
			if cfg.ActiveProfile == oldName {
				cfg.ActiveProfile = newName
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Profile %q renamed to %q.\n", oldName, newName)
			return nil
		},
	}
}

// prompt reads a line from stdin. If secret is true, it uses terminal echo suppression.
func prompt(label string, secret bool) (string, error) {
	fmt.Fprint(os.Stdout, label)
	if secret && term.IsTerminal(int(syscall.Stdin)) {
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stdout)
		if err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		return string(b), nil
	}
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	return "", scanner.Err()
}

// maskSecret shows first 4 chars then asterisks, or all asterisks if short.
func maskSecret(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}
