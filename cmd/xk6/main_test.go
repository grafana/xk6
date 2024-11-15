package main

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"go.k6.io/xk6"
)


func TestSplitWith(t *testing.T) {
	t.Parallel()
	for i, tc := range []struct {
		input         string
		expectModule  string
		expectVersion string
		expectReplace string
		expectErr     bool
	}{
		{
			input:        "module",
			expectModule: "module",
		},
		{
			input:         "module@version",
			expectModule:  "module",
			expectVersion: "version",
		},
		{
			input:         "module@version=replace",
			expectModule:  "module",
			expectVersion: "version",
			expectReplace: "replace",
		},
		{
			input:         "module=replace",
			expectModule:  "module",
			expectReplace: "replace",
		},
		{
			input:         "module@module_version=replace@replace_version",
			expectModule:  "module",
			expectReplace: "replace@replace_version",
			expectVersion: "module_version",
		},
		{
			input:     "=replace",
			expectErr: true,
		},
		{
			input:     "@version",
			expectErr: true,
		},
		{
			input:     "@version=replace",
			expectErr: true,
		},
		{
			input:     "",
			expectErr: true,
		},
	} {
		actualModule, actualVersion, actualReplace, actualErr := splitWith(tc.input)
		if actualModule != tc.expectModule {
			t.Errorf("Test %d: Expected module '%s' but got '%s' (input=%s)",
				i, tc.expectModule, actualModule, tc.input)
		}
		if tc.expectErr {
			if actualErr == nil {
				t.Errorf("Test %d: Expected error but did not get one (input='%s')", i, tc.input)
			}
			continue
		}
		if !tc.expectErr && actualErr != nil {
			t.Errorf("Test %d: Expected no error but got: %s (input='%s')", i, actualErr, tc.input)
		}
		if actualVersion != tc.expectVersion {
			t.Errorf("Test %d: Expected version '%s' but got '%s' (input='%s')",
				i, tc.expectVersion, actualVersion, tc.input)
		}
		if actualReplace != tc.expectReplace {
			t.Errorf("Test %d: Expected module '%s' but got '%s' (input='%s')",
				i, tc.expectReplace, actualReplace, tc.input)
		}
	}
}

func TestExpandPath(t *testing.T) {
	t.Run(". expands to current directory", func(t *testing.T) {
		t.Parallel()
		got, err := expandPath(".")
		if got == "." {
			t.Errorf("did not expand path")
		}
		if err != nil {
			t.Errorf("failed to expand path")
		}
	})
	t.Run("~ expands to user's home directory", func(t *testing.T) {
		t.Parallel()
		got, err := expandPath("~")
		if got == "~" {
			t.Errorf("did not expand path")
		}
		if err != nil {
			t.Errorf("failed to expand path")
		}
		switch runtime.GOOS {
		case "linux":
			if !strings.HasPrefix(got, "/home") {
				t.Errorf("did not expand home directory. want=/home/... got=%s", got)
			}
		case "darwin":
			if !strings.HasPrefix(got, "/Users") {
				t.Errorf("did not expand home directory. want=/Users/... got=%s", got)
			}
		case "windows":
			if !strings.HasPrefix(got, "C:\\Users") { // could well be another drive letter, but let's assume C:\\
				t.Errorf("did not expand home directory. want=C:\\Users\\... got=%s", got)
			}
		}
	})
}

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
	testCases := []struct {
		title     string
		args      []string
		expect    BuildOps
		expectErr string
	}{
		{
			title: "parse defaults",
			args:  []string{},
			expect: BuildOps{
				K6Version:     "",
				Extensions:     nil,
				Replacements:   nil,
				OutFile:        defaultK6OutputFile(),
				OutputOverride: false,
			},
		},
		{
			title: "override k6 path",
			args: []string{
				"--output", filepath.Join("path", "to", "k6"),
			},
			expect: BuildOps{
				K6Version:      "",
				OutFile:        filepath.Join("path", "to", "k6"),
				OutputOverride: true,
				Extensions:     nil,
				Replacements:   nil,
			},
		},
		{
			title: "parse k6 version",
			args: []string{
				"v0.0.0",
			},
			expect: BuildOps{
				K6Version:      "v0.0.0",
				OutFile:        defaultK6OutputFile(),
				OutputOverride: false,
				Extensions:     nil,
				Replacements:   nil,
			},
		},
		{
			title: "parse spurious argument",
			args: []string{
				"v0.0.0",
				"another-arg",
			},
			expect: BuildOps{},
			expectErr: "missing flag",
		},
		{
			title: "parse --with",
			args: []string{
				"--with", "github.com/repo/extension@v0.0.0",
			},
			expect: BuildOps{
				K6Version:      "",
				OutFile:        defaultK6OutputFile(),
				OutputOverride: false,
				Extensions: []xk6.Dependency{
					{
						PackagePath: "github.com/repo/extension",
						Version:     "v0.0.0",
					},
				},
				Replacements: nil,
			},
		},
		{
			title: "parse --with with missing value",
			args: []string{
				"--with",
			},
			expect: BuildOps{},
				expectErr: "expected value after --with flag",
		},
		{
			title: "parse --with with replacement",
			args: []string{
				"--with", "github.com/repo/extension@=github.com/another-repo/extension@v0.0.0",
			},
			expect: BuildOps{
				K6Version:      "",
				OutFile:        defaultK6OutputFile(),
				OutputOverride: false,
				Extensions: []xk6.Dependency{
					{
						PackagePath: "github.com/repo/extension",
					},
				},
				Replacements: []xk6.Replace{
					{
						Old: "github.com/repo/extension",
						New: "github.com/another-repo/extension@v0.0.0",
					},
				},
			},
		},
		{
			title: "parse --replace",
			args: []string{
				"--replace", "github.com/repo/extension=github.com/another-repo/extension",
			},
			expect: BuildOps{
				K6Version:      "",
				OutFile:        defaultK6OutputFile(),
				OutputOverride: false,
				Extensions:     nil,
				Replacements: []xk6.Replace{
					{
						Old: "github.com/repo/extension",
						New: "github.com/another-repo/extension",
					},
				},
			},
		},
		{
			title: "parse --replace with missing replace value",
			args: []string{
				"--replace", "github.com/repo/extension",
			},
			expect: BuildOps{},
				expectErr: "replace value must be of format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			got, err := parseBuildOpts(tc.args)
			if err != nil && !strings.Contains(err.Error(), tc.expectErr) {
				t.Errorf("expected error %v, got %v", tc.expectErr, err)
			}
			if err == nil && !reflect.DeepEqual(got, tc.expect) {
				t.Errorf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}