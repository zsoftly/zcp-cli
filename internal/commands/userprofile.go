package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/userprofile"
)

// NewUserProfileCmd returns the 'profile-info' cobra command.
func NewUserProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile-info",
		Short: "Manage user profile, company details, time settings, and API access",
	}
	cmd.AddCommand(newProfileInfoGetCmd())
	cmd.AddCommand(newProfileInfoUpdateCmd())
	cmd.AddCommand(newProfileInfoCompanyCmd())
	cmd.AddCommand(newProfileInfoTimeSettingsCmd())
	cmd.AddCommand(newProfileInfoEnableAPICmd())
	cmd.AddCommand(newProfileInfoDisableAPICmd())
	cmd.AddCommand(newProfileInfoLoginActivityCmd())
	cmd.AddCommand(newProfileInfoActivityLogsCmd())
	return cmd
}

// ─── Get ────────────────────────────────────────────────────────────────────

func newProfileInfoGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Show user profile",
		Example: `  zcp profile-info get`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileInfoGet(cmd)
		},
	}
	return cmd
}

func runProfileInfoGet(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := userprofile.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	p, err := svc.Get(ctx)
	if err != nil {
		return fmt.Errorf("profile-info get: %w", err)
	}

	u := p.User
	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", u.ID},
		{"Name", u.Name},
		{"Email", u.Email},
		{"User Type", u.UserType},
		{"Domain", u.Domain},
		{"2FA Enabled", fmt.Sprintf("%v", u.IsTwoFactor)},
		{"Last Login", u.LastLogin},
		{"Created", u.CreatedAt},
		{"Account CRN", u.Account.CRN},
		{"Account Status", u.Account.AccountStatus},
		{"Timezone", u.Account.Timezone},
	}
	if u.Company != nil {
		rows = append(rows, []string{"Company", u.Company.Name})
		rows = append(rows, []string{"Website", u.Company.Website})
	}
	return printer.PrintTable(headers, rows)
}

// ─── Update ─────────────────────────────────────────────────────────────────

func newProfileInfoUpdateCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update user profile",
		Example: `  zcp profile-info update --name "New Name"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return runProfileInfoUpdate(cmd, userprofile.UpdateProfileRequest{
				Name: name,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New display name (required)")
	return cmd
}

func runProfileInfoUpdate(cmd *cobra.Command, req userprofile.UpdateProfileRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := userprofile.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	p, err := svc.Update(ctx, req)
	if err != nil {
		return fmt.Errorf("profile-info update: %w", err)
	}

	u := p.User
	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", u.ID},
		{"Name", u.Name},
		{"Email", u.Email},
		{"Updated", u.UpdatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ─── Company ────────────────────────────────────────────────────────────────

func newProfileInfoCompanyCmd() *cobra.Command {
	var billingName, country, state, city, postalCode, line1, line2, gst string

	cmd := &cobra.Command{
		Use:     "company",
		Short:   "Update company/billing details",
		Example: `  zcp profile-info company --billing-name "ZSoftly Inc" --country CA --city Ottawa`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var req userprofile.UpdateCompanyRequest
			changed := false
			if cmd.Flags().Changed("billing-name") {
				req.BillingName = billingName
				changed = true
			}
			if cmd.Flags().Changed("country") {
				req.Country = country
				changed = true
			}
			if cmd.Flags().Changed("state") {
				req.State = state
				changed = true
			}
			if cmd.Flags().Changed("city") {
				req.City = city
				changed = true
			}
			if cmd.Flags().Changed("postal-code") {
				req.PostalCode = postalCode
				changed = true
			}
			if cmd.Flags().Changed("line1") {
				req.Line1 = line1
				changed = true
			}
			if cmd.Flags().Changed("line2") {
				req.Line2 = line2
				changed = true
			}
			if cmd.Flags().Changed("gst") {
				req.GST = gst
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one flag is required (e.g. --billing-name, --country)")
			}
			return runProfileInfoCompany(cmd, req)
		},
	}
	cmd.Flags().StringVar(&billingName, "billing-name", "", "Billing name")
	cmd.Flags().StringVar(&country, "country", "", "Country code")
	cmd.Flags().StringVar(&state, "state", "", "State/province")
	cmd.Flags().StringVar(&city, "city", "", "City")
	cmd.Flags().StringVar(&postalCode, "postal-code", "", "Postal code")
	cmd.Flags().StringVar(&line1, "line1", "", "Address line 1")
	cmd.Flags().StringVar(&line2, "line2", "", "Address line 2")
	cmd.Flags().StringVar(&gst, "gst", "", "GST/tax number")
	return cmd
}

func runProfileInfoCompany(cmd *cobra.Command, req userprofile.UpdateCompanyRequest) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := userprofile.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.UpdateCompany(ctx, req); err != nil {
		return fmt.Errorf("profile-info company: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Company details updated.\n")
	return nil
}

// ─── Time Settings ──────────────────────────────────────────────────────────

func newProfileInfoTimeSettingsCmd() *cobra.Command {
	var timezone, dateTimeFormat string

	cmd := &cobra.Command{
		Use:   "time-settings",
		Short: "Update time settings (timezone and date format)",
		Example: `  zcp profile-info time-settings --timezone "America/Toronto"
  zcp profile-info time-settings --timezone "UTC" --date-format "YYYY-MM-DD HH:mm:ss"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("timezone") && !cmd.Flags().Changed("date-format") {
				return fmt.Errorf("at least one flag is required (--timezone or --date-format)")
			}
			var req userprofile.TimeSettingsRequest
			if cmd.Flags().Changed("timezone") {
				req.Timezone = timezone
			}
			if cmd.Flags().Changed("date-format") {
				req.DateTimeFormat = dateTimeFormat
			}
			return runProfileInfoTimeSettings(cmd, req)
		},
	}
	cmd.Flags().StringVar(&timezone, "timezone", "", "Timezone (required, e.g. America/Toronto)")
	cmd.Flags().StringVar(&dateTimeFormat, "date-format", "", "Date-time format string")
	return cmd
}

