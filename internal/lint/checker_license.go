package lint

import (
	"context"
	"sort"

	"github.com/go-enry/go-license-detector/v4/licensedb"
)

func checkerLicense(_ context.Context, dir string) *checkResult {
	results := licensedb.Analyse(dir)

	const minConfidence = 0.9

	for _, result := range results {
		sort.Slice(result.Matches, func(i, j int) bool {
			return result.Matches[i].Confidence > result.Matches[j].Confidence
		})

		for _, match := range result.Matches {
			if match.Confidence < minConfidence {
				continue
			}

			if _, found := validLicenses[match.License]; found {
				return checkPassed("found `%s` as `%s` license", match.File, match.License)
			}
		}
	}

	return checkFailed("no accepted license found")
}

// source: https://grafana.com/legal/plugins/#accepted-licenses
var validLicenses = map[string]struct{}{ //nolint:gochecknoglobals
	// AGPL-3.0
	"AGPL-3.0":          {},
	"AGPL-3.0-only":     {},
	"AGPL-3.0-or-later": {},
	// Apache-2.0
	"Apache-2.0": {},
	// BSD
	"BSD-2-Clause": {},
	"BSD-3-Clause": {},
	// GPL-3.0
	"GPL-3.0-only":     {},
	"GPL-3.0":          {},
	"GPL-3.0-or-later": {},
	// LGPL-3.0
	"LGPL-3.0":          {},
	"LGPL-3.0-only":     {},
	"LGPL-3.0-or-later": {},
	// MIT
	"MIT": {},
}
