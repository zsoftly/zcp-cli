// Package objectstorage provides ZCP Ceph object storage API operations.
package objectstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// RegionSetupConfig holds the cloud-provider-setup configuration embedded in a region.
type RegionSetupConfig struct {
	S3Endpoint         string `json:"s3_endpoint"`
	S3FallbackEndpoint string `json:"s3_fallback_endpoint"`
}

// RegionCloudProviderSetup holds the setup embedded in a region response.
type RegionCloudProviderSetup struct {
	Config RegionSetupConfig `json:"config"`
}

// Region represents the region where the object storage is deployed.
type Region struct {
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	Slug               string                    `json:"slug"`
	Country            string                    `json:"country"`
	CloudProviderSetup *RegionCloudProviderSetup `json:"cloud_provider_setup"`
}

// OSStats holds object-count and byte-total stats for a storage instance.
type OSStats struct {
	TotalFiles int `json:"total_files"`
	TotalSize  int `json:"total_size"`
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
	StorageUsage         json.Number    `json:"storage_usage"`
	AllTimeConsumption   float64        `json:"all_time_consumption"`
	ServiceName          string         `json:"service_name"`
	ServiceDisplayName   string         `json:"service_display_name"`
	APIKey               string         `json:"api_key"`
	APISecret            string         `json:"api_secret"`
	IsAutoscale          bool           `json:"is_autoscale"`
	Stats                *OSStats       `json:"stats"`
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

// S3Endpoint returns the S3 endpoint URL, resolved from the nested region
// cloud-provider-setup config (the live API does not expose it as a top-level field).
func (o *ObjectStorage) S3Endpoint() string {
	if o.Region != nil && o.Region.CloudProviderSetup != nil {
		return o.Region.CloudProviderSetup.Config.S3Endpoint
	}
	return ""
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
// Size is a quoted string in the live API (e.g. "38644").
// Permission is "Private" or "Public".
type Object struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Size         string `json:"size"`
	LastModified string `json:"last_modified"`
	Permission   string `json:"permission"`
}

// IsPublic reports whether the object has public read permission.
func (o Object) IsPublic() bool {
	return strings.EqualFold(o.Permission, "public")
}

// Directory represents a virtual directory (common prefix) within a bucket.
// The live API returns objects here, not plain strings.
type Directory struct {
	Prefix string `json:"prefix"`
	Name   string `json:"name"`
}

// ObjectListPagination holds the cursor token for the next page.
type ObjectListPagination struct {
	NextToken *string `json:"next_token"`
}

// ObjectListData is the inner data envelope for the object list API response.
type ObjectListData struct {
	CurrentPrefix string               `json:"current_prefix"`
	Directories   []Directory          `json:"directories"`
	Files         []Object             `json:"files"`
	Pagination    ObjectListPagination `json:"pagination"`
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

// objectListResponse is the API envelope for the object list endpoint.
// The live API wraps files inside data.files[], not data[].
type objectListResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Data    ObjectListData `json:"data"`
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

// ListObjects returns all objects in a bucket, including those inside subdirectory
// prefixes (e.g. "tests/"). It uses the S3 protocol (minio-go Recursive=true)
// rather than the REST API, because the REST listing endpoint cannot navigate into
// subdirectory prefixes on the server side. S3 pagination is handled by minio-go.
// Keys are deduplicated — the live API is known to echo objects across pages.
// Virtual directory markers (zero-byte keys ending in "/") are skipped.
func (s *Service) ListObjects(ctx context.Context, slug, bucketSlug string) ([]Object, error) {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("listing objects in bucket %s/%s: %w", slug, bucketSlug, err)
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return nil, fmt.Errorf("listing objects in bucket %s/%s: %w", slug, bucketSlug, err)
	}

	seen := map[string]bool{}
	var all []Object
	for obj := range mc.ListObjects(ctx, bucketSlug, minio.ListObjectsOptions{Recursive: true}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("listing objects in bucket %s/%s: %w", slug, bucketSlug, obj.Err)
		}
		if strings.HasSuffix(obj.Key, "/") || seen[obj.Key] {
			continue
		}
		seen[obj.Key] = true
		all = append(all, Object{
			Key:          obj.Key,
			Name:         filepath.Base(obj.Key),
			Size:         strconv.FormatInt(obj.Size, 10),
			LastModified: obj.LastModified.UTC().Format(time.RFC3339),
			Permission:   "Private",
		})
	}
	return all, nil
}

// GetObject returns a single object's metadata by key.
// The live API's GET /objects/{key} returns only the key string, so this
// fetches the full list and filters to avoid a useless second round-trip.
func (s *Service) GetObject(ctx context.Context, slug, bucketSlug, objectKey string) (*Object, error) {
	objects, err := s.ListObjects(ctx, slug, bucketSlug)
	if err != nil {
		return nil, fmt.Errorf("getting object %s in %s/%s: %w", objectKey, slug, bucketSlug, err)
	}
	for i := range objects {
		if objects[i].Key == objectKey {
			return &objects[i], nil
		}
	}
	return nil, fmt.Errorf("object %q not found in bucket %s/%s", objectKey, slug, bucketSlug)
}

// PutObject uploads a local file to a bucket via the S3 protocol.
// objectKey defaults to the base name of localPath when empty.
// contentType is auto-detected from the file extension when empty.
func (s *Service) PutObject(ctx context.Context, slug, bucketName, localPath, objectKey, contentType string) (int64, error) {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return 0, err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(localPath); err != nil {
		return 0, fmt.Errorf("local file not found or not readable: %s: %w", localPath, err)
	}
	if objectKey == "" {
		objectKey = filepath.Base(localPath)
	}
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	info, err := mc.FPutObject(ctx, bucketName, objectKey, localPath, opts)
	if err != nil {
		return 0, fmt.Errorf("uploading %q to %s/%s: %w", localPath, bucketName, objectKey, err)
	}
	return info.Size, nil
}

// DeleteObject removes an object from a bucket via the S3 protocol.
func (s *Service) DeleteObject(ctx context.Context, slug, bucketName, objectKey string) error {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return err
	}
	if err := mc.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("deleting %q from %s: %w", objectKey, bucketName, err)
	}
	return nil
}
