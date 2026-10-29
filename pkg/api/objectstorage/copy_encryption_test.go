package objectstorage_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
						metadata := map[string]string{
							"Content-Type": "application/example", "Cache-Control": "max-age=3600",
							"Content-Disposition": `inline; filename="data.bin"`, "Content-Encoding": "gzip",
							"Content-Language": "en-CA", "Expires": "Wed, 01 Jan 2031 00:00:00 GMT",
							"X-Amz-Meta-Owner": "alice", "X-Amz-Meta-Purpose": "copy-test",
						}
						assertMetadata := func(r *http.Request) {
							for header, want := range metadata {
								// The SDK adds its streaming transport encoding to empty PUTs.
								if header == "Content-Encoding" && size == 0 {
									want = "aws-chunked," + want
								}
								if got := r.Header.Get(header); got != want {
									t.Errorf("%s=%q want %q", header, got, want)
								}
							}
							if r.Header.Get("X-Amz-Tagging") != "owner=alice" {
								t.Errorf("tags=%q", r.Header.Get("X-Amz-Tagging"))
							}
						}
						var initiated, complete, verified, deleted bool
						var ranges []string
						mgmt, _ := newS3TestPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							q := r.URL.Query()
							switch {
							case r.Method == http.MethodHead && r.URL.Path == "/source/key":
								copyHead(w, size, sourceAlgorithm, "service-key")
								for header, value := range metadata {
									w.Header().Set(header, value)
								}
								w.Header().Set("X-Amz-Tagging-Count", "1")
							case r.Method == http.MethodGet && q.Has("tagging"):
								fmt.Fprint(w, `<Tagging><TagSet><Tag><Key>owner</Key><Value>alice</Value></Tag></TagSet></Tagging>`)
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
								assertMetadata(r)
								complete = true
								w.Header().Set("ETag", `"copy-etag"`)
							case r.Method == http.MethodPost && q.Has("uploads"):
								initiated = true
								if r.Header.Get("X-Amz-Server-Side-Encryption") != algorithm || r.Header.Get("X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id") != key {
									t.Errorf("init encryption: %v", r.Header)
								}
								assertMetadata(r)
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
	for _, fault := range []string{"source-head", "empty-put", "init", "part", "part-embedded", "part-empty", "complete", "algorithm", "key", "verification-head", "cleanup", "abort", "complete-empty", "head-size", "head-etag", "cancel"} {
		for _, version := range []string{"", "null", "our-version"} {
			for _, move := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/version=%s/move=%t", fault, version, move), func(t *testing.T) {
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
							w.Header().Set("X-Amz-Version-Id", version)
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
							if q.Get("versionId") != "our-version" {
								t.Errorf("deleted unowned destination version: %s", r.URL)
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
					if removed != (completed && version == "our-version") {
						t.Fatalf("completed=%t version=%s removed=%t err=%v", completed, version, removed, err)
					}
					if completed {
						want := "destination may not be encrypted as requested and was left in place"
						if version == "our-version" {
							want = "removed this operation's unverified destination version"
							if fault == "cleanup" {
								want = "removing this operation's unverified destination version failed"
							}
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

func TestEncryptedCopyCompetingDestination(t *testing.T) {
	for _, size := range []int64{0, 10} {
		for _, version := range []string{"", "null", "our-version"} {
			for _, move := range []bool{false, true} {
				t.Run(fmt.Sprintf("size=%d/version=%s/move=%t", size, version, move), func(t *testing.T) {
					completed, competitorPresent, removedOwnVersion, sourceRemoved := false, false, false, false
					mgmt, _ := newS3TestPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						q := r.URL.Query()
						switch {
						case r.Method == http.MethodHead && r.URL.Path == "/source/key":
							copyHead(w, size, "", "")
						case r.Method == http.MethodHead:
							// A competing PUT became current after our write and before verification.
							if !completed || !competitorPresent {
								t.Error("verification before writes")
							}
							if q.Get("versionId") != version {
								t.Errorf("verification version=%q want %q", q.Get("versionId"), version)
							}
							if version == "our-version" {
								// Pinned HEAD reads our version, whose SSE is incorrect.
								copyHead(w, size, "", "")
								w.Header().Set("ETag", `"ours"`)
							} else {
								copyHead(w, 77, "", "")
								w.Header().Set("ETag", `"another-writer"`)
							}
						case r.Method == http.MethodPost && q.Has("uploads"):
							fmt.Fprint(w, `<InitiateMultipartUploadResult><UploadId>ours</UploadId></InitiateMultipartUploadResult>`)
						case r.Method == http.MethodPut && size == 0:
							completed = true
							competitorPresent = true
							w.Header().Set("ETag", `"ours"`)
							w.Header().Set("X-Amz-Version-Id", version)
						case r.Method == http.MethodPut:
							fmt.Fprint(w, `<CopyPartResult><ETag>"part"</ETag></CopyPartResult>`)
						case r.Method == http.MethodPost:
							completed = true
							competitorPresent = true
							w.Header().Set("X-Amz-Version-Id", version)
							fmt.Fprint(w, `<CompleteMultipartUploadResult><Bucket>destination</Bucket><Key>copy</Key><ETag>"ours"</ETag></CompleteMultipartUploadResult>`)
						case r.Method == http.MethodDelete && r.URL.Path == "/destination/copy":
							if q.Get("versionId") != "our-version" {
								competitorPresent = false
								t.Error("deleted competing writer")
							}
							removedOwnVersion = true
							w.WriteHeader(http.StatusNoContent)
						case r.Method == http.MethodDelete:
							sourceRemoved = true
							w.WriteHeader(http.StatusNoContent)
						default:
							t.Errorf("unexpected %s %s", r.Method, r.URL)
							w.WriteHeader(http.StatusBadRequest)
						}
					}))
					svc := objectstorage.NewService(newTestClient(t, mgmt))
					op := svc.CopyObject
					if move {
						op = svc.MoveObject
					}
					err := op(context.Background(), "my-storage-1", "source", "key", "destination", "copy", objectstorage.EncryptionOptions{Algorithm: "AES256"})
					if err == nil || sourceRemoved || !competitorPresent || removedOwnVersion != (version == "our-version") {
						t.Fatalf("err=%v sourceRemoved=%t competitorPresent=%t removedOwnVersion=%t", err, sourceRemoved, competitorPresent, removedOwnVersion)
					}
					if version != "our-version" && !strings.Contains(err.Error(), "destination may not be encrypted as requested and was left in place") {
						t.Fatalf("missing retained-destination diagnostic: %v", err)
					}
				})
			}
		}
	}
}

func TestEncryptedCopyPinsSourceTags(t *testing.T) {
	for _, size := range []int64{0, 10} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			sourceHead, readV1Tags, copiedV1 := false, false, false
			mgmt, _ := newS3TestPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				switch {
				case r.Method == http.MethodHead && r.URL.Path == "/source/key":
					sourceHead = true
					copyHead(w, size, "", "")
					w.Header().Set("X-Amz-Version-Id", "v1")
					w.Header().Set("X-Amz-Tagging-Count", "1")
				case r.Method == http.MethodGet && q.Has("tagging"):
					// V2 appears after HEAD. An unpinned read returns its different tags.
					if !sourceHead {
						t.Error("tag read before HEAD")
					}
					generation := "v2"
					if q.Get("versionId") == "v1" {
						generation = "v1"
						readV1Tags = true
					}
					fmt.Fprintf(w, `<Tagging><TagSet><Tag><Key>generation</Key><Value>%s</Value></Tag></TagSet></Tagging>`, generation)
				case r.Method == http.MethodHead:
					copyHead(w, size, "AES256", "")
					w.Header().Set("ETag", `"ours"`)
				case (r.Method == http.MethodPost && q.Has("uploads")) || (r.Method == http.MethodPut && size == 0):
					tags, _ := url.ParseQuery(r.Header.Get("X-Amz-Tagging"))
					if tags.Get("generation") != "v1" {
						t.Errorf("copied tags=%v", tags)
					}
					if size == 0 {
						copiedV1 = true
						w.Header().Set("ETag", `"ours"`)
					} else {
						fmt.Fprint(w, `<InitiateMultipartUploadResult><UploadId>ours</UploadId></InitiateMultipartUploadResult>`)
					}
				case r.Method == http.MethodPut:
					copiedV1 = strings.Contains(r.Header.Get("X-Amz-Copy-Source"), "versionId=v1")
					fmt.Fprint(w, `<CopyPartResult><ETag>"part"</ETag></CopyPartResult>`)
				case r.Method == http.MethodPost:
					fmt.Fprint(w, `<CompleteMultipartUploadResult><Bucket>destination</Bucket><Key>copy</Key><ETag>"ours"</ETag></CompleteMultipartUploadResult>`)
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			err := objectstorage.NewService(newTestClient(t, mgmt)).CopyObject(context.Background(), "my-storage-1", "source", "key", "destination", "copy", objectstorage.EncryptionOptions{Algorithm: "AES256"})
			if err != nil || !readV1Tags || !copiedV1 {
				t.Fatalf("err=%v readV1Tags=%t copiedV1=%t", err, readV1Tags, copiedV1)
			}
		})
	}
}
