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
	"github.com/zsoftly/zcp-cli/internal/api/support"
)

// NewSupportCmd returns the 'support' cobra command.
func NewSupportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "support",
		Short: "Manage support tickets, replies, feedback, and FAQs",
	}
	cmd.AddCommand(newTicketCmd())
	cmd.AddCommand(newFAQCmd())
	return cmd
}

// ---------- ticket subcommand ----------

func newTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "Manage support tickets",
	}
	cmd.AddCommand(newTicketListCmd())
	cmd.AddCommand(newTicketCreateCmd())
	cmd.AddCommand(newTicketShowCmd())
	cmd.AddCommand(newTicketDeleteCmd())
	cmd.AddCommand(newTicketSummaryCmd())
	cmd.AddCommand(newTicketReplyCmd())
	cmd.AddCommand(newTicketRepliesCmd())
	cmd.AddCommand(newTicketFeedbackCmd())
	cmd.AddCommand(newTicketFeedbackSubmitCmd())
	return cmd
}

// ---------- ticket list ----------

func newTicketListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List support tickets",
		Example: `  zcp support ticket list
  zcp support ticket list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTicketList(cmd)
		},
	}
	return cmd
}

func runTicketList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	tickets, err := svc.ListTickets(ctx)
	if err != nil {
		return fmt.Errorf("support ticket list: %w", err)
	}

	headers := []string{"ID", "SUBJECT", "STATUS", "PRIORITY", "DEPARTMENT", "CREATED"}
	rows := make([][]string, 0, len(tickets))
	for _, t := range tickets {
		rows = append(rows, []string{
			t.ID,
			t.Subject,
			t.Status,
			t.Priority,
			t.Department,
			t.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ---------- ticket create ----------

func newTicketCreateCmd() *cobra.Command {
	var subject, description, priority, department string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a support ticket",
		Example: `  zcp support ticket create --subject "Cannot SSH" --description "Connection refused on port 22"
  zcp support ticket create --subject "Billing" --description "Wrong charge" --priority high --department billing`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if subject == "" {
				return fmt.Errorf("--subject is required")
			}
			if description == "" {
				return fmt.Errorf("--description is required")
			}
			return runTicketCreate(cmd, support.CreateTicketRequest{
				Subject:     subject,
				Description: description,
				Priority:    priority,
				Department:  department,
			})
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "", "Ticket subject (required)")
	cmd.Flags().StringVar(&description, "description", "", "Ticket description (required)")
	cmd.Flags().StringVar(&priority, "priority", "", "Ticket priority (e.g. low, medium, high)")
	cmd.Flags().StringVar(&department, "department", "", "Department to route the ticket to")
	return cmd
}

func runTicketCreate(cmd *cobra.Command, req support.CreateTicketRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	ticket, err := svc.CreateTicket(ctx, req)
	if err != nil {
		return fmt.Errorf("support ticket create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", ticket.ID},
		{"Subject", ticket.Subject},
		{"Description", ticket.Description},
		{"Status", ticket.Status},
		{"Priority", ticket.Priority},
		{"Department", ticket.Department},
		{"Created", ticket.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ---------- ticket show ----------

func newTicketShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a support ticket",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp support ticket show <id>
  zcp support ticket show <id> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTicketShow(cmd, args[0])
		},
	}
	return cmd
}

func runTicketShow(cmd *cobra.Command, id string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	ticket, err := svc.GetTicket(ctx, id)
	if err != nil {
		return fmt.Errorf("support ticket show: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", ticket.ID},
		{"Subject", ticket.Subject},
		{"Description", ticket.Description},
		{"Status", ticket.Status},
		{"Priority", ticket.Priority},
		{"Department", ticket.Department},
		{"Created", ticket.CreatedAt},
		{"Updated", ticket.UpdatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ---------- ticket delete ----------

func newTicketDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a support ticket",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp support ticket delete <id>
  zcp support ticket delete <id> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTicketDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runTicketDelete(cmd *cobra.Command, id string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete support ticket %q? [y/N]: ", id)
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

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DeleteTicket(ctx, id); err != nil {
		return fmt.Errorf("support ticket delete: %w", err)
	}

	printer.Fprintf("Support ticket %q deleted.\n", id)
	return nil
}

// ---------- ticket summary ----------

func newTicketSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Show ticket count summary",
		Example: `  zcp support ticket summary
  zcp support ticket summary --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTicketSummary(cmd)
		},
	}
	return cmd
}

