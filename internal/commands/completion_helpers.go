package commands

import (
	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/config"
)

// completeProfileNames returns profile names from the local config for shell completion.
// Degrades gracefully if config is missing or unreadable.
func completeProfileNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
