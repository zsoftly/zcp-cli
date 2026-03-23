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
	profileFlag string
	outputFlag  string
	apiURLFlag  string
	timeoutFlag int
	debugFlag   bool
	noColorFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "zcp",
	Short: "ZCP CLI — ZSoftly Cloud Platform command-line interface",
	Long: `zcp is the official command-line interface for the ZSoftly Cloud Platform (ZCP).

It provides a scriptable, cross-platform way to manage cloud resources including
instances, volumes, networks, VPCs, Kubernetes clusters, and more.

Get started:
  zcp profile add          Configure your API credentials
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
  # Add to ~/.zshrc:
  $ echo 'source <(zcp completion zsh)' >> ~/.zshrc

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

// GlobalFlags returns the current global flag values for use in subcommand constructors.
// Note: this is called from PersistentPreRunE hooks in subcommands if needed.
func GlobalFlags() config.GlobalFlags {
	return config.GlobalFlags{
		Profile: profileFlag,
		Output:  outputFlag,
		APIURL:  apiURLFlag,
		Timeout: timeoutFlag,
		Debug:   debugFlag,
		NoColor: noColorFlag,
	}
}
