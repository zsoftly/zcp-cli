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
		Short: "Manage VPN gateways, connections, and users",
	}
	cmd.AddCommand(newVPNGatewayCmd())
	cmd.AddCommand(newVPNCustomerGatewayCmd())
	cmd.AddCommand(newVPNConnectionCmd())
	cmd.AddCommand(newVPNUserCmd())
	return cmd
}

// ─── VPN Gateway ──────────────────────────────────────────────────────────────

func newVPNGatewayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "Manage VPN gateways",
	}
	cmd.AddCommand(newVPNGatewayListCmd())
	cmd.AddCommand(newVPNGatewayCreateCmd())
	cmd.AddCommand(newVPNGatewayDeleteCmd())
	return cmd
}

func newVPNGatewayListCmd() *cobra.Command {
	var zoneUUID, vpcUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VPN gateways in a zone",
		Example: `  zcp vpn gateway list --zone <uuid>
  zcp vpn gateway list --zone <uuid> --vpc <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runVPNGatewayList(cmd, zoneUUID, vpcUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&vpcUUID, "vpc", "", "Filter by VPC UUID")
	return cmd
}

func runVPNGatewayList(cmd *cobra.Command, zoneUUID, vpcUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewGatewayService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	gateways, err := svc.List(ctx, zoneUUID, "", vpcUUID)
	if err != nil {
		return fmt.Errorf("vpn gateway list: %w", err)
	}

	headers := []string{"UUID", "PUBLIC IP", "VPC", "STATUS"}
	rows := make([][]string, 0, len(gateways))
	for _, g := range gateways {
		rows = append(rows, []string{
			g.UUID,
			g.PublicIP,
			g.VPCUUID,
			g.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVPNGatewayCreateCmd() *cobra.Command {
	var vpcUUID string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a VPN gateway for a VPC",
		Example: `  zcp vpn gateway create --vpc <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vpcUUID == "" {
				return fmt.Errorf("--vpc is required")
			}
			return runVPNGatewayCreate(cmd, vpcUUID)
		},
	}
	cmd.Flags().StringVar(&vpcUUID, "vpc", "", "VPC UUID (required)")
	return cmd
}

func runVPNGatewayCreate(cmd *cobra.Command, vpcUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewGatewayService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	g, err := svc.Create(ctx, vpcUUID)
	if err != nil {
		return fmt.Errorf("vpn gateway create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", g.UUID},
		{"Public IP", g.PublicIP},
		{"VPC UUID", g.VPCUUID},
		{"Zone UUID", g.ZoneUUID},
		{"Status", g.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newVPNGatewayDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a VPN gateway",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpn gateway delete <uuid>
  zcp vpn gateway delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNGatewayDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPNGatewayDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete VPN gateway %q? This action cannot be undone. [y/N]: ", uuid)
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

	svc := vpn.NewGatewayService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("vpn gateway delete: %w", err)
	}

	printer.Fprintf("VPN gateway %q deleted.\n", uuid)
	return nil
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

	cgs, err := svc.List(ctx, "")
	if err != nil {
		return fmt.Errorf("vpn customer-gateway list: %w", err)
	}

	headers := []string{"UUID", "IKE POLICY", "ESP LIFETIME", "CIDR"}
	rows := make([][]string, 0, len(cgs))
	for _, cg := range cgs {
		rows = append(rows, []string{
			cg.UUID,
			cg.IKEPolicy,
			cg.ESPLifetime,
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
	)

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new VPN customer gateway",
		Example: `  zcp vpn customer-gateway create --name remote-gw --gateway 203.0.113.1 --cidr 192.168.1.0/24 --psk mykey --ike-policy aes128-sha1-dh5 --esp-policy aes128-sha1`,
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
			})
		},
	}
	addCustomerGatewayFlags(cmd, &name, &gateway, &cidr, &psk, &ikePolicy, &espPolicy,
		&ikeLifetime, &espLifetime, &ikeEncryption, &ikeHash, &ikeVersion, &espEncryption, &espHash,
		&forceEncap, &splitConnection, &dpd)
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
		{"UUID", cg.UUID},
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
		Use:     "update <uuid>",
		Short:   "Update a VPN customer gateway",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpn customer-gateway update <uuid> --name new-name --psk newkey`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNCGUpdate(cmd, vpn.CustomerGatewayUpdateRequest{
				UUID: args[0],
				CustomerGatewayRequest: vpn.CustomerGatewayRequest{
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
				},
			})
		},
	}
	addCustomerGatewayFlags(cmd, &name, &gateway, &cidr, &psk, &ikePolicy, &espPolicy,
		&ikeLifetime, &espLifetime, &ikeEncryption, &ikeHash, &ikeVersion, &espEncryption, &espHash,
		&forceEncap, &splitConnection, &dpd)
	return cmd
}

func runVPNCGUpdate(cmd *cobra.Command, req vpn.CustomerGatewayUpdateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewCustomerGatewayService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cg, err := svc.Update(ctx, req)
	if err != nil {
		return fmt.Errorf("vpn customer-gateway update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", cg.UUID},
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
		Use:   "delete <uuid>",
		Short: "Delete a VPN customer gateway",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpn customer-gateway delete <uuid>
  zcp vpn customer-gateway delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNCGDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPNCGDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete VPN customer gateway %q? This action cannot be undone. [y/N]: ", uuid)
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

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("vpn customer-gateway delete: %w", err)
	}

	printer.Fprintf("VPN customer gateway %q deleted.\n", uuid)
	return nil
}

// ─── VPN Connection ───────────────────────────────────────────────────────────

func newVPNConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connection",
		Short: "Manage VPN connections",
	}
	cmd.AddCommand(newVPNConnListCmd())
	cmd.AddCommand(newVPNConnCreateCmd())
	cmd.AddCommand(newVPNConnResetCmd())
	cmd.AddCommand(newVPNConnDeleteCmd())
	return cmd
}

func newVPNConnListCmd() *cobra.Command {
	var zoneUUID, vpcUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VPN connections in a zone",
		Example: `  zcp vpn connection list --zone <uuid>
  zcp vpn connection list --zone <uuid> --vpc <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runVPNConnList(cmd, zoneUUID, vpcUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&vpcUUID, "vpc", "", "Filter by VPC UUID")
	return cmd
}

