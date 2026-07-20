package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/sshkey"
)

// sshKeyNameMaxLen is the API's limit on SSH key names; exceeding it returns
// a 422 ("The name field must not be greater than 20 characters.").
const sshKeyNameMaxLen = 20

// NewSSHKeyCmd returns the 'ssh-key' cobra command.
func NewSSHKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage SSH keys",
	}
	cmd.AddCommand(newSSHKeyListCmd())
	cmd.AddCommand(newSSHKeyImportCmd())
	cmd.AddCommand(newSSHKeyDeleteCmd())
	return cmd
}

func newSSHKeyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List SSH keys",
		Example: `  zcp ssh-key list
  zcp ssh-key list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSSHKeyList(cmd)
		},
	}
	return cmd
}

func runSSHKeyList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := sshkey.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	keys, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("ssh-key list: %w", err)
	}

	headers := []string{"ID", "NAME", "SLUG", "CREATED"}
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{
			k.ID,
			k.Name,
			k.Slug,
			k.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newSSHKeyImportCmd() *cobra.Command {
	var name, publicKey, keyFile, project, region string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an SSH public key",
		Long: `Import an SSH public key for the account.

--project and --region are required: the API derives the cloud provider from
them, and omitting either fails with 500 "Attempt to read property \"id\" on null".
The key can afterwards be referenced by name at VM-create time
('zcp instance create --ssh-key <name>').

Keys must be unique — both fields are validated server-side:
  - The name must be at most 20 characters and unique for the account.
  - The public key material itself must not already be registered. Re-importing
    a key you already have (even under a different name) is rejected with
    "The public key has already been taken." To rename or replace a key, delete
    the existing one first ('zcp ssh-key delete <slug>'), then re-import.`,
		Example: `  zcp ssh-key import --name mykey --public-key "ssh-rsa AAAA..." --project default-9 --region yul-1
  zcp ssh-key import --name mykey --key-file ~/.ssh/id_rsa.pub --project default-9 --region yul-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if len(name) > sshKeyNameMaxLen {
				return fmt.Errorf("--name must be at most %d characters (got %d)", sshKeyNameMaxLen, len(name))
			}
			if keyFile != "" {
				data, err := os.ReadFile(keyFile)
				if err != nil {
					return fmt.Errorf("reading key file %q: %w", keyFile, err)
				}
				publicKey = strings.TrimSpace(string(data))
			}
			if publicKey == "" {
				return fmt.Errorf("--public-key or --key-file is required")
			}
			// Fall back to the active profile's defaults (SSH keys use the
			// compute region, so the profile default applies), matching how the
			// scope gate resolves region/project for non-exempt commands.
			project = resolveProject(project)
			region = resolveRegion(region)
			if project == "" || region == "" {
				pr, pp := profileScopeDefaults(cmd)
				if region == "" {
					region = pr
				}
				if project == "" {
					project = pp
				}
			}
			if project == "" {
				return fmt.Errorf("--project is required (or set ZCP_PROJECT, or `zcp profile add` a default)")
			}
			if region == "" {
				return fmt.Errorf("--region is required (or set ZCP_REGION, or `zcp profile add` a default)")
			}
			return runSSHKeyImport(cmd, sshkey.CreateRequest{
				Name:      name,
				PublicKey: publicKey,
				Project:   project,
				Region:    region,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name for the SSH key (required)")
	cmd.Flags().StringVar(&publicKey, "public-key", "", "SSH public key string")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "Path to a file containing the SSH public key")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required; or set ZCP_PROJECT)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug, e.g. yul-1 (required; or set ZCP_REGION)")
	return cmd
}

func runSSHKeyImport(cmd *cobra.Command, req sshkey.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := sshkey.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	key, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("ssh-key import: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", key.ID},
		{"Name", key.Name},
		{"Slug", key.Slug},
		{"Created", key.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newSSHKeyDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete an SSH key",
		Args:  exactArgs(1),
		Long: `Delete an SSH key. <key> may be the key's slug, name, or ID as shown by
'zcp ssh-key list'. The API deletes by slug, so any other identifier is
resolved to the slug first.`,
		Example: `  zcp ssh-key delete my-key
  zcp ssh-key delete my-key --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSSHKeyDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runSSHKeyDelete(cmd *cobra.Command, identifier string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete SSH key %q? [y/N]: ", identifier)
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

	svc := sshkey.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	// The delete endpoint addresses keys by slug only; the ID (UUID) shown by
	// 'ssh-key list' is rejected as not-found (verified live 2026-07-19). Resolve
	// whatever identifier the user passed to the slug so slug, name, and ID all
	// work.
	keys, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("ssh-key delete: resolving %q: %w", identifier, err)
	}
	slug := ""
	for _, k := range keys {
		if k.Slug == identifier || k.ID == identifier || k.Name == identifier {
			slug = k.Slug
			break
		}
	}
	if slug == "" {
		fmt.Fprintf(os.Stderr, "SSH key %q not found — already deleted.\n", identifier)
		return nil
	}

	if err := svc.Delete(ctx, slug); err != nil {
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "SSH key %q not found — already deleted.\n", identifier)
			return nil
		}
		return fmt.Errorf("ssh-key delete: %w", err)
	}

	printer.Fprintf("SSH key %q deleted.\n", identifier)
	return nil
}
