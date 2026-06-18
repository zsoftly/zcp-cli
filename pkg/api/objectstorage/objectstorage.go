// Package objectstorage provides ZCP Ceph object storage API operations.
package objectstorage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/sse"
	"github.com/minio/minio-go/v7/pkg/tags"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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

// BucketCreateRequest holds parameters for creating a bucket. The API requires
// an initial ACL grant (acl_grantee + acl_permission) alongside the name.
type BucketCreateRequest struct {
	Name          string `json:"name"`
	ACLGrantee    string `json:"acl_grantee"`
	ACLPermission string `json:"acl_permission"`
}

// BucketUpdateRequest holds parameters for updating bucket settings.
type BucketUpdateRequest struct {
	ACL string `json:"acl,omitempty"`
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

// CreateBucket creates a new bucket within an object storage instance. The
// bucket is created private, granting the owner full control; visibility can be
// changed afterward with bucket set-acl (SetBucketVisibility).
func (s *Service) CreateBucket(ctx context.Context, slug, name string) (*Bucket, error) {
	req := BucketCreateRequest{Name: name, ACLGrantee: "Owner", ACLPermission: "FULL_CONTROL"}
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
// userMetadata (may be nil) sets x-amz-meta-* headers on the object.
func (s *Service) PutObject(ctx context.Context, slug, bucketName, localPath, objectKey, contentType string, userMetadata map[string]string) (int64, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(localPath); err != nil {
		return 0, fmt.Errorf("local file not found or not readable: %s: %w", localPath, err)
	}
	if objectKey == "" {
		objectKey = filepath.Base(localPath)
	}
	opts := minio.PutObjectOptions{UserMetadata: userMetadata}
	if contentType != "" {
		opts.ContentType = contentType
	}
	info, err := mc.FPutObject(ctx, bucketName, objectKey, localPath, opts)
	if err != nil {
		return 0, fmt.Errorf("uploading %q to %s/%s: %w", localPath, bucketName, objectKey, err)
	}
	return info.Size, nil
}

// DownloadObject downloads an object from a bucket to a local path via the S3
// protocol. When destPath is empty it writes to the object key's base name in
// the current directory; when destPath is an existing directory it writes the
// base name inside it. Returns the path written and the number of bytes.
// versionID (may be "") selects a specific object version.
func (s *Service) DownloadObject(ctx context.Context, slug, bucketName, objectKey, destPath, versionID string) (string, int64, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return "", 0, err
	}
	if destPath == "" {
		destPath = filepath.Base(objectKey)
	} else if fi, statErr := os.Stat(destPath); statErr == nil && fi.IsDir() {
		destPath = filepath.Join(destPath, filepath.Base(objectKey))
	}
	if err := mc.FGetObject(ctx, bucketName, objectKey, destPath, minio.GetObjectOptions{VersionID: versionID}); err != nil {
		return "", 0, fmt.Errorf("downloading %q from %s: %w", objectKey, bucketName, err)
	}
	var size int64
	if fi, statErr := os.Stat(destPath); statErr == nil {
		size = fi.Size()
	}
	return destPath, size, nil
}

// DeleteObject removes an object from a bucket via the S3 protocol. versionID
// (may be "") deletes a specific version; otherwise the current version (which,
// on a versioned bucket, adds a delete marker).
func (s *Service) DeleteObject(ctx context.Context, slug, bucketName, objectKey, versionID string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if err := mc.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{VersionID: versionID}); err != nil {
		return fmt.Errorf("deleting %q from %s: %w", objectKey, bucketName, err)
	}
	return nil
}

// BucketVisibility values accepted by SetBucketVisibility.
const (
	VisibilityPrivate         = "private"
	VisibilityPublicRead      = "public-read"
	VisibilityPublicReadWrite = "public-read-write"
)

