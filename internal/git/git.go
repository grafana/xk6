// Package git provides the small subset of Git operations used by xk6.
package git

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Clone clones url into dir using the Git executable available on PATH.
func Clone(ctx context.Context, url, dir string) error {
	parent := filepath.Dir(dir)
	name := filepath.Base(dir)

	_, err := run(ctx, parent, "clone", "--", url, name)

	return err
}

// RemoteURL returns the URL configured for remote on dir.
func RemoteURL(ctx context.Context, dir, remote string) (string, error) {
	out, err := run(ctx, dir, "remote", "get-url", remote)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

// IsWorktree verifies that dir is inside a non-bare Git worktree.
func IsWorktree(ctx context.Context, dir string) error {
	out, err := run(ctx, dir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return err
	}

	if strings.TrimSpace(out) != "true" {
		return fmt.Errorf("directory is not a Git worktree")
	}

	return nil
}

// Tags returns all tag names in dir.
func Tags(ctx context.Context, dir string) ([]string, error) {
	out, err := run(ctx, dir, "for-each-ref", "--format=%(refname:strip=2)", "refs/tags")
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, tag := range strings.Split(out, "\n") {
		tag = strings.TrimSuffix(tag, "\r")
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

// Checkout checks out ref in detached HEAD mode in dir.
func Checkout(ctx context.Context, dir, ref string) error {
	_, err := run(ctx, dir, "checkout", "--quiet", "--detach", ref)

	return err
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...) // #nosec G204 -- arguments are passed directly without a shell
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, detail)
		}

		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return string(out), nil
}
