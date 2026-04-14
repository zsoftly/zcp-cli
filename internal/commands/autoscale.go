package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/autoscale"
)

// NewAutoscaleCmd returns the 'autoscale' cobra command.
func NewAutoscaleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autoscale",
		Short: "Manage VM autoscale groups, policies, and conditions",
	}
	cmd.AddCommand(newAutoscaleListCmd())
	cmd.AddCommand(newAutoscaleCreateCmd())
	cmd.AddCommand(newAutoscaleEnableCmd())
	cmd.AddCommand(newAutoscaleDisableCmd())
	cmd.AddCommand(newAutoscaleChangePlanCmd())
	cmd.AddCommand(newAutoscaleChangeTemplateCmd())
	cmd.AddCommand(newAutoscalePolicyCmd())
	cmd.AddCommand(newAutoscaleConditionCmd())
	return cmd
}

// ---------------------------------------------------------------------------
// autoscale list
// ---------------------------------------------------------------------------

func newAutoscaleListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List autoscale groups",
		Example: `  zcp autoscale list
  zcp autoscale list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAutoscaleList(cmd)
		},
	}
	return cmd
}

func runAutoscaleList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	groups, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("autoscale list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "STATE", "PLAN", "TEMPLATE", "MIN", "MAX", "CURRENT", "ZONE"}
	rows := make([][]string, 0, len(groups))
	for _, g := range groups {
		rows = append(rows, []string{
			g.Slug,
			g.Name,
			g.State,
			g.Plan,
			g.Template,
			strconv.Itoa(g.MinInstances),
			strconv.Itoa(g.MaxInstances),
			strconv.Itoa(g.CurrentCount),
			g.ZoneSlug,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale create
// ---------------------------------------------------------------------------

func newAutoscaleCreateCmd() *cobra.Command {
	var (
		name           string
		plan           string
		template       string
		minInstances   int
		maxInstances   int
		cooldownPeriod int
		zoneSlug       string
		networkSlug    string
		cloudProvider  string
		region         string
		project        string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new autoscale group",
		Example: `  zcp autoscale create --name web-group --plan small --template ubuntu-22 --min 1 --max 5 --zone yow-1 --cloud-provider <slug> --region <slug> --project <slug>
  zcp autoscale create --name web-group --plan small --template ubuntu-22 --min 2 --max 10 --zone yow-1 --network default --cooldown 300 --cloud-provider <slug> --region <slug> --project <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			if template == "" {
				return fmt.Errorf("--template is required")
			}
			if zoneSlug == "" {
				return fmt.Errorf("--zone is required")
			}
			cloudProvider = resolveCloudProvider(cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			region = resolveRegion(region)
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			if minInstances < 0 {
				return fmt.Errorf("--min must be >= 0")
			}
			if maxInstances < minInstances {
				return fmt.Errorf("--max must be >= --min")
			}
			return runAutoscaleCreate(cmd, autoscale.CreateRequest{
				Name:           name,
				Plan:           plan,
				Template:       template,
				MinInstances:   minInstances,
				MaxInstances:   maxInstances,
				CooldownPeriod: cooldownPeriod,
				ZoneSlug:       zoneSlug,
				NetworkSlug:    networkSlug,
				CloudProvider:  cloudProvider,
				Region:         region,
				Project:        project,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Autoscale group name (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Compute plan slug (required)")
	cmd.Flags().StringVar(&template, "template", "", "Template slug (required)")
	cmd.Flags().IntVar(&minInstances, "min", 1, "Minimum number of instances")
	cmd.Flags().IntVar(&maxInstances, "max", 1, "Maximum number of instances")
	cmd.Flags().IntVar(&cooldownPeriod, "cooldown", 0, "Cooldown period in seconds between scaling actions")
	cmd.Flags().StringVar(&zoneSlug, "zone", "", "Zone slug (required)")
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	return cmd
}

func runAutoscaleCreate(cmd *cobra.Command, req autoscale.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	group, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("autoscale create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", group.Slug},
		{"Name", group.Name},
		{"State", group.State},
		{"Plan", group.Plan},
		{"Template", group.Template},
		{"Min Instances", strconv.Itoa(group.MinInstances)},
		{"Max Instances", strconv.Itoa(group.MaxInstances)},
		{"Current Count", strconv.Itoa(group.CurrentCount)},
		{"Cooldown Period", strconv.Itoa(group.CooldownPeriod)},
		{"Zone", group.ZoneSlug},
		{"Network", group.NetworkSlug},
	}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale enable / disable
// ---------------------------------------------------------------------------

func newAutoscaleEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable <slug>",
		Short:   "Enable an autoscale group",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp autoscale enable web-group`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAutoscaleEnable(cmd, args[0])
		},
	}
	return cmd
}

func runAutoscaleEnable(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	group, err := svc.Enable(ctx, slug)
	if err != nil {
		return fmt.Errorf("autoscale enable: %w", err)
	}

	headers := []string{"SLUG", "NAME", "STATE"}
	rows := [][]string{{group.Slug, group.Name, group.State}}
	return printer.PrintTable(headers, rows)
}

func newAutoscaleDisableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disable <slug>",
		Short:   "Disable an autoscale group",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp autoscale disable web-group`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAutoscaleDisable(cmd, args[0])
		},
	}
	return cmd
}

