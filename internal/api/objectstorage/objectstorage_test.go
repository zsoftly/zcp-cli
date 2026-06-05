package objectstorage_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/objectstorage"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newTestClient(t *testing.T, srv *httptest.Server) *httpclient.Client {
	t.Helper()
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

type listResponse struct {
	Status  string                        `json:"status"`
	Message string                        `json:"message"`
	Data    []objectstorage.ObjectStorage `json:"data"`
	Total   int                           `json:"total"`
}

type singleResponse struct {
	Status  string                      `json:"status"`
	Message string                      `json:"message"`
	Data    objectstorage.ObjectStorage `json:"data"`
}

type bucketListResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    []objectstorage.Bucket `json:"data"`
	Total   int                    `json:"total"`
}

type bucketSingleResponse struct {
	Status  string               `json:"status"`
	Message string               `json:"message"`
	Data    objectstorage.Bucket `json:"data"`
}

type objectListResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    []objectstorage.Object `json:"data"`
	Total   int                    `json:"total"`
}

type objectSingleResponse struct {
	Status  string               `json:"status"`
	Message string               `json:"message"`
	Data    objectstorage.Object `json:"data"`
}

func TestList(t *testing.T) {
	expected := []objectstorage.ObjectStorage{
		{ID: "os-1", Name: "my-storage", Slug: "my-storage-1", Status: "Active"},
		{ID: "os-2", Name: "backup-storage", Slug: "backup-storage-1", Status: "Active"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object-storages" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "want GET", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResponse{Status: "Success", Message: "OK", Data: expected, Total: 2})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	stores, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(stores) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(stores))
	}
	if stores[0].ID != "os-1" {
		t.Errorf("stores[0].ID = %q, want %q", stores[0].ID, "os-1")
	}
}

func TestGet(t *testing.T) {
	expected := objectstorage.ObjectStorage{ID: "os-1", Name: "my-storage", Slug: "my-storage-1", Status: "Active"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object-storages/my-storage-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Message: "OK", Data: expected})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	store, err := svc.Get(context.Background(), "my-storage-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if store.ID != "os-1" {
		t.Errorf("store.ID = %q, want %q", store.ID, "os-1")
	}
}

func TestCreate(t *testing.T) {
	expected := objectstorage.ObjectStorage{ID: "os-new", Name: "new-storage", Slug: "new-storage-1"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/object-storages" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Message: "OK", Data: expected})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	req := objectstorage.CreateRequest{
		Name:            "new-storage",
		Project:         "default",
		CloudProvider:   "ceph",
		Region:          "yul-1",
		BillingCycle:    "hourly",
		StorageCategory: "premium-ssd",
		CustomPlan:      &objectstorage.CustomPlan{Storage: 100},
	}
	store, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store.ID != "os-new" {
		t.Errorf("store.ID = %q, want %q", store.ID, "os-new")
	}
	if gotBody["cloud_provider"] != "ceph" {
		t.Errorf("body cloud_provider = %v, want %q", gotBody["cloud_provider"], "ceph")
	}
	if gotBody["region"] != "yul-1" {
		t.Errorf("body region = %v, want %q", gotBody["region"], "yul-1")
	}
	if cp, ok := gotBody["custom_plan"].(map[string]interface{}); !ok {
		t.Error("body custom_plan not present or wrong type")
	} else if cp["storage"] != float64(100) {
		t.Errorf("body custom_plan.storage = %v, want 100", cp["storage"])
	}
}

func TestDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "my-storage-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/object-storages/my-storage-1" {
		t.Errorf("path = %q, want /object-storages/my-storage-1", gotPath)
	}
}

func TestResize(t *testing.T) {
	expected := objectstorage.ObjectStorage{ID: "os-1", Slug: "my-storage-1"}

	var gotBody map[string]interface{}
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Message: "OK", Data: expected})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	store, err := svc.Resize(context.Background(), "my-storage-1", 200)
	if err != nil {
		t.Fatalf("Resize() error = %v", err)
	}
	if gotPath != "/object-storages/my-storage-1/resize" {
		t.Errorf("path = %q, want /object-storages/my-storage-1/resize", gotPath)
	}
	if cp, ok := gotBody["custom_plan"].(map[string]interface{}); !ok {
		t.Error("body custom_plan not present")
	} else if cp["storage"] != float64(200) {
		t.Errorf("body custom_plan.storage = %v, want 200", cp["storage"])
	}
	if store.ID != "os-1" {
		t.Errorf("store.ID = %q, want os-1", store.ID)
	}
}

