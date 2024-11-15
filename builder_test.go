// Copyright 2020 Matthew Holt
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

package xk6

import (
	"reflect"
	"testing"
)

func TestReplacementPath_Param(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		r    ReplacementPath
		want string
	}{
		{
			"Empty",
			ReplacementPath(""),
			"",
		},
		{
			"ModulePath",
			ReplacementPath("github.com/x/y"),
			"github.com/x/y",
		},
		{
			"ModulePath Version Pinned",
			ReplacementPath("github.com/x/y v0.0.0-20200101000000-xxxxxxxxxxxx"),
			"github.com/x/y@v0.0.0-20200101000000-xxxxxxxxxxxx",
		},
		{
			"FilePath",
			ReplacementPath("/x/y/z"),
			"/x/y/z",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.r.Param(); got != tt.want {
				t.Errorf("ReplacementPath.Param() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewReplace(t *testing.T) {
	t.Parallel()
	type args struct {
		old string
		new string
	}
	tests := []struct {
		name string
		args args
		want Replace
	}{
		{
			"Empty",
			args{"", ""},
			Replace{"", ""},
		},
		{
			"Constructor",
			args{"a", "b"},
			Replace{"a", "b"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NewReplace(tt.args.old, tt.args.new); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewReplace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildCommandArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		buildFlags string
		want       []string
	}{
		{
			buildFlags: "",
			want: []string{
				"build", "-o", "binfile",
			},
		},
		{
			buildFlags: "-ldflags='-w -s'",
			want: []string{
				"build", "-o", "binfile", "-ldflags=-w -s",
			},
		},
		{
			buildFlags: "-race -buildvcs=false",
			want: []string{
				"build", "-o", "binfile", "-race", "-buildvcs=false",
			},
		},
		{
			buildFlags: `-buildvcs=false -ldflags="-s -w" -race`,
			want: []string{
				"build", "-o", "binfile", "-buildvcs=false", "-ldflags=-s -w", "-race",
			},
		},
		{
			buildFlags: `-ldflags="-s -w" -race -buildvcs=false`,
			want: []string{
				"build", "-o", "binfile", "-ldflags=-s -w", "-race", "-buildvcs=false",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.buildFlags, func(t *testing.T) {
			t.Parallel()
			if got := buildCommandArgs(tt.buildFlags, "binfile"); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildCommandArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
