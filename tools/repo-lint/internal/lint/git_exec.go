package lint

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ExecGitClient invokes the system git CLI against RepoRoot.
type ExecGitClient struct{}

func (ExecGitClient) runGit(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, msg) // txthc-ignore
	}
	return stdout.String(), nil
}

// TrackedFiles returns normalized slash-separated paths from `git ls-files`.
func (g ExecGitClient) TrackedFiles(root string) ([]string, error) {
	out, err := g.runGit(root, "ls-files")
	if err != nil {
		return nil, err
	}
	return splitNormalizedLines(out), nil
}

// StagedFiles returns normalized slash-separated paths staged for commit.
func (g ExecGitClient) StagedFiles(root string) ([]string, error) {
	out, err := g.runGit(root, "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	if err != nil {
		return nil, err
	}
	return splitNormalizedLines(out), nil
}

// StagedDiffUnified0 returns `git diff --cached -U0 -- <relPath>`.
func (g ExecGitClient) StagedDiffUnified0(root, relPath string) (string, error) {
	return g.runGit(root, "diff", "--cached", "-U0", "--", relPath)
}

func splitNormalizedLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, strings.ReplaceAll(line, "\\", "/"))
	}
	return out
}
