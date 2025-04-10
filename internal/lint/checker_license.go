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

	return checkFailed("no valid license found")
}

// source: https://spdx.org/licenses/
// both FSF Free and OSI Approved licenses.
var validLicenses = map[string]struct{}{ //nolint:gochecknoglobals
	"AFL-1.1":           {},
	"AFL-1.2":           {},
	"AFL-2.0":           {},
	"AFL-2.1":           {},
	"AFL-3.0":           {},
	"AGPL-3.0":          {},
	"AGPL-3.0-only":     {},
	"AGPL-3.0-or-later": {},
	"Apache-1.1":        {},
	"Apache-2.0":        {},
	"APSL-2.0":          {},
	"Artistic-2.0":      {},
	"BSD-2-Clause":      {},
	"BSD-3-Clause":      {},
	"BSL-1.0":           {},
	"CDDL-1.0":          {},
	"CPAL-1.0":          {},
	"CPL-1.0":           {},
	"ECL-2.0":           {},
	"EFL-2.0":           {},
	"EPL-1.0":           {},
	"EPL-2.0":           {},
	"EUDatagrid":        {},
	"EUPL-1.1":          {},
	"EUPL-1.2":          {},
	"GPL-2.0-only":      {},
	"GPL-2.0":           {},
	"GPL-2.0-or-later":  {},
	"GPL-3.0-only":      {},
	"GPL-3.0":           {},
	"GPL-3.0-or-later":  {},
	"HPND":              {},
	"Intel":             {},
	"IPA":               {},
	"IPL-1.0":           {},
	"ISC":               {},
	"LGPL-2.1":          {},
	"LGPL-2.1-only":     {},
	"LGPL-2.1-or-later": {},
	"LGPL-3.0":          {},
	"LGPL-3.0-only":     {},
	"LGPL-3.0-or-later": {},
	"LPL-1.02":          {},
	"MIT":               {},
	"MPL-1.1":           {},
	"MPL-2.0":           {},
	"MS-PL":             {},
	"MS-RL":             {},
	"NCSA":              {},
	"Nokia":             {},
	"OFL-1.1":           {},
	"OSL-1.0":           {},
	"OSL-2.0":           {},
	"OSL-2.1":           {},
	"OSL-3.0":           {},
	"PHP-3.01":          {},
	"Python-2.0":        {},
	"QPL-1.0":           {},
	"RPSL-1.0":          {},
	"SISSL":             {},
	"Sleepycat":         {},
	"SPL-1.0":           {},
	"Unlicense":         {},
	"UPL-1.0":           {},
	"W3C":               {},
	"Zlib":              {},
	"ZPL-2.0":           {},
	"ZPL-2.1":           {},
}
