package objectstorage_test

import (
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/objectstorage"
)

func TestNewS3Client_NoRegion(t *testing.T) {
	store := &objectstorage.ObjectStorage{
		Slug:      "my-store",
		APIKey:    "key",
		APISecret: "secret",
	}
	_, err := objectstorage.NewS3Client(store)
	if err == nil {
		t.Fatal("expected error for missing region/endpoint, got nil")
	}
}

func TestNewS3Client_MissingAPIKey(t *testing.T) {
	store := &objectstorage.ObjectStorage{
		Slug:      "my-store",
		APISecret: "secret",
		Region: &objectstorage.Region{
			CloudProviderSetup: &objectstorage.RegionCloudProviderSetup{
				Config: objectstorage.RegionSetupConfig{S3Endpoint: "https://s3.example.com"},
			},
		},
	}
	_, err := objectstorage.NewS3Client(store)
	if err == nil {
		t.Fatal("expected error for missing api_key, got nil")
	}
}

func TestNewS3Client_MissingAPISecret(t *testing.T) {
	store := &objectstorage.ObjectStorage{
		Slug:   "my-store",
		APIKey: "key",
		Region: &objectstorage.Region{
			CloudProviderSetup: &objectstorage.RegionCloudProviderSetup{
				Config: objectstorage.RegionSetupConfig{S3Endpoint: "https://s3.example.com"},
			},
		},
	}
	_, err := objectstorage.NewS3Client(store)
	if err == nil {
		t.Fatal("expected error for missing api_secret, got nil")
	}
}

func TestNewS3Client_HTTPS(t *testing.T) {
	store := &objectstorage.ObjectStorage{
		Slug:      "my-store",
		APIKey:    "key",
		APISecret: "secret",
		Region: &objectstorage.Region{
			CloudProviderSetup: &objectstorage.RegionCloudProviderSetup{
				Config: objectstorage.RegionSetupConfig{S3Endpoint: "https://s3.yul-1.zsoftly.ca"},
			},
		},
	}
	mc, err := objectstorage.NewS3Client(store)
	if err != nil {
		t.Fatalf("NewS3Client() error = %v", err)
	}
	if mc == nil {
		t.Fatal("NewS3Client() returned nil client")
	}
}

func TestNewS3Client_HTTP(t *testing.T) {
	store := &objectstorage.ObjectStorage{
		Slug:      "my-store",
		APIKey:    "key",
		APISecret: "secret",
		Region: &objectstorage.Region{
			CloudProviderSetup: &objectstorage.RegionCloudProviderSetup{
				Config: objectstorage.RegionSetupConfig{S3Endpoint: "http://10.18.20.21:7480"},
			},
		},
	}
	mc, err := objectstorage.NewS3Client(store)
	if err != nil {
		t.Fatalf("NewS3Client() error = %v", err)
	}
	if mc == nil {
		t.Fatal("NewS3Client() returned nil client")
	}
}

func TestNewS3Client_InvalidEndpoint(t *testing.T) {
	store := &objectstorage.ObjectStorage{
		Slug:      "my-store",
		APIKey:    "key",
		APISecret: "secret",
		Region: &objectstorage.Region{
			CloudProviderSetup: &objectstorage.RegionCloudProviderSetup{
				Config: objectstorage.RegionSetupConfig{S3Endpoint: "://bad url"},
			},
		},
	}
	_, err := objectstorage.NewS3Client(store)
	if err == nil {
		t.Fatal("expected error for invalid endpoint URL, got nil")
	}
}

func TestNewS3Client_NoScheme(t *testing.T) {
	store := &objectstorage.ObjectStorage{
		Slug:      "my-store",
		APIKey:    "key",
		APISecret: "secret",
		Region: &objectstorage.Region{
			CloudProviderSetup: &objectstorage.RegionCloudProviderSetup{
				Config: objectstorage.RegionSetupConfig{S3Endpoint: "s3.example.com"},
			},
		},
	}
	_, err := objectstorage.NewS3Client(store)
	if err == nil {
		t.Fatal("expected error for endpoint without scheme, got nil")
	}
}
