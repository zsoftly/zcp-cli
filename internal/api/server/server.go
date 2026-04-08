// Package server provides ZCP server API operations (STKCNSL).
package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Server represents a STKCNSL server (e.g. "Cloud Compute").
type Server struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      bool   `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Icon        string `json:"icon"`
}

// envelope is the STKCNSL response wrapper.
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Service provides server API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new server Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all servers.
func (s *Service) List(ctx context.Context) ([]Server, error) {
	var env envelope
	if err := s.client.Get(ctx, "/servers", nil, &env); err != nil {
		return nil, fmt.Errorf("listing servers: %w", err)
	}

	var servers []Server
	if err := json.Unmarshal(env.Data, &servers); err != nil {
		return nil, fmt.Errorf("decoding servers: %w", err)
	}

	return servers, nil
}
