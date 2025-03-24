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

func TestParseEnv(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		env       map[string]string
		expect    Builder
		expectErr string
	}{
		{
			title:  "parse defaults",
			env:    map[string]string{},
			expect: Builder{BuildFlags: defaultBuildFlags},
		},
		{
			title: "parse k6 version",
			env:   map[string]string{"K6_VERSION": "v0.0.0"},
			expect: Builder{
				K6Version:  "v0.0.0",
				BuildFlags: defaultBuildFlags,
			},
		},
		{
			title: "parse k6 repo",
			env:   map[string]string{"XK6_K6_REPO": "github.com/another/repo"},
			expect: Builder{
				K6Repo:     "github.com/another/repo",
				BuildFlags: defaultBuildFlags,
			},
		},
		{
			title: "parse GO environment variables",
			env:   map[string]string{"GOARCH": "amd64", "GOOS": "linux"},
			expect: Builder{
				Compile: Compile{
					Platform: Platform{
						Arch: "amd64",
						OS:   "linux",
					},
				},
				BuildFlags: defaultBuildFlags,
			},
		},
		{
			title:  "parse build opts",
			env:    map[string]string{"XK6_BUILD_FLAGS": "-buildvcs"},
			expect: Builder{BuildFlags: "-buildvcs"},
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
