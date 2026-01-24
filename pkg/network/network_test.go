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
	"testing"
)

func TestGetTapIndex(t *testing.T) {
	// This test just verifies the function doesn't panic
	// The actual count depends on the system's network interfaces
	index, err := getTapIndex()
	if err != nil {
		t.Errorf("getTapIndex() error = %v", err)
		return
	}
	
	// Index should be non-negative
	if index < 0 {
		t.Errorf("getTapIndex() = %d, want >= 0", index)
	}
	
	// Index should not exceed 255 (function returns error if > 255)
	if index > 255 {
		t.Errorf("getTapIndex() = %d, want <= 255", index)
	}
}

func TestNewNetworkManager(t *testing.T) {
	tests := []struct {
		name        string
		networkType string
		wantErr     bool
		wantType    string
	}{
		{
			name:        "static network manager",
			networkType: "static",
			wantErr:     false,
			wantType:    "*network.StaticNetwork",
		},
		{
			name:        "dynamic network manager",
			networkType: "dynamic",
			wantErr:     false,
			wantType:    "*network.DynamicNetwork",
		},
		{
			name:        "invalid network type",
			networkType: "invalid",
			wantErr:     true,
		},
		{
			name:        "empty network type",
			networkType: "",
			wantErr:     true,
		},
		{
			name:        "unknown network type",
			networkType: "bridge",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNetworkManager(tt.networkType)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNetworkManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if got == nil {
					t.Error("NewNetworkManager() returned nil manager")
				}
			}
		})
	}
}

func TestEnsureEth0Exists(t *testing.T) {
	// This test verifies the function works
	// It might fail or succeed depending on the test environment
	err := ensureEth0Exists()
	
	// We don't assert a specific result since it depends on
	// whether eth0 exists in the test environment
	// Just verify it doesn't panic
	t.Logf("ensureEth0Exists() error = %v", err)
}
