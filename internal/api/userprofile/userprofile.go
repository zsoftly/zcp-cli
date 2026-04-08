// Package userprofile provides STKCNSL user profile, MFA, password,
// user management, time-settings, and API access operations.
package userprofile

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Profile represents the authenticated user's profile from GET /profile.
type Profile struct {
	User User `json:"user"`
}

// User holds the top-level user data within the profile.
type User struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	EmailVerifiedAt  *string  `json:"email_verified_at"`
	RegistrationType string   `json:"registration_type"`
	UserType         string   `json:"user_type"`
	Domain           string   `json:"domain"`
	IsTwoFactor      bool     `json:"is_two_factor"`
	IsBlocked        bool     `json:"is_blocked"`
	IsInvited        bool     `json:"is_invited"`
	LastLogin        string   `json:"last_login"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	Account          Account  `json:"account"`
	Company          *Company `json:"company"`
	Address          *Address `json:"address"`
}

// Account holds account-level metadata.
type Account struct {
	ID             string `json:"id"`
	CRN            string `json:"crn"`
	Status         string `json:"status"`
	AccountStatus  string `json:"account_status"`
	PaymentMode    string `json:"payment_mode"`
	Timezone       string `json:"timezone"`
	DateTimeFormat string `json:"date_time_format"`
	Enforce2FA     bool   `json:"enforce_2fa_to_all"`
	OwnerName      string `json:"owner_name"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// Company holds company details.
type Company struct {
	Name    string `json:"name"`
	Website string `json:"website"`
}

// Address holds an address record.
type Address struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Country     string `json:"country"`
	State       string `json:"state"`
	City        string `json:"city"`
	PostalCode  string `json:"postal_code"`
	Line1       string `json:"line_1"`
	Line2       string `json:"line_2"`
	BillingName string `json:"billing_name"`
}

// UpdateProfileRequest holds fields for PUT /profile.
type UpdateProfileRequest struct {
	Name string `json:"name,omitempty"`
}

// UpdateCompanyRequest holds fields for PUT /profile/company-details.
type UpdateCompanyRequest struct {
	BillingName string `json:"billing_name,omitempty"`
	Country     string `json:"country,omitempty"`
	State       string `json:"state,omitempty"`
	City        string `json:"city,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
	Line1       string `json:"line_1,omitempty"`
	Line2       string `json:"line_2,omitempty"`
	GST         string `json:"GST,omitempty"`
}

// TimeSettingsRequest holds fields for POST /profile/time-settings.
type TimeSettingsRequest struct {
	Timezone       string `json:"timezone"`
	DateTimeFormat string `json:"date_time_format,omitempty"`
}

// ChangePasswordRequest holds fields for POST /users/change-password.
type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current_password"`
	NewPassword        string `json:"password"`
	NewPasswordConfirm string `json:"password_confirmation"`
}

// MFASendOTPRequest holds fields for MFA OTP send endpoints.
type MFASendOTPRequest struct {
	Type string `json:"type,omitempty"` // "email" or "sms"
}

// MFAVerifyOTPRequest holds fields for MFA OTP verify endpoints.
type MFAVerifyOTPRequest struct {
	OTP string `json:"otp"`
}

// Enforce2FARequest holds fields for POST /account/enforce2fa.
type Enforce2FARequest struct {
	Enforce bool `json:"enforce_2fa_to_all"`
}

// CreateUserRequest holds fields for POST /api/users.
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	RoleID   string `json:"role_id,omitempty"`
}

// UpdateUserRequest holds fields for PUT /api/users/{id}.
type UpdateUserRequest struct {
	Name   string `json:"name,omitempty"`
	RoleID string `json:"role_id,omitempty"`
}

// ManagedUser represents a user managed under this account.
type ManagedUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	UserType  string `json:"user_type"`
	IsBlocked bool   `json:"is_blocked"`
	IsInvited bool   `json:"is_invited"`
	LastLogin string `json:"last_login"`
	CreatedAt string `json:"created_at"`
}

// LogEntry represents a login or activity log entry.
type LogEntry struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	IPAddress string `json:"ip_address"`
	Details   string `json:"details"`
	CreatedAt string `json:"created_at"`
}

// DocLinks holds the API documentation URL.
type DocLinks struct {
	URL string `json:"url"`
}

// StatusResponse is a generic response for operations that return a status message.
type StatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Service provides user profile API operations against STKCNSL.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new user profile Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// Get returns the authenticated user's profile.
func (s *Service) Get(ctx context.Context) (*Profile, error) {
	var p Profile
	if err := s.client.GetEnvelope(ctx, "/profile", nil, &p); err != nil {
		return nil, fmt.Errorf("getting profile: %w", err)
	}
	return &p, nil
}

// Update updates the authenticated user's profile.
func (s *Service) Update(ctx context.Context, req UpdateProfileRequest) (*Profile, error) {
	var p Profile
	if err := s.client.PutEnvelope(ctx, "/profile", nil, req, &p); err != nil {
		return nil, fmt.Errorf("updating profile: %w", err)
	}
	return &p, nil
}

// UpdateCompany updates the company/billing details.
func (s *Service) UpdateCompany(ctx context.Context, req UpdateCompanyRequest) error {
	if err := s.client.PutEnvelope(ctx, "/profile/company-details", nil, req, nil); err != nil {
		return fmt.Errorf("updating company details: %w", err)
	}
	return nil
}