func TestListBuckets(t *testing.T) {
	expected := []objectstorage.Bucket{
		{ID: "b-1", Name: "my-bucket", Slug: "my-bucket"},
		{ID: "b-2", Name: "logs", Slug: "logs"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object-storages/my-storage-1/buckets" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bucketListResponse{Status: "Success", Message: "OK", Data: expected, Total: 2})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	buckets, err := svc.ListBuckets(context.Background(), "my-storage-1")
	if err != nil {
		t.Fatalf("ListBuckets() error = %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("ListBuckets() returned %d buckets, want 2", len(buckets))
	}
	if buckets[0].Name != "my-bucket" {
		t.Errorf("buckets[0].Name = %q, want my-bucket", buckets[0].Name)
	}
}

func TestGetBucket(t *testing.T) {
	expected := objectstorage.Bucket{ID: "b-1", Name: "my-bucket", Slug: "my-bucket"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object-storages/my-storage-1/buckets/my-bucket" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bucketSingleResponse{Status: "Success", Message: "OK", Data: expected})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	bucket, err := svc.GetBucket(context.Background(), "my-storage-1", "my-bucket")
	if err != nil {
		t.Fatalf("GetBucket() error = %v", err)
	}
	if bucket.ID != "b-1" {
		t.Errorf("bucket.ID = %q, want b-1", bucket.ID)
	}
}

func TestCreateBucket(t *testing.T) {
	expected := objectstorage.Bucket{ID: "b-new", Name: "new-bucket", Slug: "new-bucket"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/object-storages/my-storage-1/buckets" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bucketSingleResponse{Status: "Success", Message: "OK", Data: expected})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	bucket, err := svc.CreateBucket(context.Background(), "my-storage-1", "new-bucket")
	if err != nil {
		t.Fatalf("CreateBucket() error = %v", err)
	}
	if bucket.ID != "b-new" {
		t.Errorf("bucket.ID = %q, want b-new", bucket.ID)
	}
	if gotBody["name"] != "new-bucket" {
		t.Errorf("body name = %v, want new-bucket", gotBody["name"])
	}
}

func TestDeleteBucket(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	err := svc.DeleteBucket(context.Background(), "my-storage-1", "my-bucket")
	if err != nil {
		t.Fatalf("DeleteBucket() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/object-storages/my-storage-1/buckets/my-bucket" {
		t.Errorf("path = %q, want /object-storages/my-storage-1/buckets/my-bucket", gotPath)
	}
}

func TestUpdateBucket(t *testing.T) {
	expected := objectstorage.Bucket{ID: "b-1", Name: "my-bucket", Slug: "my-bucket"}

	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bucketSingleResponse{Status: "Success", Message: "OK", Data: expected})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	req := objectstorage.BucketUpdateRequest{ACL: "public-read"}
	bucket, err := svc.UpdateBucket(context.Background(), "my-storage-1", "my-bucket", req)
	if err != nil {
		t.Fatalf("UpdateBucket() error = %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/object-storages/my-storage-1/buckets/my-bucket" {
		t.Errorf("path = %q, want /object-storages/my-storage-1/buckets/my-bucket", gotPath)
	}
	if gotBody["acl"] != "public-read" {
		t.Errorf("body acl = %v, want public-read", gotBody["acl"])
	}
	if bucket.ID != "b-1" {
		t.Errorf("bucket.ID = %q, want b-1", bucket.ID)
	}
}

func TestSetBucketACL(t *testing.T) {
	expected := objectstorage.Bucket{ID: "b-1", Name: "my-bucket", Slug: "my-bucket"}

	var gotBody map[string]interface{}
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bucketSingleResponse{Status: "Success", Message: "OK", Data: expected})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	bucket, err := svc.SetBucketACL(context.Background(), "my-storage-1", "my-bucket", "private")
	if err != nil {
		t.Fatalf("SetBucketACL() error = %v", err)
	}
	if gotPath != "/object-storages/my-storage-1/buckets/my-bucket/acl" {
		t.Errorf("path = %q, want /object-storages/my-storage-1/buckets/my-bucket/acl", gotPath)
	}
	if gotBody["acl"] != "private" {
		t.Errorf("body acl = %v, want private", gotBody["acl"])
	}
	if bucket.ID != "b-1" {
		t.Errorf("bucket.ID = %q, want b-1", bucket.ID)
	}
}

func TestListObjects(t *testing.T) {
	expected := []objectstorage.Object{
		{Key: "file.txt", Name: "file.txt", ContentType: "text/plain"},
		{Key: "images/logo.png", Name: "logo.png", ContentType: "image/png"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object-storages/my-storage-1/buckets/my-bucket/objects" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(objectListResponse{Status: "Success", Message: "OK", Data: expected, Total: 2})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	objects, err := svc.ListObjects(context.Background(), "my-storage-1", "my-bucket")
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("ListObjects() returned %d items, want 2", len(objects))
	}
	if objects[0].Key != "file.txt" {
		t.Errorf("objects[0].Key = %q, want file.txt", objects[0].Key)
	}
	if objects[1].ContentType != "image/png" {
		t.Errorf("objects[1].ContentType = %q, want image/png", objects[1].ContentType)
	}
}

func TestGetObject(t *testing.T) {
	expected := objectstorage.Object{Key: "file.txt", Name: "file.txt", ContentType: "text/plain", URL: "https://s3.example.com/my-bucket/file.txt"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object-storages/my-storage-1/buckets/my-bucket/objects/file.txt" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(objectSingleResponse{Status: "Success", Message: "OK", Data: expected})
	}))
	defer srv.Close()

	svc := objectstorage.NewService(newTestClient(t, srv))
	obj, err := svc.GetObject(context.Background(), "my-storage-1", "my-bucket", "file.txt")
	if err != nil {
		t.Fatalf("GetObject() error = %v", err)
	}
	if obj.Key != "file.txt" {
		t.Errorf("obj.Key = %q, want file.txt", obj.Key)
	}
	if obj.URL != "https://s3.example.com/my-bucket/file.txt" {
		t.Errorf("obj.URL = %q, want https://s3.example.com/my-bucket/file.txt", obj.URL)
	}
}

func TestS3EndpointFromRegion(t *testing.T) {
	setup := &objectstorage.RegionCloudProviderSetup{
		Config: objectstorage.RegionSetupConfig{
			S3Endpoint: "https://s3.yul-1.zsoftly.ca",
		},
	}
	store := objectstorage.ObjectStorage{
		Region: &objectstorage.Region{
			CloudProviderSetup: setup,
		},
	}

	if got := store.S3Endpoint(); got != "https://s3.yul-1.zsoftly.ca" {
		t.Errorf("S3Endpoint() = %q, want https://s3.yul-1.zsoftly.ca", got)
	}
}

func TestS3EndpointNilRegion(t *testing.T) {
	store := objectstorage.ObjectStorage{}
	if got := store.S3Endpoint(); got != "" {
		t.Errorf("S3Endpoint() with nil region = %q, want empty string", got)
	}
}

func TestCredentialsDecoding(t *testing.T) {
	payload := `{
		"status": "Success",
		"message": "OK",
		"data": {
			"id": "os-1",
			"slug": "my-storage-1",
			"name": "my-storage",
			"status": "Active",
			"api_key": "AKIAIOSFODNN7EXAMPLE",
			"api_secret": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region": {
				"id": "r-1",
				"name": "YUL-1",
				"slug": "yul-1",
				"cloud_provider_setup": {
					"config": {
						"s3_endpoint": "https://s3.yul-1.zsoftly.ca",
						"s3_fallback_endpoint": "http://10.18.20.21:7480"
					}
				}
			}
		}
	}`

	var resp singleResponse
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	store := resp.Data
	if store.APIKey != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("APIKey = %q, want AKIAIOSFODNN7EXAMPLE", store.APIKey)
	}
	if store.APISecret != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("APISecret = %q, want wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", store.APISecret)
	}
	if store.S3Endpoint() != "https://s3.yul-1.zsoftly.ca" {
		t.Errorf("S3Endpoint() = %q, want https://s3.yul-1.zsoftly.ca", store.S3Endpoint())
	}
}
