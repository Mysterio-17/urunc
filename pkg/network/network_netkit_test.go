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

package network

import (
	"os"
	"testing"
)

func TestIsNetkitEnabled(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "enabled with 1",
			envValue: "1",
			want:     true,
		},
		{
			name:     "enabled with true",
			envValue: "true",
			want:     true,
		},
		{
			name:     "enabled with TRUE",
			envValue: "TRUE",
			want:     true,
		},
		{
			name:     "disabled with 0",
			envValue: "0",
			want:     false,
		},
		{
			name:     "disabled with false",
			envValue: "false",
			want:     false,
		},
		{
			name:     "disabled with empty",
			envValue: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalValue := os.Getenv("URUNC_ENABLE_NETKIT")
			defer func() {
				if originalValue != "" {
					os.Setenv("URUNC_ENABLE_NETKIT", originalValue)
				} else {
					os.Unsetenv("URUNC_ENABLE_NETKIT")
				}
			}()

			// Set test value
			if tt.envValue != "" {
				os.Setenv("URUNC_ENABLE_NETKIT", tt.envValue)
			} else {
				os.Unsetenv("URUNC_ENABLE_NETKIT")
			}

			if got := isNetkitEnabled(); got != tt.want {
				t.Errorf("isNetkitEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNetkitSupported(t *testing.T) {
	// This test will check if the kernel version detection works
	// The result depends on the actual kernel version
	supported := isNetkitSupported()
	t.Logf("Netkit support detected: %v", supported)

	// We can't assert a specific value since it depends on the kernel
	// but we can verify the function doesn't panic
	if supported {
		t.Log("Kernel version >= 6.8 detected, netkit is supported")
	} else {
		t.Log("Kernel version < 6.8 detected, netkit is not supported")
	}
}

func TestGetLinkType(t *testing.T) {
	tests := []struct {
		name string
		link interface{}
		want string
	}{
		{
			name: "nil link",
			link: nil,
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getLinkType(nil); got != tt.want {
				t.Errorf("getLinkType() = %v, want %v", got, tt.want)
			}
		})
	}
}
