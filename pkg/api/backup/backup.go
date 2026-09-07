// Package backup provides ZCP block storage backup API operations
// targeting the STKCNSL API.
package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/pkg/api/response"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// BackupVolume is the nested block storage volume embedded in a backup list
// item. List responses carry the volume here instead of a top-level
// blockstorage_id; Size is a string in the API response (e.g. "8").
type BackupVolume struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Size string `json:"size"`
}

// Backup represents a STKCNSL block storage backup.
type Backup struct {
	ID                   string        `json:"id"`
	Name                 string        `json:"name"`
	Slug                 string        `json:"slug"`
	BlockstorageID       string        `json:"blockstorage_id"`
	Blockstorage         *BackupVolume `json:"blockstorage"`
	UserID               string        `json:"user_id"`
	AccountID            string        `json:"account_id"`
	ProjectID            string        `json:"project_id"`
	RegionID             string        `json:"region_id"`
	CloudProviderID      string        `json:"cloud_provider_id"`
	CloudProviderSetupID string        `json:"cloud_provider_setup_id"`
	RequestStatus        bool          `json:"request_status"`
	Interval             string        `json:"interval"`
	Day                  *string       `json:"day"`
	At                   int           `json:"at"`
	ScheduledAt          string        `json:"scheduled_at"`
	Immediate            bool          `json:"immediate"`
	ServiceName          string        `json:"service_name"`
	ServiceDisplayName   string        `json:"service_display_name"`
	AllTimeConsumption   float64       `json:"all_time_consumption"`
	HasContract          bool          `json:"has_contract"`
	FrozenAt             *string       `json:"frozen_at"`
	SuspendedAt          *string       `json:"suspended_at"`
	TerminatedAt         *string       `json:"terminated_at"`
	CreatedAt            string        `json:"created_at"`
	UpdatedAt            string        `json:"updated_at"`
	DeletedAt            *string       `json:"deleted_at"`
}

// VolumeSlug returns the volume slug for this backup. List responses nest
// the volume under "blockstorage" with no top-level blockstorage_id; create
// responses carry blockstorage_id directly. Prefer the nested slug and fall
// back to blockstorage_id so callers get a usable value either way.
func (b *Backup) VolumeSlug() string {
	if b.Blockstorage != nil && b.Blockstorage.Slug != "" {
		return b.Blockstorage.Slug
	}
	return b.BlockstorageID
}

// backupAlias avoids infinite recursion when Backup.UnmarshalJSON re-decodes
// the payload through json.Unmarshal.
type backupAlias Backup

// UnmarshalJSON decodes a Backup, tolerating the "at" field being either a
// JSON number (create responses, e.g. 3) or a quoted numeric string (list
// responses, e.g. "3"). A null or empty "at" decodes to 0.
func (b *Backup) UnmarshalJSON(data []byte) error {
	aux := struct {
		At json.RawMessage `json:"at"`
		*backupAlias
	}{
		backupAlias: (*backupAlias)(b),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	at, err := response.ParseFlexInt(aux.At)
	if err != nil {
		return fmt.Errorf("backup: field \"at\": %w", err)
	}
	b.At = at
	return nil
}

// listResponse is the STKCNSL paginated envelope for block storage backups.
type listResponse struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentPage int      `json:"current_page"`
	Data        []Backup `json:"data"`
	LastPage    int      `json:"last_page"`
	Total       int      `json:"total"`
}

// singleResponse is used when the API returns a single backup in `data`.
type singleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Backup `json:"data"`
}

// CreateRequest holds parameters for creating a block storage backup.
type CreateRequest struct {
	Interval      string `json:"interval"`
	At            int    `json:"at"`
	Immediate     int    `json:"immediate"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	BillingCycle  string `json:"billing_cycle"`
	Plan          string `json:"plan"`
	PseudoService string `json:"psudo_service"`
	Project       string `json:"project"`
}

// Service provides block storage backup API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new backup Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// maxListPages bounds the paginated List loop so a server that misreports
// last_page can't loop forever.
const maxListPages = 1000

// List returns block storage backups, walking every page of the listing.
func (s *Service) List(ctx context.Context, region, project string) ([]Backup, error) {
	var all []Backup
	page := 1
	for ; page <= maxListPages; page++ {
		q := url.Values{}
		if region != "" {
			q.Set("filter[region]", region)
		}
		if project != "" {
			q.Set("filter[project]", project)
		}
		if page > 1 {
			q.Set("page", fmt.Sprintf("%d", page))
		}

		var resp listResponse
		if err := s.client.Get(ctx, "/blockstorages/backups", q, &resp); err != nil {
			return nil, fmt.Errorf("listing block storage backups: %w", err)
		}
		all = append(all, resp.Data...)

		// The loop counter, not the server-echoed current_page, drives
		// pagination: a server that ignores ?page and always echoes
		// current_page=1 would otherwise never advance past the first page.
		if len(resp.Data) == 0 || resp.LastPage <= 0 || page >= resp.LastPage {
			return all, nil
		}
	}
	return nil, fmt.Errorf("listing block storage backups: exceeded %d pages without reaching the last page", maxListPages)
}

// Create creates a new block storage backup.
func (s *Service) Create(ctx context.Context, blockstorageSlug string, req CreateRequest) (*Backup, error) {
	var resp singleResponse
	path := fmt.Sprintf("/blockstorages/%s/backups", blockstorageSlug)
	if err := s.client.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("creating block storage backup: %w", err)
	}
	return &resp.Data, nil
}

// Delete permanently deletes a block storage backup schedule.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/blockstorages/backups/"+slug, nil); err != nil {
		return fmt.Errorf("deleting block storage backup %s: %w", slug, err)
	}
	return nil
}
