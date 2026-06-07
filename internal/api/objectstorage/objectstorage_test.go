package objectstorage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// s3ListXML returns a minimal S3 ListBucketResult XML body for the given key/size pairs.
func s3ListXML(bucket string, entries [][2]string) string {
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult><Name>%s</Name><IsTruncated>false</IsTruncated>`, bucket)
	for _, e := range entries {
		body += fmt.Sprintf(`<Contents><Key>%s</Key><Size>%s</Size><LastModified>2026-01-01T00:00:00.000Z</LastModified><ETag>"x"</ETag></Contents>`, e[0], e[1])
	}
	body += `</ListBucketResult>`
	return body
}

func TestListObjects(t *testing.T) {
	// Includes a subdirectory-prefixed key ("tests/logo.png") to verify recursion.
	s3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, s3ListXML("my-bucket", [][2]string{
			{"file.txt", "1024"},
			{"tests/logo.png", "38644"},
		}))
	})

	mgmt, _ := newS3TestPair(t, s3Handler)
	svc := objectstorage.NewService(newTestClient(t, mgmt))

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
	if objects[0].Size != "1024" {
		t.Errorf("objects[0].Size = %q, want 1024", objects[0].Size)
	}
	if objects[1].Key != "tests/logo.png" {
		t.Errorf("objects[1].Key = %q, want tests/logo.png", objects[1].Key)
	}
	if objects[1].Name != "logo.png" {
		t.Errorf("objects[1].Name = %q, want logo.png (base of key)", objects[1].Name)
	}
}

// TestListObjects_Dedup verifies that duplicate keys returned by the S3 server
// (known live-API quirk) appear only once in the result.
func TestListObjects_Dedup(t *testing.T) {
	s3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, s3ListXML("my-bucket", [][2]string{
			{"dup.txt", "10"},
			{"dup.txt", "10"},
		}))
	})

	mgmt, _ := newS3TestPair(t, s3Handler)
	svc := objectstorage.NewService(newTestClient(t, mgmt))

	objects, err := svc.ListObjects(context.Background(), "my-storage-1", "my-bucket")
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("ListObjects() returned %d items, want 1 after dedup", len(objects))
	}
}

// TestListObjects_SkipsDirectoryMarkers verifies that virtual directory keys
// (zero-byte entries ending in "/") are excluded from results.
func TestListObjects_SkipsDirectoryMarkers(t *testing.T) {
	s3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, s3ListXML("my-bucket", [][2]string{
			{"tests/", "0"},
			{"tests/real.txt", "512"},
		}))
	})

	mgmt, _ := newS3TestPair(t, s3Handler)
	svc := objectstorage.NewService(newTestClient(t, mgmt))

	objects, err := svc.ListObjects(context.Background(), "my-storage-1", "my-bucket")
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("ListObjects() returned %d items, want 1 (directory marker skipped)", len(objects))
	}
	if objects[0].Key != "tests/real.txt" {
		t.Errorf("objects[0].Key = %q, want tests/real.txt", objects[0].Key)
	}
}

func TestGetObject(t *testing.T) {
	s3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, s3ListXML("my-bucket", [][2]string{
			{"other.txt", "100"},
			{"file.txt", "1024"},
		}))
	})

	mgmt, _ := newS3TestPair(t, s3Handler)
	svc := objectstorage.NewService(newTestClient(t, mgmt))

	obj, err := svc.GetObject(context.Background(), "my-storage-1", "my-bucket", "file.txt")
	if err != nil {
		t.Fatalf("GetObject() error = %v", err)
	}
	if obj.Key != "file.txt" {
		t.Errorf("obj.Key = %q, want file.txt", obj.Key)
	}
	if obj.Size != "1024" {
		t.Errorf("obj.Size = %q, want 1024", obj.Size)
	}
}

func TestGetObject_NotFound(t *testing.T) {
	s3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, s3ListXML("my-bucket", nil))
	})

	mgmt, _ := newS3TestPair(t, s3Handler)
	svc := objectstorage.NewService(newTestClient(t, mgmt))

	_, err := svc.GetObject(context.Background(), "my-storage-1", "my-bucket", "missing.txt")
	if err == nil {
		t.Fatal("expected error for missing object, got nil")
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

// newS3TestPair spins up two httptest servers: one for the management API and
// one acting as an S3-compatible endpoint. The management API Get response
// points at the S3 server so that NewS3Client connects to it.
//
// s3Handler receives actual S3 wire requests. minio-go always sends
// GET /?location= before any bucket operation; the helper wraps s3Handler
// to answer that preflight automatically so individual tests don't need to.
func newS3TestPair(t *testing.T, s3Handler http.Handler) (mgmt *httptest.Server, s3srv *httptest.Server) {
	t.Helper()

	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body) //nolint: errcheck
		// minio-go always resolves the bucket region before operating on it.
		if r.Method == http.MethodGet && r.URL.RawQuery == "location=" {
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><LocationConstraint></LocationConstraint>`)
			return
		}
		s3Handler.ServeHTTP(w, r)
	})

	s3srv = httptest.NewServer(wrapped)

	mgmt = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		store := objectstorage.ObjectStorage{
			ID:        "os-1",
			Slug:      "my-storage-1",
			Name:      "my-storage",
			Status:    "Active",
			APIKey:    "testkey",
			APISecret: "testsecret",
			Region: &objectstorage.Region{
				ID:   "r-1",
				Slug: "yul-1",
				Name: "YUL-1",
				CloudProviderSetup: &objectstorage.RegionCloudProviderSetup{
					Config: objectstorage.RegionSetupConfig{
						S3Endpoint: s3srv.URL,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Data: store})
	}))

	t.Cleanup(func() {
		mgmt.Close()
		s3srv.Close()
	})
	return mgmt, s3srv
}

