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


func TestParseEnv(t *testing.T) {
	testCases := []struct {
		title     string
		env       map[string]string
		expect    Builder
		expectErr string
	}{
		{
			title: "parse defaults",
			env:   map[string]string{},
			expect: Builder{
				Compile: Compile{
					Cgo: false,
					Platform: Platform{
						Arch: "",
						OS:   "",
						ARM:  "",
					},
				},
				K6Repo:         "",
				K6Version:      "",
				BuildFlags:     defaultBuildFlags,
				RaceDetector:   false,
				SkipCleanup:    false,
				Extensions:     nil,
				Replacements:   nil,
			},
		},
		{
			title: "parse k6 version",
			env:   map[string]string{
				"K6_VERSION": "v0.0.0",
			},
			expect: Builder{
				Compile: Compile{
					Cgo: false,
					Platform: Platform{
						Arch: "",
						OS:   "",
						ARM:  "",
					},
				},
				K6Repo:         "",
				K6Version:      "v0.0.0",
				BuildFlags:     defaultBuildFlags,
				RaceDetector:   false,
				SkipCleanup:    false,
				Extensions:     nil,
				Replacements:   nil,
			},
		},
		{
			title: "parse k6 repo",
			env:   map[string]string{
				"XK6_K6_REPO": "github.com/another/repo",
			},
			expect: Builder{
				Compile: Compile{
					Cgo: false,
					Platform: Platform{
						Arch: "",
						OS:   "",
						ARM:  "",
					},
				},
				K6Repo:         "github.com/another/repo",
				K6Version:      "",
				BuildFlags:     defaultBuildFlags,
				RaceDetector:   false,
				SkipCleanup:    false,
				Extensions:     nil,
				Replacements:   nil,
			},
		},
		{
			title: "parse GO environment variables",
			env: map[string]string{
				"GOARCH": "amd64",
				"GOOS":   "linux",
			},
			expect: Builder{
				Compile: Compile{
					Cgo: false,
					Platform: Platform{
						Arch: "amd64",
						OS:   "linux",
						ARM:  "",
					},
				},
				K6Repo:         "",
				K6Version:      "",
				BuildFlags:     defaultBuildFlags,
				RaceDetector:   false,
				SkipCleanup:    false,
				Extensions:     nil,
				Replacements:   nil,
			},
		},
		{
			title: "parse build opts",
			env:   map[string]string{
				"XK6_BUILD_FLAGS": "-buildvcs",
			},
			expect: Builder{
				Compile: Compile{
					Cgo: false,
					Platform: Platform{
						Arch: "",
						OS:   "",
						ARM:  "",
					},
				},
				K6Repo:         "",
				K6Version:      "",
				BuildFlags:     "-buildvcs",
				RaceDetector:   false,
				SkipCleanup:    false,
				Extensions:     nil,
				Replacements:   nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			got := parseEnv(tc.env)
			if !reflect.DeepEqual(got, tc.expect) {
				t.Errorf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}
