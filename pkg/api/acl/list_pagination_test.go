package acl_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/acl"
	"github.com/zsoftly/zcp-cli/pkg/api/vpc"
)

// Exercise both public list paths against the same contract fixtures.
func TestACLListPagination(t *testing.T) {
	for _, service := range []string{"List", "ListACLs"} {
		for _, scenario := range []struct {
			name     string
			metadata string
			count    int
			requests int
		}{
			{"total", "total", 12, 2},
			{"last page", "last_page", 12, 2},
			{"empty page", "current_page", 12, 3},
			{"documented meta", "meta", 12, 2},
			{"one page", "total", 2, 1},
			{"legacy one page", "", 2, 1},
			{"empty list", "total", 0, 1},
		} {
			t.Run(service+"/"+scenario.name, func(t *testing.T) {
				requests := 0
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					if r.URL.Path != "/vpcs/my-vpc/network-acl-list" || r.Method != http.MethodGet {
						t.Errorf("request = %s %s", r.Method, r.URL.Path)
					}
					page, err := strconv.Atoi(r.URL.Query().Get("page"))
					if err != nil || page != requests || r.URL.Query().Get("per_page") != "100" {
						t.Errorf("query = %v, request %d", r.URL.Query(), requests)
					}
					if requests > scenario.requests {
						http.Error(w, "unexpected extra page", http.StatusBadRequest)
						return
					}
					data := make([]acl.NetworkACL, 0)
					// Model a server capped at 10 records despite per_page=100.
					for i := (requests - 1) * 10; i < requests*10 && i < scenario.count; i++ {
						data = append(data, acl.NetworkACL{
							ID: fmt.Sprintf("id-%d", i), Name: fmt.Sprintf("name-%d", i),
							Slug: fmt.Sprintf("slug-%d", i), Description: "description", Status: "Enabled",
						})
					}
					envelope := map[string]interface{}{"status": "Success", "data": data}
					switch scenario.metadata {
					case "total":
						envelope["current_page"], envelope["total"] = page, scenario.count
					case "last_page":
						envelope["current_page"], envelope["last_page"] = page, 2
					case "current_page":
						envelope["current_page"] = page
					case "meta":
						envelope["meta"] = map[string]int{"page": page, "total": scenario.count, "per_page": 10}
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(envelope)
				}))
				defer srv.Close()
				ids, err := listACLIDs(service, srv.URL)
				if err != nil {
					t.Fatalf("%s() error = %v", service, err)
				}
				if len(ids) != scenario.count || requests != scenario.requests {
					t.Fatalf("got %d ACLs in %d requests, want %d in %d", len(ids), requests, scenario.count, scenario.requests)
				}
				for i, id := range ids {
					if id != fmt.Sprintf("id-%d", i) {
						t.Errorf("ACL %d = %q", i, id)
					}
				}
			})
		}
	}
}

func listACLIDs(service, baseURL string) ([]string, error) {
	var ids []string
	if service == "List" {
		items, err := acl.NewService(newClient(baseURL)).List(context.Background(), "my-vpc")
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		return ids, err
	}
	items, err := vpc.NewService(newClient(baseURL)).ListACLs(context.Background(), "my-vpc")
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids, err
}

func TestACLListPaginationDoesNotRepeatPages(t *testing.T) {
	for _, service := range []string{"List", "ListACLs"} {
		for _, staleMetadata := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/stale-metadata=%t", service, staleMetadata), func(t *testing.T) {
				requests := 0
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					if requests > 2 {
						http.Error(w, "pagination failed to stop", http.StatusBadRequest)
						return
					}
					page := requests
					if staleMetadata {
						page = 1
					}
					fmt.Fprintf(w, `{"current_page":%d,"total":12,"data":[{"id":"same-first-id"}]}`, page)
				}))
				defer srv.Close()
				ids, err := listACLIDs(service, srv.URL)
				if err == nil || len(ids) != 0 || requests != 2 {
					t.Fatalf("got ids=%v err=%v requests=%d; want error without partial results after two requests", ids, err, requests)
				}
				if !strings.Contains(err.Error(), "returned page 1") && !strings.Contains(err.Error(), "repeated") {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	}
}

func TestResolveACLOnSecondPage(t *testing.T) {
	for _, reference := range []string{"oldest-id", "oldest-name", "oldest-slug"} {
		t.Run(reference, func(t *testing.T) {
			requests := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				page := r.URL.Query().Get("page")
				var items []acl.NetworkACL
				switch page {
				case "1":
					for i := 0; i < 10; i++ {
						items = append(items, acl.NetworkACL{ID: fmt.Sprintf("newer-%d", i)})
					}
				case "2":
					items = []acl.NetworkACL{{ID: "older-id"}, {ID: "oldest-id", Name: "oldest-name", Slug: "oldest-slug"}}
				default:
					http.Error(w, "unexpected page", http.StatusBadRequest)
					return
				}
				json.NewEncoder(w).Encode(map[string]interface{}{"current_page": requests, "total": 12, "data": items})
			}))
			defer srv.Close()
			id, err := acl.NewService(newClient(srv.URL)).Resolve(context.Background(), "my-vpc", reference)
			if err != nil || id != "oldest-id" || requests != 2 {
				t.Fatalf("Resolve(%q) = %q, %v; requests=%d", reference, id, err, requests)
			}
		})
	}
}

func TestACLListPaginationRequestFailure(t *testing.T) {
	for _, service := range []string{"List", "ListACLs"} {
		t.Run(service, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") == "1" {
					fmt.Fprint(w, `{"current_page":1,"total":12,"data":[{"id":"first"}]}`)
					return
				}
				http.Error(w, "page unavailable", http.StatusBadRequest)
			}))
			defer srv.Close()
			ids, err := listACLIDs(service, srv.URL)
			if err == nil || len(ids) != 0 {
				t.Fatalf("got %v, %v; want error without partial results", ids, err)
			}
		})
	}
}

func TestACLListPaginationPageLimit(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprintf(w, `{"current_page":%d,"data":[{"id":"id-%d"}]}`, requests, requests)
	}))
	defer srv.Close()
	ids, err := listACLIDs("List", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "exceeded 1000 pages") || len(ids) != 0 || requests != 1000 {
		t.Fatalf("got %d ACLs, %v, %d requests; want page-limit error after 1000 requests", len(ids), err, requests)
	}
}