func TestPutObject(t *testing.T) {
	var gotMethod, gotPath string

	s3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("ETag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
	})

	mgmt, _ := newS3TestPair(t, s3Handler)

	f, err := os.CreateTemp(t.TempDir(), "upload-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("hello zcp")
	f.Write(content)
	f.Close()

	svc := objectstorage.NewService(newTestClient(t, mgmt))
	size, err := svc.PutObject(context.Background(), "my-storage-1", "my-bucket", f.Name(), "hello.txt", "")
	if err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("S3 method = %q, want PUT", gotMethod)
	}
	if gotPath != "/my-bucket/hello.txt" {
		t.Errorf("S3 path = %q, want /my-bucket/hello.txt", gotPath)
	}
	if size != int64(len(content)) {
		t.Errorf("returned size = %d, want %d", size, len(content))
	}
}

func TestPutObject_DefaultKey(t *testing.T) {
	var gotPath string

	s3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("ETag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
	})

	mgmt, _ := newS3TestPair(t, s3Handler)

	f, err := os.CreateTemp(t.TempDir(), "myfile-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("content")
	f.Close()

	svc := objectstorage.NewService(newTestClient(t, mgmt))
	_, err = svc.PutObject(context.Background(), "my-storage-1", "my-bucket", f.Name(), "", "")
	if err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}
	wantPath := fmt.Sprintf("/my-bucket/%s", filepath.Base(f.Name()))
	if gotPath != wantPath {
		t.Errorf("S3 path = %q, want %q", gotPath, wantPath)
	}
}

func TestDeleteObject(t *testing.T) {
	var gotMethod, gotPath string

	s3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})

	mgmt, _ := newS3TestPair(t, s3Handler)

	svc := objectstorage.NewService(newTestClient(t, mgmt))
	err := svc.DeleteObject(context.Background(), "my-storage-1", "my-bucket", "report.pdf")
	if err != nil {
		t.Fatalf("DeleteObject() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("S3 method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/my-bucket/report.pdf" {
		t.Errorf("S3 path = %q, want /my-bucket/report.pdf", gotPath)
	}
}
