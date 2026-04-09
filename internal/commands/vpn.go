package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/vpn"
	"golang.org/x/term"
)

// NewVPNCmd returns the 'vpn' cobra command.
func NewVPNCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpn",
		Short: "Manage VPN users and customer gateways",
	}
	cmd.AddCommand(newVPNCustomerGatewayCmd())
	cmd.AddCommand(newVPNUserCmd())
	return cmd
}

// ─── VPN Customer Gateway ─────────────────────────────────────────────────────

func newVPNCustomerGatewayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "customer-gateway",
		Short: "Manage VPN customer gateways",
	}
	cmd.AddCommand(newVPNCGListCmd())
	cmd.AddCommand(newVPNCGCreateCmd())
	cmd.AddCommand(newVPNCGUpdateCmd())
	cmd.AddCommand(newVPNCGDeleteCmd())
	return cmd
}

func newVPNCGListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List VPN customer gateways",
		Example: `  zcp vpn customer-gateway list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNCGList(cmd)
		},
	}
	return cmd
}

func runVPNCGList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewCustomerGatewayService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cgs, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("vpn customer-gateway list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "GATEWAY", "IKE POLICY", "CIDR"}
	rows := make([][]string, 0, len(cgs))
	for _, cg := range cgs {
		rows = append(rows, []string{
			cg.Slug,
			cg.Name,
			cg.Gateway,
			cg.IKEPolicy,
			cg.CIDRList,
		})
	}
	return printer.PrintTable(headers, rows)
}

func addCustomerGatewayFlags(cmd *cobra.Command, name, gateway, cidr, psk, ikePolicy, espPolicy *string,
	ikeLifetime, espLifetime, ikeEncryption, ikeHash, ikeVersion, espEncryption, espHash *string,
	forceEncap, splitConnection, dpd *bool) {
	cmd.Flags().StringVar(name, "name", "", "Customer gateway name")
	cmd.Flags().StringVar(gateway, "gateway", "", "Remote gateway IP address")
	cmd.Flags().StringVar(cidr, "cidr", "", "Remote CIDR list (comma-separated)")
	cmd.Flags().StringVar(psk, "psk", "", "IPSec pre-shared key")
	cmd.Flags().StringVar(ikePolicy, "ike-policy", "", "IKE policy")
	cmd.Flags().StringVar(espPolicy, "esp-policy", "", "ESP policy")
	cmd.Flags().StringVar(ikeLifetime, "ike-lifetime", "", "IKE lifetime")
	cmd.Flags().StringVar(espLifetime, "esp-lifetime", "", "ESP lifetime")
	cmd.Flags().StringVar(ikeEncryption, "ike-encryption", "", "IKE encryption algorithm")
	cmd.Flags().StringVar(ikeHash, "ike-hash", "", "IKE hash algorithm")
	cmd.Flags().StringVar(ikeVersion, "ike-version", "", "IKE version (optional)")
	cmd.Flags().StringVar(espEncryption, "esp-encryption", "", "ESP encryption algorithm")
	cmd.Flags().StringVar(espHash, "esp-hash", "", "ESP hash algorithm")
	cmd.Flags().BoolVar(forceEncap, "force-encap", false, "Force UDP encapsulation")
	cmd.Flags().BoolVar(splitConnection, "split-connection", false, "Enable split connection")
	cmd.Flags().BoolVar(dpd, "dpd", false, "Enable dead peer detection")
}

func newVPNCGCreateCmd() *cobra.Command {
	var (
		name, gateway, cidr, psk, ikePolicy, espPolicy string
		ikeLifetime, espLifetime                       string
		ikeEncryption, ikeHash, ikeVersion             string
		espEncryption, espHash                         string
		forceEncap, splitConnection, dpd               bool
		cloudProvider, region, project                 string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new VPN customer gateway",
		Example: `  zcp vpn customer-gateway create --name remote-gw --gateway 203.0.113.1 --cidr 192.168.1.0/24 --psk mykey --ike-policy aes128-sha1-dh5 --esp-policy aes128-sha1 --cloud-provider <slug> --region <slug> --project <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if gateway == "" {
				return fmt.Errorf("--gateway is required")
			}
			if cidr == "" {
				return fmt.Errorf("--cidr is required")
			}
			if psk == "" {
				return fmt.Errorf("--psk is required")
			}
			if ikePolicy == "" {
				return fmt.Errorf("--ike-policy is required")
			}
			if espPolicy == "" {
				return fmt.Errorf("--esp-policy is required")
			}
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			return runVPNCGCreate(cmd, vpn.CustomerGatewayRequest{
				Name:            name,
				Gateway:         gateway,
				CIDRList:        cidr,
				IPSecPSK:        psk,
				IKEPolicy:       ikePolicy,
				ESPPolicy:       espPolicy,
				IKELifetime:     ikeLifetime,
				ESPLifetime:     espLifetime,
				IKEEncryption:   ikeEncryption,
				IKEHash:         ikeHash,
				IKEVersion:      ikeVersion,
				ESPEncryption:   espEncryption,
				ESPHash:         espHash,
				ForceEncap:      forceEncap,
				SplitConnection: splitConnection,
				DPD:             dpd,
				CloudProvider:   cloudProvider,
				Region:          region,
				Project:         project,
			})
		},
	}
	addCustomerGatewayFlags(cmd, &name, &gateway, &cidr, &psk, &ikePolicy, &espPolicy,
		&ikeLifetime, &espLifetime, &ikeEncryption, &ikeHash, &ikeVersion, &espEncryption, &espHash,
		&forceEncap, &splitConnection, &dpd)
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	return cmd
}

