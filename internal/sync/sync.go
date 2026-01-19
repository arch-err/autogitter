package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	gosync "sync"

	"github.com/arch-err/autogitter/internal/config"
	"github.com/arch-err/autogitter/internal/git"
	"github.com/arch-err/autogitter/internal/ui"
)

type SyncOptions struct {
	Prune      bool
	Add        bool
	Force      bool
	ConfigPath string
	Jobs       int
}

type cloneJob struct {
	status RepoStatus
	source *config.Source
}

type cloneResult struct {
	name    string
	success bool
	err     error
}

type SyncResult struct {
	Cloned  int
	Pruned  int
	Skipped int
	Added   int
}

type RepoStatus struct {
	Name       string
	FullName   string
	LocalPath  string
	Status     ui.DiffStatus
	InConfig   bool
	ExistsLocal bool
}

func Run(cfg *config.Config, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{}

	for i := range cfg.Sources {
		source := &cfg.Sources[i]
		if source.Strategy != config.StrategyManual {
			ui.Warn("skipping source with unsupported strategy", "source", source.Name, "strategy", source.Strategy)
			continue
		}

		sourceResult, err := syncSource(source, cfg, opts)
		if err != nil {
			ui.Error("failed to sync source", "source", source.Name, "error", err)
			continue
		}

		result.Cloned += sourceResult.Cloned
		result.Pruned += sourceResult.Pruned
		result.Skipped += sourceResult.Skipped
		result.Added += sourceResult.Added
	}

	return result, nil
}

func syncSource(source *config.Source, cfg *config.Config, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{}

	// Get configured repos
	configuredRepos := make(map[string]bool)
	for _, repo := range source.Repos {
		repoName := repoNameFromFullName(repo)
		configuredRepos[repoName] = true
	}

	// Scan local directory
	localRepos, err := scanLocalRepos(source.LocalPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to scan local repos: %w", err)
	}

	// Build status list
	var statuses []RepoStatus

	// Add configured repos
	for _, repo := range source.Repos {
		repoName := repoNameFromFullName(repo)
		localPath := filepath.Join(source.LocalPath, repoName)
		exists := localRepos[repoName]

		status := ui.StatusAdded
		if exists {
			status = ui.StatusUnchanged
		}

		statuses = append(statuses, RepoStatus{
			Name:        repoName,
			FullName:    repo,
			LocalPath:   localPath,
			Status:      status,
			InConfig:    true,
			ExistsLocal: exists,
		})
	}

	// Add orphaned repos (in local but not in config)
	for repoName := range localRepos {
		if !configuredRepos[repoName] {
			localPath := filepath.Join(source.LocalPath, repoName)
			statuses = append(statuses, RepoStatus{
				Name:        repoName,
				LocalPath:   localPath,
				Status:      ui.StatusRemoved,
				InConfig:    false,
				ExistsLocal: true,
			})
		}
	}

	// Check if there are any changes
	hasNew := false
	hasOrphaned := false
	for _, s := range statuses {
		if s.Status == ui.StatusAdded {
			hasNew = true
		}
		if s.Status == ui.StatusRemoved {
			hasOrphaned = true
		}
	}

	if !hasNew && !hasOrphaned {
		ui.Info("source is up to date", "source", source.Name)
		return result, nil
	}

	// Print diff
	entries := make([]ui.DiffEntry, len(statuses))
	for i, s := range statuses {
		entries[i] = ui.DiffEntry{Name: s.Name, Status: s.Status}
	}
	ui.PrintDiff(source.Name, entries)

	// Handle orphaned repos
	if hasOrphaned {
		action := "skip"

		if opts.Prune {
			action = "prune"
		} else if opts.Add {
			action = "add"
		} else {
			// Interactive mode
			var err error
			action, err = ui.ConfirmAction()
			if err != nil {
				return nil, fmt.Errorf("failed to get user input: %w", err)
			}
		}

		switch action {
		case "prune":
			orphaned := getOrphanedRepos(statuses)
			if !opts.Force {
				names := make([]string, len(orphaned))
				for i, r := range orphaned {
					names[i] = r.Name
				}
				confirm, err := ui.ConfirmPrune(names)
				if err != nil {
					return nil, fmt.Errorf("failed to get confirmation: %w", err)
				}
				if !confirm {
					ui.Info("prune cancelled")
					break
				}
			}

			for _, repo := range orphaned {
				ui.Info("removing", "repo", repo.Name)
				if err := os.RemoveAll(repo.LocalPath); err != nil {
					ui.Error("failed to remove repo", "repo", repo.Name, "error", err)
					continue
				}
				result.Pruned++
			}

		case "add":
			orphaned := getOrphanedRepos(statuses)
			for _, repo := range orphaned {
				fullName := guessFullName(source.Source, repo.Name)
				source.Repos = append(source.Repos, fullName)
				result.Added++
				ui.Info("added to config", "repo", fullName)
			}

			// Save updated config
			if opts.ConfigPath != "" {
				if err := cfg.Save(opts.ConfigPath); err != nil {
					ui.Error("failed to save config", "error", err)
				} else {
					ui.Info("config saved", "path", opts.ConfigPath)
				}
			}
		}
	}

	// Clone new repos in parallel
	var toClone []RepoStatus
	for _, status := range statuses {
		if status.Status == ui.StatusAdded {
			toClone = append(toClone, status)
		}
	}

	if len(toClone) > 0 {
		cloned := cloneReposParallel(toClone, source, opts.Jobs)
		result.Cloned = cloned
	}

	return result, nil
}

