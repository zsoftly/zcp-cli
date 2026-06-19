package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/commands"
	"github.com/zsoftly/zcp-cli/internal/config"
	"github.com/zsoftly/zcp-cli/internal/version"
)

var (
	profileFlag     string
	outputFlag      string
	apiURLFlag      string
	timeoutFlag     int
	debugFlag       bool
	noColorFlag     bool
	pagerFlag       bool
	autoApproveFlag bool
	regionFlag      string
	projectFlag     string
)

// scopeExemptTop lists top-level commands that are NOT region/project-scoped, so
// they are not gated by the mandatory region+project requirement. These are
// account-level data (dns), account/billing/meta commands, credential and
// discovery commands (you use these to find a region/project in the first
// place), and mixed groups whose region/project-scoped subcommands enforce it
// themselves while their other subcommands are account-wide:
//   - ssh-key: list/delete are account-wide (the key registry is not
//     region-scoped); import validates region+project itself.
//   - object-storage: get/credentials/delete act on a slug, and create/list use
//     object-storage regions (os-yul/os-yow) that differ from the profile's
//     compute region, so the generic gate must not force the compute default
//     onto them — the scoped subcommands validate their own os-* region.
var scopeExemptTop = map[string]bool{
	"auth": true, "profile": true, "profile-info": true, "region": true,
	"project": true, "cloud-provider": true, "currency": true, "billing-cycle": true,
	"server": true, "dns": true, "support": true, "dashboard": true, "billing": true,
	"product": true, "store": true, "version": true, "completion": true, "help": true,
	"ssh-key": true, "object-storage": true,
}

// topLevelName returns the name of cmd's ancestor directly under the root.
func topLevelName(cmd *cobra.Command) string {
	c := cmd
	for c.HasParent() && c.Parent().Name() != rootCmd.Name() {
		c = c.Parent()
	}
	return c.Name()
}

