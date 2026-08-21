package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRepositoryOperations(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	ctx := context.Background()
	source := t.TempDir()
	t.Setenv("GIT_AUTHOR_NAME", "xk6 tests")
	t.Setenv("GIT_AUTHOR_EMAIL", "xk6@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "xk6 tests")
	t.Setenv("GIT_COMMITTER_EMAIL", "xk6@example.com")
	runGit(t, source, "init")
	runGit(t, source, "config", "user.email", "xk6@example.com")
	runGit(t, source, "config", "user.name", "xk6 tests")
	runGit(t, source, "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("v1\n"), 0o644); err != nil { //nolint:forbidigo
		t.Fatal(err)
	}
	runGit(t, source, "add", "README.md")
	runGit(t, source, "commit", "-m", "initial")
	runGit(t, source, "tag", "v1.0.0")

	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("v2\n"), 0o644); err != nil { //nolint:forbidigo
		t.Fatal(err)
	}
	runGit(t, source, "add", "README.md")
	runGit(t, source, "commit", "-m", "second")
	runGit(t, source, "tag", "v2.0.0")
	runGit(t, source, "tag", "not-a-version")

	clone := filepath.Join(t.TempDir(), "clone")
	if err := Clone(ctx, source, clone); err != nil {
		t.Fatal(err)
	}

	remote, err := RemoteURL(ctx, clone, "origin")
	if err != nil {
		t.Fatal(err)
	}
	if remote != source {
		t.Fatalf("remote URL = %q, want %q", remote, source)
	}

	if err := IsWorktree(ctx, clone); err != nil {
		t.Fatal(err)
	}

	tags, err := Tags(ctx, clone)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(tags, "v1.0.0") || !contains(tags, "v2.0.0") || !contains(tags, "not-a-version") {
		t.Fatalf("tags = %v, want all test tags", tags)
	}

	if err := Checkout(ctx, clone, "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(clone, "README.md")) //nolint:forbidigo
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != "v1" {
		t.Fatalf("README after checkout = %q, want v1", content)
	}
}

func contains(values []string, want string) bool {
	return slices.Contains(values, want)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
