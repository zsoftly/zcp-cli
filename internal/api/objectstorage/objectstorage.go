// Package objectstorage provides ZCP Ceph object storage API operations.
package objectstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Region represents the region where the object storage is deployed.
type Region struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Country string `json:"country"`
}

// CloudProvider represents the cloud provider backing the object storage.
type CloudProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
}

// Project represents the project the object storage belongs to.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// BillingCycle represents a billing cycle on an offering.
type BillingCycle struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Offering represents the billing plan attached to the object storage.
type Offering struct {
	ID           string        `json:"id"`
	Size         json.Number   `json:"size"`
	Price        string        `json:"price"`
	BillingCycle *BillingCycle `json:"billing_cycle"`
	RenewAt      string        `json:"renew_at"`
}

// ObjectStorage represents a Ceph object storage instance.
type ObjectStorage struct {
	ID                   string         `json:"id"`
	Slug                 string         `json:"slug"`
	Name                 string         `json:"name"`
	Status               string         `json:"status"`
	Size                 json.Number    `json:"size"`
	UsedSpace            json.Number    `json:"used_space"`
	S3Endpoint           string         `json:"s3_endpoint"`
	ServiceName          string         `json:"service_name"`
	ServiceDisplayName   string         `json:"service_display_name"`
	ProjectID            string         `json:"project_id"`
	RegionID             string         `json:"region_id"`
	CloudProviderID      string         `json:"cloud_provider_id"`
	CloudProviderSetupID string         `json:"cloud_provider_setup_id"`
	FrozenAt             *string        `json:"frozen_at"`
	SuspendedAt          *string        `json:"suspended_at"`
	TerminatedAt         *string        `json:"terminated_at"`
	CreatedAt            string         `json:"created_at"`
	UpdatedAt            string         `json:"updated_at"`
	Region               *Region        `json:"region"`
	CloudProvider        *CloudProvider `json:"cloud_provider"`
	Project              *Project       `json:"project"`
	Offering             *Offering      `json:"offering"`
}

