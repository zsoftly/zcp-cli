package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/tags"
)

// NewTagCmd returns the 'tag' cobra command.
func NewTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage resource tags",
	}
	cmd.AddCommand(newTagListCmd())
	cmd.AddCommand(newTagCreateCmd())
	cmd.AddCommand(newTagDeleteCmd())
	return cmd
}

func newTagListCmd() *cobra.Command {
	var resourceUUID, resourceType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List resource tags",
		Example: `  zcp tag list
  zcp tag list --resource <uuid>
  zcp tag list --type VirtualMachine`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTagList(cmd, resourceUUID, resourceType)
		},
	}
	cmd.Flags().StringVar(&resourceUUID, "resource", "", "Filter by resource UUID")
	cmd.Flags().StringVar(&resourceType, "type", "", "Filter by resource type")
	return cmd
}

func runTagList(cmd *cobra.Command, resourceUUID, resourceType string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := tags.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	tagList, err := svc.List(ctx, resourceUUID, resourceType)
	if err != nil {
		return fmt.Errorf("tag list: %w", err)
	}

	headers := []string{"UUID", "KEY", "VALUE", "RESOURCE TYPE", "RESOURCE UUID"}
	rows := make([][]string, 0, len(tagList))
	for _, t := range tagList {
		rows = append(rows, []string{
			t.UUID,
			t.Key,
			t.Value,
			t.ResourceType,
			t.ResourceUUID,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newTagCreateCmd() *cobra.Command {
	var zoneUUID, resourceUUID, resourceType, key, value string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a resource tag",
		Example: `  zcp tag create --zone <uuid> --resource <uuid> --type VirtualMachine --key env --value prod`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			if resourceUUID == "" {
				return fmt.Errorf("--resource is required")
			}
			if resourceType == "" {
				return fmt.Errorf("--type is required")
			}
			if key == "" {
				return fmt.Errorf("--key is required")
			}
			if value == "" {
				return fmt.Errorf("--value is required")
			}
			return runTagCreate(cmd, zoneUUID, resourceType, tags.CreateRequest{
				Key:          key,
				Value:        value,
				ResourceUUID: resourceUUID,
			})
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&resourceUUID, "resource", "", "Resource UUID to tag (required)")
	cmd.Flags().StringVar(&resourceType, "type", "", "Resource type (e.g. VirtualMachine) (required)")
	cmd.Flags().StringVar(&key, "key", "", "Tag key (required)")
	cmd.Flags().StringVar(&value, "value", "", "Tag value (required)")
	return cmd
}

func runTagCreate(cmd *cobra.Command, zoneUUID, resourceType string, req tags.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := tags.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	tag, err := svc.Create(ctx, resourceType, zoneUUID, req)
	if err != nil {
		return fmt.Errorf("tag create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", tag.UUID},
		{"Key", tag.Key},
		{"Value", tag.Value},
		{"Resource Type", tag.ResourceType},
		{"Resource UUID", tag.ResourceUUID},
	}
	return printer.PrintTable(headers, rows)
}

func newTagDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a resource tag",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp tag delete <uuid>
  zcp tag delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTagDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runTagDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete tag %q? [y/N]: ", uuid)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
	}

	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := tags.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("tag delete: %w", err)
	}

	printer.Fprintf("Tag %q deleted.\n", uuid)
	return nil
}