// UpdateTimeSettings updates timezone and date format preferences.
func (s *Service) UpdateTimeSettings(ctx context.Context, req TimeSettingsRequest) error {
	if err := s.client.PostEnvelope(ctx, "/profile/time-settings", req, nil); err != nil {
		return fmt.Errorf("updating time settings: %w", err)
	}
	return nil
}

// EnableAPI enables API access for the account.
func (s *Service) EnableAPI(ctx context.Context) error {
	if err := s.client.PostEnvelope(ctx, "/profile/api/enable", nil, nil); err != nil {
		return fmt.Errorf("enabling API access: %w", err)
	}
	return nil
}

// DisableAPI disables API access for the account.
func (s *Service) DisableAPI(ctx context.Context) error {
	if err := s.client.Delete(ctx, "/profile/api/disable", nil); err != nil {
		return fmt.Errorf("disabling API access: %w", err)
	}
	return nil
}

// GetDocLinks returns the API documentation URL.
func (s *Service) GetDocLinks(ctx context.Context) (*DocLinks, error) {
	var d DocLinks
	if err := s.client.GetEnvelope(ctx, "/profile/swagger-doc-links", nil, &d); err != nil {
		return nil, fmt.Errorf("getting doc links: %w", err)
	}
	return &d, nil
}

// LoginActivity returns login activity logs for the given CRN.
func (s *Service) LoginActivity(ctx context.Context, crn string) ([]LogEntry, error) {
	var entries []LogEntry
	if err := s.client.GetEnvelope(ctx, "/loggers/"+crn, nil, &entries); err != nil {
		return nil, fmt.Errorf("getting login activity: %w", err)
	}
	return entries, nil
}

// ActivityLogs returns activity logs for the given CRN.
func (s *Service) ActivityLogs(ctx context.Context, crn string) ([]LogEntry, error) {
	var entries []LogEntry
	if err := s.client.GetEnvelope(ctx, "/loggers/activity/"+crn, nil, &entries); err != nil {
		return nil, fmt.Errorf("getting activity logs: %w", err)
	}
	return entries, nil
}

// ChangePassword changes the authenticated user's password.
func (s *Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	if err := s.client.PostEnvelope(ctx, "/users/change-password", req, nil); err != nil {
		return fmt.Errorf("changing password: %w", err)
	}
	return nil
}

// MFAEnableSendOTP sends an OTP to begin enabling 2FA.
func (s *Service) MFAEnableSendOTP(ctx context.Context, req MFASendOTPRequest) error {
	if err := s.client.PostEnvelope(ctx, "/mfa/enable/send-otp", req, nil); err != nil {
		return fmt.Errorf("sending 2FA enable OTP: %w", err)
	}
	return nil
}

// MFAEnableVerifyOTP verifies the OTP to complete enabling 2FA.
func (s *Service) MFAEnableVerifyOTP(ctx context.Context, req MFAVerifyOTPRequest) error {
	if err := s.client.PostEnvelope(ctx, "/mfa/enable/verify-otp", req, nil); err != nil {
		return fmt.Errorf("verifying 2FA enable OTP: %w", err)
	}
	return nil
}

// MFADisableSendOTP sends an OTP to begin disabling 2FA.
func (s *Service) MFADisableSendOTP(ctx context.Context, req MFASendOTPRequest) error {
	if err := s.client.PostEnvelope(ctx, "/mfa/disable/send-otp", req, nil); err != nil {
		return fmt.Errorf("sending 2FA disable OTP: %w", err)
	}
	return nil
}

// MFADisableVerifyOTP verifies the OTP to complete disabling 2FA.
func (s *Service) MFADisableVerifyOTP(ctx context.Context, req MFAVerifyOTPRequest) error {
	if err := s.client.PostEnvelope(ctx, "/mfa/disable/verify-otp", req, nil); err != nil {
		return fmt.Errorf("disabling 2FA: %w", err)
	}
	return nil
}

// Enforce2FA sets whether 2FA is enforced for all users on the account.
func (s *Service) Enforce2FA(ctx context.Context, req Enforce2FARequest) error {
	if err := s.client.PostEnvelope(ctx, "/account/enforce2fa", req, nil); err != nil {
		return fmt.Errorf("enforcing 2FA: %w", err)
	}
	return nil
}

// CreateUser creates a new user under the account.
func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (*ManagedUser, error) {
	var u ManagedUser
	if err := s.client.PostEnvelope(ctx, "/api/users", req, &u); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return &u, nil
}

// UpdateUser updates an existing user by ID.
func (s *Service) UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*ManagedUser, error) {
	var u ManagedUser
	if err := s.client.PutEnvelope(ctx, "/api/users/"+id, nil, req, &u); err != nil {
		return nil, fmt.Errorf("updating user %s: %w", id, err)
	}
	return &u, nil
}

// ReInviteUser re-sends an invitation to a user.
func (s *Service) ReInviteUser(ctx context.Context, id string) error {
	if err := s.client.PostEnvelope(ctx, "/users/re-invite/"+id, nil, nil); err != nil {
		return fmt.Errorf("re-inviting user %s: %w", id, err)
	}
	return nil
}

// DeleteUser removes a user by ID.
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	q := url.Values{}
	if err := s.client.Delete(ctx, "/users/"+id, q); err != nil {
		return fmt.Errorf("deleting user %s: %w", id, err)
	}
	return nil
}