func runAutoscaleDisable(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	group, err := svc.Disable(ctx, slug)
	if err != nil {
		return fmt.Errorf("autoscale disable: %w", err)
	}

	headers := []string{"SLUG", "NAME", "STATE"}
	rows := [][]string{{group.Slug, group.Name, group.State}}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale change-plan
// ---------------------------------------------------------------------------

func newAutoscaleChangePlanCmd() *cobra.Command {
	var plan string

	cmd := &cobra.Command{
		Use:     "change-plan <slug>",
		Short:   "Change the compute plan of an autoscale group",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp autoscale change-plan web-group --plan medium`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			return runAutoscaleChangePlan(cmd, args[0], plan)
		},
	}
	cmd.Flags().StringVar(&plan, "plan", "", "New compute plan slug (required)")
	return cmd
}

func runAutoscaleChangePlan(cmd *cobra.Command, slug, plan string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	group, err := svc.ChangePlan(ctx, slug, plan)
	if err != nil {
		return fmt.Errorf("autoscale change-plan: %w", err)
	}

	headers := []string{"SLUG", "NAME", "PLAN", "STATE"}
	rows := [][]string{{group.Slug, group.Name, group.Plan, group.State}}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale change-template
// ---------------------------------------------------------------------------

func newAutoscaleChangeTemplateCmd() *cobra.Command {
	var template string

	cmd := &cobra.Command{
		Use:     "change-template <slug>",
		Short:   "Change the template of an autoscale group",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp autoscale change-template web-group --template ubuntu-24`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if template == "" {
				return fmt.Errorf("--template is required")
			}
			return runAutoscaleChangeTemplate(cmd, args[0], template)
		},
	}
	cmd.Flags().StringVar(&template, "template", "", "New template slug (required)")
	return cmd
}

func runAutoscaleChangeTemplate(cmd *cobra.Command, slug, template string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	group, err := svc.ChangeTemplate(ctx, slug, template)
	if err != nil {
		return fmt.Errorf("autoscale change-template: %w", err)
	}

	headers := []string{"SLUG", "NAME", "TEMPLATE", "STATE"}
	rows := [][]string{{group.Slug, group.Name, group.Template, group.State}}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale policy (subcommand group)
// ---------------------------------------------------------------------------

func newAutoscalePolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage scale-up policies",
	}
	cmd.AddCommand(newPolicyCreateCmd())
	cmd.AddCommand(newPolicyUpdateCmd())
	cmd.AddCommand(newPolicyDeleteCmd())
	return cmd
}

// ---------------------------------------------------------------------------
// autoscale policy create
// ---------------------------------------------------------------------------

