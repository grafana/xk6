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
	id    Checker
	fn    checkFunc
	score int
}

//nolint:mnd
func checkDefinitions(official bool) []checkDefinition {
	modCheck := newModuleChecker()
	gitCheck := newGitChecker()

	defs := []checkDefinition{
		{id: CheckerSecurity, score: 0, fn: checkerSecurity},
		{id: CheckerVulnerability, score: 0, fn: checkerVulnerability},
		{id: CheckerModule, score: 2, fn: modCheck.hasGoModule},
		{id: CheckerReplace, score: 2, fn: modCheck.hasNoReplace},
		{id: CheckerReadme, score: 5, fn: checkerReadme},
		{id: CheckerLicense, score: 5, fn: checkerLicense},
		{id: CheckerGit, score: 1, fn: gitCheck.isWorkDir},
		{id: CheckerVersions, score: 5, fn: gitCheck.hasVersions},
		{id: CheckerBuild, score: 5, fn: modCheck.canBuild},
		{id: CheckerSmoke, score: 2, fn: modCheck.smoke},
		{id: CheckerExamples, score: 2, fn: modCheck.examples},
		{id: CheckerTypes, score: 2, fn: modCheck.types},
	}

	if !official {
		return defs
	}

	extra := []checkDefinition{
		{id: CheckerCodeowners, score: 2, fn: checkerCodeowners},
	}

	defs = append(defs, extra...)

	return defs
}

func runChecks(ctx context.Context, dir string, opts *Options) ([]Check, int) {
	checkDefs := checkDefinitions(opts.Official)
	results := make([]Check, 0, len(checkDefs))
	passed := passedChecks(opts.Passed)

	var (
		total, sum float64
		blocking   bool
	)

	for _, checker := range checkDefs {
		total += float64(checker.score)

		var check Check

		if c, found := passed[checker.id]; found {
			check = c
		} else {
			res := checker.fn(ctx, dir)

			check.ID = checker.id
			check.Passed = res.passed
			check.Details = res.details
		}

		if check.Passed {
			sum += float64(checker.score)
		} else {
			blocking = blocking || (checker.score == 0)
		}

		results = append(results, check)
	}

	if blocking {
		sum = 0
	}

	const hundredPercent = 100.0

	return results, int((sum / total) * hundredPercent)
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