// bucketPublicPolicy builds an S3 bucket policy granting anonymous read of every
// object in the bucket (and listing the bucket). When write is true it also
// grants anonymous write/delete. The policy is marshaled with encoding/json so
// the bucket name is escaped rather than interpolated raw.
func bucketPublicPolicy(bucketName string, write bool) string {
	objectActions := []string{"s3:GetObject"}
	if write {
		objectActions = append(objectActions, "s3:PutObject", "s3:DeleteObject")
	}
	type principal struct {
		AWS []string `json:"AWS"`
	}
	type statement struct {
		Effect    string    `json:"Effect"`
		Principal principal `json:"Principal"`
		Action    []string  `json:"Action"`
		Resource  []string  `json:"Resource"`
	}
	policy := struct {
		Version   string      `json:"Version"`
		Statement []statement `json:"Statement"`
	}{
		Version: "2012-10-17",
		Statement: []statement{
			{Effect: "Allow", Principal: principal{AWS: []string{"*"}}, Action: []string{"s3:ListBucket"}, Resource: []string{"arn:aws:s3:::" + bucketName}},
			{Effect: "Allow", Principal: principal{AWS: []string{"*"}}, Action: objectActions, Resource: []string{"arn:aws:s3:::" + bucketName + "/*"}},
		},
	}
	b, _ := json.Marshal(policy)
	return string(b)
}

// SetBucketVisibility makes a bucket public or private using an S3 bucket policy
// (the mechanism Ceph RGW honors for anonymous object access; a bucket canned
// ACL does not grant s3:GetObject on objects). "private" removes the policy.
func (s *Service) SetBucketVisibility(ctx context.Context, slug, bucketName, visibility string) error {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return err
	}
	var policy string
	switch visibility {
	case VisibilityPrivate:
		policy = "" // empty policy removes public access
	case VisibilityPublicRead:
		policy = bucketPublicPolicy(bucketName, false)
	case VisibilityPublicReadWrite:
		policy = bucketPublicPolicy(bucketName, true)
	default:
		return fmt.Errorf("unsupported visibility %q (use private, public-read, or public-read-write)", visibility)
	}
	if err := mc.SetBucketPolicy(ctx, bucketName, policy); err != nil {
		return fmt.Errorf("setting visibility %q on bucket %s: %w", visibility, bucketName, err)
	}
	return nil
}

// SetBucketVersioning enables or suspends object versioning on a bucket (S3
// protocol). Ceph RGW supports versioning; once enabled, overwritten and deleted
// objects retain prior versions.
func (s *Service) SetBucketVersioning(ctx context.Context, slug, bucketName string, enabled bool) error {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return err
	}
	if enabled {
		err = mc.EnableVersioning(ctx, bucketName)
	} else {
		err = mc.SuspendVersioning(ctx, bucketName)
	}
	if err != nil {
		return fmt.Errorf("setting versioning on bucket %s: %w", bucketName, err)
	}
	return nil
}

// GetBucketVersioning returns the bucket's versioning status ("Enabled",
// "Suspended", or "" when never configured).
func (s *Service) GetBucketVersioning(ctx context.Context, slug, bucketName string) (string, error) {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return "", err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return "", err
	}
	cfg, err := mc.GetBucketVersioning(ctx, bucketName)
	if err != nil {
		return "", fmt.Errorf("getting versioning on bucket %s: %w", bucketName, err)
	}
	return cfg.Status, nil
}

// GetBucketPolicy returns the bucket's S3 policy as a JSON string ("" if none).
func (s *Service) GetBucketPolicy(ctx context.Context, slug, bucketName string) (string, error) {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return "", err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return "", err
	}
	policy, err := mc.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		return "", fmt.Errorf("getting policy on bucket %s: %w", bucketName, err)
	}
	return policy, nil
}

// PutBucketPolicy sets a raw S3 bucket policy (JSON). An empty policy removes it.
func (s *Service) PutBucketPolicy(ctx context.Context, slug, bucketName, policyJSON string) error {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return err
	}
	if err := mc.SetBucketPolicy(ctx, bucketName, policyJSON); err != nil {
		return fmt.Errorf("setting policy on bucket %s: %w", bucketName, err)
	}
	return nil
}

// s3 fetches the object-storage instance and returns a ready S3 client for it.
func (s *Service) s3(ctx context.Context, slug string) (*minio.Client, error) {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	return NewS3Client(store)
}

// s3NotConfigured reports whether an S3 error means "this subresource has no
// configuration yet" (so callers can treat it as empty rather than an error).
func s3NotConfigured(err error) bool {
	switch minio.ToErrorResponse(err).Code {
	case "NoSuchTagSet", "NoSuchTagSetError",
		"NoSuchLifecycleConfiguration",
		"ServerSideEncryptionConfigurationNotFoundError",
		"NoSuchCORSConfiguration", "NoSuchBucketPolicy":
		return true
	}
	return false
}

