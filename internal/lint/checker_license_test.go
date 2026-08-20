package lint

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckerLicense(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contents string
		want     string
		passed   bool
	}{
		{
			name:     "apache",
			contents: "Apache License\nVersion 2.0, January 2004",
			want:     "Apache-2.0",
			passed:   true,
		},
		{
			name:     "spdx mit",
			contents: "SPDX-License-Identifier: MIT",
			want:     "MIT",
			passed:   true,
		},
		{
			name:     "bsd two clause",
			contents: "Redistribution and use in source and binary forms are permitted.",
			want:     "BSD-2-Clause",
			passed:   true,
		},
		{
			name:     "bsd three clause",
			contents: "Redistribution and use in source and binary forms are permitted. Neither the name of the project nor the names of its contributors may be used.",
			want:     "BSD-3-Clause",
			passed:   true,
		},
		{
			name:     "unsupported license",
			contents: "SPDX-License-Identifier: MPL-2.0",
			passed:   false,
		},
		{
			name:     "missing root license",
			contents: "",
			passed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if tt.name != "missing root license" {
				err := os.WriteFile(filepath.Join(dir, licenseFilename), []byte(tt.contents), 0o600)
				if err != nil {
					t.Fatal(err)
				}
			}

			result := checkerLicense(context.Background(), dir)
			if result.passed != tt.passed {
				t.Fatalf("passed = %t, want %t (%s)", result.passed, tt.passed, result.details)
			}

			if result.passed && result.details != "found `LICENSE` as `"+tt.want+"` license" {
				t.Fatalf("details = %q, want license %q", result.details, tt.want)
			}
		})
	}
}

func TestCheckerLicenseDoesNotScanSubdirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := os.Mkdir(filepath.Join(dir, "nested"), 0o700)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(dir, "nested", licenseFilename), []byte("Apache License\nVersion 2.0"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	result := checkerLicense(context.Background(), dir)
	if result.passed {
		t.Fatalf("license in a subdirectory unexpectedly passed: %s", result.details)
	}
}
