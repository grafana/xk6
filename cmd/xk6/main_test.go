package main

import (
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestNormalizeImportPath(t *testing.T) {
	t.Parallel()

	type (
		args struct {
			currentModule string
			cwd           string
			moduleDir     string
		}
		testCaseType []struct {
			name string
			args args
			want string
		}
	)

	tests := testCaseType{
		{"linux-path", args{
			currentModule: "go.k6.io/xk6",
			cwd:           "/xk6",
			moduleDir:     "/xk6",
		}, "go.k6.io/xk6"},
		{"linux-subpath", args{
			currentModule: "go.k6.io/xk6",
			cwd:           "/xk6/subdir",
			moduleDir:     "/xk6",
		}, "go.k6.io/xk6/subdir"},
	}
	windowsTests := testCaseType{
		{"windows-path", args{
			currentModule: "go.k6.io/xk6",
			cwd:           "c:\\xk6",
			moduleDir:     "c:\\xk6",
		}, "go.k6.io/xk6"},
		{"windows-subpath", args{
			currentModule: "go.k6.io/xk6",
			cwd:           "c:\\xk6\\subdir",
			moduleDir:     "c:\\xk6",
		}, "go.k6.io/xk6/subdir"},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, windowsTests...)
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeImportPath(tt.args.currentModule, tt.args.cwd, tt.args.moduleDir); got != tt.want {
				t.Errorf("normalizeImportPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBuildOpts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title  string
		args   []string
		expect buildOps
	}{
		{
			title: "parse defaults", args: []string{},
			expect: buildOps{OutFile: defaultK6OutputFile()},
		},
		{
			title: "override k6 path", args: []string{"--output", filepath.Join("path", "to", "k6")},
			expect: buildOps{OutFile: filepath.Join("path", "to", "k6"), OutputOverride: true},
		},
		{
			title: "parse k6 version", args: []string{"v0.0.0"},
			expect: buildOps{K6Version: "v0.0.0", OutFile: defaultK6OutputFile()},
		},
		{
			title: "parse --with", args: []string{"--with", "github.com/repo/extension@v0.0.0"},
			expect: buildOps{OutFile: defaultK6OutputFile(), Extensions: []string{"github.com/repo/extension@v0.0.0"}},
		},
		{
			title: "parse --with with replacement",
			args:  []string{"--with", "github.com/repo/extension=github.com/another-repo/extension@v0.0.0"},
			expect: buildOps{
				OutFile:    defaultK6OutputFile(),
				Extensions: []string{"github.com/repo/extension=github.com/another-repo/extension@v0.0.0"},
			},
		},
		{
			title: "parse --replace",
			args:  []string{"--replace", "github.com/repo/extension=github.com/another-repo/extension"},
			expect: buildOps{
				OutFile:      defaultK6OutputFile(),
				Replacements: []string{"github.com/repo/extension=github.com/another-repo/extension"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			got, err := parseBuildOpts(tc.args)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

			if err == nil && !reflect.DeepEqual(got, tc.expect) {
				t.Errorf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}

func TestParseBuildOptsWithError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		title     string
		args      []string
		expectErr error
	}{
		{
			title: "parse spurious argument", args: []string{"v0.0.0", "another-arg"},
			expectErr: errMissingFlag,
		},
		{
			title: "parse --with with missing value", args: []string{"--with"},
			expectErr: errExpectedValue,
		},
		{
			title: "parse --replace with missing replace value", args: []string{"--replace", "github.com/repo/extension"},
			expectErr: errMissingReplace,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			_, err := parseBuildOpts(tc.args)
			if !errors.Is(err, tc.expectErr) {
				t.Errorf("expected error %v, got %v", tc.expectErr, err)
			}
		})
	}
}
