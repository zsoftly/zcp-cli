package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/subuser"
)

// errSubUserNotFound marks the "no match in the listing" case so callers can
// distinguish a genuinely-absent user (idempotent delete) from a List failure
// (auth/network/5xx), which must not be reported as a successful deletion.
var errSubUserNotFound = errors.New("sub-user not found")

// NewSubUserCmd returns the 'sub-user' cobra command.
func NewSubUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sub-user",
		Aliases: []string{"subuser"},
		Short:   "Manage account sub-users (additional users under your account)",
	}
	cmd.AddCommand(newSubUserListCmd())
	cmd.AddCommand(newSubUserCreateCmd())
	cmd.AddCommand(newSubUserUpdateCmd())
	cmd.AddCommand(newSubUserBlockCmd(true))
	cmd.AddCommand(newSubUserBlockCmd(false))
	cmd.AddCommand(newSubUserDeleteCmd())
	return cmd
}

// resolveSubUser maps a reference (UUID or email) to a sub-user. The API has no
// single-user GET, so resolution is done by listing and matching client-side.
func resolveSubUser(ctx context.Context, svc *subuser.Service, ref string) (*subuser.SubUser, error) {
	users, err := svc.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].ID == ref || strings.EqualFold(users[i].Email, ref) {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %q — run 'zcp sub-user list' to see IDs and emails", errSubUserNotFound, ref)
}

// ---------- list ----------

func newSubUserListCmd() *cobra.Command {
	var roleSlug string
	var blocked bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sub-users",
		Example: `  zcp sub-user list
  zcp sub-user list --role service-administrator
  zcp sub-user list --blocked`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSubUserList(cmd, roleSlug, blocked, cmd.Flags().Changed("blocked"))
		},
	}
	cmd.Flags().StringVar(&roleSlug, "role", "", "Only show sub-users with this role slug")
	cmd.Flags().BoolVar(&blocked, "blocked", false, "Only show blocked sub-users")
	return cmd
}

func runSubUserList(cmd *cobra.Command, roleSlug string, blocked, blockedSet bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := subuser.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	users, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("sub-user list: %w", err)
	}

	// Server-side filters are ignored by the API, so filter client-side.
	if roleSlug != "" || blockedSet {
		filtered := users[:0:0]
		for _, u := range users {
			if roleSlug != "" && u.RoleSlug() != roleSlug {
				continue
			}
			if blockedSet && u.IsBlocked != blocked {
				continue
			}
			filtered = append(filtered, u)
		}
		users = filtered
	}

	if printer.Format() == output.FormatJSON || printer.Format() == output.FormatYAML {
		return printer.Print(users)
	}

	headers := []string{"ID", "NAME", "EMAIL", "ROLE", "STATUS", "BLOCKED"}
	rows := make([][]string, 0, len(users))
	for _, u := range users {
		rows = append(rows, []string{
			u.ID, u.Name, u.Email, u.RoleSlug(), u.UserStatus, fmt.Sprintf("%t", u.IsBlocked),
		})
	}
	return printer.PrintTable(headers, rows)
}

// ---------- create ----------

