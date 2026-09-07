package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/instance"
)

func strPtr(s string) *string { return &s }

// ─── sshTargetIP ─────────────────────────────────────────────────────────

func TestSSHTargetIP(t *testing.T) {
	tests := []struct {
		name       string
		vm         *instance.VirtualMachine
		usePublic  bool
		usePrivate bool
		wantIP     string
		wantKind   string
		wantErr    string
	}{
		{
			name: "public via ipaddresses list, top-level public_ip null",
			vm: &instance.VirtualMachine{
				Slug:     "adj-headscale",
				PublicIP: nil,
				IPAddresses: []instance.IPAddresses{
					{IPAddress: "23.229.49.87", Type: "IPv4", IPType: "Public IP"},
				},
			},
			wantIP:   "23.229.49.87",
			wantKind: "public",
		},
		{
			name: "public via top-level public_ip",
			vm: &instance.VirtualMachine{
				Slug:     "my-vm",
				PublicIP: strPtr("198.51.100.10"),
			},
			wantIP:   "198.51.100.10",
			wantKind: "public",
		},
		{
			name: "private only via private_ip",
			vm: &instance.VirtualMachine{
				Slug:      "my-vm",
				PrivateIP: strPtr("10.0.0.5"),
			},
			wantIP:   "10.0.0.5",
			wantKind: "private",
		},
		{
			name: "private only via network pivot",
			vm: &instance.VirtualMachine{
				Slug: "adj-headscale",
				Networks: []instance.VMNetwork{
					{IsDefault: true, Pivot: &instance.VMNetworkIP{IPAddress: "10.0.0.214"}},
				},
			},
			wantIP:   "10.0.0.214",
			wantKind: "private",
		},
		{
			name: "use-private when both public and private exist",
			vm: &instance.VirtualMachine{
				Slug:      "adj-headscale",
				PrivateIP: strPtr("10.0.0.214"),
				IPAddresses: []instance.IPAddresses{
					{IPAddress: "23.229.49.87", Type: "IPv4", IPType: "Public IP"},
				},
			},
			usePrivate: true,
			wantIP:     "10.0.0.214",
			wantKind:   "private",
		},
		{
			name: "use-public when only private exists errors",
			vm: &instance.VirtualMachine{
				Slug:      "my-vm",
				PrivateIP: strPtr("10.0.0.5"),
			},
			usePublic: true,
			wantErr:   "instance my-vm has no public IP address. Use --use-private to connect over the VPC/VPN",
		},
		{
			name: "use-private when only public exists errors",
			vm: &instance.VirtualMachine{
				Slug:     "my-vm",
				PublicIP: strPtr("198.51.100.10"),
			},
			usePrivate: true,
			wantErr:    "instance my-vm has no private IP address",
		},
		{
			name:    "no IPs at all errors",
			vm:      &instance.VirtualMachine{Slug: "my-vm"},
			wantErr: "instance my-vm has no usable IP address",
		},
		{
			name:       "use-public and use-private together errors",
			vm:         &instance.VirtualMachine{Slug: "my-vm"},
			usePublic:  true,
			usePrivate: true,
			wantErr:    "--use-public and --use-private are mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, kind, err := sshTargetIP(tt.vm, tt.usePublic, tt.usePrivate)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want containing %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ip != tt.wantIP {
				t.Errorf("ip = %q, want %q", ip, tt.wantIP)
			}
			if kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", kind, tt.wantKind)
			}
		})
	}
}

// ─── sshUser ─────────────────────────────────────────────────────────────

func TestSSHUser(t *testing.T) {
	tests := []struct {
		name       string
		flagUser   string
		flagSet    bool
		vmUsername string
		want       string
	}{
		{
			name:       "explicit root stays root",
			flagUser:   "root",
			flagSet:    true,
			vmUsername: "ubuntu",
			want:       "root",
		},
		{
			name:       "unset flag uses VM username",
			flagUser:   "root",
			flagSet:    false,
			vmUsername: "ubuntu",
			want:       "ubuntu",
		},
		{
			name:       "unset flag and empty VM username falls back to root",
			flagUser:   "root",
			flagSet:    false,
			vmUsername: "",
			want:       "root",
		},
		{
			name:       "explicitly set but empty flag falls back to VM username",
			flagUser:   "",
			flagSet:    true,
			vmUsername: "ubuntu",
			want:       "ubuntu",
		},
		{
			name:       "explicitly set but empty flag and empty VM username falls back to root",
			flagUser:   "",
			flagSet:    true,
			vmUsername: "",
			want:       "root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sshUser(tt.flagUser, tt.flagSet, tt.vmUsername)
			if got != tt.want {
				t.Errorf("sshUser(%q, %v, %q) = %q, want %q", tt.flagUser, tt.flagSet, tt.vmUsername, got, tt.want)
			}
		})
	}
}

// ─── nil vm guard ────────────────────────────────────────────────────────

func TestSSHTargetIPNilVM(t *testing.T) {
	_, _, err := sshTargetIP(nil, false, false)
	if err == nil {
		t.Fatal("expected error for nil vm, got nil")
	}
	if !strings.Contains(err.Error(), "no usable IP address") {
		t.Errorf("error = %q, want containing %q", err.Error(), "no usable IP address")
	}
}

// ─── cobra-level mutual exclusion ───────────────────────────────────────────

func TestInstanceSSHMutuallyExclusiveFlags(t *testing.T) {
	cmd := NewInstanceCmd()
	root := newTestRoot()
	root.AddCommand(cmd)

	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs([]string{"instance", "ssh", "x", "--use-public", "--use-private"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "use-public") || !strings.Contains(err.Error(), "use-private") {
		t.Errorf("error = %q, want containing mutual-exclusion message", err.Error())
	}
}