func runVPNCGCreate(cmd *cobra.Command, req vpn.CustomerGatewayRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewCustomerGatewayService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cg, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("vpn customer-gateway create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", cg.Slug},
		{"Name", cg.Name},
		{"Gateway", cg.Gateway},
		{"IKE Policy", cg.IKEPolicy},
		{"IKE Lifetime", cg.IKELifetime},
		{"ESP Lifetime", cg.ESPLifetime},
		{"CIDR List", cg.CIDRList},
		{"IKE Version", cg.IKEVersion},
	}
	return printer.PrintTable(headers, rows)
}

func newVPNCGUpdateCmd() *cobra.Command {
	var (
		name, gateway, cidr, psk, ikePolicy, espPolicy string
		ikeLifetime, espLifetime                       string
		ikeEncryption, ikeHash, ikeVersion             string
		espEncryption, espHash                         string
		forceEncap, splitConnection, dpd               bool
	)

	cmd := &cobra.Command{
		Use:     "update <slug>",
		Short:   "Update a VPN customer gateway",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpn customer-gateway update <slug> --name new-name --psk newkey`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNCGUpdate(cmd, args[0], vpn.CustomerGatewayRequest{
				Name:            name,
				Gateway:         gateway,
				CIDRList:        cidr,
				IPSecPSK:        psk,
				IKEPolicy:       ikePolicy,
				ESPPolicy:       espPolicy,
				IKELifetime:     ikeLifetime,
				ESPLifetime:     espLifetime,
				IKEEncryption:   ikeEncryption,
				IKEHash:         ikeHash,
				IKEVersion:      ikeVersion,
				ESPEncryption:   espEncryption,
				ESPHash:         espHash,
				ForceEncap:      forceEncap,
				SplitConnection: splitConnection,
				DPD:             dpd,
			})
		},
	}
	addCustomerGatewayFlags(cmd, &name, &gateway, &cidr, &psk, &ikePolicy, &espPolicy,
		&ikeLifetime, &espLifetime, &ikeEncryption, &ikeHash, &ikeVersion, &espEncryption, &espHash,
		&forceEncap, &splitConnection, &dpd)
	return cmd
}

func runVPNCGUpdate(cmd *cobra.Command, slug string, req vpn.CustomerGatewayRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewCustomerGatewayService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cg, err := svc.Update(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("vpn customer-gateway update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", cg.Slug},
		{"Name", cg.Name},
		{"Gateway", cg.Gateway},
		{"IKE Policy", cg.IKEPolicy},
		{"IKE Lifetime", cg.IKELifetime},
		{"ESP Lifetime", cg.ESPLifetime},
		{"CIDR List", cg.CIDRList},
	}
	return printer.PrintTable(headers, rows)
}

func newVPNCGDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a VPN customer gateway",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpn customer-gateway delete <slug>
  zcp vpn customer-gateway delete <slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNCGDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPNCGDelete(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete VPN customer gateway %q? This action cannot be undone. [y/N]: ", slug)
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

	svc := vpn.NewCustomerGatewayService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		return fmt.Errorf("vpn customer-gateway delete: %w", err)
	}

	printer.Fprintf("VPN customer gateway %q deleted.\n", slug)
	return nil
}

// ─── VPN User ─────────────────────────────────────────────────────────────────

func newVPNUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage VPN users",
	}
	cmd.AddCommand(newVPNUserListCmd())
	cmd.AddCommand(newVPNUserCreateCmd())
	cmd.AddCommand(newVPNUserDeleteCmd())
	return cmd
}

func newVPNUserListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List VPN users",
		Example: `  zcp vpn user list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNUserList(cmd)
		},
	}
	return cmd
}

func runVPNUserList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewUserService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	users, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("vpn user list: %w", err)
	}

	headers := []string{"SLUG", "USERNAME", "STATUS"}
	rows := make([][]string, 0, len(users))
	for _, u := range users {
		rows = append(rows, []string{
			u.Slug,
			u.UserName,
			u.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVPNUserCreateCmd() *cobra.Command {
	var username, password string
	var cloudProvider, region, project string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new VPN user",
		Example: `  zcp vpn user create --username alice --cloud-provider <slug> --region <slug> --project <slug>
  zcp vpn user create --username alice --password secret --cloud-provider <slug> --region <slug> --project <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return fmt.Errorf("--username is required")
			}
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			if password == "" {
				// Prompt securely for password
				fmt.Fprint(os.Stderr, "Password: ")
				raw, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					// Fallback to plain stdin if terminal is not available
					fmt.Fprint(os.Stderr, "\nPassword (plain): ")
					scanner := bufio.NewScanner(os.Stdin)
					scanner.Scan()
					password = scanner.Text()
				} else {
					fmt.Fprintln(os.Stderr)
					password = string(raw)
				}
				if password == "" {
					return fmt.Errorf("password cannot be empty")
				}
			}
			return runVPNUserCreate(cmd, vpn.UserCreateRequest{
				Username:      username,
				Password:      password,
				CloudProvider: cloudProvider,
				Region:        region,
				Project:       project,
			})
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "VPN username (required)")
	cmd.Flags().StringVar(&password, "password", "", "VPN password (prompted if not provided)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	return cmd
}

func runVPNUserCreate(cmd *cobra.Command, req vpn.UserCreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewUserService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	u, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("vpn user create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", u.Slug},
		{"Username", u.UserName},
		{"Status", u.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newVPNUserDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a VPN user by slug",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpn user delete <slug>
  zcp vpn user delete <slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNUserDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPNUserDelete(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete VPN user %q? This action cannot be undone. [y/N]: ", slug)
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

	svc := vpn.NewUserService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		return fmt.Errorf("vpn user delete: %w", err)
	}

	printer.Fprintf("VPN user %q deleted.\n", slug)
	return nil
}
