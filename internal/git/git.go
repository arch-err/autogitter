package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

type CloneOptions struct {
	URL        string
	Path       string
	Branch     string
	PrivateKey string
	Submodules bool
}

type PullOptions struct {
	Path       string
	PrivateKey string
	Submodules bool
}

func Clone(opts CloneOptions) error {
	if opts.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if opts.Path == "" {
		return fmt.Errorf("path is required")
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(opts.Path)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	args := []string{"clone"}

	if opts.Submodules {
		args = append(args, "--recurse-submodules")
	}

	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	args = append(args, opts.URL, opts.Path)

	cmd := exec.Command("git", args...)

	// Handle custom SSH key
	if opts.PrivateKey != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new", opts.PrivateKey)
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(output))
	}

	log.Debug("cloned repository", "url", opts.URL, "path", opts.Path)
	return nil
}

func Pull(opts PullOptions) error {
	if opts.Path == "" {
		return fmt.Errorf("path is required")
	}

	cmd := exec.Command("git", "-C", opts.Path, "pull")

	// Handle custom SSH key
	if opts.PrivateKey != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new", opts.PrivateKey)
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %w\n%s", err, string(output))
	}

	log.Debug("pulled repository", "path", opts.Path)

	if opts.Submodules {
		subCmd := exec.Command("git", "-C", opts.Path, "submodule", "update", "--init", "--recursive")
		if opts.PrivateKey != "" {
			sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new", opts.PrivateKey)
			subCmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
		}
		subOutput, subErr := subCmd.CombinedOutput()
		if subErr != nil {
			return fmt.Errorf("git submodule update failed: %w\n%s", subErr, string(subOutput))
		}
		log.Debug("updated submodules", "path", opts.Path)
	}

	return nil
}

func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func GetRemoteURL(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func GetCurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func RepoNameFromPath(path string) string {
	return filepath.Base(path)
}

func RepoNameFromURL(url string) string {
	// Handle SSH URLs like git@github.com:user/repo.git
	if strings.Contains(url, ":") && strings.Contains(url, "@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			repo := parts[1]
			repo = strings.TrimSuffix(repo, ".git")
			return filepath.Base(repo)
		}
	}

	// Handle HTTPS URLs
	url = strings.TrimSuffix(url, ".git")
	return filepath.Base(url)
}
