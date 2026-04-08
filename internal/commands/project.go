package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/project"
)

// NewProjectCmd returns the 'project' cobra command.
func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects, project icons, and project users",
	}
	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectCreateCmd())
	cmd.AddCommand(newProjectUpdateCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	cmd.AddCommand(newProjectDashboardCmd())
	cmd.AddCommand(newProjectIconCmd())
	cmd.AddCommand(newProjectUserCmd())
	return cmd
}

// ─── Project List ────────────────────────────────────────────────────────────

func newProjectListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all projects",
		Example: `  zcp project list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectList(cmd)
		},
	}
	return cmd
}

func runProjectList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := project.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	projects, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("project list: %w", err)
	}

	headers := []string{"ID", "NAME", "SLUG", "DESCRIPTION", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(projects))
	for _, p := range projects {
		rows = append(rows, []string{
			p.ID,
			p.Name,
			p.Slug,
			p.Description,
			fmt.Sprintf("%v", p.Status),
			p.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ─── Project Create ──────────────────────────────────────────────────────────

func newProjectCreateCmd() *cobra.Command {
	var name, description, icon, purpose string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Example: `  zcp project create --name my-project --icon cloud-15 --purpose "Development"
  zcp project create --name my-project --description "My project" --icon cloud-13 --purpose "Testing"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if icon == "" {
				icon = "cloud-13"
			}
			if purpose == "" {
				purpose = "Development & Testing"
			}
			return runProjectCreate(cmd, project.CreateRequest{
				Name:        name,
				Description: description,
				Icon:        icon,
				Purpose:     purpose,
				Status:      1,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Project description")
	cmd.Flags().StringVar(&icon, "icon", "cloud-13", "Icon name (see: zcp project icon list)")
	cmd.Flags().StringVar(&purpose, "purpose", "Development & Testing", "Project purpose")
	return cmd
}

func runProjectCreate(cmd *cobra.Command, req project.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := project.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	p, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("project create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", p.ID},
		{"Name", p.Name},
		{"Slug", p.Slug},
		{"Description", p.Description},
		{"Default", fmt.Sprintf("%v", p.Status)},
		{"Created At", p.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ─── Project Update ──────────────────────────────────────────────────────────

func newProjectUpdateCmd() *cobra.Command {
	var name, description string
	var iconID string

	cmd := &cobra.Command{
		Use:   "update <slug>",
		Short: "Update an existing project",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp project update my-project --name "New Name"
  zcp project update my-project --description "Updated description" --icon 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectUpdate(cmd, args[0], project.UpdateRequest{
				Name:        name,
				Description: description,
				IconID:      iconID,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New project name")
	cmd.Flags().StringVar(&description, "description", "", "New project description")
	cmd.Flags().StringVar(&iconID, "icon", "", "New icon ID (see: zcp project icon list)")
	return cmd
}

func runProjectUpdate(cmd *cobra.Command, slug string, req project.UpdateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := project.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	p, err := svc.Update(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("project update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", p.ID},
		{"Name", p.Name},
		{"Slug", p.Slug},
		{"Description", p.Description},
		{"Default", fmt.Sprintf("%v", p.Status)},
		{"Updated At", p.UpdatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ─── Project Delete ─────────────────────────────────────────────────────────

func newProjectDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp project delete my-project
  zcp project delete my-project --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !confirmAction(cmd, "Delete project %q?", slug) {
				fmt.Fprintln(cmd.ErrOrStderr(), "Cancelled.")
				return nil
			}
			return runProjectDelete(cmd, slug)
		},
	}
	return cmd
}

func runProjectDelete(cmd *cobra.Command, slug string) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := project.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		return fmt.Errorf("project delete: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Project %q deleted.\n", slug)
	return nil
}

// ─── Project Dashboard ───────────────────────────────────────────────────────

func newProjectDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dashboard <slug>",
		Short:   "Show project dashboard services",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp project dashboard my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectDashboard(cmd, args[0])
		},
	}
	return cmd
}

func runProjectDashboard(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := project.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	services, err := svc.Dashboard(ctx, slug)
	if err != nil {
		return fmt.Errorf("project dashboard: %w", err)
	}

	headers := []string{"NAME", "TYPE", "STATUS", "COUNT"}
	rows := make([][]string, 0, len(services))
	for _, s := range services {
		rows = append(rows, []string{
			s.Name,
			s.Type,
			s.Status,
			strconv.Itoa(s.Count),
		})
	}
	return printer.PrintTable(headers, rows)
}

// ─── Project Icon ────────────────────────────────────────────────────────────

func newProjectIconCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "icon",
		Short: "Manage project icons",
	}
	cmd.AddCommand(newProjectIconListCmd())
	return cmd
}

func newProjectIconListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List available project icons",
		Example: `  zcp project icon list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectIconList(cmd)
		},
	}
	return cmd
}

func runProjectIconList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := project.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	icons, err := svc.ListIcons(ctx)
	if err != nil {
		return fmt.Errorf("project icon list: %w", err)
	}

	headers := []string{"ID", "NAME", "URL"}
	rows := make([][]string, 0, len(icons))
	for _, ic := range icons {
		rows = append(rows, []string{
			ic.ID,
			ic.Name,
			ic.URL,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ─── Project User ────────────────────────────────────────────────────────────

func newProjectUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage project users",
	}
	cmd.AddCommand(newProjectUserListCmd())
	cmd.AddCommand(newProjectUserAddCmd())
	return cmd
}

func newProjectUserListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <slug>",
		Short:   "List users in a project",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp project user list my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectUserList(cmd, args[0])
		},
	}
	return cmd
}

func runProjectUserList(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := project.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	users, err := svc.ListUsers(ctx, slug)
	if err != nil {
		return fmt.Errorf("project user list: %w", err)
	}

	headers := []string{"ID", "NAME", "EMAIL", "ROLE"}
	rows := make([][]string, 0, len(users))
	for _, u := range users {
		rows = append(rows, []string{
			u.ID,
			u.Name,
			u.Email,
			u.Role,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newProjectUserAddCmd() *cobra.Command {
	var email, role string

	cmd := &cobra.Command{
		Use:   "add <slug>",
		Short: "Add a user to a project",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp project user add my-project --email alice@example.com
  zcp project user add my-project --email alice@example.com --role admin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			return runProjectUserAdd(cmd, args[0], project.AddUserRequest{
				Email: email,
				Role:  role,
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "User email address (required)")
	cmd.Flags().StringVar(&role, "role", "", "User role (e.g. admin, member)")
	return cmd
}

func runProjectUserAdd(cmd *cobra.Command, slug string, req project.AddUserRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := project.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	u, err := svc.AddUser(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("project user add: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", u.ID},
		{"Name", u.Name},
		{"Email", u.Email},
		{"Role", u.Role},
	}
	return printer.PrintTable(headers, rows)
}
