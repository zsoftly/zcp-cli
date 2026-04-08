package support_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/support"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

// ---------- Ticket tests ----------

func TestListTickets(t *testing.T) {
	expected := []support.Ticket{
		{ID: "t-1", Subject: "Cannot SSH", Status: "open", Priority: "high"},
		{ID: "t-2", Subject: "Billing question", Status: "closed", Priority: "low"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/support/tickets" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   expected,
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	tickets, err := svc.ListTickets(context.Background())
	if err != nil {
		t.Fatalf("ListTickets() error = %v", err)
	}
	if len(tickets) != 2 {
		t.Fatalf("ListTickets() returned %d tickets, want 2", len(tickets))
	}
	if tickets[0].ID != "t-1" {
		t.Errorf("tickets[0].ID = %q, want %q", tickets[0].ID, "t-1")
	}
	if tickets[1].Subject != "Billing question" {
		t.Errorf("tickets[1].Subject = %q, want %q", tickets[1].Subject, "Billing question")
	}
}

func TestCreateTicket(t *testing.T) {
	created := support.Ticket{ID: "t-new", Subject: "New issue", Status: "open", Priority: "medium"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/support/tickets" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   created,
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	ticket, err := svc.CreateTicket(context.Background(), support.CreateTicketRequest{
		Subject:     "New issue",
		Description: "Details here",
		Priority:    "medium",
	})
	if err != nil {
		t.Fatalf("CreateTicket() error = %v", err)
	}
	if ticket.ID != "t-new" {
		t.Errorf("ticket.ID = %q, want %q", ticket.ID, "t-new")
	}
	if gotBody["subject"] != "New issue" {
		t.Errorf("body subject = %v, want %q", gotBody["subject"], "New issue")
	}
}

func TestGetTicket(t *testing.T) {
	expected := support.Ticket{ID: "t-1", Subject: "Cannot SSH", Status: "open"}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   expected,
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	ticket, err := svc.GetTicket(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("GetTicket() error = %v", err)
	}
	if gotPath != "/support/tickets/t-1" {
		t.Errorf("path = %q, want %q", gotPath, "/support/tickets/t-1")
	}
	if ticket.Subject != "Cannot SSH" {
		t.Errorf("ticket.Subject = %q, want %q", ticket.Subject, "Cannot SSH")
	}
}

func TestDeleteTicket(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "Success",
				"data":   support.Ticket{ID: "t-del"},
			})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	err := svc.DeleteTicket(context.Background(), "t-del")
	if err != nil {
		t.Fatalf("DeleteTicket() error = %v", err)
	}
	if len(methods) < 2 {
		t.Fatalf("expected at least 2 requests (GET + DELETE), got %d", len(methods))
	}
	if methods[len(methods)-1] != http.MethodDelete {
		t.Errorf("last method = %q, want %q", methods[len(methods)-1], http.MethodDelete)
	}
}

func TestSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/services/Ticket/summary" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   support.TicketSummary{Total: 10, Open: 3, Closed: 7},
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	summary, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Total != 10 {
		t.Errorf("Total = %d, want 10", summary.Total)
	}
	if summary.Open != 3 {
		t.Errorf("Open = %d, want 3", summary.Open)
	}
}

// ---------- Reply tests ----------

func TestListReplies(t *testing.T) {
	expected := []support.Reply{
		{ID: "r-1", TicketID: "t-1", Message: "Thanks for reporting"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/support/tickets-reply/t-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   expected,
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	replies, err := svc.ListReplies(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("ListReplies() error = %v", err)
	}
	if len(replies) != 1 {
		t.Fatalf("ListReplies() returned %d, want 1", len(replies))
	}
	if replies[0].Message != "Thanks for reporting" {
		t.Errorf("replies[0].Message = %q, want %q", replies[0].Message, "Thanks for reporting")
	}
}

func TestCreateReply(t *testing.T) {
	created := support.Reply{ID: "r-new", TicketID: "t-1", Message: "Here is more info"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/support/tickets-reply/t-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   created,
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	reply, err := svc.CreateReply(context.Background(), "t-1", support.CreateReplyRequest{Message: "Here is more info"})
	if err != nil {
		t.Fatalf("CreateReply() error = %v", err)
	}
	if reply.ID != "r-new" {
		t.Errorf("reply.ID = %q, want %q", reply.ID, "r-new")
	}
}

// ---------- Feedback tests ----------

func TestGetFeedback(t *testing.T) {
	expected := support.Feedback{ID: "fb-1", TicketID: "t-1", Rating: 5, Comment: "Great support"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/support/ticket-feedbacks/t-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   expected,
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	fb, err := svc.GetFeedback(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("GetFeedback() error = %v", err)
	}
	if fb.Rating != 5 {
		t.Errorf("Rating = %d, want 5", fb.Rating)
	}
}

func TestSubmitFeedback(t *testing.T) {
	created := support.Feedback{ID: "fb-new", TicketID: "t-1", Rating: 4, Comment: "Good"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/support/ticket-feedbacks/t-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   created,
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	fb, err := svc.SubmitFeedback(context.Background(), "t-1", support.SubmitFeedbackRequest{Rating: 4, Comment: "Good"})
	if err != nil {
		t.Fatalf("SubmitFeedback() error = %v", err)
	}
	if fb.ID != "fb-new" {
		t.Errorf("fb.ID = %q, want %q", fb.ID, "fb-new")
	}
}

// ---------- FAQ tests ----------

func TestListFAQs(t *testing.T) {
	expected := []support.FAQ{
		{ID: "faq-1", Question: "How do I reset?", Answer: "Go to settings", Category: "Account"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/faqs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   expected,
		})
	}))
	defer srv.Close()

	svc := support.NewService(newClient(srv.URL))
	faqs, err := svc.ListFAQs(context.Background())
	if err != nil {
		t.Fatalf("ListFAQs() error = %v", err)
	}
	if len(faqs) != 1 {
		t.Fatalf("ListFAQs() returned %d, want 1", len(faqs))
	}
	if faqs[0].Question != "How do I reset?" {
		t.Errorf("faqs[0].Question = %q, want %q", faqs[0].Question, "How do I reset?")
	}
}
