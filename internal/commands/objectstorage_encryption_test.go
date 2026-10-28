package commands

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/objectstorage"
)

func TestObjectStorageEncryptionCommands(t *testing.T) {
	for _, algorithm := range []string{"AES256", "aws:kms"} {
		t.Run(algorithm, func(t *testing.T) {
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")
			t.Setenv("ZCP_S3_ACCESS_KEY", "test-access")
			t.Setenv("ZCP_S3_SECRET_KEY", "test-secret")
			var gotBody, gotAlgorithm, gotKey string
			completed := false
			s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					if r.URL.Path == "/destination/target" && !completed {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.Header().Set("Content-Length", "12")
					w.Header().Set("Last-Modified", "Thu, 01 Jan 2026 00:00:00 GMT")
					w.Header().Set("ETag", `"abc"`)
					if r.URL.Path == "/destination/target" {
						w.Header().Set("X-Amz-Server-Side-Encryption", algorithm)
						if algorithm == "aws:kms" {
							w.Header().Set("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id", "service-key")
						}
					}
					return
				}
				if r.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				body, _ := io.ReadAll(r.Body)
				gotBody, gotAlgorithm, gotKey = string(body), r.Header.Get("X-Amz-Server-Side-Encryption"), r.Header.Get("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id")
				switch {
				case r.Method == http.MethodGet && r.URL.Query().Has("encryption"):
					fmt.Fprintf(w, `<ServerSideEncryptionConfiguration><Rule><ApplyServerSideEncryptionByDefault><SSEAlgorithm>%s</SSEAlgorithm><KMSMasterKeyID>service-key</KMSMasterKeyID></ApplyServerSideEncryptionByDefault></Rule></ServerSideEncryptionConfiguration>`, algorithm)
				case r.Method == http.MethodPost && r.URL.Query().Has("uploads"):
					fmt.Fprint(w, `<InitiateMultipartUploadResult><UploadId>copy-upload</UploadId></InitiateMultipartUploadResult>`)
				case r.Method == http.MethodPost && r.URL.Query().Has("uploadId"):
					completed = true
					fmt.Fprint(w, `<CompleteMultipartUploadResult><Bucket>destination</Bucket><Key>target</Key><ETag>"abc"</ETag></CompleteMultipartUploadResult>`)
				case r.Header.Get("X-Amz-Copy-Source") != "":
					fmt.Fprint(w, `<CopyObjectResult><LastModified>2026-01-01T00:00:00.000Z</LastModified><ETag>"abc"</ETag></CopyObjectResult>`)
				default:
					w.Header().Set("ETag", `"abc"`)
				}
			}))
			defer s3.Close()
			mgmt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/object-storages/store/buckets/bucket" {
					fmt.Fprint(w, `{"status":"Success","data":{"slug":"bucket","name":"bucket"}}`)
					return
				}
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
				// Final HEAD and DELETE requests retain the multipart initiation observations.
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

func TestCopyEncryptionWarningAndHelp(t *testing.T) {
	for _, operation := range []string{"copy", "move"} {
		for _, selection := range []objectstorage.EncryptionOptions{{}, {Algorithm: "AES256"}, {Algorithm: "aws:kms", KMSKeyID: "service-key"}} {
			cmd := NewObjectStorageCmd()
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			warnOSCopyEncryption(cmd, selection)
			if got := strings.Contains(stderr.String(), "do not inherit destination bucket default encryption"); got != (selection.Algorithm == "") {
				t.Fatalf("selection=%+v warning=%q", selection, stderr.String())
			}
		}
		stdout, _, err := execCmd(t, NewObjectStorageCmd(), "object", operation, "--help")
		if err != nil || !strings.Contains(stdout, "destination bucket default") || !strings.Contains(stdout, "--sse") {
			t.Fatalf("%s help=%q err=%v", operation, stdout, err)
		}
	}
}

func TestBucketEncryptionCommandsResolveSlug(t *testing.T) {
	for _, operation := range []struct{ command, method string }{
		{"enable", http.MethodPut}, {"status", http.MethodGet}, {"disable", http.MethodDelete},
	} {
		for _, found := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/found=%t", operation.command, found), func(t *testing.T) {
				t.Setenv("ZCP_BEARER_TOKEN", "test-token")
				t.Setenv("ZCP_S3_ACCESS_KEY", "test-access")
				t.Setenv("ZCP_S3_SECRET_KEY", "test-secret")
				var requests []string
				s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests = append(requests, r.Method+" "+r.URL.Path)
					if r.Method != operation.method || strings.TrimSuffix(r.URL.Path, "/") != "/media" || !r.URL.Query().Has("encryption") {
						t.Errorf("unexpected S3 request: %s %s", r.Method, r.URL)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if r.Method == http.MethodGet {
						fmt.Fprint(w, `<ServerSideEncryptionConfiguration><Rule><ApplyServerSideEncryptionByDefault><SSEAlgorithm>AES256</SSEAlgorithm></ApplyServerSideEncryptionByDefault></Rule></ServerSideEncryptionConfiguration>`)
					} else if r.Method == http.MethodDelete {
						w.WriteHeader(http.StatusNoContent)
					}
				}))
				defer s3.Close()
				mgmt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests = append(requests, r.Method+" "+r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/object-storages/assets-x1/buckets/media-b1":
						if !found {
							w.WriteHeader(http.StatusNotFound)
							fmt.Fprint(w, `{"status":"Error","message":"Bucket not found."}`)
							return
						}
						fmt.Fprint(w, `{"status":"Success","data":{"slug":"media-b1","name":"media"}}`)
					case "/object-storages/assets-x1":
						fmt.Fprintf(w, `{"status":"Success","data":{"slug":"assets-x1","region":{"slug":"os-test","cloud_provider_setup":{"config":{"s3_endpoint":%q}}}}}`, s3.URL)
					case "/object-storages/assets-x1/keys":
						fmt.Fprint(w, `{"status":"Success","data":[{"api_key":"test-access","status":"active"}]}`)
					default:
						t.Errorf("unexpected management request: %s %s", r.Method, r.URL)
						w.WriteHeader(http.StatusNotFound)
					}
				}))
				defer mgmt.Close()
				stdout, _, err := execCmd(t, NewObjectStorageCmd(), "bucket", "encryption", operation.command, "assets-x1", "media-b1", "--api-url", mgmt.URL)
				want := "GET /object-storages/assets-x1/buckets/media-b1"
				if found {
					if err != nil || (operation.command == "status" && !strings.Contains(stdout, "media-b1")) {
						t.Fatalf("output=%q error=%v", stdout, err)
					}
					want += ",GET /object-storages/assets-x1,GET /object-storages/assets-x1/keys," + operation.method + " /media/"
				} else if err == nil || !strings.Contains(err.Error(), "getting bucket media-b1") {
					t.Fatalf("unresolved bucket error=%v", err)
				}
				if got := strings.Join(requests, ","); got != want {
					t.Fatalf("requests=%q, want %q", got, want)
				}
			})
		}
	}
}