func runTicketSummary(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	summary, err := svc.Summary(ctx)
	if err != nil {
		return fmt.Errorf("support ticket summary: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Total", strconv.Itoa(summary.Total)},
		{"Open", strconv.Itoa(summary.Open)},
		{"Closed", strconv.Itoa(summary.Closed)},
	}
	return printer.PrintTable(headers, rows)
}

// ---------- ticket reply (create) ----------

func newTicketReplyCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:     "reply <ticket-id>",
		Short:   "Reply to a support ticket",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp support ticket reply <ticket-id> --message "Here is more detail..."`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if message == "" {
				return fmt.Errorf("--message is required")
			}
			return runTicketReply(cmd, args[0], support.CreateReplyRequest{Message: message})
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "Reply message (required)")
	return cmd
}

func runTicketReply(cmd *cobra.Command, ticketID string, req support.CreateReplyRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	reply, err := svc.CreateReply(ctx, ticketID, req)
	if err != nil {
		return fmt.Errorf("support ticket reply: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", reply.ID},
		{"Ticket ID", reply.TicketID},
		{"Message", reply.Message},
		{"Author", reply.Author},
		{"Created", reply.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ---------- ticket replies (list) ----------

func newTicketRepliesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replies <ticket-id>",
		Short: "List replies for a support ticket",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp support ticket replies <ticket-id>
  zcp support ticket replies <ticket-id> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTicketReplies(cmd, args[0])
		},
	}
	return cmd
}

func runTicketReplies(cmd *cobra.Command, ticketID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	replies, err := svc.ListReplies(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("support ticket replies: %w", err)
	}

	headers := []string{"ID", "AUTHOR", "MESSAGE", "CREATED"}
	rows := make([][]string, 0, len(replies))
	for _, r := range replies {
		rows = append(rows, []string{
			r.ID,
			r.Author,
			r.Message,
			r.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ---------- ticket feedback (get) ----------

func newTicketFeedbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feedback <ticket-id>",
		Short: "Get feedback for a support ticket",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp support ticket feedback <ticket-id>
  zcp support ticket feedback <ticket-id> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTicketFeedback(cmd, args[0])
		},
	}
	return cmd
}

func runTicketFeedback(cmd *cobra.Command, ticketID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	fb, err := svc.GetFeedback(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("support ticket feedback: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", fb.ID},
		{"Ticket ID", fb.TicketID},
		{"Rating", strconv.Itoa(fb.Rating)},
		{"Comment", fb.Comment},
		{"Created", fb.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ---------- ticket feedback-submit ----------

func newTicketFeedbackSubmitCmd() *cobra.Command {
	var rating int
	var comment string

	cmd := &cobra.Command{
		Use:   "feedback-submit <ticket-id>",
		Short: "Submit feedback for a support ticket",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp support ticket feedback-submit <ticket-id> --rating 5
  zcp support ticket feedback-submit <ticket-id> --rating 4 --comment "Quick resolution"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if rating < 1 || rating > 5 {
				return fmt.Errorf("--rating must be between 1 and 5")
			}
			return runTicketFeedbackSubmit(cmd, args[0], support.SubmitFeedbackRequest{
				Rating:  rating,
				Comment: comment,
			})
		},
	}
	cmd.Flags().IntVar(&rating, "rating", 0, "Rating from 1 to 5 (required)")
	cmd.Flags().StringVar(&comment, "comment", "", "Optional feedback comment")
	return cmd
}

func runTicketFeedbackSubmit(cmd *cobra.Command, ticketID string, req support.SubmitFeedbackRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	fb, err := svc.SubmitFeedback(ctx, ticketID, req)
	if err != nil {
		return fmt.Errorf("support ticket feedback-submit: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", fb.ID},
		{"Ticket ID", fb.TicketID},
		{"Rating", strconv.Itoa(fb.Rating)},
		{"Comment", fb.Comment},
		{"Created", fb.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ---------- faq subcommand ----------

func newFAQCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "faq",
		Short: "View frequently asked questions",
	}
	cmd.AddCommand(newFAQListCmd())
	return cmd
}

func newFAQListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List FAQs",
		Example: `  zcp support faq list
  zcp support faq list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFAQList(cmd)
		},
	}
	return cmd
}

func runFAQList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := support.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	faqs, err := svc.ListFAQs(ctx)
	if err != nil {
		return fmt.Errorf("support faq list: %w", err)
	}

	headers := []string{"ID", "CATEGORY", "QUESTION", "ANSWER"}
	rows := make([][]string, 0, len(faqs))
	for _, f := range faqs {
		rows = append(rows, []string{
			f.ID,
			f.Category,
			f.Question,
			f.Answer,
		})
	}
	return printer.PrintTable(headers, rows)
}
