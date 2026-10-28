package objectstorage_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/zsoftly/zcp-cli/pkg/api/objectstorage"
)

// The HTTP double checks the real SDK request sequence.
func TestEncryptedCopyAndMove(t *testing.T) {
	for _, sourceAlgorithm := range []string{"", "AES256", "aws:kms"} {
		for _, selection := range []objectstorage.EncryptionOptions{{}, {Algorithm: "AES256"}, {Algorithm: "aws:kms", KMSKeyID: "service-key"}} {
			if sourceAlgorithm == "" && selection.Algorithm == "" {
				continue
			}
			for _, size := range []int64{0, 10, 33 * 1024 * 1024} {
				for _, move := range []bool{false, true} {
					t.Run(fmt.Sprintf("source=%s/destination=%s/size=%d/move=%t", sourceAlgorithm, selection.Algorithm, size, move), func(t *testing.T) {
						algorithm, key := selection.Algorithm, selection.KMSKeyID
						if algorithm == "" {
							algorithm = sourceAlgorithm
							if algorithm == "aws:kms" {
								key = "service-key"
							}
						}
						var initiated, complete, verified, deleted bool
						var ranges []string
						mgmt, _ := newS3TestPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							q := r.URL.Query()
							switch {
							case r.Method == http.MethodHead && r.URL.Path == "/source/key":
								copyHead(w, size, sourceAlgorithm, "service-key")
								w.Header().Set("Content-Type", "application/octet-stream")
								w.Header().Set("X-Amz-Meta-Owner", "alice")
							case r.Method == http.MethodHead && !complete:
								w.WriteHeader(http.StatusNotFound)
							case r.Method == http.MethodHead:
								verified = true
								copyHead(w, size, algorithm, key)
								w.Header().Set("ETag", `"copy-etag"`)
							case r.Method == http.MethodPut && !q.Has("uploadId") && size == 0:
								if (r.ContentLength != 0 && r.Header.Get("X-Amz-Decoded-Content-Length") != "0") || r.Header.Get("X-Amz-Server-Side-Encryption") != algorithm || r.Header.Get("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id") != key || r.Header.Get("X-Amz-Meta-Owner") != "alice" {
									t.Errorf("empty PUT: %v", r.Header)
								}
								complete = true
								w.Header().Set("ETag", `"copy-etag"`)
							case r.Method == http.MethodPost && q.Has("uploads"):
								initiated = true
								if r.Header.Get("X-Amz-Server-Side-Encryption") != algorithm || r.Header.Get("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id") != key {
									t.Errorf("init encryption: %v", r.Header)
								}
								if r.Header.Get("X-Amz-Meta-Owner") != "alice" || r.Header.Get("Content-Type") != "application/octet-stream" {
									t.Errorf("lost source metadata: %v", r.Header)
								}
								fmt.Fprint(w, `<InitiateMultipartUploadResult><UploadId>ours</UploadId></InitiateMultipartUploadResult>`)
							case r.Method == http.MethodPut && q.Get("uploadId") == "ours":
								if !initiated || r.Header.Get("X-Amz-Copy-Source") != "source/key" || r.Header.Get("X-Amz-Copy-Source-If-Match") != "source-etag" {
									t.Errorf("part headers: %v", r.Header)
								}
								ranges = append(ranges, r.Header.Get("X-Amz-Copy-Source-Range"))
								fmt.Fprint(w, `<CopyPartResult><ETag>"part"</ETag><LastModified>2026-01-01T00:00:00Z</LastModified></CopyPartResult>`)
							case r.Method == http.MethodPost && q.Get("uploadId") == "ours":
								complete = true
								fmt.Fprint(w, `<CompleteMultipartUploadResult><Bucket>destination</Bucket><Key>copy</Key><ETag>"copy-etag"</ETag></CompleteMultipartUploadResult>`)
							case r.Method == http.MethodDelete && r.URL.Path == "/source/key":
								if !verified {
									t.Error("source deleted before encryption verification")
								}
								deleted = true
								w.WriteHeader(http.StatusNoContent)
							default:
								t.Errorf("unexpected request: %s %s", r.Method, r.URL)
								w.WriteHeader(http.StatusBadRequest)
							}
						}))
						svc := objectstorage.NewService(newTestClient(t, mgmt))
						op := svc.CopyObject
						if move {
							op = svc.MoveObject
						}
						err := op(context.Background(), "my-storage-1", "source", "key", "destination", "copy", selection)
						if err != nil || !complete || !verified || deleted != move {
							t.Fatalf("complete=%t verified=%t deleted=%t err=%v", complete, verified, deleted, err)
						}
						want := []string{"bytes=0-9"}
						if size == 0 {
							want = nil
						}
						if size > 10 {
							want = []string{"bytes=0-16777215", "bytes=16777216-33554431", "bytes=33554432-34603007"}
						}
						if strings.Join(ranges, ",") != strings.Join(want, ",") {
							t.Fatalf("ranges=%v want=%v", ranges, want)
						}
					})
				}
			}
		}
	}
}

func copyHead(w http.ResponseWriter, size int64, algorithm, key string) {
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Last-Modified", "Thu, 01 Jan 2026 00:00:00 GMT")
	w.Header().Set("ETag", `"source-etag"`)
	if algorithm != "" {
		w.Header().Set("X-Amz-Server-Side-Encryption", algorithm)
	}
	if algorithm == "aws:kms" {
		w.Header().Set("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id", key)
	}
}

