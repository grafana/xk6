package lint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

var reSmoke = regexp.MustCompile(`(?i)^smoke(\.test)?\.(?:js|ts)`)

func checkerSmoke(ctx context.Context, dir string) *checkResult {
	s := getState(ctx)

	exe, err := s.exePath(ctx)
	if err != nil {
		return checkError(err)
	}

	js, err := s.isJS(ctx)
	if err != nil {
		return checkError(err)
	}

	if !js {
		return checkPassed("skipped due to non JavaScript extension")
	}

	filename, shortname, err := findFile(reSmoke,
		dir,
		filepath.Join(dir, "test"),
		filepath.Join(dir, "tests"),
		filepath.Join(dir, "examples"),
		filepath.Join(dir, "scripts"),
	)
	if err != nil {
		return checkError(err)
	}

	if len(shortname) == 0 {
		return checkFailed("no smoke test file found")
	}

	cmd := exec.CommandContext(ctx, exe, "run", "--no-usage-report", "--no-summary", "--quiet", filename) // #nosec G204

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, string(out))

		return checkError(err)
	}

	return checkPassed("`%s` successfully run with k6", shortname)
}
