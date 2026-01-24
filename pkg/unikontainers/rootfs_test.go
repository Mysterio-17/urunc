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

	"github.com/urunc-dev/urunc/pkg/unikontainers/types"
)

func TestNewRootfsResult(t *testing.T) {
	tests := []struct {
		name        string
		rootfsType  string
		path        string
		mountedPath string
		monRootfs   string
		want        types.RootfsParams
	}{
		{
			name:        "initrd rootfs",
			rootfsType:  "initrd",
			path:        "/path/to/initrd",
			mountedPath: "",
			monRootfs:   "/run/urunc/mon",
			want: types.RootfsParams{
				Type:        "initrd",
				Path:        "/path/to/initrd",
				MountedPath: "",
				MonRootfs:   "/run/urunc/mon",
			},
		},
		{
			name:        "block rootfs",
			rootfsType:  "block",
			path:        "/dev/dm-0",
			mountedPath: "/mnt/rootfs",
			monRootfs:   "/run/urunc/mon",
			want: types.RootfsParams{
				Type:        "block",
				Path:        "/dev/dm-0",
				MountedPath: "/mnt/rootfs",
				MonRootfs:   "/run/urunc/mon",
			},
		},
		{
			name:        "9pfs rootfs",
			rootfsType:  "9pfs",
			path:        "/container/rootfs",
			mountedPath: "",
			monRootfs:   "/run/urunc/mon",
			want: types.RootfsParams{
				Type:        "9pfs",
				Path:        "/container/rootfs",
				MountedPath: "",
				MonRootfs:   "/run/urunc/mon",
			},
		},
		{
			name:        "empty paths",
			rootfsType:  "none",
			path:        "",
			mountedPath: "",
			monRootfs:   "",
			want: types.RootfsParams{
				Type:        "none",
				Path:        "",
				MountedPath: "",
				MonRootfs:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newRootfsResult(tt.rootfsType, tt.path, tt.mountedPath, tt.monRootfs)
			
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Path != tt.want.Path {
				t.Errorf("Path = %v, want %v", got.Path, tt.want.Path)
			}
			if got.MountedPath != tt.want.MountedPath {
				t.Errorf("MountedPath = %v, want %v", got.MountedPath, tt.want.MountedPath)
			}
			if got.MonRootfs != tt.want.MonRootfs {
				t.Errorf("MonRootfs = %v, want %v", got.MonRootfs, tt.want.MonRootfs)
			}
		})
	}
}

func TestRootfsSelector_TryInitrd(t *testing.T) {
	tests := []struct {
		name       string
		annot      map[string]string
		wantFound  bool
		wantType   string
		wantPath   string
	}{
		{
			name: "initrd present",
			annot: map[string]string{
				annotInitrd: "/path/to/initrd.img",
			},
			wantFound: true,
			wantType:  "initrd",
			wantPath:  "/path/to/initrd.img",
		},
		{
			name:      "initrd missing",
			annot:     map[string]string{},
			wantFound: false,
		},
		{
			name: "initrd empty",
			annot: map[string]string{
				annotInitrd: "",
			},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &rootfsSelector{
				annot:      tt.annot,
				cntrRootfs: "/container/rootfs",
			}
			
			got, found := rs.tryInitrd()
			
			if found != tt.wantFound {
				t.Errorf("tryInitrd() found = %v, want %v", found, tt.wantFound)
				return
			}
			
			if found {
				if got.Type != tt.wantType {
					t.Errorf("tryInitrd() Type = %v, want %v", got.Type, tt.wantType)
				}
				if got.Path != tt.wantPath {
					t.Errorf("tryInitrd() Path = %v, want %v", got.Path, tt.wantPath)
				}
			}
		})
	}
}

func TestRootfsSelector_ShouldMountContainerRootfs(t *testing.T) {
	tests := []struct {
		name  string
		annot map[string]string
		want  bool
	}{
		{
			name: "mount rootfs true",
			annot: map[string]string{
				annotMountRootfs: "true",
			},
			want: true,
		},
		{
			name: "mount rootfs 1",
			annot: map[string]string{
				annotMountRootfs: "1",
			},
			want: true,
		},
		{
			name: "mount rootfs false",
			annot: map[string]string{
				annotMountRootfs: "false",
			},
			want: false,
		},
		{
			name: "mount rootfs 0",
			annot: map[string]string{
				annotMountRootfs: "0",
			},
			want: false,
		},
		{
			name:  "mount rootfs missing",
			annot: map[string]string{},
			want:  false,
		},
		{
			name: "mount rootfs empty",
			annot: map[string]string{
				annotMountRootfs: "",
			},
			want: false,
		},
		{
			name: "mount rootfs invalid",
			annot: map[string]string{
				annotMountRootfs: "invalid",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &rootfsSelector{
				annot: tt.annot,
			}
			
			got := rs.shouldMountContainerRootfs()
			if got != tt.want {
				t.Errorf("shouldMountContainerRootfs() = %v, want %v", got, tt.want)
			}
		})
	}
}