func newPolicyCreateCmd() *cobra.Command {
	var (
		name        string
		metric      string
		operator    string
		threshold   int
		duration    int
		scaleAmount int
		cooldown    int
	)

	cmd := &cobra.Command{
		Use:   "create <group-slug>",
		Short: "Create a scale-up policy for an autoscale group",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp autoscale policy create web-group --name cpu-high --metric cpu --operator gte --threshold 80 --duration 300 --scale-amount 2
  zcp autoscale policy create web-group --name mem-high --metric memory --operator gte --threshold 90 --duration 120 --scale-amount 1 --cooldown 600`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if metric == "" {
				return fmt.Errorf("--metric is required")
			}
			if operator == "" {
				return fmt.Errorf("--operator is required")
			}
			if threshold <= 0 {
				return fmt.Errorf("--threshold must be > 0")
			}
			if duration <= 0 {
				return fmt.Errorf("--duration must be > 0")
			}
			if scaleAmount <= 0 {
				return fmt.Errorf("--scale-amount must be > 0")
			}
			return runPolicyCreate(cmd, args[0], autoscale.PolicyRequest{
				Name:        name,
				Metric:      metric,
				Operator:    operator,
				Threshold:   threshold,
				Duration:    duration,
				ScaleAmount: scaleAmount,
				Cooldown:    cooldown,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Policy name (required)")
	cmd.Flags().StringVar(&metric, "metric", "", "Metric to monitor, e.g. cpu, memory (required)")
	cmd.Flags().StringVar(&operator, "operator", "", "Comparison operator, e.g. gte, lte, gt, lt (required)")
	cmd.Flags().IntVar(&threshold, "threshold", 0, "Threshold value as a percentage (required)")
	cmd.Flags().IntVar(&duration, "duration", 0, "Duration in seconds the condition must hold (required)")
	cmd.Flags().IntVar(&scaleAmount, "scale-amount", 0, "Number of instances to add (required)")
	cmd.Flags().IntVar(&cooldown, "cooldown", 0, "Cooldown in seconds before this policy can trigger again")
	return cmd
}

func runPolicyCreate(cmd *cobra.Command, slug string, req autoscale.PolicyRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	policy, err := svc.CreatePolicy(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("autoscale policy create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", policy.ID},
		{"Name", policy.Name},
		{"Metric", policy.Metric},
		{"Operator", policy.Operator},
		{"Threshold", strconv.Itoa(policy.Threshold)},
		{"Duration", strconv.Itoa(policy.Duration)},
		{"Scale Amount", strconv.Itoa(policy.ScaleAmount)},
		{"Cooldown", strconv.Itoa(policy.Cooldown)},
	}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale policy update
// ---------------------------------------------------------------------------

func newPolicyUpdateCmd() *cobra.Command {
	var (
		policyID    int
		name        string
		metric      string
		operator    string
		threshold   int
		duration    int
		scaleAmount int
		cooldown    int
	)

	cmd := &cobra.Command{
		Use:     "update <group-slug>",
		Short:   "Update a scale-up policy for an autoscale group",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp autoscale policy update web-group --policy-id 42 --name cpu-high --metric cpu --operator gte --threshold 85 --duration 300 --scale-amount 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if policyID <= 0 {
				return fmt.Errorf("--policy-id is required and must be > 0")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if metric == "" {
				return fmt.Errorf("--metric is required")
			}
			if operator == "" {
				return fmt.Errorf("--operator is required")
			}
			if threshold <= 0 {
				return fmt.Errorf("--threshold must be > 0")
			}
			if duration <= 0 {
				return fmt.Errorf("--duration must be > 0")
			}
			if scaleAmount <= 0 {
				return fmt.Errorf("--scale-amount must be > 0")
			}
			return runPolicyUpdate(cmd, args[0], policyID, autoscale.PolicyRequest{
				Name:        name,
				Metric:      metric,
				Operator:    operator,
				Threshold:   threshold,
				Duration:    duration,
				ScaleAmount: scaleAmount,
				Cooldown:    cooldown,
			})
		},
	}
	cmd.Flags().IntVar(&policyID, "policy-id", 0, "Policy ID to update (required)")
	cmd.Flags().StringVar(&name, "name", "", "Policy name (required)")
	cmd.Flags().StringVar(&metric, "metric", "", "Metric to monitor (required)")
	cmd.Flags().StringVar(&operator, "operator", "", "Comparison operator (required)")
	cmd.Flags().IntVar(&threshold, "threshold", 0, "Threshold value as a percentage (required)")
	cmd.Flags().IntVar(&duration, "duration", 0, "Duration in seconds (required)")
	cmd.Flags().IntVar(&scaleAmount, "scale-amount", 0, "Number of instances to add (required)")
	cmd.Flags().IntVar(&cooldown, "cooldown", 0, "Cooldown in seconds")
	return cmd
}

func runPolicyUpdate(cmd *cobra.Command, slug string, policyID int, req autoscale.PolicyRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	policy, err := svc.UpdatePolicy(ctx, slug, policyID, req)
	if err != nil {
		return fmt.Errorf("autoscale policy update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", policy.ID},
		{"Name", policy.Name},
		{"Metric", policy.Metric},
		{"Operator", policy.Operator},
		{"Threshold", strconv.Itoa(policy.Threshold)},
		{"Duration", strconv.Itoa(policy.Duration)},
		{"Scale Amount", strconv.Itoa(policy.ScaleAmount)},
		{"Cooldown", strconv.Itoa(policy.Cooldown)},
	}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale policy delete
// ---------------------------------------------------------------------------

func newPolicyDeleteCmd() *cobra.Command {
	var (
		policyID int
		yes      bool
	)

	cmd := &cobra.Command{
		Use:   "delete <group-slug>",
		Short: "Delete a scale-up policy from an autoscale group",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp autoscale policy delete web-group --policy-id 42
  zcp autoscale policy delete web-group --policy-id 42 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if policyID <= 0 {
				return fmt.Errorf("--policy-id is required and must be > 0")
			}
			return runPolicyDelete(cmd, args[0], policyID, yes)
		},
	}
	cmd.Flags().IntVar(&policyID, "policy-id", 0, "Policy ID to delete (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runPolicyDelete(cmd *cobra.Command, slug string, policyID int, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete scale-up policy %d from autoscale group %q? [y/N]: ", policyID, slug)
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

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DeletePolicy(ctx, slug, policyID); err != nil {
		return fmt.Errorf("autoscale policy delete: %w", err)
	}

	printer.Fprintf("Scale-up policy %d deleted from autoscale group %q.\n", policyID, slug)
	return nil
}

// ---------------------------------------------------------------------------
// autoscale condition (subcommand group)
// ---------------------------------------------------------------------------

func newAutoscaleConditionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "condition",
		Short: "Manage scale-down conditions",
	}
	cmd.AddCommand(newConditionCreateCmd())
	cmd.AddCommand(newConditionUpdateCmd())
	cmd.AddCommand(newConditionDeleteCmd())
	return cmd
}

// ---------------------------------------------------------------------------
// autoscale condition create
// ---------------------------------------------------------------------------

func newConditionCreateCmd() *cobra.Command {
	var (
		name        string
		metric      string
		operator    string
		threshold   int
		duration    int
		scaleAmount int
		cooldown    int
	)

	cmd := &cobra.Command{
		Use:   "create <group-slug>",
		Short: "Create a scale-down condition for an autoscale group",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp autoscale condition create web-group --name cpu-low --metric cpu --operator lte --threshold 20 --duration 600 --scale-amount 1
  zcp autoscale condition create web-group --name mem-low --metric memory --operator lte --threshold 30 --duration 300 --scale-amount 1 --cooldown 600`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if metric == "" {
				return fmt.Errorf("--metric is required")
			}
			if operator == "" {
				return fmt.Errorf("--operator is required")
			}
			if threshold <= 0 {
				return fmt.Errorf("--threshold must be > 0")
			}
			if duration <= 0 {
				return fmt.Errorf("--duration must be > 0")
			}
			if scaleAmount <= 0 {
				return fmt.Errorf("--scale-amount must be > 0")
			}
			return runConditionCreate(cmd, args[0], autoscale.ConditionRequest{
				Name:        name,
				Metric:      metric,
				Operator:    operator,
				Threshold:   threshold,
				Duration:    duration,
				ScaleAmount: scaleAmount,
				Cooldown:    cooldown,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Condition name (required)")
	cmd.Flags().StringVar(&metric, "metric", "", "Metric to monitor, e.g. cpu, memory (required)")
	cmd.Flags().StringVar(&operator, "operator", "", "Comparison operator, e.g. gte, lte, gt, lt (required)")
	cmd.Flags().IntVar(&threshold, "threshold", 0, "Threshold value as a percentage (required)")
	cmd.Flags().IntVar(&duration, "duration", 0, "Duration in seconds the condition must hold (required)")
	cmd.Flags().IntVar(&scaleAmount, "scale-amount", 0, "Number of instances to remove (required)")
	cmd.Flags().IntVar(&cooldown, "cooldown", 0, "Cooldown in seconds before this condition can trigger again")
	return cmd
}

func runConditionCreate(cmd *cobra.Command, slug string, req autoscale.ConditionRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cond, err := svc.CreateCondition(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("autoscale condition create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", cond.ID},
		{"Name", cond.Name},
		{"Metric", cond.Metric},
		{"Operator", cond.Operator},
		{"Threshold", strconv.Itoa(cond.Threshold)},
		{"Duration", strconv.Itoa(cond.Duration)},
		{"Scale Amount", strconv.Itoa(cond.ScaleAmount)},
		{"Cooldown", strconv.Itoa(cond.Cooldown)},
	}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale condition update
// ---------------------------------------------------------------------------

func newConditionUpdateCmd() *cobra.Command {
	var (
		conditionID int
		name        string
		metric      string
		operator    string
		threshold   int
		duration    int
		scaleAmount int
		cooldown    int
	)

	cmd := &cobra.Command{
		Use:     "update <group-slug>",
		Short:   "Update a scale-down condition for an autoscale group",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp autoscale condition update web-group --condition-id 7 --name cpu-low --metric cpu --operator lte --threshold 15 --duration 600 --scale-amount 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if conditionID <= 0 {
				return fmt.Errorf("--condition-id is required and must be > 0")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if metric == "" {
				return fmt.Errorf("--metric is required")
			}
			if operator == "" {
				return fmt.Errorf("--operator is required")
			}
			if threshold <= 0 {
				return fmt.Errorf("--threshold must be > 0")
			}
			if duration <= 0 {
				return fmt.Errorf("--duration must be > 0")
			}
			if scaleAmount <= 0 {
				return fmt.Errorf("--scale-amount must be > 0")
			}
			return runConditionUpdate(cmd, args[0], conditionID, autoscale.ConditionRequest{
				Name:        name,
				Metric:      metric,
				Operator:    operator,
				Threshold:   threshold,
				Duration:    duration,
				ScaleAmount: scaleAmount,
				Cooldown:    cooldown,
			})
		},
	}
	cmd.Flags().IntVar(&conditionID, "condition-id", 0, "Condition ID to update (required)")
	cmd.Flags().StringVar(&name, "name", "", "Condition name (required)")
	cmd.Flags().StringVar(&metric, "metric", "", "Metric to monitor (required)")
	cmd.Flags().StringVar(&operator, "operator", "", "Comparison operator (required)")
	cmd.Flags().IntVar(&threshold, "threshold", 0, "Threshold value as a percentage (required)")
	cmd.Flags().IntVar(&duration, "duration", 0, "Duration in seconds (required)")
	cmd.Flags().IntVar(&scaleAmount, "scale-amount", 0, "Number of instances to remove (required)")
	cmd.Flags().IntVar(&cooldown, "cooldown", 0, "Cooldown in seconds")
	return cmd
}

func runConditionUpdate(cmd *cobra.Command, slug string, conditionID int, req autoscale.ConditionRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cond, err := svc.UpdateCondition(ctx, slug, conditionID, req)
	if err != nil {
		return fmt.Errorf("autoscale condition update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", cond.ID},
		{"Name", cond.Name},
		{"Metric", cond.Metric},
		{"Operator", cond.Operator},
		{"Threshold", strconv.Itoa(cond.Threshold)},
		{"Duration", strconv.Itoa(cond.Duration)},
		{"Scale Amount", strconv.Itoa(cond.ScaleAmount)},
		{"Cooldown", strconv.Itoa(cond.Cooldown)},
	}
	return printer.PrintTable(headers, rows)
}

// ---------------------------------------------------------------------------
// autoscale condition delete
// ---------------------------------------------------------------------------

func newConditionDeleteCmd() *cobra.Command {
	var (
		conditionID int
		yes         bool
	)

	cmd := &cobra.Command{
		Use:   "delete <group-slug>",
		Short: "Delete a scale-down condition from an autoscale group",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp autoscale condition delete web-group --condition-id 7
  zcp autoscale condition delete web-group --condition-id 7 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if conditionID <= 0 {
				return fmt.Errorf("--condition-id is required and must be > 0")
			}
			return runConditionDelete(cmd, args[0], conditionID, yes)
		},
	}
	cmd.Flags().IntVar(&conditionID, "condition-id", 0, "Condition ID to delete (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runConditionDelete(cmd *cobra.Command, slug string, conditionID int, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete scale-down condition %d from autoscale group %q? [y/N]: ", conditionID, slug)
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

	svc := autoscale.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DeleteCondition(ctx, slug, conditionID); err != nil {
		return fmt.Errorf("autoscale condition delete: %w", err)
	}

	printer.Fprintf("Scale-down condition %d deleted from autoscale group %q.\n", conditionID, slug)
	return nil
}