// enforceScope requires --region and --project (or ZCP_REGION/ZCP_PROJECT) for
// every action command except the account-level/meta commands in scopeExemptTop.
// Resources and the catalog are region- and project-specific; running unscoped
// returns or targets entries from other regions/projects.
func enforceScope(cmd *cobra.Command, _ []string) error {
	// Group commands with no action (e.g. bare `zcp instance`) just print help.
	if cmd.RunE == nil && cmd.Run == nil {
		return nil
	}
	if scopeExemptTop[topLevelName(cmd)] {
		return nil
	}
	flagOrEnv := func(flag, env string) string {
		if f := cmd.Flags().Lookup(flag); f != nil && f.Value.String() != "" {
			return f.Value.String()
		}
		return os.Getenv(env)
	}
	region := flagOrEnv("region", "ZCP_REGION")
	project := flagOrEnv("project", "ZCP_PROJECT")

	// Fall back to the active profile's stored defaults (set by `profile add`,
	// like `aws configure`), so a configured user need not repeat the flags. This
	// MUST use the same resolver the command layer uses (config.ScopeDefaults), or
	// the gate would accept a profile default that the command then ignores —
	// silently running unscoped. See scopedRegionProject/requireRegion.
	if region == "" || project == "" {
		pr, pp := config.ScopeDefaults(profileFlag)
		if region == "" {
			region = pr
		}
		if project == "" {
			project = pp
		}
	}

	if region == "" {
		return fmt.Errorf("--region is required (or set ZCP_REGION, or `zcp profile add` a default) for %q — "+
			"resources and the catalog are region-specific; only DNS and account-level commands are region-free. "+
			"See 'zcp region list'", cmd.CommandPath())
	}
	if project == "" {
		return fmt.Errorf("--project is required (or set ZCP_PROJECT, or `zcp profile add` a default) for %q. "+
			"See 'zcp project list'", cmd.CommandPath())
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "zcp",
	Short: "ZCP CLI — ZSoftly Cloud Platform command-line interface",
	Long: `zcp is the official command-line interface for the ZSoftly Cloud Platform (ZCP).

It provides a scriptable, cross-platform way to manage cloud resources including
instances, volumes, networks, VPCs, Kubernetes clusters, and more.

Get started:
  zcp profile add default  Configure your API credentials
  zcp auth validate        Verify your credentials work
  zcp region list           List available regions
  zcp instance list        List your instances

Environment variables:
  ZCP_BEARER_TOKEN    Bearer token for zero-config or CI use
  ZCP_API_URL         API base URL override
  ZCP_PROFILE         Profile name when --profile is not provided
  ZCP_PROJECT         Default project slug for create commands
  ZCP_REGION          Default region slug for create commands
  ZCP_CLOUD_PROVIDER  Default cloud provider slug for create commands
  ZCP_OUTPUT          Default output format: table, json, or yaml
  ZCP_DEBUG           Enable debug output when true, 1, or yes

Documentation: https://docs.zsoftly.com/zcp-cli`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "Configuration profile to use (overrides active profile)")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "table", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().StringVar(&apiURLFlag, "api-url", "", "Override API base URL")
	rootCmd.PersistentFlags().IntVar(&timeoutFlag, "timeout", 30, "Request timeout in seconds")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug output (written to stderr)")
	rootCmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "Disable color output")
	rootCmd.PersistentFlags().BoolVar(&pagerFlag, "pager", false, "Pipe table output through less (requires less in PATH)")
	rootCmd.PersistentFlags().BoolVarP(&autoApproveFlag, "auto-approve", "y", false, "Skip all confirmation prompts (useful for automation/CI)")
	rootCmd.PersistentFlags().StringVar(&regionFlag, "region", "", "Region slug (required for all but account-level commands; or set ZCP_REGION)")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Project slug (required for all but account-level commands; or set ZCP_PROJECT)")

	// Mandatory region+project scoping for every action command except the
	// account-level/meta commands in scopeExemptTop.
	rootCmd.PersistentPreRunE = enforceScope

	// Version subcommand
	rootCmd.AddCommand(newVersionCmd())

	// Completion subcommand
	rootCmd.AddCommand(newCompletionCmd())

	// Subcommands from the commands package
	rootCmd.AddCommand(commands.NewProfileCmd())
	rootCmd.AddCommand(commands.NewAuthCmd())
	rootCmd.AddCommand(commands.NewTemplateCmd())
	// Phase 2: compute, storage, networking
	rootCmd.AddCommand(commands.NewInstanceCmd())
	rootCmd.AddCommand(commands.NewVolumeCmd())
	rootCmd.AddCommand(commands.NewSnapshotCmd())
	rootCmd.AddCommand(commands.NewVMSnapshotCmd())
	rootCmd.AddCommand(commands.NewNetworkCmd())
	rootCmd.AddCommand(commands.NewIPCmd())
	rootCmd.AddCommand(commands.NewFirewallCmd())
	rootCmd.AddCommand(commands.NewEgressCmd())
	rootCmd.AddCommand(commands.NewPortForwardCmd())
	// Phase 3: advanced networking, kubernetes
	rootCmd.AddCommand(commands.NewVPCCmd())
	rootCmd.AddCommand(commands.NewACLCmd())
	rootCmd.AddCommand(commands.NewLoadBalancerCmd())
	rootCmd.AddCommand(commands.NewVPNCmd())
	rootCmd.AddCommand(commands.NewSSHKeyCmd())
	rootCmd.AddCommand(commands.NewKubernetesCmd())

	// STKCNSL API — new feature commands
	rootCmd.AddCommand(commands.NewRegionCmd())
	rootCmd.AddCommand(commands.NewProjectCmd())
	rootCmd.AddCommand(commands.NewSupportCmd())
	rootCmd.AddCommand(commands.NewDNSCmd())
	rootCmd.AddCommand(commands.NewAutoscaleCmd())
	rootCmd.AddCommand(commands.NewISOCmd())
	rootCmd.AddCommand(commands.NewAffinityGroupCmd())
	rootCmd.AddCommand(commands.NewMonitoringCmd())
	rootCmd.AddCommand(commands.NewStoreCmd())
	rootCmd.AddCommand(commands.NewProductCmd())
	rootCmd.AddCommand(commands.NewMarketplaceCmd())
	rootCmd.AddCommand(commands.NewPlanCmd())
	rootCmd.AddCommand(commands.NewDashboardCmd())
	rootCmd.AddCommand(commands.NewBillingCmd())
	rootCmd.AddCommand(commands.NewBackupCmd())
	rootCmd.AddCommand(commands.NewUserProfileCmd())
	rootCmd.AddCommand(commands.NewVMBackupCmd())
	rootCmd.AddCommand(commands.NewCloudProviderCmd())
	rootCmd.AddCommand(commands.NewServerCmd())
	rootCmd.AddCommand(commands.NewCurrencyCmd())
	rootCmd.AddCommand(commands.NewBillingCycleCmd())
	rootCmd.AddCommand(commands.NewStorageCategoryCmd())
	rootCmd.AddCommand(commands.NewObjectStorageCmd())

	// Flag completions — static values, no network calls
	rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.RegisterFlagCompletionFunc("profile", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names := make([]string, 0, len(cfg.Profiles))
		for name := range cfg.Profiles {
			names = append(names, name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})

	// Make command groups reject unknown subcommands with an actionable error
	// instead of silently printing help with a success exit code. Must run after
	// every subcommand is registered above.
	commands.EnforceSubcommandErrors(rootCmd)

	// --cloud-provider is auto-detected and stored on the profile (see
	// `zcp auth validate`); hide it from help while keeping it as an override.
	commands.HideFlagEverywhere(rootCmd, "cloud-provider")
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the zcp CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("zcp version %s\n", version.Version)
		},
	}
}

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for zcp.

To load completions:

Bash:
  $ source <(zcp completion bash)

Zsh:
  # Add to ~/.zshrc (takes effect in new shells):
  $ echo 'source <(zcp completion zsh)' >> ~/.zshrc
  # Or load immediately in the current shell:
  $ source <(zcp completion zsh)

Fish:
  $ zcp completion fish | source

PowerShell:
  PS> zcp completion powershell | Out-String | Invoke-Expression`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}
