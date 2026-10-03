package objectstorage

import (
	"fmt"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// NewS3Client builds a minio S3 client from an ObjectStorage instance.
// The instance must have been fetched with the region+cloud_provider_setup includes
// so that its endpoint and an explicitly supplied S3 credential pair are populated.
func NewS3Client(store *ObjectStorage) (*minio.Client, error) {
	endpoint := store.S3Endpoint()
	if endpoint == "" {
		return nil, fmt.Errorf("object storage %q has no S3 endpoint (region.cloud_provider_setup missing)", store.Slug)
	}
	accessKey, secretKey, err := s3Credentials(store)
	if err != nil {
		return nil, err
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
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       u.Scheme == "https",
		BucketLookup: minio.BucketLookupPath,
		Region:       store.S3Region,
	})
}

func s3Credentials(store *ObjectStorage) (string, string, error) {
	if store.APIKey != "" && store.APISecret != "" {
		return store.APIKey, store.APISecret, nil
	}
	return "", "", fmt.Errorf("object storage %q is missing S3 credentials", store.Slug)
}
