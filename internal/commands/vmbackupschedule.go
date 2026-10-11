package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/instance"
	"github.com/zsoftly/zcp-cli/pkg/api/scheduler"
)

func newVMBackupScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "schedule", Short: "Manage VM backup schedules"}
	cmd.AddCommand(newVMBackupScheduleListCmd(), newVMBackupScheduleGetCmd(), newVMBackupScheduleCreateCmd(), newVMBackupScheduleUpdateCmd(), newVMBackupScheduleDeleteCmd(), newVMBackupSchedulePauseCmd(), newVMBackupScheduleResumeCmd(), newVMBackupScheduleRunNowCmd())
	return cmd
}

func schedulerService(cmd *cobra.Command) (*scheduler.Service, context.Context, context.CancelFunc, error) {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	return scheduler.NewService(client), ctx, cancel, nil
}

func scopedSchedule(cmd *cobra.Command, svc *scheduler.Service, ctx context.Context, id string) (*scheduler.Policy, error) {
	p, err := svc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	region, project := scopedRegionProject(cmd)
	if p.Action != "Virtual Machine Backup" || p.Region == nil || p.Project == nil || p.Region.Slug != region || p.Project.Slug != project {
		return nil, fmt.Errorf("schedule %q is not a VM backup in region %q and project %q", id, region, project)
	}
	return p, nil
}

func scopedVM(cmd *cobra.Command, ctx context.Context, vmSlug string) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	region, project := scopedRegionProject(cmd)
	vms, err := instance.NewService(client).List(ctx, region, project)
	if err != nil {
		return fmt.Errorf("listing VMs in scope: %w", err)
	}
	for _, vm := range vms {
		if vm.Slug == vmSlug {
			return nil
		}
	}
	return fmt.Errorf("VM %q was not found in region %q and project %q", vmSlug, region, project)
}

func printSchedule(cmd *cobra.Command, p *scheduler.Policy) error {
	_, _, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	if outputFormat(cmd) != output.FormatTable {
		return printer.Print(p)
	}
	day := "-"
	if p.Day != nil {
		day = fmt.Sprint(*p.Day)
	}
	return printer.PrintTable([]string{"ID", "NAME", "SLUG", "VM", "INTERVAL", "DAY", "AT", "TIMEZONE", "RETENTION", "STATUS", "LAST RUN", "LAST RESULT", "LAST ERROR", "NEXT SCHEDULED"}, [][]string{{p.ID, p.Name, p.Slug, scheduleVM(p), p.Interval, day, p.At, p.Timezone, fmt.Sprint(p.RetentionPolicy), p.Status, p.LastRunTime, p.LastRunStatus, p.LastErrorMessage, p.NextScheduledAt}})
}

func scheduleVM(p *scheduler.Policy) string {
	if p.Actionable != nil {
		return p.Actionable.Slug
	}
	return "-"
}

func outputFormat(cmd *cobra.Command) output.Format {
	if v := os.Getenv("ZCP_OUTPUT"); v != "" {
		return output.ParseFormat(v)
	}
	v, _ := cmd.Root().PersistentFlags().GetString("output")
	return output.ParseFormat(v)
}

func newVMBackupScheduleListCmd() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List VM backup schedules", RunE: func(cmd *cobra.Command, args []string) error {
		svc, ctx, cancel, err := schedulerService(cmd)
		if err != nil {
			return err
		}
		defer cancel()
		region, project := scopedRegionProject(cmd)
		all, err := svc.List(ctx, region, project)
		if err != nil {
			return fmt.Errorf("vm-backup schedule list: %w", err)
		}
		_, _, printer, err := buildClientAndPrinter(cmd)
		if err != nil {
			return err
		}
		if outputFormat(cmd) != output.FormatTable {
			return printer.Print(all)
		}
		rows := make([][]string, 0, len(all))
		for _, p := range all {
			day := "-"
			if p.Day != nil {
				day = fmt.Sprint(*p.Day)
			}
			rows = append(rows, []string{p.ID, p.Name, p.Slug, scheduleVM(&p), p.Interval, day, p.At, p.Timezone, fmt.Sprint(p.RetentionPolicy), p.Status, p.LastRunTime, p.LastRunStatus, p.LastErrorMessage, p.NextScheduledAt})
		}
		return printer.PrintTable([]string{"ID", "NAME", "SLUG", "VM", "INTERVAL", "DAY", "AT", "TIMEZONE", "RETENTION", "STATUS", "LAST RUN", "LAST RESULT", "LAST ERROR", "NEXT SCHEDULED"}, rows)
	}}
}
func newVMBackupScheduleGetCmd() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Show a VM backup schedule", Args: exactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		svc, ctx, cancel, err := schedulerService(cmd)
		if err != nil {
			return err
		}
		defer cancel()
		p, err := scopedSchedule(cmd, svc, ctx, args[0])
		if err != nil {
			return fmt.Errorf("vm-backup schedule get: %w", err)
		}
		return printSchedule(cmd, p)
	}}
}

