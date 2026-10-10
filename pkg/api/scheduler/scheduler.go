// Package scheduler provides scheduler-action API operations.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/zsoftly/zcp-cli/pkg/api/response"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

const maxListPages = 1000

// Policy is the safe, user-facing portion of a scheduler action response.
// It retains only the actionable VM reference and omits provider and account
// objects, which contain unrelated and sometimes sensitive response data.
type Policy struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	Slug             string    `json:"slug"`
	Action           string    `json:"action"`
	Interval         string    `json:"interval"`
	Day              *int      `json:"day,omitempty"`
	At               string    `json:"at"`
	RetentionPolicy  int       `json:"retention_policy"`
	Status           string    `json:"status"`
	Timezone         string    `json:"timezone"`
	LastExecutedAt   string    `json:"last_executed_at,omitempty"`
	NextScheduledAt  string    `json:"next_scheduled_at,omitempty"`
	CreatedAt        string    `json:"created_at,omitempty"`
	UpdatedAt        string    `json:"updated_at,omitempty"`
	Actionable       *VMRef    `json:"actionable,omitempty"`
	Project          *ScopeRef `json:"project,omitempty"`
	Region           *ScopeRef `json:"region,omitempty"`
	LastRunStatus    string    `json:"last_run_status,omitempty"`
	LastRunTime      string    `json:"last_run_time,omitempty"`
	LastErrorMessage string    `json:"last_error_message,omitempty"`
	IsHealthy        *bool     `json:"is_healthy,omitempty"`
}

type ScopeRef struct {
	Slug string `json:"slug"`
}

// VMRef is the safe VM reference embedded in a scheduler action.
type VMRef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// UnmarshalJSON accepts both the numeric day sent on create and the quoted
// day returned by scheduler responses. A missing or null day remains nil.
func (p *Policy) UnmarshalJSON(data []byte) error {
	type policyAlias Policy
	aux := struct {
		Day json.RawMessage `json:"day"`
		*policyAlias
	}{policyAlias: (*policyAlias)(p)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Day) == 0 || string(aux.Day) == "null" {
		p.Day = nil
		return nil
	}
	v, err := response.ParseFlexInt(aux.Day)
	if err != nil {
		return fmt.Errorf("scheduler: field \"day\": %w", err)
	}
	p.Day = &v
	return nil
}

// CreateRequest creates a virtual-machine backup policy.
type CreateRequest struct {
	Service            string  `json:"service"`
	Slug               string  `json:"slug"`
	Action             string  `json:"action"`
	Name               *string `json:"name"`
	Description        *string `json:"description"`
	Interval           string  `json:"interval"`
	Day                *int    `json:"day"`
	At                 string  `json:"at"`
	Timezone           string  `json:"timezone"`
	RetentionPolicy    int     `json:"retention_policy"`
	TakeOneImmediately int     `json:"take_one_immediately"`
	Config             Config  `json:"config"`
}

// Config holds the documented VM-backup creation setting.
type Config struct {
	TakeOneImmediately int `json:"take_one_immediately"`
}

// UpdateRequest contains the scheduler fields accepted by the dashboard's
// update endpoint. A nil Day is serialized as null for non-day intervals.
type UpdateRequest struct {
	Interval        string `json:"interval"`
	Day             *int   `json:"day"`
	At              string `json:"at"`
	Timezone        string `json:"timezone"`
	RetentionPolicy int    `json:"retention_policy"`
}

type envelope struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Policy `json:"data"`
}

// ActionResponse is returned by asynchronous scheduler triggers and deletes.
type ActionResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type listEnvelope struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentPage int      `json:"current_page"`
	Data        []Policy `json:"data"`
	Total       *int     `json:"total"`
}

// Service provides scheduler-action API operations.
type Service struct{ client *httpclient.Client }

