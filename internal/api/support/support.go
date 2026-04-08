// Package support provides STKCNSL support ticket, reply, feedback, and FAQ API operations.
package support

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// ---------- Models ----------

// Ticket represents a support ticket.
type Ticket struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Department  string `json:"department"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// CreateTicketRequest holds parameters for creating a support ticket.
type CreateTicketRequest struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Priority    string `json:"priority,omitempty"`
	Department  string `json:"department,omitempty"`
}

// Reply represents a reply on a support ticket.
type Reply struct {
	ID        string `json:"id"`
	TicketID  string `json:"ticketId"`
	Message   string `json:"message"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
}

// CreateReplyRequest holds parameters for replying to a ticket.
type CreateReplyRequest struct {
	Message string `json:"message"`
}

// Feedback represents feedback on a support ticket.
type Feedback struct {
	ID        string `json:"id"`
	TicketID  string `json:"ticketId"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"createdAt"`
}

// SubmitFeedbackRequest holds parameters for submitting ticket feedback.
type SubmitFeedbackRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

// FAQ represents a frequently asked question.
type FAQ struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Category string `json:"category"`
}

// TicketSummary represents the ticket count summary.
type TicketSummary struct {
	Total  int `json:"total"`
	Open   int `json:"open"`
	Closed int `json:"closed"`
}

// ---------- Response envelopes ----------
// STKCNSL API wraps responses in {"status": "Success", "data": ...}.

type listTicketsResponse struct {
	Status string   `json:"status"`
	Data   []Ticket `json:"data"`
}

type singleTicketResponse struct {
	Status string `json:"status"`
	Data   Ticket `json:"data"`
}

type ticketSummaryResponse struct {
	Status string        `json:"status"`
	Data   TicketSummary `json:"data"`
}

type listRepliesResponse struct {
	Status string  `json:"status"`
	Data   []Reply `json:"data"`
}

type singleReplyResponse struct {
	Status string `json:"status"`
	Data   Reply  `json:"data"`
}

type feedbackResponse struct {
	Status string   `json:"status"`
	Data   Feedback `json:"data"`
}

type listFAQsResponse struct {
	Status string `json:"status"`
	Data   []FAQ  `json:"data"`
}

type deleteResponse struct {
	Status string `json:"status"`
}

// ---------- Service ----------

// Service provides support ticket API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new support Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// ListTickets returns all support tickets for the authenticated user.
func (s *Service) ListTickets(ctx context.Context) ([]Ticket, error) {
	var resp listTicketsResponse
	if err := s.client.Get(ctx, "/support/tickets", url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("listing support tickets: %w", err)
	}
	return resp.Data, nil
}

// CreateTicket creates a new support ticket.
func (s *Service) CreateTicket(ctx context.Context, req CreateTicketRequest) (*Ticket, error) {
	var resp singleTicketResponse
	if err := s.client.Post(ctx, "/support/tickets", req, &resp); err != nil {
		return nil, fmt.Errorf("creating support ticket: %w", err)
	}
	return &resp.Data, nil
}

// GetTicket returns a single support ticket by ID.
func (s *Service) GetTicket(ctx context.Context, id string) (*Ticket, error) {
	var resp singleTicketResponse
	if err := s.client.Get(ctx, "/support/tickets/"+id, url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("getting support ticket %s: %w", id, err)
	}
	return &resp.Data, nil
}

// DeleteTicket deletes a support ticket by ID.
func (s *Service) DeleteTicket(ctx context.Context, id string) error {
	var resp deleteResponse
	if err := s.client.Get(ctx, "/support/tickets/"+id, url.Values{}, &resp); err != nil {
		// Verify the ticket exists before attempting delete.
		return fmt.Errorf("verifying support ticket %s: %w", id, err)
	}
	if err := s.client.Delete(ctx, "/support/tickets/"+id, nil); err != nil {
		return fmt.Errorf("deleting support ticket %s: %w", id, err)
	}
	return nil
}

// Summary returns a count summary of support tickets.
func (s *Service) Summary(ctx context.Context) (*TicketSummary, error) {
	var resp ticketSummaryResponse
	if err := s.client.Get(ctx, "/services/Ticket/summary", url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("getting ticket summary: %w", err)
	}
	return &resp.Data, nil
}

// ListReplies returns all replies for a given ticket ID.
func (s *Service) ListReplies(ctx context.Context, ticketID string) ([]Reply, error) {
	var resp listRepliesResponse
	if err := s.client.Get(ctx, "/support/tickets-reply/"+ticketID, url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("listing replies for ticket %s: %w", ticketID, err)
	}
	return resp.Data, nil
}

// CreateReply adds a reply to a support ticket.
func (s *Service) CreateReply(ctx context.Context, ticketID string, req CreateReplyRequest) (*Reply, error) {
	var resp singleReplyResponse
	if err := s.client.Post(ctx, "/support/tickets-reply/"+ticketID, req, &resp); err != nil {
		return nil, fmt.Errorf("replying to ticket %s: %w", ticketID, err)
	}
	return &resp.Data, nil
}

// GetFeedback returns feedback for a given ticket ID.
func (s *Service) GetFeedback(ctx context.Context, ticketID string) (*Feedback, error) {
	var resp feedbackResponse
	if err := s.client.Get(ctx, "/support/ticket-feedbacks/"+ticketID, url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("getting feedback for ticket %s: %w", ticketID, err)
	}
	return &resp.Data, nil
}

// SubmitFeedback submits feedback for a support ticket.
func (s *Service) SubmitFeedback(ctx context.Context, ticketID string, req SubmitFeedbackRequest) (*Feedback, error) {
	var resp feedbackResponse
	if err := s.client.Post(ctx, "/support/ticket-feedbacks/"+ticketID, req, &resp); err != nil {
		return nil, fmt.Errorf("submitting feedback for ticket %s: %w", ticketID, err)
	}
	return &resp.Data, nil
}

// ListFAQs returns all FAQs.
func (s *Service) ListFAQs(ctx context.Context) ([]FAQ, error) {
	var resp listFAQsResponse
	if err := s.client.Get(ctx, "/faqs", url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("listing FAQs: %w", err)
	}
	return resp.Data, nil
}