func scheduleFlags(cmd *cobra.Command, interval, at, timezone *string, day *int, retention *int) {
	cmd.Flags().StringVar(interval, "interval", "", "Schedule interval: hourly, dailyAt, everyOtherDay, weeklyOn, or monthlyOn")
	cmd.Flags().StringVar(at, "at", "", "Time as a whole hour, HH:00 (defaults to 00:00 for hourly)")
	cmd.Flags().StringVar(timezone, "timezone", "", "IANA timezone, for example America/Toronto or UTC")
	cmd.Flags().IntVar(day, "day", -1, "Weekday 0-6 for weeklyOn, or day 1-28 for monthlyOn")
	cmd.Flags().IntVar(retention, "retention", 0, "Maximum snapshots/backups to retain (minimum 1)")
}
func validateSchedule(interval, at, timezone string, day, retention int) (*int, error) {
	switch interval {
	case "hourly", "dailyAt", "everyOtherDay", "weeklyOn", "monthlyOn":
	default:
		return nil, fmt.Errorf("--interval must be hourly, dailyAt, everyOtherDay, weeklyOn, or monthlyOn")
	}
	if strings.TrimSpace(timezone) == "" {
		return nil, fmt.Errorf("--timezone is required")
	}
	if timezone == "Local" {
		return nil, fmt.Errorf("--timezone must be an IANA timezone, not Local")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, fmt.Errorf("--timezone must be a valid IANA timezone: %w", err)
	}
	if retention < 1 {
		return nil, fmt.Errorf("--retention must be at least 1")
	}
	if at != "" {
		if _, err := time.Parse("15:04", at); err != nil {
			return nil, fmt.Errorf("--at must be HH:MM")
		}
		if !strings.HasSuffix(at, ":00") {
			return nil, fmt.Errorf("--at must be a whole hour (HH:00)")
		}
	} else if interval != "hourly" {
		return nil, fmt.Errorf("--at must be HH:MM")
	}
	if interval == "weeklyOn" {
		if day < 0 || day > 6 {
			return nil, fmt.Errorf("--day must be 0 through 6 for weeklyOn")
		}
		return &day, nil
	}
	if interval == "monthlyOn" {
		if day < 1 || day > 28 {
			return nil, fmt.Errorf("--day must be 1 through 28 for monthlyOn")
		}
		return &day, nil
	}
	if day != -1 {
		return nil, fmt.Errorf("--day is valid only with weeklyOn or monthlyOn")
	}
	return nil, nil
}

