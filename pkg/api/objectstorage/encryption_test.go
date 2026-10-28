package objectstorage_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/zsoftly/zcp-cli/pkg/api/objectstorage"
)

func TestEncryptionRegionalErrorsPreserveS3Cause(t *testing.T) {
	for _, tc := range []struct {
		code, message string
		status        int
		guidance      bool
	}{
		{"NotImplemented", "unsupported operation", 501, true},
		{"InvalidArgument", "Request specifies Server Side Encryption but server configuration does not support this.", 400, true},
		{"InvalidArgument", "invalid key selection", 400, false},
		{"AccessDenied", "denied", 403, false},
	} {
		t.Run(tc.code+"/"+tc.message, func(t *testing.T) {
			for _, operation := range []string{"get", "set", "delete", "put", "copy", "move"} {
				t.Run(operation, func(t *testing.T) {
					deleted := false
					mgmt, _ := newS3TestPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete && !r.URL.Query().Has("encryption") {
							deleted = true
						}
						w.WriteHeader(tc.status)
						fmt.Fprintf(w, "<Error><Code>%s</Code><Message>%s</Message></Error>", tc.code, tc.message)
					}))
					svc := objectstorage.NewService(newTestClient(t, mgmt))
					ctx := context.Background()
					selection := objectstorage.EncryptionOptions{Algorithm: "AES256"}
					var err error
					switch operation {
					case "get":
						_, err = svc.GetBucketEncryption(ctx, "my-storage-1", "bucket")
					case "set":
						err = svc.SetBucketEncryption(ctx, "my-storage-1", "bucket", selection)
					case "delete":
						err = svc.DisableBucketEncryption(ctx, "my-storage-1", "bucket")
					case "put":
						file := filepath.Join(t.TempDir(), "upload")
						if err := os.WriteFile(file, []byte("data"), 0600); err != nil {
							t.Fatal(err)
						}
						_, err = svc.PutObject(ctx, "my-storage-1", "bucket", file, "key", "", nil, selection)
					case "copy":
						err = svc.CopyObject(ctx, "my-storage-1", "source", "key", "destination", "key", selection)
					case "move":
						err = svc.MoveObject(ctx, "my-storage-1", "source", "key", "destination", "key", selection)
					}
					if err == nil {
						t.Fatal("expected S3 error")
					}
					if strings.Contains(err.Error(), "regional encryption support") != tc.guidance {
						t.Fatalf("regional guidance mismatch: %v", err)
					}
					var cause minio.ErrorResponse
					if !errors.As(err, &cause) || cause.Code != tc.code {
						t.Fatalf("S3 cause lost: %v", err)
					}
					if deleted {
						t.Fatal("failed copy deleted source")
					}
				})
			}
		})
	}
}

func TestCopyObjectRejectsInPlaceReencryption(t *testing.T) {
	requests := 0
	mgmt, _ := newS3TestPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++ }))
	err := objectstorage.NewService(newTestClient(t, mgmt)).CopyObject(context.Background(), "my-storage-1", "bucket", "key", "bucket", "key", objectstorage.EncryptionOptions{Algorithm: "AES256"})
	if err == nil || !strings.Contains(err.Error(), "in-place re-encryption") || requests != 0 {
		t.Fatalf("requests=%d err=%v", requests, err)
	}
}
