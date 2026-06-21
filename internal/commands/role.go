package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/role"
)

// predefinedRoles are the built-in roles the API refuses to update or delete.
// Catching them client-side gives a clear message instead of an opaque 403.
var predefinedRoles = map[string]bool{
	"owner": true, "service-administrator": true, "service-viewer": true,
}

// NewRoleCmd returns the 'role' cobra command.
func NewRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage roles and their permissions",
	}
	cmd.AddCommand(newRoleListCmd())
	cmd.AddCommand(newRoleGetCmd())
	cmd.AddCommand(newRoleCreateCmd())
	cmd.AddCommand(newRoleUpdateCmd())
	cmd.AddCommand(newRoleDeleteCmd())
	return cmd
}

// ---------- list ----------

func newRoleListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List roles",
		Example: `  zcp role list`,
		RunE:    func(cmd *cobra.Command, args []string) error { return runRoleList(cmd) },
	}
	return cmd
}

func runRoleList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := role.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	roles, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("role list: %w", err)
	}

	if printer.Format() == output.FormatJSON || printer.Format() == output.FormatYAML {
		return printer.Print(roles)
	}

	headers := []string{"SLUG", "NAME", "DESCRIPTION", "USERS", "PREDEFINED"}
	rows := make([][]string, 0, len(roles))
	for _, r := range roles {
		predefined := "no"
		if predefinedRoles[r.Slug] {
			predefined = "yes"
		}
		rows = append(rows, []string{r.Slug, r.Name, r.Description, fmt.Sprintf("%d", len(r.Users)), predefined})
	}
	return printer.PrintTable(headers, rows)
}

// ---------- get ----------

func newRoleGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <slug>",
		Short:   "Show a role with its permissions and assigned users",
		Args:    exactArgs(1),
		Example: `  zcp role get service-administrator`,
		RunE:    func(cmd *cobra.Command, args []string) error { return runRoleGet(cmd, args[0]) },
	}
	return cmd
}

func runRoleGet(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := role.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	r, err := svc.Get(ctx, slug)
	if err != nil {
		return fmt.Errorf("role get: %w", err)
	}

	if printer.Format() == output.FormatJSON || printer.Format() == output.FormatYAML {
		return printer.Print(r)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Role:        %s (%s)\n", r.Name, r.Slug)
	fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", r.Description)
	if predefinedRoles[r.Slug] {
		fmt.Fprintf(cmd.OutOrStdout(), "Predefined:  yes (cannot be edited or deleted)\n")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nPermissions (%d):\n", len(r.Permissions))
	perms := make([]string, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, p.Slug)
	}
	sort.Strings(perms)
	for _, p := range perms {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", p)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nAssigned users (%d):\n", len(r.Users))
	for _, u := range r.Users {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s <%s>\n", u.Name, u.Email)
	}
	return nil
}

// ---------- create ----------

func newRoleCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		permissions []string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a role",
		Long: `Create a role from a set of permissions.

Permission slugs come from 'zcp permission list'. At least one permission is
required — a role with no permissions is rejected by the API and can leave an
unusable record behind.`,
		Example: `  zcp role create --name "VM Operator" \
    --permission virtual-machine-read --permission virtual-machine-manage \
    --description "Can run and manage VMs"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("--name is required")
			}
			if len(permissions) == 0 {
				return fmt.Errorf("at least one --permission is required (see 'zcp permission list')")
			}
			return runRoleCreate(cmd, role.CreateRequest{
				Name: name, Description: description, Permissions: permissions,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Role name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Role description")
	cmd.Flags().StringArrayVar(&permissions, "permission", nil, "Permission slug to grant (repeatable; at least one required)")
	return cmd
}

func runRoleCreate(cmd *cobra.Command, req role.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := role.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	r, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("role create: %w", err)
	}
	printer.Fprintf("Role %q created with %d permissions.\n", r.Slug, len(req.Permissions))
	return nil
}

// ---------- update ----------

func newRoleUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		permissions []string
	)
	cmd := &cobra.Command{
		Use:   "update <slug>",
		Short: "Update a role's name, description, or permissions",
		Long: `Update a role. Only the flags you pass change; the rest are preserved from
the current role. --permission REPLACES the role's full permission set (it is
not additive), so pass every permission the role should end up with.

Predefined roles (owner, service-administrator, service-viewer) cannot be edited.`,
		Args: exactArgs(1),
		Example: `  zcp role update vm-operator --description "Updated"
  zcp role update vm-operator --permission virtual-machine-read --permission dns-read`,
		RunE: func(cmd *cobra.Command, args []string) error {
			nameSet := cmd.Flags().Changed("name")
			descSet := cmd.Flags().Changed("description")
			permsSet := cmd.Flags().Changed("permission")
			if !nameSet && !descSet && !permsSet {
				return fmt.Errorf("nothing to update: pass --name, --description, and/or --permission")
			}
			return runRoleUpdate(cmd, args[0], name, description, permissions, nameSet, descSet, permsSet)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New role name")
	cmd.Flags().StringVar(&description, "description", "", "New role description")
	cmd.Flags().StringArrayVar(&permissions, "permission", nil, "Permission slugs to set (repeatable; REPLACES the current set)")
	return cmd
}

func runRoleUpdate(cmd *cobra.Command, slug, name, description string, permissions []string, nameSet, descSet, permsSet bool) error {
	if predefinedRoles[slug] {
		return fmt.Errorf("role %q is predefined and cannot be edited", slug)
	}

	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := role.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	// Load the current role so unchanged fields are preserved (PUT replaces the
	// whole role, including its permission set).
	current, err := svc.Get(ctx, slug)
	if err != nil {
		return fmt.Errorf("role update: %w", err)
	}

	req := role.UpdateRequest{Name: current.Name, Description: current.Description}
	req.Permissions = make([]string, 0, len(current.Permissions))
	for _, p := range current.Permissions {
		req.Permissions = append(req.Permissions, p.Slug)
	}
	if nameSet {
		req.Name = name
	}
	if descSet {
		req.Description = description
	}
	if permsSet {
		if len(permissions) == 0 {
			return fmt.Errorf("--permission cannot be set to an empty list; a role needs at least one permission")
		}
		req.Permissions = permissions
	}

	r, err := svc.Update(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("role update: %w", err)
	}
	printer.Fprintf("Role %q updated (%d permissions).\n", r.Slug, len(req.Permissions))
	return nil
}

// ---------- delete ----------

func newRoleDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a role",
		Args:  exactArgs(1),
		Example: `  zcp role delete vm-operator
  zcp role delete vm-operator --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoleDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runRoleDelete(cmd *cobra.Command, slug string, yes bool) error {
	if predefinedRoles[slug] {
		return fmt.Errorf("role %q is predefined and cannot be deleted", slug)
	}
	if !yes && !confirmAction(cmd, "Delete role %q?", slug) {
		fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
		return nil
	}

	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := role.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		// A missing role surfaces as a 500 "No query results for model" rather
		// than a clean 404, so match that too to keep delete idempotent.
		if apierrors.IsResourceNotFound(err) || strings.Contains(err.Error(), "No query results") {
			fmt.Fprintf(cmd.ErrOrStderr(), "Role %q not found — already deleted.\n", slug)
			return nil
		}
		return fmt.Errorf("role delete: %w", err)
	}
	printer.Fprintf("Role %q deleted.\n", slug)
	return nil
}