func newVMBackupScheduleCreateCmd() *cobra.Command {
	var interval, at, tz, name, description string
	var day, retention int
	var immediate bool
	cmd := &cobra.Command{Use: "create <vm-slug>", Short: "Create a VM backup schedule", Args: exactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if interval == "hourly" && at == "" {
			at = "00:00"
		}
		d, err := validateSchedule(interval, at, tz, day, retention)
		if err != nil {
			return err
		}
		svc, ctx, cancel, err := schedulerService(cmd)
		if err != nil {
			return err
		}
		defer cancel()
		if err := scopedVM(cmd, ctx, args[0]); err != nil {
			return err
		}
		i := 0
		if immediate {
			i = 1
		}
		var nameValue, descriptionValue *string
		if v := strings.TrimSpace(name); v != "" {
			nameValue = &v
		}
		if v := strings.TrimSpace(description); v != "" {
			descriptionValue = &v
		}
		p, err := svc.Create(ctx, scheduler.CreateRequest{Service: "Virtual Machine", Slug: args[0], Action: "VM Backup", Name: nameValue, Description: descriptionValue, Interval: interval, Day: d, At: at, Timezone: tz, RetentionPolicy: retention, TakeOneImmediately: i, Config: scheduler.Config{TakeOneImmediately: i}})
		if err != nil {
			return fmt.Errorf("vm-backup schedule create: the API may have accepted the request before returning an error (%w). Run 'zcp vm-backup schedule list' before retrying", err)
		}
		return printSchedule(cmd, p)
	}}
	scheduleFlags(cmd, &interval, &at, &tz, &day, &retention)
	cmd.Flags().StringVar(&name, "name", "", "Optional policy name")
	cmd.Flags().StringVar(&description, "description", "", "Policy description")
	cmd.Flags().BoolVar(&immediate, "immediate", false, "Request one immediate backup")
	_ = cmd.MarkFlagRequired("interval")
	_ = cmd.MarkFlagRequired("timezone")
	_ = cmd.MarkFlagRequired("retention")
	return cmd
}
func newVMBackupScheduleUpdateCmd() *cobra.Command {
	var interval, at, tz string
	var day, retention int
	cmd := &cobra.Command{Use: "update <id>", Short: "Update a VM backup schedule", Args: exactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if interval == "hourly" && at == "" {
			at = "00:00"
		}
		d, err := validateSchedule(interval, at, tz, day, retention)
		if err != nil {
			return err
		}
		svc, ctx, cancel, err := schedulerService(cmd)
		if err != nil {
			return err
		}
		defer cancel()
		if _, err := scopedSchedule(cmd, svc, ctx, args[0]); err != nil {
			return err
		}
		p, err := svc.Update(ctx, args[0], scheduler.UpdateRequest{Interval: interval, Day: d, At: at, Timezone: tz, RetentionPolicy: retention})
		if err != nil {
			return fmt.Errorf("vm-backup schedule update: %w", err)
		}
		return printSchedule(cmd, p)
	}}
	scheduleFlags(cmd, &interval, &at, &tz, &day, &retention)
	_ = cmd.MarkFlagRequired("interval")
	_ = cmd.MarkFlagRequired("timezone")
	_ = cmd.MarkFlagRequired("retention")
	return cmd
}
func newVMBackupScheduleDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "delete <id>", Short: "Delete a VM backup schedule", Args: exactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !yes && !confirmAction(cmd, "Delete VM backup schedule %q? This cannot be undone.", args[0]) {
			return nil
		}
		svc, ctx, cancel, err := schedulerService(cmd)
		if err != nil {
			return err
		}
		defer cancel()
		if _, err := scopedSchedule(cmd, svc, ctx, args[0]); err != nil {
			return err
		}
		result, err := svc.Delete(ctx, args[0])
		if err != nil {
			return fmt.Errorf("vm-backup schedule delete: %w", err)
		}
		_, _, printer, err := buildClientAndPrinter(cmd)
		if err != nil {
			return err
		}
		if outputFormat(cmd) != output.FormatTable {
			return printer.Print(result)
		}
		printer.Fprintf("%s\n", result.Message)
		return nil
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
func scheduleActionCmd(use, short string, action func(*scheduler.Service, context.Context, string) (*scheduler.Policy, error)) *cobra.Command {
	return &cobra.Command{Use: use + " <id>", Short: short, Args: exactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		svc, ctx, cancel, err := schedulerService(cmd)
		if err != nil {
			return err
		}
		defer cancel()
		if _, err := scopedSchedule(cmd, svc, ctx, args[0]); err != nil {
			return err
		}
		p, err := action(svc, ctx, args[0])
		if err != nil {
			return fmt.Errorf("vm-backup schedule %s: %w", use, err)
		}
		return printSchedule(cmd, p)
	}}
}
func newVMBackupSchedulePauseCmd() *cobra.Command {
	return scheduleActionCmd("pause", "Pause a VM backup schedule", func(s *scheduler.Service, c context.Context, id string) (*scheduler.Policy, error) {
		return s.Pause(c, id)
	})
}
func newVMBackupScheduleResumeCmd() *cobra.Command {
	return scheduleActionCmd("resume", "Resume a VM backup schedule", func(s *scheduler.Service, c context.Context, id string) (*scheduler.Policy, error) {
		return s.Resume(c, id)
	})
}
func newVMBackupScheduleRunNowCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "run-now <id>", Short: "Run a VM backup schedule now", Args: exactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !yes && !confirmAction(cmd, "Run VM backup schedule %q now?", args[0]) {
			return nil
		}
		svc, ctx, cancel, err := schedulerService(cmd)
		if err != nil {
			return err
		}
		defer cancel()
		if _, err := scopedSchedule(cmd, svc, ctx, args[0]); err != nil {
			return err
		}
		result, err := svc.RunNow(ctx, args[0])
		if err != nil {
			return fmt.Errorf("vm-backup schedule run-now: %w", err)
		}
		_, _, printer, err := buildClientAndPrinter(cmd)
		if err != nil {
			return err
		}
		if outputFormat(cmd) != output.FormatTable {
			return printer.Print(result)
		}
		printer.Fprintf("%s\n", result.Message)
		return nil
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