func TestEncryptedCopyFailuresPreserveSource(t *testing.T) {
	for _, fault := range []string{"source-head", "destination-head", "empty-put", "init", "part", "part-embedded", "part-empty", "complete", "algorithm", "key", "verification-head", "cleanup", "abort", "complete-empty", "head-size", "head-etag", "cancel"} {
		for _, existing := range []bool{false, true} {
			for _, move := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/existing=%t/move=%t", fault, existing, move), func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					var initiated, completed, aborted, removed bool
					mgmt, _ := newS3TestPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						q := r.URL.Query()
						fail := func() {
							w.WriteHeader(http.StatusForbidden)
							fmt.Fprint(w, `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`)
						}
						switch {
						case r.Method == http.MethodHead && r.URL.Path == "/source/key":
							if fault == "source-head" {
								fail()
								return
							}
							size := int64(10)
							if fault == "empty-put" {
								size = 0
							}
							copyHead(w, size, "", "")
						case r.Method == http.MethodHead && !completed:
							if fault == "destination-head" {
								fail()
								return
							}
							if existing {
								copyHead(w, 10, "", "")
							} else {
								w.WriteHeader(http.StatusNotFound)
							}
						case r.Method == http.MethodHead:
							if fault == "verification-head" {
								fail()
								return
							}
							algorithm, key := "aws:kms", "service-key"
							if fault == "algorithm" || fault == "cleanup" {
								algorithm = ""
							}
							if fault == "key" {
								key = "wrong-key"
							}
							copyHead(w, 10, algorithm, key)
							w.Header().Set("ETag", `"copy-etag"`)
							if fault == "head-size" {
								w.Header().Set("Content-Length", "1")
							}
							if fault == "head-etag" {
								w.Header().Set("ETag", `"other"`)
							}
						case r.Method == http.MethodPut && !q.Has("uploadId") && fault == "empty-put":
							fail()
						case r.Method == http.MethodPost && q.Has("uploads"):
							if fault == "init" {
								fail()
								return
							}
							initiated = true
							fmt.Fprint(w, `<InitiateMultipartUploadResult><UploadId>ours</UploadId></InitiateMultipartUploadResult>`)
						case r.Method == http.MethodPut && q.Get("uploadId") == "ours":
							switch fault {
							case "part", "abort", "cancel":
								if fault == "cancel" {
									cancel()
								}
								fail()
							case "part-embedded":
								fmt.Fprint(w, `<Error><Code>NotImplemented</Code><Message>copy unavailable</Message><RequestId>part-request</RequestId></Error>`)
							case "part-empty":
								fmt.Fprint(w, `<CopyPartResult/>`)
							default:
								fmt.Fprint(w, `<CopyPartResult><ETag>"part"</ETag></CopyPartResult>`)
							}
						case r.Method == http.MethodPost && q.Get("uploadId") == "ours":
							if fault == "complete" {
								fail()
								return
							}
							completed = true
							if fault == "complete-empty" {
								fmt.Fprint(w, `<CompleteMultipartUploadResult><Bucket>destination</Bucket></CompleteMultipartUploadResult>`)
								return
							}
							fmt.Fprint(w, `<CompleteMultipartUploadResult><Bucket>destination</Bucket><Key>copy</Key><ETag>"copy-etag"</ETag></CompleteMultipartUploadResult>`)
						case r.Method == http.MethodDelete && q.Get("uploadId") == "ours":
							aborted = true
							if fault == "abort" {
								fail()
								return
							}
							w.WriteHeader(http.StatusNoContent)
						case r.Method == http.MethodDelete && r.URL.Path == "/destination/copy":
							if existing {
								t.Error("removed pre-existing destination")
							}
							removed = true
							if fault == "cleanup" {
								fail()
								return
							}
							w.WriteHeader(http.StatusNoContent)
						case r.Method == http.MethodDelete && r.URL.Path == "/source/key":
							t.Error("failure deleted source")
							w.WriteHeader(http.StatusNoContent)
						default:
							t.Errorf("unexpected %s %s", r.Method, r.URL)
							fail()
						}
					}))
					svc := objectstorage.NewService(newTestClient(t, mgmt))
					op := svc.CopyObject
					if move {
						op = svc.MoveObject
					}
					err := op(ctx, "my-storage-1", "source", "key", "destination", "copy", objectstorage.EncryptionOptions{Algorithm: "aws:kms", KMSKeyID: "service-key"})
					if err == nil {
						t.Fatal("expected failure")
					}
					if aborted != (initiated && !completed) {
						t.Fatalf("initiated=%t completed=%t aborted=%t err=%v", initiated, completed, aborted, err)
					}
					if removed != (completed && !existing) {
						t.Fatalf("completed=%t existing=%t removed=%t err=%v", completed, existing, removed, err)
					}
					if completed {
						want := "removed newly created unverified destination"
						if existing {
							want = "destination existed before this command"
						} else if fault == "cleanup" {
							want = "removing newly created unverified destination failed"
						}
						if !strings.Contains(err.Error(), want) {
							t.Fatalf("missing cleanup diagnostic: %v", err)
						}
					}
					if fault == "abort" && !strings.Contains(err.Error(), "aborting incomplete copy upload ours") {
						t.Fatalf("missing abort error: %v", err)
					}
					if fault == "part-embedded" {
						var cause minio.ErrorResponse
						if !errors.As(err, &cause) || cause.Code != "NotImplemented" || cause.RequestID != "part-request" {
							t.Fatalf("lost embedded cause: %v", err)
						}
					}
				})
			}
		}
	}
}
