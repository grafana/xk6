package lint

import (
	"context"
	"fmt"
	"os"
)

type checkFunc func(ctx context.Context, dir string) *checkResult

type checkResult struct {
	passed  bool
	details string
}

func checkFailed(details string) *checkResult {
	return &checkResult{passed: false, details: details}
}

func checkPassed(details string, args ...any) *checkResult {
	return &checkResult{passed: true, details: fmt.Sprintf(details, args...)}
}

func checkError(err error) *checkResult {
	return &checkResult{passed: false, details: "error: " + err.Error()}
}

type checkDefinition struct {
	id Checker
	fn checkFunc
}

//nolint:mnd
func checkDefinitions(official bool) []checkDefinition {
	modCheck := newModuleChecker()
	gitCheck := newGitChecker()

	defs := []checkDefinition{
		{id: CheckerSecurity, fn: checkerSecurity},
		{id: CheckerVulnerability, fn: checkerVulnerability},
		{id: CheckerModule, fn: modCheck.hasGoModule},
		{id: CheckerReplace, fn: modCheck.hasNoReplace},
		{id: CheckerReadme, fn: checkerReadme},
		{id: CheckerLicense, fn: checkerLicense},
		{id: CheckerGit, fn: gitCheck.isWorkDir},
		{id: CheckerVersions, fn: gitCheck.hasVersions},
		{id: CheckerBuild, fn: modCheck.canBuild},
		{id: CheckerSmoke, fn: modCheck.smoke},
		{id: CheckerExamples, fn: modCheck.examples},
		{id: CheckerTypes, fn: modCheck.types},
	}

	if !official {
		return defs
	}

	extra := []checkDefinition{
		{id: CheckerCodeowners, fn: checkerCodeowners},
	}

	defs = append(defs, extra...)

	return defs
}

func runChecks(ctx context.Context, dir string, opts *Options) ([]Check, bool) {
	checkDefs := checkDefinitions(opts.Official)
	results := make([]Check, 0, len(checkDefs))
	passed := passedChecks(opts.Passed)

	pass := true

	for _, checker := range checkDefs {
		var check Check

		if c, found := passed[checker.id]; found {
			check = c
		} else {
			res := checker.fn(ctx, dir)

			check.ID = checker.id
			check.Passed = res.passed
			check.Details = res.details
		}

		if !check.Passed {
			pass = false
		}

		results = append(results, check)
	}

	return results, pass
}

// ParseChecker parses checker name from string.
func ParseChecker(val string) (Checker, error) {
	v := Checker(val)

	switch v {
	case
		CheckerSecurity,
		CheckerVulnerability,
		CheckerModule,
		CheckerReplace,
		CheckerReadme,
		CheckerExamples,
		CheckerLicense,
		CheckerGit,
		CheckerVersions,
		CheckerBuild,
		CheckerSmoke,
		CheckerCodeowners,
		CheckerTypes:
		return v, nil
	default:
		return "", fmt.Errorf("%w: %s", os.ErrInvalid, val)
	}
}
