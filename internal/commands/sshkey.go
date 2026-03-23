package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/sshkey"
)

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

	headers := []string{"UUID", "NAME", "STATUS", "DOMAIN"}
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{
			k.UUID,
			k.Name,
			k.Status,
			k.DomainName,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newSSHKeyImportCmd() *cobra.Command {
	var name, publicKey, keyFile string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an SSH public key",
		Example: `  zcp ssh-key import --name mykey --public-key "ssh-rsa AAAA..."
  zcp ssh-key import --name mykey --key-file ~/.ssh/id_rsa.pub`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
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
			return runSSHKeyImport(cmd, sshkey.CreateRequest{
				Name:      name,
				PublicKey: publicKey,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name for the SSH key (required)")
	cmd.Flags().StringVar(&publicKey, "public-key", "", "SSH public key string")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "Path to a file containing the SSH public key")
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
		{"UUID", key.UUID},
		{"Name", key.Name},
		{"Status", key.Status},
		{"Domain", key.DomainName},
	}
	return printer.PrintTable(headers, rows)
}

func newSSHKeyDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete an SSH key",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp ssh-key delete <uuid>
  zcp ssh-key delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSSHKeyDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runSSHKeyDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete SSH key %q? [y/N]: ", uuid)
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

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("ssh-key delete: %w", err)
	}

	printer.Fprintf("SSH key %q deleted.\n", uuid)
	return nil
}