func newSubUserCreateCmd() *cobra.Command {
	var (
		name     string
		email    string
		password string
		roleSlug string
		projects []string
		partner  bool
		blocked  bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a sub-user",
		Long: `Create a sub-user under your account.

Requirements:
  --email     must be a valid company email address
  --password  at least 8 characters with upper, lower, number, and special char
  --role      a role slug (see 'zcp role list')
  --project   one or more project slugs (repeatable; see 'zcp project list')`,
		Example: `  zcp sub-user create --name "Jane Doe" --email jane@yourco.com \
    --password 'S3cret!pass' --role service-viewer --project default-9`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ordered so the first missing flag is reported deterministically.
			required := []struct{ flag, val string }{
				{"--name", name}, {"--email", email}, {"--role", roleSlug},
			}
			for _, r := range required {
				if strings.TrimSpace(r.val) == "" {
					return fmt.Errorf("%s is required", r.flag)
				}
			}
			if len(projects) == 0 {
				return fmt.Errorf("at least one --project is required (see 'zcp project list')")
			}
			// Prompt for the password when omitted so it need not appear in shell
			// history or the process list. Hidden input on a TTY (see prompt()).
			if strings.TrimSpace(password) == "" {
				entered, perr := prompt("Password: ", true)
				if perr != nil {
					return perr
				}
				password = entered
			}
			if strings.TrimSpace(password) == "" {
				return fmt.Errorf("--password is required (pass --password or enter it at the prompt)")
			}
			return runSubUserCreate(cmd, subuser.CreateRequest{
				Name:           name,
				Email:          email,
				Password:       password,
				Role:           roleSlug,
				Projects:       projects,
				AuthUser:       "customer",
				IsUserPassword: true,
				IsPartner:      partner,
				IsBlocked:      blocked,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Full name (required)")
	cmd.Flags().StringVar(&email, "email", "", "Company email address (required)")
	cmd.Flags().StringVar(&password, "password", "", "Initial password (8+ chars, mixed case, number, symbol; prompted if omitted)")
	cmd.Flags().StringVar(&roleSlug, "role", "", "Role slug (required; see 'zcp role list')")
	cmd.Flags().StringArrayVar(&projects, "project", nil, "Project slug to grant access to (repeatable; required)")
	cmd.Flags().BoolVar(&partner, "partner", false, "Mark the sub-user as a partner")
	cmd.Flags().BoolVar(&blocked, "blocked", false, "Create the sub-user in a blocked state")
	return cmd
}

func runSubUserCreate(cmd *cobra.Command, req subuser.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := subuser.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	u, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("sub-user create: %w", err)
	}
	if u.ID != "" {
		printer.Fprintf("Sub-user %q created (%s).\n", req.Email, u.ID)
	} else {
		printer.Fprintf("Sub-user %q created.\n", req.Email)
	}
	return nil
}

// ---------- update ----------

func newSubUserUpdateCmd() *cobra.Command {
	var (
		name     string
		email    string
		roleSlug string
		projects []string
		partner  bool
	)
	cmd := &cobra.Command{
		Use:   "update <id-or-email>",
		Short: "Update a sub-user's name, email, role, or projects",
		Long: `Update a sub-user. Only the flags you pass change; the rest are preserved
from the current record. --project REPLACES the project list.`,
		Args: exactArgs(1),
		Example: `  zcp sub-user update jane@yourco.com --role service-administrator
  zcp sub-user update jane@yourco.com --project default-9 --project prodacc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("email") &&
				!cmd.Flags().Changed("role") && !cmd.Flags().Changed("project") &&
				!cmd.Flags().Changed("partner") {
				return fmt.Errorf("nothing to update: pass --name, --email, --role, --project, and/or --partner")
			}
			return runSubUserUpdate(cmd, args[0], name, email, roleSlug, projects, partner)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New full name")
	cmd.Flags().StringVar(&email, "email", "", "New email address")
	cmd.Flags().StringVar(&roleSlug, "role", "", "New role slug")
	cmd.Flags().StringArrayVar(&projects, "project", nil, "Project slugs to set (repeatable; REPLACES the current set)")
	cmd.Flags().BoolVar(&partner, "partner", false, "Set the partner flag")
	return cmd
}

func runSubUserUpdate(cmd *cobra.Command, ref, name, email, roleSlug string, projects []string, partner bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := subuser.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	current, err := resolveSubUser(ctx, svc, ref)
	if err != nil {
		return fmt.Errorf("sub-user update: %w", err)
	}

	// Email and projects are required on every update; default them to the
	// current values so a single-field change does not clear them.
	req := subuser.UpdateRequest{
		Name:      current.Name,
		Email:     current.Email,
		Role:      current.RoleSlug(),
		Projects:  current.ProjectSlugs(),
		IsBlocked: current.IsBlocked,
	}
	if cmd.Flags().Changed("name") {
		req.Name = name
	}
	if cmd.Flags().Changed("email") {
		req.Email = email
	}
	if cmd.Flags().Changed("role") {
		req.Role = roleSlug
	}
	if cmd.Flags().Changed("project") {
		if len(projects) == 0 {
			return fmt.Errorf("--project cannot be set to an empty list")
		}
		req.Projects = projects
	}
	if cmd.Flags().Changed("partner") {
		req.IsPartner = &partner
	}

	if _, err := svc.Update(ctx, current.ID, req); err != nil {
		return fmt.Errorf("sub-user update: %w", err)
	}
	// Report the email the record now has (req.Email), which differs from
	// current.Email when --email was changed.
	printer.Fprintf("Sub-user %q updated.\n", req.Email)
	return nil
}

// ---------- block / unblock ----------

func newSubUserBlockCmd(block bool) *cobra.Command {
	use, short := "unblock <id-or-email>", "Unblock a sub-user (restore access)"
	if block {
		use, short = "block <id-or-email>", "Block a sub-user (revoke access without deleting)"
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSubUserSetBlocked(cmd, args[0], block)
		},
	}
	return cmd
}

func runSubUserSetBlocked(cmd *cobra.Command, ref string, block bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := subuser.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	current, err := resolveSubUser(ctx, svc, ref)
	if err != nil {
		return err
	}
	verb := "blocked"
	if !block {
		verb = "unblocked"
	}
	if current.IsBlocked == block {
		printer.Fprintf("Sub-user %q already %s.\n", current.Email, verb)
		return nil
	}

	req := subuser.UpdateRequest{
		Name:      current.Name,
		Email:     current.Email,
		Role:      current.RoleSlug(),
		Projects:  current.ProjectSlugs(),
		IsBlocked: block,
	}
	if _, err := svc.Update(ctx, current.ID, req); err != nil {
		return fmt.Errorf("sub-user %s: %w", strings.TrimSuffix(verb, "ed"), err)
	}
	printer.Fprintf("Sub-user %q %s.\n", current.Email, verb)
	return nil
}

// ---------- delete ----------

func newSubUserDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <id-or-email>",
		Short: "Delete a sub-user",
		Args:  exactArgs(1),
		Example: `  zcp sub-user delete jane@yourco.com
  zcp sub-user delete jane@yourco.com --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSubUserDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runSubUserDelete(cmd *cobra.Command, ref string, yes bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := subuser.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	current, err := resolveSubUser(ctx, svc, ref)
	if err != nil {
		// Only a genuine no-match is idempotent success; a List failure
		// (auth/network/5xx) must surface, not be reported as a deletion.
		if errors.Is(err, errSubUserNotFound) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Sub-user %q not found — already deleted.\n", ref)
			return nil
		}
		return fmt.Errorf("sub-user delete: %w", err)
	}

	if !yes && !confirmAction(cmd, "Delete sub-user %q (%s)?", current.Email, current.ID) {
		fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
		return nil
	}

	if err := svc.Delete(ctx, current.ID); err != nil {
		if apierrors.IsResourceNotFound(err) || strings.Contains(err.Error(), "No query results") {
			fmt.Fprintf(cmd.ErrOrStderr(), "Sub-user %q not found — already deleted.\n", current.Email)
			return nil
		}
		return fmt.Errorf("sub-user delete: %w", err)
	}
	printer.Fprintf("Sub-user %q deleted.\n", current.Email)
	return nil
}
