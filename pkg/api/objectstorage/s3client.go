package objectstorage

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
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

	transport, err := minio.DefaultTransport(u.Scheme == "https")
	if err != nil {
		return nil, err
	}
	return minio.New(host, &minio.Options{
		Transport:    copyErrorTransport{transport},
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

// MinIO decodes HTTP-200 CopyObject errors as empty success results. Return the
// S3 cause from the body reader so its XML decoder retains it without transport
// retries or changing the HTTP status.
type copyErrorTransport struct {
	http.RoundTripper
}

func (t copyErrorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.RoundTripper.RoundTrip(req)
	if err == nil && resp.StatusCode == http.StatusOK && req.Method == http.MethodPut && req.Header.Get("X-Amz-Copy-Source") != "" && !req.URL.Query().Has("uploadId") {
		resp.Body = &copyErrorBody{ReadCloser: resp.Body}
	}
	return resp, err
}

type copyErrorBody struct {
	io.ReadCloser
	reader *bytes.Reader
	err    error
}

func (b *copyErrorBody) Read(p []byte) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	if b.reader == nil {
		body, err := io.ReadAll(b.ReadCloser)
		if err != nil {
			b.err = err
			return 0, err
		}
		var cause minio.ErrorResponse
		if xml.Unmarshal(body, &cause) == nil && cause.Code != "" {
			cause.StatusCode = http.StatusOK
			b.err = cause
			return 0, b.err
		}
		b.reader = bytes.NewReader(body)
	}
	return b.reader.Read(p)
}