// Bucket represents an object storage bucket.
type Bucket struct {
	ID              string      `json:"id"`
	Slug            string      `json:"slug"`
	Name            string      `json:"name"`
	Status          string      `json:"status"`
	ObjectCount     int         `json:"object_count"`
	Size            json.Number `json:"size"`
	ObjectStorageID string      `json:"object_storage_id"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
}

// CustomPlan holds storage size for a custom (non-catalogue) plan.
type CustomPlan struct {
	Storage int `json:"storage"`
}

// CreateRequest holds parameters for creating an object storage instance.
type CreateRequest struct {
	Name            string      `json:"name"`
	Project         string      `json:"project"`
	CloudProvider   string      `json:"cloud_provider"`
	Region          string      `json:"region"`
	BillingCycle    string      `json:"billing_cycle"`
	StorageCategory string      `json:"storage_category"`
	Plan            string      `json:"plan,omitempty"`
	CustomPlan      *CustomPlan `json:"custom_plan,omitempty"`
	Coupon          string      `json:"coupon,omitempty"`
}

// ResizeRequest holds parameters for resizing an object storage instance.
type ResizeRequest struct {
	CustomPlan CustomPlan `json:"custom_plan"`
}

// BucketCreateRequest holds parameters for creating a bucket.
type BucketCreateRequest struct {
	Name string `json:"name"`
}

// BucketUpdateRequest holds parameters for updating bucket settings.
type BucketUpdateRequest struct {
	ACL string `json:"acl,omitempty"`
}

// ACLUpdateRequest holds parameters for updating bucket ACL.
type ACLUpdateRequest struct {
	ACL string `json:"acl"`
}

// Object represents an object stored in a bucket.
type Object struct {
	Key          string      `json:"key"`
	Name         string      `json:"name"`
	Size         json.Number `json:"size"`
	ContentType  string      `json:"content_type"`
	LastModified string      `json:"last_modified"`
	IsPublic     bool        `json:"is_public"`
	ETag         string      `json:"etag"`
	URL          string      `json:"url"`
}

// listResponse is the paginated API envelope for object storage instances.
type listResponse struct {
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	CurrentPage int             `json:"current_page"`
	Data        []ObjectStorage `json:"data"`
	Total       int             `json:"total"`
}

// singleResponse wraps a single object storage in an API envelope.
type singleResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Data    ObjectStorage `json:"data"`
}

// bucketListResponse is the paginated API envelope for buckets.
type bucketListResponse struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentPage int      `json:"current_page"`
	Data        []Bucket `json:"data"`
	Total       int      `json:"total"`
}

// bucketSingleResponse wraps a single bucket in an API envelope.
type bucketSingleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Bucket `json:"data"`
}

// objectListResponse is the paginated API envelope for objects.
type objectListResponse struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentPage int      `json:"current_page"`
	Data        []Object `json:"data"`
	Total       int      `json:"total"`
}

// objectSingleResponse wraps a single object in an API envelope.
type objectSingleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Object `json:"data"`
}

// Service provides object storage API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new object storage Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all object storage instances for the account.
func (s *Service) List(ctx context.Context) ([]ObjectStorage, error) {
	q := url.Values{
		"include": {"cloud_provider,region,project,offering"},
	}
	var resp listResponse
	if err := s.client.Get(ctx, "/object-storages", q, &resp); err != nil {
		return nil, fmt.Errorf("listing object storages: %w", err)
	}
	return resp.Data, nil
}

// Get returns a single object storage instance by slug.
func (s *Service) Get(ctx context.Context, slug string) (*ObjectStorage, error) {
	q := url.Values{
		"include": {"cloud_provider,region,project,offering"},
	}
	var resp singleResponse
	if err := s.client.Get(ctx, "/object-storages/"+slug, q, &resp); err != nil {
		return nil, fmt.Errorf("getting object storage %s: %w", slug, err)
	}
	return &resp.Data, nil
}

// Create provisions a new object storage instance.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*ObjectStorage, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/object-storages", req, &resp); err != nil {
		return nil, fmt.Errorf("creating object storage: %w", err)
	}
	return &resp.Data, nil
}

// Delete permanently deletes an object storage instance.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/object-storages/"+slug, nil); err != nil {
		return fmt.Errorf("deleting object storage %s: %w", slug, err)
	}
	return nil
}

// Resize changes the storage allocation of an object storage instance.
func (s *Service) Resize(ctx context.Context, slug string, storageGB int) (*ObjectStorage, error) {
	req := ResizeRequest{CustomPlan: CustomPlan{Storage: storageGB}}
	var resp singleResponse
	if err := s.client.Post(ctx, "/object-storages/"+slug+"/resize", req, &resp); err != nil {
		return nil, fmt.Errorf("resizing object storage %s: %w", slug, err)
	}
	return &resp.Data, nil
}

// ListBuckets returns all buckets for an object storage instance.
func (s *Service) ListBuckets(ctx context.Context, slug string) ([]Bucket, error) {
	var resp bucketListResponse
	if err := s.client.Get(ctx, "/object-storages/"+slug+"/buckets", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing buckets for %s: %w", slug, err)
	}
	return resp.Data, nil
}

// GetBucket returns a single bucket by slug within an object storage instance.
func (s *Service) GetBucket(ctx context.Context, slug, bucketSlug string) (*Bucket, error) {
	var resp bucketSingleResponse
	path := fmt.Sprintf("/object-storages/%s/buckets/%s", slug, bucketSlug)
	if err := s.client.Get(ctx, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("getting bucket %s in %s: %w", bucketSlug, slug, err)
	}
	return &resp.Data, nil
}

// CreateBucket creates a new bucket within an object storage instance.
func (s *Service) CreateBucket(ctx context.Context, slug, name string) (*Bucket, error) {
	req := BucketCreateRequest{Name: name}
	var resp bucketSingleResponse
	if err := s.client.Post(ctx, "/object-storages/"+slug+"/buckets", req, &resp); err != nil {
		return nil, fmt.Errorf("creating bucket %q in %s: %w", name, slug, err)
	}
	return &resp.Data, nil
}

// DeleteBucket permanently deletes a bucket from an object storage instance.
func (s *Service) DeleteBucket(ctx context.Context, slug, bucketSlug string) error {
	path := fmt.Sprintf("/object-storages/%s/buckets/%s", slug, bucketSlug)
	if err := s.client.Delete(ctx, path, nil); err != nil {
		return fmt.Errorf("deleting bucket %s in %s: %w", bucketSlug, slug, err)
	}
	return nil
}

// UpdateBucket updates bucket settings (e.g. ACL / visibility).
func (s *Service) UpdateBucket(ctx context.Context, slug, bucketSlug string, req BucketUpdateRequest) (*Bucket, error) {
	var resp bucketSingleResponse
	path := fmt.Sprintf("/object-storages/%s/buckets/%s", slug, bucketSlug)
	if err := s.client.Put(ctx, path, nil, req, &resp); err != nil {
		return nil, fmt.Errorf("updating bucket %s in %s: %w", bucketSlug, slug, err)
	}
	return &resp.Data, nil
}

// SetBucketACL sets the access control list on a bucket.
// Common values: "private", "public-read", "public-read-write", "authenticated-read".
func (s *Service) SetBucketACL(ctx context.Context, slug, bucketSlug, acl string) (*Bucket, error) {
	req := ACLUpdateRequest{ACL: acl}
	var resp bucketSingleResponse
	path := fmt.Sprintf("/object-storages/%s/buckets/%s/acl", slug, bucketSlug)
	if err := s.client.Put(ctx, path, nil, req, &resp); err != nil {
		return nil, fmt.Errorf("setting ACL on bucket %s in %s: %w", bucketSlug, slug, err)
	}
	return &resp.Data, nil
}

// ListObjects returns all objects in a bucket.
func (s *Service) ListObjects(ctx context.Context, slug, bucketSlug string) ([]Object, error) {
	var resp objectListResponse
	path := fmt.Sprintf("/object-storages/%s/buckets/%s/objects", slug, bucketSlug)
	if err := s.client.Get(ctx, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("listing objects in bucket %s/%s: %w", slug, bucketSlug, err)
	}
	return resp.Data, nil
}

// GetObject returns a single object by key from a bucket.
func (s *Service) GetObject(ctx context.Context, slug, bucketSlug, objectKey string) (*Object, error) {
	var resp objectSingleResponse
	path := fmt.Sprintf("/object-storages/%s/buckets/%s/objects/%s", slug, bucketSlug, url.PathEscape(objectKey))
	if err := s.client.Get(ctx, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("getting object %s in %s/%s: %w", objectKey, slug, bucketSlug, err)
	}
	return &resp.Data, nil
}
