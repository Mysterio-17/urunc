// Copyright (c) 2023-2025, Nubificus LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikontainers

import (
	"testing"
)

func TestIdToGuestCID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantMin  int
		wantMax  int
		checkCID int
	}{
		{
			name:    "empty string",
			id:      "",
			wantMin: 3,
			wantMax: 99,
		},
		{
			name:     "simple id",
			id:       "container123",
			wantMin:  3,
			wantMax:  99,
			checkCID: -1,
		},
		{
			name:     "long id",
			id:       "very-long-container-id-12345678",
			wantMin:  3,
			wantMax:  99,
			checkCID: -1,
		},
		{
			name:     "special characters",
			id:       "container-with-special_chars.123",
			wantMin:  3,
			wantMax:  99,
			checkCID: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := idToGuestCID(tt.id)
			
			// Check if CID is within valid range
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("idToGuestCID(%q) = %d, want value between %d and %d",
					tt.id, got, tt.wantMin, tt.wantMax)
			}
			
			// Test determinism - same input should always produce same output
			got2 := idToGuestCID(tt.id)
			if got != got2 {
				t.Errorf("idToGuestCID(%q) is not deterministic: first=%d, second=%d",
					tt.id, got, got2)
			}
		})
	}
}

func TestIsValidVSockAddress(t *testing.T) {
	tests := []struct {
		name        string
		rpcAddress  string
		hypervisor  string
		wantValid   bool
		wantErr     bool
		wantPath    string
	}{
		{
			name:       "valid qemu vsock address",
			rpcAddress: "vsock://2:1234",
			hypervisor: "qemu",
			wantValid:  true,
			wantErr:    false,
		},
		{
			name:       "invalid qemu vsock - wrong CID",
			rpcAddress: "vsock://3:1234",
			hypervisor: "qemu",
			wantValid:  false,
			wantErr:    true,
		},
		{
			name:       "invalid qemu vsock - no port",
			rpcAddress: "vsock://2:",
			hypervisor: "qemu",
			wantValid:  false,
			wantErr:    true,
		},
		{
			name:       "invalid qemu vsock - malformed",
			rpcAddress: "vsock://invalid",
			hypervisor: "qemu",
			wantValid:  false,
			wantErr:    true,
		},
		{
			name:       "not a vsock address",
			rpcAddress: "http://localhost:1234",
			hypervisor: "qemu",
			wantValid:  false,
			wantErr:    true,
		},
		{
			name:       "empty address",
			rpcAddress: "",
			hypervisor: "qemu",
			wantValid:  false,
			wantErr:    true,
		},
		{
			name:       "valid firecracker unix socket",
			rpcAddress: "unix:///tmp/vaccel.sock_1234",
			hypervisor: "firecracker",
			wantValid:  true,
			wantErr:    false,
			wantPath:   "/tmp",
		},
		{
			name:       "valid firecracker nested path",
			rpcAddress: "unix:///var/run/urunc/vaccel.sock_5678",
			hypervisor: "firecracker",
			wantValid:  true,
			wantErr:    false,
			wantPath:   "/var/run/urunc",
		},
		{
			name:       "firecracker invalid - wrong socket name",
			rpcAddress: "unix:///tmp/test.sock",
			hypervisor: "firecracker",
			wantValid:  false,
			wantErr:    true,
		},
		{
			name:       "firecracker invalid - no unix prefix",
			rpcAddress: "/tmp/vaccel.sock_1234",
			hypervisor: "firecracker",
			wantValid:  false,
			wantErr:    true,
		},
		{
			name:       "unsupported hypervisor",
			rpcAddress: "vsock://2:1234",
			hypervisor: "kvm",
			wantValid:  false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := tt.rpcAddress
			gotValid, gotPath, err := isValidVSockAddress(&addr, tt.hypervisor)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("isValidVSockAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if gotValid != tt.wantValid {
				t.Errorf("isValidVSockAddress() valid = %v, want %v", gotValid, tt.wantValid)
			}
			
			if tt.wantPath != "" && gotPath != tt.wantPath {
				t.Errorf("isValidVSockAddress() path = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}