// EmptyBucket deletes every object and every object version (including delete
// markers) from a bucket via the S3 protocol. This is required before a bucket
// that has ever had versioning enabled can be deleted. Returns the number of
// version entries removed.
func (s *Service) EmptyBucket(ctx context.Context, slug, bucketName string) (int, error) {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return 0, err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return 0, err
	}
	n := 0
	for obj := range mc.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true, WithVersions: true}) {
		if obj.Err != nil {
			return n, fmt.Errorf("listing objects in %s: %w", bucketName, obj.Err)
		}
		if err := mc.RemoveObject(ctx, bucketName, obj.Key, minio.RemoveObjectOptions{VersionID: obj.VersionID}); err != nil {
			return n, fmt.Errorf("removing %s (version %s): %w", obj.Key, obj.VersionID, err)
		}
		n++
	}
	return n, nil
}

// GetBucketTagging returns the bucket's tags ({} if none set).
func (s *Service) GetBucketTagging(ctx context.Context, slug, bucketName string) (map[string]string, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return nil, err
	}
	t, err := mc.GetBucketTagging(ctx, bucketName)
	if err != nil {
		if s3NotConfigured(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("getting tags on bucket %s: %w", bucketName, err)
	}
	return t.ToMap(), nil
}

// SetBucketTagging replaces the bucket's tag set.
func (s *Service) SetBucketTagging(ctx context.Context, slug, bucketName string, m map[string]string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	t, err := tags.NewTags(m, false)
	if err != nil {
		return fmt.Errorf("invalid tags: %w", err)
	}
	if err := mc.SetBucketTagging(ctx, bucketName, t); err != nil {
		return fmt.Errorf("setting tags on bucket %s: %w", bucketName, err)
	}
	return nil
}

// DeleteBucketTagging removes all tags from a bucket.
func (s *Service) DeleteBucketTagging(ctx context.Context, slug, bucketName string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if err := mc.RemoveBucketTagging(ctx, bucketName); err != nil {
		return fmt.Errorf("removing tags on bucket %s: %w", bucketName, err)
	}
	return nil
}

// GetObjectTags returns an object's tags ({} if none).
func (s *Service) GetObjectTags(ctx context.Context, slug, bucketName, objectKey string) (map[string]string, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return nil, err
	}
	t, err := mc.GetObjectTagging(ctx, bucketName, objectKey, minio.GetObjectTaggingOptions{})
	if err != nil {
		if s3NotConfigured(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("getting tags on %s/%s: %w", bucketName, objectKey, err)
	}
	return t.ToMap(), nil
}

// SetObjectTags replaces an object's tag set.
func (s *Service) SetObjectTags(ctx context.Context, slug, bucketName, objectKey string, m map[string]string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	t, err := tags.NewTags(m, true)
	if err != nil {
		return fmt.Errorf("invalid tags: %w", err)
	}
	if err := mc.PutObjectTagging(ctx, bucketName, objectKey, t, minio.PutObjectTaggingOptions{}); err != nil {
		return fmt.Errorf("setting tags on %s/%s: %w", bucketName, objectKey, err)
	}
	return nil
}

// DeleteObjectTags removes all tags from an object.
func (s *Service) DeleteObjectTags(ctx context.Context, slug, bucketName, objectKey string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if err := mc.RemoveObjectTagging(ctx, bucketName, objectKey, minio.RemoveObjectTaggingOptions{}); err != nil {
		return fmt.Errorf("removing tags on %s/%s: %w", bucketName, objectKey, err)
	}
	return nil
}

// GetBucketEncryption returns the bucket's default-encryption algorithm ("" if
// none configured, e.g. "AES256" for SSE-S3).
func (s *Service) GetBucketEncryption(ctx context.Context, slug, bucketName string) (string, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return "", err
	}
	cfg, err := mc.GetBucketEncryption(ctx, bucketName)
	if err != nil {
		if s3NotConfigured(err) {
			return "", nil
		}
		return "", fmt.Errorf("getting encryption on bucket %s: %w", bucketName, err)
	}
	if len(cfg.Rules) > 0 {
		return cfg.Rules[0].Apply.SSEAlgorithm, nil
	}
	return "", nil
}