func runProfileInfoTimeSettings(cmd *cobra.Command, req userprofile.TimeSettingsRequest) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := userprofile.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.UpdateTimeSettings(ctx, req); err != nil {
		return fmt.Errorf("profile-info time-settings: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Time settings updated.\n")
	return nil
}

// ─── Enable API ─────────────────────────────────────────────────────────────

func newProfileInfoEnableAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable-api",
		Short:   "Enable API access for the account",
		Example: `  zcp profile-info enable-api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileInfoEnableAPI(cmd)
		},
	}
	return cmd
}

func runProfileInfoEnableAPI(cmd *cobra.Command) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := userprofile.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.EnableAPI(ctx); err != nil {
		return fmt.Errorf("profile-info enable-api: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "API access enabled.\n")
	return nil
}

// ─── Disable API ────────────────────────────────────────────────────────────

func newProfileInfoDisableAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable-api",
		Short: "Disable API access for the account",
		Example: `  zcp profile-info disable-api
  zcp profile-info disable-api -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(cmd, "Disable API access for this account?") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Cancelled.")
				return nil
			}
			return runProfileInfoDisableAPI(cmd)
		},
	}
	return cmd
}

func runProfileInfoDisableAPI(cmd *cobra.Command) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := userprofile.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DisableAPI(ctx); err != nil {
		return fmt.Errorf("profile-info disable-api: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "API access disabled.\n")
	return nil
}

// ─── Login Activity ─────────────────────────────────────────────────────────

func newProfileInfoLoginActivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login-activity <crn>",
		Short:   "Show login activity logs",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp profile-info login-activity CRN-123456`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileInfoLoginActivity(cmd, args[0])
		},
	}
	return cmd
}

func runProfileInfoLoginActivity(cmd *cobra.Command, crn string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := userprofile.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	entries, err := svc.LoginActivity(ctx, crn)
	if err != nil {
		return fmt.Errorf("profile-info login-activity: %w", err)
	}

	headers := []string{"ID", "ACTION", "IP ADDRESS", "DETAILS", "CREATED"}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{
			e.ID,
			e.Action,
			e.IPAddress,
			e.Details,
			e.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ─── Activity Logs ──────────────────────────────────────────────────────────

func newProfileInfoActivityLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "activity-logs <crn>",
		Short:   "Show activity logs",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp profile-info activity-logs CRN-123456`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileInfoActivityLogs(cmd, args[0])
		},
	}
	return cmd
}

func runProfileInfoActivityLogs(cmd *cobra.Command, crn string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := userprofile.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	entries, err := svc.ActivityLogs(ctx, crn)
	if err != nil {
		return fmt.Errorf("profile-info activity-logs: %w", err)
	}

	headers := []string{"ID", "ACTION", "IP ADDRESS", "DETAILS", "CREATED"}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{
			e.ID,
			e.Action,
			e.IPAddress,
			e.Details,
			e.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}
