// Package sshkey provides STKCNSL SSH key API operations.
package sshkey

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// SSHKey represents a STKCNSL SSH key.
type SSHKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	PublicKey string `json:"public_key"`
	User      *Owner `json:"user,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Owner holds the user who owns the SSH key.
type Owner struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateRequest holds parameters for creating an SSH key. Project and Region
// are mandatory: the API derives the cloud provider from them, and omitting
// either makes the backend dereference a null and return
// 500 "Attempt to read property \"id\" on null".
type CreateRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	Project   string `json:"project"`
	Region    string `json:"region"`
}

// Service provides SSH key API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new SSH key Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// List returns all SSH keys for the authenticated user.
func (s *Service) List(ctx context.Context) ([]SSHKey, error) {
	var keys []SSHKey
	if err := s.client.GetEnvelope(ctx, "/users/ssh-keys", nil, &keys); err != nil {
		return nil, fmt.Errorf("listing SSH keys: %w", err)
	}
	return keys, nil
}

// Create imports an SSH public key with the given name. The key is registered
// for the account and can then be referenced by name at VM-create time.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*SSHKey, error) {
	var key SSHKey
	if err := s.client.PostEnvelope(ctx, "/users/ssh-keys", req, &key); err != nil {
		return nil, fmt.Errorf("creating SSH key: %w", err)
	}
	return &key, nil
}

// Delete removes an SSH key by its slug. The API rejects the key's ID (UUID)
// here as not-found (verified live 2026-07-19), so callers must pass the slug;
// the command layer resolves other identifiers to it.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/users/ssh-keys/"+slug, nil); err != nil {
		return fmt.Errorf("deleting SSH key %s: %w", slug, err)
	}
	return nil
}
