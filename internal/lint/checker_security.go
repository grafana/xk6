package lint

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

const gosecName = "gosec"

type gosecOut struct {
	Stats struct {
		Found int `json:"found"`
		Files int `json:"files"`
	} `json:"Stats"`
}

func checkerSecurity(ctx context.Context, dir string) *checkResult {
	_, err := exec.LookPath(gosecName)
	if err != nil {
		return checkFailed("the 'gosec' tool is missing")
	}

	cmd := exec.CommandContext( // #nosec G204
		ctx,
		gosecName,
		"-no-fail",
		"-fmt", "json",
		"-exclude-generated",
		"-r", dir+"/...",
	)

	data, err := cmd.Output()
	if err != nil {
		return checkError(err)
	}

	var out gosecOut

	if err := json.Unmarshal(data, &out); err != nil {
		return checkError(err)
	}

	if out.Stats.Files == 0 {
		return checkFailed("no checked files")
	}

	if out.Stats.Found != 0 {
		return checkFailed(fmt.Sprintf("%d issue(s) found", out.Stats.Found))
	}

	return checkPassed("no issues found")
}