// SetBucketEncryption enables default SSE-S3 encryption on a bucket.
func (s *Service) SetBucketEncryption(ctx context.Context, slug, bucketName string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if err := mc.SetBucketEncryption(ctx, bucketName, sse.NewConfigurationSSES3()); err != nil {
		return fmt.Errorf("enabling encryption on bucket %s: %w", bucketName, err)
	}
	return nil
}

// DisableBucketEncryption removes default encryption from a bucket.
func (s *Service) DisableBucketEncryption(ctx context.Context, slug, bucketName string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if err := mc.RemoveBucketEncryption(ctx, bucketName); err != nil {
		return fmt.Errorf("disabling encryption on bucket %s: %w", bucketName, err)
	}
	return nil
}

// Lifecycle XML types with a guaranteed <Filter><Prefix> element. minio-go omits
// the <Filter> entirely for an empty prefix, which strict S3 / Ceph RGW rejects
// ("not well-formed against schema"), so for the "all objects" case we marshal
// our own body and PUT it directly (see putBucketSubresourceXML). Optional
// actions are pointers so they are omitted when unset.
type lcExpiration struct {
	Days int `xml:"Days,omitempty"`
}
type lcNoncurrent struct {
	NoncurrentDays int `xml:"NoncurrentDays,omitempty"`
}
type lcAbort struct {
	DaysAfterInitiation int `xml:"DaysAfterInitiation,omitempty"`
}
type lcFilter struct {
	Prefix string `xml:"Prefix"` // always emitted, even when empty (match all)
}
type lcRule struct {
	ID         string        `xml:"ID"`
	Filter     lcFilter      `xml:"Filter"`
	Status     string        `xml:"Status"`
	Expiration *lcExpiration `xml:"Expiration,omitempty"`
	Noncurrent *lcNoncurrent `xml:"NoncurrentVersionExpiration,omitempty"`
	Abort      *lcAbort      `xml:"AbortIncompleteMultipartUpload,omitempty"`
}
type lcConfig struct {
	XMLName xml.Name `xml:"LifecycleConfiguration"`
	Rules   []lcRule `xml:"Rule"`
}

// SetBucketExpiry sets a single lifecycle rule (optionally scoped to a key
// prefix). Each duration is applied only when > 0:
//   - days: expire current object versions after N days
//   - noncurrentDays: expire noncurrent (old) versions after N days
//   - abortMultipartDays: abort incomplete multipart uploads after N days
func (s *Service) SetBucketExpiry(ctx context.Context, slug, bucketName, prefix string, days, noncurrentDays, abortMultipartDays int) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	rule := lcRule{ID: "zcp-expire", Status: "Enabled", Filter: lcFilter{Prefix: prefix}}
	if days > 0 {
		rule.Expiration = &lcExpiration{Days: days}
	}
	if noncurrentDays > 0 {
		rule.Noncurrent = &lcNoncurrent{NoncurrentDays: noncurrentDays}
	}
	if abortMultipartDays > 0 {
		rule.Abort = &lcAbort{DaysAfterInitiation: abortMultipartDays}
	}
	body, err := xml.Marshal(lcConfig{Rules: []lcRule{rule}})
	if err != nil {
		return err
	}
	if err := s.putBucketSubresourceXML(ctx, mc, bucketName, "lifecycle", body); err != nil {
		return fmt.Errorf("setting lifecycle on bucket %s: %w", bucketName, err)
	}
	return nil
}

