package objectstorage

import (
	"fmt"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// NewS3Client builds a minio S3 client from an ObjectStorage instance.
// The instance must have been fetched with the region+cloud_provider_setup includes
// so that S3Endpoint(), APIKey, and APISecret are populated.
func NewS3Client(store *ObjectStorage) (*minio.Client, error) {
	endpoint := store.S3Endpoint()
	if endpoint == "" {
		return nil, fmt.Errorf("object storage %q has no S3 endpoint (region.cloud_provider_setup missing)", store.Slug)
	}
	if store.APIKey == "" {
		return nil, fmt.Errorf("object storage %q is missing api_key", store.Slug)
	}
	if store.APISecret == "" {
		return nil, fmt.Errorf("object storage %q is missing api_secret", store.Slug)
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 endpoint %q: %w", endpoint, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid S3 endpoint %q: scheme must be http or https", endpoint)
	}
	host := u.Host
	if host == "" {
		return nil, fmt.Errorf("invalid S3 endpoint %q: missing host", endpoint)
	}

	return minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(store.APIKey, store.APISecret, ""),
		Secure:       u.Scheme == "https",
		BucketLookup: minio.BucketLookupPath,
	})
}