func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// List returns every VM backup scheduler action in the selected region and project.
func (s *Service) List(ctx context.Context, region, project string) ([]Policy, error) {
	var policies []Policy
	var reportedTotal *int
	for page := 1; page <= maxListPages; page++ {
		q := url.Values{}
		if region != "" {
			q.Set("filter[region]", region)
		}
		if project != "" {
			q.Set("filter[project]", project)
		}
		q.Set("actionable_type", "Virtual Machine Backup")
		q.Set("page", strconv.Itoa(page))
		var resp listEnvelope
		if err := s.client.Get(ctx, "/scheduler-actions", q, &resp); err != nil {
			return nil, fmt.Errorf("listing scheduler actions: %w", err)
		}
		if page > 1 && resp.CurrentPage == 0 {
			return nil, fmt.Errorf("listing scheduler actions: requested page %d but the API did not return pagination metadata", page)
		}
		if resp.CurrentPage > 0 && resp.CurrentPage != page {
			return nil, fmt.Errorf("listing scheduler actions: requested page %d but the API returned page %d", page, resp.CurrentPage)
		}
		if resp.Total != nil && *resp.Total < 0 {
			return nil, fmt.Errorf("listing scheduler actions: API returned invalid negative total %d", *resp.Total)
		}
		if reportedTotal == nil {
			reportedTotal = resp.Total
		} else if resp.Total == nil || *reportedTotal != *resp.Total {
			return nil, fmt.Errorf("listing scheduler actions: pagination total changed or was omitted on page %d", page)
		}
		if reportedTotal == nil {
			return resp.Data, nil
		}
		if *reportedTotal == 0 && len(resp.Data) > 0 {
			return nil, fmt.Errorf("listing scheduler actions: API reported total 0 with %d actions", len(resp.Data))
		}
		if len(policies)+len(resp.Data) > *reportedTotal {
			return nil, fmt.Errorf("listing scheduler actions: API returned more actions than reported total %d", *reportedTotal)
		}
		policies = append(policies, resp.Data...)
		if len(policies) == *reportedTotal {
			return policies, nil
		}
		if len(resp.Data) == 0 {
			return nil, fmt.Errorf("listing scheduler actions: page %d was empty before reaching reported total %d", page, *reportedTotal)
		}
	}
	return nil, fmt.Errorf("listing scheduler actions: exceeded %d pages without reaching the reported total", maxListPages)
}

func (s *Service) Get(ctx context.Context, id string) (*Policy, error) {
	var r envelope
	if err := s.client.Get(ctx, "/scheduler-actions/"+id, nil, &r); err != nil {
		return nil, fmt.Errorf("getting scheduler action %s: %w", id, err)
	}
	return &r.Data, nil
}
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Policy, error) {
	var r envelope
	if err := s.client.Post(ctx, "/scheduler-actions", req, &r); err != nil {
		return nil, fmt.Errorf("creating scheduler action: %w", err)
	}
	return &r.Data, nil
}
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*Policy, error) {
	var r envelope
	if err := s.client.Put(ctx, "/scheduler-actions/"+id, nil, req, &r); err != nil {
		return nil, fmt.Errorf("updating scheduler action %s: %w", id, err)
	}
	return &r.Data, nil
}
func (s *Service) Delete(ctx context.Context, id string) (*ActionResponse, error) {
	var r ActionResponse
	if err := s.client.DeleteWithResult(ctx, "/scheduler-actions/"+id, nil, &r); err != nil {
		return nil, fmt.Errorf("deleting scheduler action %s: %w", id, err)
	}
	return &r, nil
}
func (s *Service) Pause(ctx context.Context, id string) (*Policy, error) {
	var r envelope
	if err := s.client.Patch(ctx, "/scheduler-actions/"+id+"/pause", nil, &r); err != nil {
		return nil, fmt.Errorf("pausing scheduler action %s: %w", id, err)
	}
	return &r.Data, nil
}
func (s *Service) Resume(ctx context.Context, id string) (*Policy, error) {
	var r envelope
	if err := s.client.Patch(ctx, "/scheduler-actions/"+id+"/resume", nil, &r); err != nil {
		return nil, fmt.Errorf("resuming scheduler action %s: %w", id, err)
	}
	return &r.Data, nil
}
func (s *Service) RunNow(ctx context.Context, id string) (*ActionResponse, error) {
	var r ActionResponse
	if err := s.client.Post(ctx, "/scheduler/"+id+"/run-now", nil, &r); err != nil {
		return nil, fmt.Errorf("running scheduler action %s: %w", id, err)
	}
	return &r, nil
}