func cloneReposParallel(repos []RepoStatus, source *config.Source, numWorkers int) int {
	if numWorkers <= 0 {
		numWorkers = 4
	}

	// Don't use more workers than repos
	if numWorkers > len(repos) {
		numWorkers = len(repos)
	}

	jobs := make(chan cloneJob, len(repos))
	results := make(chan cloneResult, len(repos))

	// Start progress spinner
	progress := ui.NewProgress(len(repos), "Cloning repos")

	// Start workers
	var wg gosync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go cloneWorker(jobs, results, &wg)
	}

	// Send jobs
	for _, repo := range repos {
		jobs <- cloneJob{status: repo, source: source}
	}
	close(jobs)

	// Wait for workers to finish, then close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	cloned := 0
	var errors []cloneResult
	for res := range results {
		progress.Increment()
		if res.success {
			cloned++
		} else {
			errors = append(errors, res)
		}
	}

	// Stop spinner before printing results
	progress.Finish()

	// Print results
	for _, res := range errors {
		ui.Error("failed to clone", "repo", res.name, "error", res.err)
	}
	if cloned > 0 {
		ui.Info("cloned repos", "count", cloned)
	}

	return cloned
}

func cloneWorker(jobs <-chan cloneJob, results chan<- cloneResult, wg *gosync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		err := git.Clone(git.CloneOptions{
			URL:        job.source.GetRepoURL(job.status.FullName),
			Path:       job.status.LocalPath,
			Branch:     job.source.GetBranch(),
			PrivateKey: job.source.PrivateKey,
		})
		results <- cloneResult{
			name:    job.status.FullName,
			success: err == nil,
			err:     err,
		}
	}
}

func scanLocalRepos(path string) (map[string]bool, error) {
	repos := make(map[string]bool)

	entries, err := os.ReadDir(path)
	if err != nil {
		return repos, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(path, entry.Name())
		if git.IsGitRepo(fullPath) {
			repos[entry.Name()] = true
		}
	}

	return repos, nil
}

func repoNameFromFullName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

func guessFullName(source, repoName string) string {
	// Try to extract the default user/org from the source
	// e.g., github.com/arch-err -> arch-err/repoName
	parts := strings.Split(source, "/")
	if len(parts) >= 2 {
		user := parts[len(parts)-1]
		return user + "/" + repoName
	}
	return repoName
}

func getOrphanedRepos(statuses []RepoStatus) []RepoStatus {
	var orphaned []RepoStatus
	for _, s := range statuses {
		if s.Status == ui.StatusRemoved {
			orphaned = append(orphaned, s)
		}
	}
	return orphaned
}