// putBucketSubresourceXML PUTs a raw XML body to a bucket subresource (e.g.
// ?lifecycle) by way of a short-lived presigned URL, so minio-go handles request
// signing (and region) while we control the exact body. Used where minio-go's
// typed marshaling can't produce the body RGW requires.
func (s *Service) putBucketSubresourceXML(ctx context.Context, mc *minio.Client, bucketName, subresource string, body []byte) error {
	u, err := mc.Presign(ctx, http.MethodPut, bucketName, "", 5*time.Minute, url.Values{subresource: []string{""}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	sum := md5.Sum(body)
	req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(sum[:]))
	req.Header.Set("Content-Type", "application/xml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT ?%s: HTTP %d: %s", subresource, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// GetBucketLifecycle returns the bucket's lifecycle configuration as JSON ("" if
// none).
func (s *Service) GetBucketLifecycle(ctx context.Context, slug, bucketName string) (string, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return "", err
	}
	cfg, err := mc.GetBucketLifecycle(ctx, bucketName)
	if err != nil {
		if s3NotConfigured(err) {
			return "", nil
		}
		return "", fmt.Errorf("getting lifecycle on bucket %s: %w", bucketName, err)
	}
	if cfg == nil || len(cfg.Rules) == 0 {
		return "", nil
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DeleteBucketLifecycle removes the bucket's lifecycle configuration.
func (s *Service) DeleteBucketLifecycle(ctx context.Context, slug, bucketName string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if err := mc.SetBucketLifecycle(ctx, bucketName, &lifecycle.Configuration{}); err != nil {
		return fmt.Errorf("removing lifecycle on bucket %s: %w", bucketName, err)
	}
	return nil
}

// GetBucketCORS returns the bucket's CORS rules as JSON ("" if none).
func (s *Service) GetBucketCORS(ctx context.Context, slug, bucketName string) (string, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return "", err
	}
	cfg, err := mc.GetBucketCors(ctx, bucketName)
	if err != nil {
		if s3NotConfigured(err) {
			return "", nil
		}
		return "", fmt.Errorf("getting CORS on bucket %s: %w", bucketName, err)
	}
	if cfg == nil || len(cfg.CORSRules) == 0 {
		return "", nil
	}
	b, err := json.MarshalIndent(cfg.CORSRules, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SetBucketCORS sets a single CORS rule allowing the given origins and methods
// (and optionally headers / max-age) — the common case. Replaces any existing
// CORS configuration.
func (s *Service) SetBucketCORS(ctx context.Context, slug, bucketName string, origins, methods, headers []string, maxAgeSeconds int) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	cfg := cors.NewConfig([]cors.Rule{{
		AllowedOrigin: origins,
		AllowedMethod: methods,
		AllowedHeader: headers,
		MaxAgeSeconds: maxAgeSeconds,
	}})
	if err := mc.SetBucketCors(ctx, bucketName, cfg); err != nil {
		return fmt.Errorf("setting CORS on bucket %s: %w", bucketName, err)
	}
	return nil
}

// DeleteBucketCORS removes the bucket's CORS configuration.
func (s *Service) DeleteBucketCORS(ctx context.Context, slug, bucketName string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if err := mc.SetBucketCors(ctx, bucketName, nil); err != nil {
		return fmt.Errorf("removing CORS on bucket %s: %w", bucketName, err)
	}
	return nil
}

// PresignObjectURL returns a time-limited, pre-signed HTTPS URL that a client can
// use to download the object without ZCP credentials, via the S3 protocol.
func (s *Service) PresignObjectURL(ctx context.Context, slug, bucketName, objectKey string, expires time.Duration) (string, error) {
	store, err := s.Get(ctx, slug)
	if err != nil {
		return "", err
	}
	mc, err := NewS3Client(store)
	if err != nil {
		return "", err
	}
	u, err := mc.PresignedGetObject(ctx, bucketName, objectKey, expires, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presigning %q in %s: %w", objectKey, bucketName, err)
	}
	return u.String(), nil
}

// PresignPutURL returns a time-limited, pre-signed HTTPS URL a client can use to
// upload an object via HTTP PUT, without ZCP/S3 credentials.
func (s *Service) PresignPutURL(ctx context.Context, slug, bucketName, objectKey string, expires time.Duration) (string, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return "", err
	}
	u, err := mc.PresignedPutObject(ctx, bucketName, objectKey, expires)
	if err != nil {
		return "", fmt.Errorf("presigning upload for %q in %s: %w", objectKey, bucketName, err)
	}
	return u.String(), nil
}

// ObjectVersion describes one version (or delete marker) of an object.
type ObjectVersion struct {
	Key            string
	VersionID      string
	IsLatest       bool
	IsDeleteMarker bool
	Size           int64
	LastModified   time.Time
}

// ListObjectVersions lists every version and delete-marker in a bucket (optionally
// under a key prefix).
func (s *Service) ListObjectVersions(ctx context.Context, slug, bucketName, prefix string) ([]ObjectVersion, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return nil, err
	}
	var out []ObjectVersion
	for obj := range mc.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: prefix, Recursive: true, WithVersions: true}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("listing versions in %s: %w", bucketName, obj.Err)
		}
		out = append(out, ObjectVersion{obj.Key, obj.VersionID, obj.IsLatest, obj.IsDeleteMarker, obj.Size, obj.LastModified})
	}
	return out, nil
}

