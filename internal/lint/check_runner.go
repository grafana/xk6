package lint

import (
	"context"
	"fmt"
)

func runChecks(ctx context.Context, dir string, opts *Options) ([]Check, bool) {
	checkers := checkersFor(opts)
	results := make([]Check, 0, len(checkers))
	funcs := checkFunctions()
	// passed := passedChecks(opts.Passed)

	ctx, cleanup := withState(ctx, dir)
	defer cleanup()

	pass := true

	for _, checker := range checkers {
		var check Check

		fn, ok := funcs[checker.ID]
		if !ok {
			// This should never happen, but just in case...
			check = Check{
				ID:      checker.ID,
				Passed:  false,
				Details: fmt.Sprintf("checker function not found for %s", checker.ID),
			}

			pass = false

			results = append(results, check)

			continue
		}

		res := fn(ctx, dir)

		check.ID = checker.ID
		check.Passed = res.passed
		check.Details = res.details
		check.Definition = &checker

		if !check.Passed {
			pass = false
		}

		results = append(results, check)
	}

	return results, pass
}

type checkFunc func(ctx context.Context, dir string) *checkResult

type checkResult struct {
	passed  bool
	details string
}

func addToFilter(filter map[CheckID]struct{}, ids []CheckID) {
	for _, id := range ids {
		filter[id] = struct{}{}
	}
}

func checkersFor(opts *Options) []CheckDefinition {
	all := checkDefinitions()

	defs := make([]CheckDefinition, 0)

	if len(opts.EnableOnly) > 0 {
		for _, id := range opts.EnableOnly {
			if fn, found := all[id]; found {
				defs = append(defs, fn)
			}
		}

		return defs
	}

	for _, id := range opts.Disable {
		delete(all, id)
	}

	filter := make(map[CheckID]struct{}, len(opts.Enable))

	addToFilter(filter, opts.Enable)

	if preset, ok := presetDefinitions()[opts.Preset]; ok {
		addToFilter(filter, preset.Checks)
	}

	for _, id := range CheckIDs { // preserve order
		if _, found := filter[id]; !found {
			continue
		}

		if def, found := all[id]; found {
			defs = append(defs, def)
		}
	}

	return defs
}

func presetDefinitions() map[PresetID]PresetDefinition {
	m := make(map[PresetID]PresetDefinition, len(PresetDefinitions))

	for _, p := range PresetDefinitions {
		m[p.ID] = p
	}

	return m
}

func checkDefinitions() map[CheckID]CheckDefinition {
	m := make(map[CheckID]CheckDefinition, len(CheckDefinitions))

	for _, def := range CheckDefinitions {
		m[def.ID] = def
	}

	return m
}

func checkFunctions() map[CheckID]checkFunc {
	defs := map[CheckID]checkFunc{
		CheckIDSecurity:      checkerSecurity,
		CheckIDVulnerability: checkerVulnerability,
		CheckIDModule:        checkerModule,
		CheckIDReplace:       checkerReplace,
		CheckIDReadme:        checkerReadme,
		CheckIDLicense:       checkerLicense,
		CheckIDGit:           checkerGit,
		CheckIDVersions:      checkerVersions,
		CheckIDBuild:         checkerBuild,
		CheckIDSmoke:         checkerSmoke,
		CheckIDExamples:      checkerExamples,
		CheckIDTypes:         checkerTypes,
		CheckIDCodeowners:    checkerCodeowners,
	}

	return defs
}

func checkFailed(details string, args ...any) *checkResult {
	return &checkResult{passed: false, details: fmt.Sprintf(details, args...)}
}

func checkPassed(details string, args ...any) *checkResult {
	return &checkResult{passed: true, details: fmt.Sprintf(details, args...)}
}

func checkError(err error) *checkResult {
	return &checkResult{passed: false, details: "error: " + err.Error()}
}
