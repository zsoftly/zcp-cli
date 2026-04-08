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
)

var rootCmd = &cobra.Command{
	Use:   "zcp",
	Short: "ZCP CLI — ZSoftly Cloud Platform command-line interface",
	Long: `zcp is the official command-line interface for the ZSoftly Cloud Platform (ZCP).

It provides a scriptable, cross-platform way to manage cloud resources including
instances, volumes, networks, VPCs, Kubernetes clusters, and more.

Get started:
  zcp profile add default  Configure your API credentials
  zcp auth validate        Verify your credentials work
  zcp zone list            List available zones
  zcp instance list        List your instances

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

	// Version subcommand
	rootCmd.AddCommand(newVersionCmd())

	// Completion subcommand
	rootCmd.AddCommand(newCompletionCmd())

	// Subcommands from the commands package
	rootCmd.AddCommand(commands.NewProfileCmd())
	rootCmd.AddCommand(commands.NewAuthCmd())
	rootCmd.AddCommand(commands.NewZoneCmd())
	rootCmd.AddCommand(commands.NewOfferingCmd())
	rootCmd.AddCommand(commands.NewTemplateCmd())
	rootCmd.AddCommand(commands.NewResourceCmd())
	// Phase 2: compute, storage, networking
	rootCmd.AddCommand(commands.NewInstanceCmd())
	rootCmd.AddCommand(commands.NewVolumeCmd())
	rootCmd.AddCommand(commands.NewSnapshotCmd())
	rootCmd.AddCommand(commands.NewVMSnapshotCmd())
	rootCmd.AddCommand(commands.NewSnapshotPolicyCmd())
	rootCmd.AddCommand(commands.NewNetworkCmd())
	rootCmd.AddCommand(commands.NewIPCmd())
	rootCmd.AddCommand(commands.NewFirewallCmd())
	rootCmd.AddCommand(commands.NewEgressCmd())
	rootCmd.AddCommand(commands.NewPortForwardCmd())
	rootCmd.AddCommand(commands.NewTagCmd())
	// Phase 3: advanced networking, security, kubernetes, billing/admin
	rootCmd.AddCommand(commands.NewVPCCmd())
	rootCmd.AddCommand(commands.NewACLCmd())
	rootCmd.AddCommand(commands.NewLoadBalancerCmd())
	rootCmd.AddCommand(commands.NewInternalLBCmd())
	rootCmd.AddCommand(commands.NewVPNCmd())
	rootCmd.AddCommand(commands.NewSSHKeyCmd())
	rootCmd.AddCommand(commands.NewSecurityGroupCmd())
	rootCmd.AddCommand(commands.NewKubernetesCmd())
	rootCmd.AddCommand(commands.NewUsageCmd())
	rootCmd.AddCommand(commands.NewCostCmd())
	rootCmd.AddCommand(commands.NewHostCmd())
	rootCmd.AddCommand(commands.NewAdminCmd())

	// STKCNSL API — new feature commands
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
		Args:                  cobra.ExactValidArgs(1),
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