// RestoreObject undeletes an object by removing its latest delete marker, so the
// previous version becomes current again. Errors if the object is not currently
// deleted (no latest delete marker).
func (s *Service) RestoreObject(ctx context.Context, slug, bucketName, objectKey string) (string, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return "", err
	}
	for obj := range mc.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Prefix: objectKey, Recursive: true, WithVersions: true}) {
		if obj.Err != nil {
			return "", fmt.Errorf("listing versions in %s: %w", bucketName, obj.Err)
		}
		if obj.Key == objectKey && obj.IsDeleteMarker && obj.IsLatest {
			if err := mc.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{VersionID: obj.VersionID}); err != nil {
				return "", fmt.Errorf("removing delete marker for %q: %w", objectKey, err)
			}
			return obj.VersionID, nil
		}
	}
	return "", fmt.Errorf("object %q has no current delete marker to restore (it is not deleted)", objectKey)
}

// CopyObject server-side copies an object within the same instance (no
// download/upload round-trip).
func (s *Service) CopyObject(ctx context.Context, slug, srcBucket, srcKey, dstBucket, dstKey string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if _, err := mc.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: dstBucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: srcBucket, Object: srcKey}); err != nil {
		return fmt.Errorf("copying %s/%s to %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}
	return nil
}

// MoveObject server-side copies an object then deletes the source.
func (s *Service) MoveObject(ctx context.Context, slug, srcBucket, srcKey, dstBucket, dstKey string) error {
	// Guard against moving an object onto itself: the copy-to-self would be
	// followed by deleting that same key, destroying the object.
	if srcBucket == dstBucket && srcKey == dstKey {
		return fmt.Errorf("source and destination are the same object (%s/%s)", srcBucket, srcKey)
	}
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if _, err := mc.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: dstBucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: srcBucket, Object: srcKey}); err != nil {
		return fmt.Errorf("copying %s/%s to %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}
	if err := mc.RemoveObject(ctx, srcBucket, srcKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("deleting source %s/%s after copy: %w", srcBucket, srcKey, err)
	}
	return nil
}

// ObjectStat is full S3 metadata for an object.
type ObjectStat struct {
	Key          string
	VersionID    string
	Size         int64
	ContentType  string
	ETag         string
	StorageClass string
	LastModified time.Time
	UserMetadata map[string]string
}

// StatObject returns full S3 metadata (HEAD) for an object. versionID may be "".
func (s *Service) StatObject(ctx context.Context, slug, bucketName, objectKey, versionID string) (*ObjectStat, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return nil, err
	}
	info, err := mc.StatObject(ctx, bucketName, objectKey, minio.StatObjectOptions{VersionID: versionID})
	if err != nil {
		return nil, fmt.Errorf("stat %s/%s: %w", bucketName, objectKey, err)
	}
	return &ObjectStat{
		Key:          info.Key,
		VersionID:    info.VersionID,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		StorageClass: info.StorageClass,
		LastModified: info.LastModified,
		UserMetadata: info.UserMetadata,
	}, nil
}

// IncompleteUpload describes a started-but-unfinished multipart upload.
type IncompleteUpload struct {
	Key       string
	UploadID  string
	Size      int64
	Initiated time.Time
}

// ListIncompleteUploads lists incomplete multipart uploads (storage consumed by
// failed/aborted large uploads), optionally under a key prefix.
func (s *Service) ListIncompleteUploads(ctx context.Context, slug, bucketName, prefix string) ([]IncompleteUpload, error) {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return nil, err
	}
	var out []IncompleteUpload
	for u := range mc.ListIncompleteUploads(ctx, bucketName, prefix, true) {
		if u.Err != nil {
			return nil, fmt.Errorf("listing incomplete uploads in %s: %w", bucketName, u.Err)
		}
		out = append(out, IncompleteUpload{u.Key, u.UploadID, u.Size, u.Initiated})
	}
	return out, nil
}

// AbortIncompleteUpload removes the incomplete multipart upload(s) for a key,
// reclaiming the consumed storage.
func (s *Service) AbortIncompleteUpload(ctx context.Context, slug, bucketName, objectKey string) error {
	mc, err := s.s3(ctx, slug)
	if err != nil {
		return err
	}
	if err := mc.RemoveIncompleteUpload(ctx, bucketName, objectKey); err != nil {
		return fmt.Errorf("aborting incomplete upload %q in %s: %w", objectKey, bucketName, err)
	}
	return nil
}
