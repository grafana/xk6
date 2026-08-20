package lint

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const licenseFilename = "LICENSE"

func checkerLicense(_ context.Context, dir string) *checkResult {
	filename := filepath.Join(dir, licenseFilename)

	data, err := os.ReadFile(filepath.Clean(filename)) //nolint:forbidigo
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return checkFailed("no root `%s` file found", licenseFilename)
		}

		return checkError(err)
	}

	license := identifyLicense(string(data))
	if license == "" {
		return checkFailed("`%s` is not an accepted license", licenseFilename)
	}

	return checkPassed("found `%s` as `%s` license", licenseFilename, license)
}

func identifyLicense(contents string) string {
	text := strings.ToUpper(contents)

	// SPDX identifiers are unambiguous and allow projects to use a customized
	// license file while still declaring the applicable standard license.
	for _, line := range strings.Split(text, "\n") {
		const prefix = "SPDX-LICENSE-IDENTIFIER:"
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		identifier := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if license, found := spdxLicenses[identifier]; found {
			return license
		}

		return ""
	}

	// These signatures identify the standard license texts without attempting
	// fuzzy matching or scanning files outside the extension root.
	switch {
	case strings.Contains(text, "GNU AFFERO GENERAL PUBLIC LICENSE") && strings.Contains(text, "VERSION 3"):
		return "AGPL-3.0"
	case strings.Contains(text, "GNU LESSER GENERAL PUBLIC LICENSE") && strings.Contains(text, "VERSION 3"):
		return "LGPL-3.0"
	case strings.Contains(text, "GNU GENERAL PUBLIC LICENSE") && strings.Contains(text, "VERSION 3"):
		return "GPL-3.0"
	case strings.Contains(text, "APACHE LICENSE") && strings.Contains(text, "VERSION 2.0"):
		return "Apache-2.0"
	case strings.Contains(text, "PERMISSION IS HEREBY GRANTED, FREE OF CHARGE, TO ANY PERSON OBTAINING A COPY OF THIS SOFTWARE"):
		return "MIT"
	case strings.Contains(text, "REDISTRIBUTION AND USE IN SOURCE AND BINARY FORMS") &&
		strings.Contains(text, "NEITHER THE NAME"):
		return "BSD-3-Clause"
	case strings.Contains(text, "REDISTRIBUTION AND USE IN SOURCE AND BINARY FORMS"):
		return "BSD-2-Clause"
	default:
		return ""
	}
}

// source: https://grafana.com/legal/plugins/#accepted-licenses
var spdxLicenses = map[string]string{ //nolint:gochecknoglobals
	"AGPL-3.0":          "AGPL-3.0",
	"AGPL-3.0-ONLY":     "AGPL-3.0",
	"AGPL-3.0-OR-LATER": "AGPL-3.0",
	"APACHE-2.0":        "Apache-2.0",
	"BSD-2-CLAUSE":      "BSD-2-Clause",
	"BSD-3-CLAUSE":      "BSD-3-Clause",
	"GPL-3.0-ONLY":      "GPL-3.0",
	"GPL-3.0":           "GPL-3.0",
	"GPL-3.0-OR-LATER":  "GPL-3.0",
	"LGPL-3.0":          "LGPL-3.0",
	"LGPL-3.0-ONLY":     "LGPL-3.0",
	"LGPL-3.0-OR-LATER": "LGPL-3.0",
	"MIT":               "MIT",
}
