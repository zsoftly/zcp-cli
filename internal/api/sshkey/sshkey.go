// Package sshkey provides ZCP SSH key API operations.
package sshkey

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// SSHKey represents a ZCP SSH key.
type SSHKey struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	IsActive   bool   `json:"isActive"`
	DomainName string `json:"domainName"`
}

// CreateRequest holds parameters for creating/importing an SSH key.
type CreateRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

type listSSHKeyResponse struct {
	Count              int      `json:"count"`
	ListSSHKeyResponse []SSHKey `json:"listSSHKeyResponse"`
}

// Service provides SSH key API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new SSH key Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// List returns all SSH keys for the authenticated domain.
func (s *Service) List(ctx context.Context) ([]SSHKey, error) {
	var resp listSSHKeyResponse
	if err := s.client.Get(ctx, "/restapi/sshkey/sshkeyList", url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("listing SSH keys: %w", err)
	}
	return resp.ListSSHKeyResponse, nil
}

// Create imports an SSH public key with the given name.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*SSHKey, error) {
	var resp listSSHKeyResponse
	if err := s.client.Post(ctx, "/restapi/sshkey/createSSHkey", req, &resp); err != nil {
		return nil, fmt.Errorf("creating SSH key: %w", err)
	}
	if len(resp.ListSSHKeyResponse) == 0 {
		return nil, fmt.Errorf("create SSH key returned empty response")
	}
	return &resp.ListSSHKeyResponse[0], nil
}

// Delete removes an SSH key by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/sshkey/deleteSSHkey/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting SSH key %s: %w", uuid, err)
	}
	return nil
}