func runVPNConnList(cmd *cobra.Command, zoneUUID, vpcUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewConnectionService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	conns, err := svc.List(ctx, zoneUUID, "", vpcUUID)
	if err != nil {
		return fmt.Errorf("vpn connection list: %w", err)
	}

	headers := []string{"UUID", "STATE", "IKE POLICY", "CUSTOMER GW", "VPN GW"}
	rows := make([][]string, 0, len(conns))
	for _, c := range conns {
		rows = append(rows, []string{
			c.UUID,
			c.State,
			c.IKEPolicy,
			c.CustomerGatewayUUID,
			c.VPNGatewayUUID,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVPNConnCreateCmd() *cobra.Command {
	var vpcUUID, customerGatewayUUID string
	var passive bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VPN connection",
		Example: `  zcp vpn connection create --vpc <uuid> --customer-gateway <uuid>
  zcp vpn connection create --vpc <uuid> --customer-gateway <uuid> --passive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vpcUUID == "" {
				return fmt.Errorf("--vpc is required")
			}
			if customerGatewayUUID == "" {
				return fmt.Errorf("--customer-gateway is required")
			}
			return runVPNConnCreate(cmd, vpn.ConnectionCreateRequest{
				VPCUUID:             vpcUUID,
				CustomerGatewayUUID: customerGatewayUUID,
				Passive:             passive,
			})
		},
	}
	cmd.Flags().StringVar(&vpcUUID, "vpc", "", "VPC UUID (required)")
	cmd.Flags().StringVar(&customerGatewayUUID, "customer-gateway", "", "Customer gateway UUID (required)")
	cmd.Flags().BoolVar(&passive, "passive", false, "Create connection in passive mode")
	return cmd
}

func runVPNConnCreate(cmd *cobra.Command, req vpn.ConnectionCreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewConnectionService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	c, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("vpn connection create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", c.UUID},
		{"State", c.State},
		{"IKE Policy", c.IKEPolicy},
		{"ESP Policy", c.ESPPolicy},
		{"Customer Gateway UUID", c.CustomerGatewayUUID},
		{"VPN Gateway UUID", c.VPNGatewayUUID},
		{"Zone UUID", c.ZoneUUID},
	}
	return printer.PrintTable(headers, rows)
}

func newVPNConnResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reset <uuid>",
		Short:   "Reset a VPN connection",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpn connection reset <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNConnReset(cmd, args[0])
		},
	}
	return cmd
}

func runVPNConnReset(cmd *cobra.Command, uuid string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewConnectionService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	c, err := svc.Reset(ctx, uuid)
	if err != nil {
		return fmt.Errorf("vpn connection reset: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", c.UUID},
		{"State", c.State},
		{"IKE Policy", c.IKEPolicy},
	}
	return printer.PrintTable(headers, rows)
}

func newVPNConnDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a VPN connection",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpn connection delete <uuid>
  zcp vpn connection delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNConnDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPNConnDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete VPN connection %q? This action cannot be undone. [y/N]: ", uuid)
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

	svc := vpn.NewConnectionService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("vpn connection delete: %w", err)
	}

	printer.Fprintf("VPN connection %q deleted.\n", uuid)
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

	users, err := svc.List(ctx, "")
	if err != nil {
		return fmt.Errorf("vpn user list: %w", err)
	}

	headers := []string{"UUID", "USERNAME", "STATUS"}
	rows := make([][]string, 0, len(users))
	for _, u := range users {
		rows = append(rows, []string{
			u.UUID,
			u.UserName,
			u.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVPNUserCreateCmd() *cobra.Command {
	var username, password string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new VPN user",
		Example: `  zcp vpn user create --username alice
  zcp vpn user create --username alice --password secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return fmt.Errorf("--username is required")
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
			return runVPNUserCreate(cmd, username, password)
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "VPN username (required)")
	cmd.Flags().StringVar(&password, "password", "", "VPN password (prompted if not provided)")
	return cmd
}

func runVPNUserCreate(cmd *cobra.Command, username, password string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpn.NewUserService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	u, err := svc.Create(ctx, username, password)
	if err != nil {
		return fmt.Errorf("vpn user create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", u.UUID},
		{"Username", u.UserName},
		{"Status", u.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newVPNUserDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <username>",
		Short: "Delete a VPN user by username",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpn user delete alice
  zcp vpn user delete alice --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNUserDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPNUserDelete(cmd *cobra.Command, username string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete VPN user %q? This action cannot be undone. [y/N]: ", username)
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

	if err := svc.Delete(ctx, username); err != nil {
		return fmt.Errorf("vpn user delete: %w", err)
	}

	printer.Fprintf("VPN user %q deleted.\n", username)
	return nil
}
