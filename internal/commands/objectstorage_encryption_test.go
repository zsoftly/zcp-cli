package commands

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestObjectStorageEncryptionCommands(t *testing.T) {
	for _, algorithm := range []string{"AES256", "aws:kms"} {
		t.Run(algorithm, func(t *testing.T) {
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")
			t.Setenv("ZCP_S3_ACCESS_KEY", "test-access")
			t.Setenv("ZCP_S3_SECRET_KEY", "test-secret")
			var gotBody, gotAlgorithm, gotKey string
			s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				body, _ := io.ReadAll(r.Body)
				gotBody, gotAlgorithm, gotKey = string(body), r.Header.Get("X-Amz-Server-Side-Encryption"), r.Header.Get("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id")
				switch {
				case r.Method == http.MethodGet && r.URL.Query().Has("encryption"):
					fmt.Fprintf(w, `<ServerSideEncryptionConfiguration><Rule><ApplyServerSideEncryptionByDefault><SSEAlgorithm>%s</SSEAlgorithm><KMSMasterKeyID>service-key</KMSMasterKeyID></ApplyServerSideEncryptionByDefault></Rule></ServerSideEncryptionConfiguration>`, algorithm)
				case r.Header.Get("X-Amz-Copy-Source") != "":
					fmt.Fprint(w, `<CopyObjectResult><LastModified>2026-01-01T00:00:00.000Z</LastModified><ETag>"abc"</ETag></CopyObjectResult>`)
				default:
					w.Header().Set("ETag", `"abc"`)
				}
			}))
			defer s3.Close()
			mgmt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/object-storages/store/keys" {
					fmt.Fprint(w, `{"status":"Success","data":[{"api_key":"test-access","status":"active"}]}`)
					return
				}
				fmt.Fprintf(w, `{"status":"Success","data":{"slug":"store","region":{"slug":"os-test","cloud_provider_setup":{"config":{"s3_endpoint":%q}}}}}`, s3.URL)
			}))
			defer mgmt.Close()
			flags := []string{"--sse", algorithm, "--api-url", mgmt.URL}
			if algorithm == "aws:kms" {
				flags = append(flags, "--sse-kms-key-id", "service-key")
			}
			file := filepath.Join(t.TempDir(), "upload.txt")
			if err := os.WriteFile(file, []byte("upload bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{
				{"bucket", "encryption", "enable", "store", "bucket"},
				{"object", "put", "store", "bucket", file},
				{"object", "copy", "store", "bucket", "source", "destination", "target"},
				{"object", "move", "store", "bucket", "source", "destination", "target"},
			} {
				gotBody, gotAlgorithm, gotKey = "", "", ""
				// Move's final DELETE carries no encryption; retain copy observations.
				if args[0] == "bucket" && algorithm == "AES256" {
					args = append(args, "--api-url", mgmt.URL)
				} else {
					args = append(args, flags...)
				}
				_, _, err := execCmd(t, NewObjectStorageCmd(), args...)
				if err != nil {
					t.Fatalf("%v: %v", args, err)
				}
				if args[0] == "bucket" {
					if !strings.Contains(gotBody, "<SSEAlgorithm>"+algorithm+"</SSEAlgorithm>") {
						t.Fatalf("bucket XML = %q", gotBody)
					}
					if algorithm == "aws:kms" && !strings.Contains(gotBody, "<KMSMasterKeyID>service-key</KMSMasterKeyID>") {
						t.Fatalf("bucket XML = %q", gotBody)
					}
				} else {
					if gotAlgorithm != algorithm {
						t.Fatalf("algorithm = %q, want %q", gotAlgorithm, algorithm)
					}
					if algorithm == "aws:kms" && gotKey != "service-key" {
						t.Fatalf("KMS key = %q", gotKey)
					}
				}
			}
			stdout, _, err := execCmd(t, NewObjectStorageCmd(), "bucket", "encryption", "status", "store", "bucket", "--api-url", mgmt.URL)
			if err != nil || !strings.Contains(stdout, algorithm) || !strings.Contains(stdout, "service-key") {
				t.Fatalf("status = %q, %v", stdout, err)
			}
			if _, _, err := execCmd(t, NewObjectStorageCmd(), "bucket", "encryption", "disable", "store", "bucket", "--api-url", mgmt.URL); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestObjectStorageEncryptionValidationBeforeRequests(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; w.WriteHeader(http.StatusBadRequest) }))
	defer srv.Close()
	for _, command := range [][]string{
		{"bucket", "encryption", "enable", "store", "bucket"},
		{"object", "put", "store", "bucket", "unused"},
		{"object", "copy", "store", "src", "key", "dst", "key"},
		{"object", "move", "store", "src", "key", "dst", "key"},
	} {
		for _, flags := range [][]string{
			{"--sse", "invalid"}, {"--sse", "aws:kms"},
			{"--sse", "AES256", "--sse-kms-key-id", "key"},
			{"--sse", "aws:kms", "--sse-kms-key-id", "mount/key"},
		} {
			args := append(append(append([]string{}, command...), flags...), "--api-url", srv.URL)
			if _, _, err := execCmd(t, NewObjectStorageCmd(), args...); err == nil || !strings.Contains(err.Error(), "--sse") {
				t.Fatalf("%v: %v", args, err)
			}
		}
	}
	if requests != 0 {
		t.Fatalf("invalid selections made %d requests", requests)
	}
}
