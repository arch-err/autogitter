package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	gosync "sync"

	"github.com/arch-err/autogitter/internal/config"
	"github.com/arch-err/autogitter/internal/connector"
	"github.com/arch-err/autogitter/internal/git"
	"github.com/arch-err/autogitter/internal/ui"
)

type SyncOptions struct {
	Prune      bool
	Add        bool
	Force      bool
	ConfigPath string
	Jobs       int
	DryRun     bool
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

	// Load credentials from credentials.env if it exists
	credPath := connector.DefaultCredentialsPath()
	if err := connector.LoadCredentialsEnv(credPath); err != nil {
		ui.Debug("failed to load credentials file", "error", err)
	}

	for i := range cfg.Sources {
		source := &cfg.Sources[i]

		// Handle strategy-specific logic
		switch source.Strategy {
		case config.StrategyManual:
			// Manual strategy uses the repos from config
		case config.StrategyAll:
			// Fetch repos from API
			repos, err := fetchReposFromAPI(source)
			if err != nil {
				ui.Warn("skipping source - failed to fetch repos", "source", source.Name, "error", err)
				continue
			}
			source.Repos = repos
			ui.Debug("fetched repos from API", "source", source.Name, "count", len(repos))
		case config.StrategyRegex:
			// Fetch repos from API, then filter by regex pattern
			repos, err := fetchReposFromAPI(source)
			if err != nil {
				ui.Warn("skipping source - failed to fetch repos", "source", source.Name, "error", err)
				continue
			}
			filtered, err := filterReposByRegex(repos, source.RegexStrategy.Pattern)
			if err != nil {
				ui.Warn("skipping source - invalid regex pattern", "source", source.Name, "error", err)
				continue
			}
			source.Repos = filtered
			ui.Debug("fetched and filtered repos from API", "source", source.Name, "total", len(repos), "matched", len(filtered))
		case config.StrategyFile:
			ui.Warn("skipping source with unsupported strategy", "source", source.Name, "strategy", source.Strategy)
			continue
		default:
			ui.Warn("skipping source with unknown strategy", "source", source.Name, "strategy", source.Strategy)
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

// fetchReposFromAPI fetches repository list from the Git provider API
func fetchReposFromAPI(source *config.Source) ([]string, error) {
	connType := source.GetConnectorType()
	token := connector.GetToken(connType)

	if token == "" {
		envVar := connector.GetEnvVarName(connType)
		return nil, fmt.Errorf("no token found - set %s or run 'ag connect'", envVar)
	}

	host := source.GetHost()
	userOrOrg := source.GetUserOrOrg()

	if userOrOrg == "" {
		return nil, fmt.Errorf("source must include user/org (e.g., github.com/username)")
	}

	conn, err := connector.New(connType, host, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create connector: %w", err)
	}

	ctx := context.Background()
	repos, err := conn.ListRepos(ctx, userOrOrg)
	if err != nil {
		return nil, fmt.Errorf("failed to list repos: %w", err)
	}

	return repos, nil
}

// filterReposByRegex filters a list of repo names by a regex pattern.
// The pattern is matched against the full repo name (user/repo format).
func filterReposByRegex(repos []string, pattern string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}

	var filtered []string
	for _, repo := range repos {
		if re.MatchString(repo) {
			filtered = append(filtered, repo)
		}
	}

	return filtered, nil
}

func syncSource(source *config.Source, cfg *config.Config, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{}

	// Check if local_path exists, prompt to create if not
	if _, err := os.Stat(source.LocalPath); os.IsNotExist(err) {
		if opts.DryRun {
			ui.Info("would create directory", "path", source.LocalPath)
		} else {
			if !opts.Force {
				create, promptErr := ui.ConfirmCreateDir(source.LocalPath)
				if promptErr != nil {
					return nil, fmt.Errorf("failed to get user input: %w", promptErr)
				}
				if !create {
					ui.Info("skipping source", "source", source.Name, "reason", "directory not created")
					return result, nil
				}
			}
			if err := os.MkdirAll(source.LocalPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
			ui.Info("created directory", "path", source.LocalPath)
		}
	}

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

		if opts.DryRun {
			// In dry-run mode, just report what would happen based on flags
			orphaned := getOrphanedRepos(statuses)
			if opts.Prune {
				for _, repo := range orphaned {
					ui.Info("would prune", "repo", repo.Name)
				}
			} else if opts.Add {
				for _, repo := range orphaned {
					fullName := guessFullName(source.Source, repo.Name)
					ui.Info("would add to config", "repo", fullName)
				}
			}
		} else {
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
	}

	// Clone new repos in parallel
	var toClone []RepoStatus
	for _, status := range statuses {
		if status.Status == ui.StatusAdded {
			toClone = append(toClone, status)
		}
	}

	if len(toClone) > 0 {
		if opts.DryRun {
			for _, repo := range toClone {
				ui.Info("would clone", "repo", repo.FullName, "path", repo.LocalPath)
			}
		} else {
			cloned := cloneReposParallel(toClone, source, opts.Jobs)
			result.Cloned = cloned
		}
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
			PrivateKey: job.source.GetPrivateKey(),
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

// ComputeSourceStatus computes the status of repos for a single source
// without performing any actions. Returns the list of repo statuses.
func ComputeSourceStatus(source *config.Source) ([]RepoStatus, error) {
	// Load credentials from credentials.env if it exists
	credPath := connector.DefaultCredentialsPath()
	if err := connector.LoadCredentialsEnv(credPath); err != nil {
		ui.Debug("failed to load credentials file", "error", err)
	}

	// Handle strategy-specific logic
	switch source.Strategy {
	case config.StrategyManual:
		// Manual strategy uses the repos from config
	case config.StrategyAll:
		repos, err := fetchReposFromAPI(source)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repos: %w", err)
		}
		source.Repos = repos
	case config.StrategyRegex:
		repos, err := fetchReposFromAPI(source)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repos: %w", err)
		}
		filtered, err := filterReposByRegex(repos, source.RegexStrategy.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		source.Repos = filtered
	case config.StrategyFile:
		return nil, fmt.Errorf("file strategy not yet supported")
	default:
		return nil, fmt.Errorf("unknown strategy: %s", source.Strategy)
	}

	// Build configured repos map
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

	return statuses, nil
}

// PullOptions contains options for the pull command
type PullOptions struct {
	Force bool
	Jobs  int
}

// PullResult contains the results of a pull operation
type PullResult struct {
	Updated int
	Failed  int
	Skipped int
}

type pullJob struct {
	path       string
	name       string
	privateKey string
}

type pullResult struct {
	name    string
	success bool
	err     error
}

// RunPull pulls all repos for all configured sources
func RunPull(cfg *config.Config, opts PullOptions) (*PullResult, error) {
	result := &PullResult{}

	// Load credentials from credentials.env if it exists
	credPath := connector.DefaultCredentialsPath()
	if err := connector.LoadCredentialsEnv(credPath); err != nil {
		ui.Debug("failed to load credentials file", "error", err)
	}

	var allJobs []pullJob

	for i := range cfg.Sources {
		source := &cfg.Sources[i]

		// Check if local_path exists
		if _, err := os.Stat(source.LocalPath); os.IsNotExist(err) {
			ui.Warn("skipping source - directory does not exist", "source", source.Name, "path", source.LocalPath)
			continue
		}

		// Scan local directory for repos
		localRepos, err := scanLocalRepos(source.LocalPath)
		if err != nil {
			ui.Warn("skipping source - failed to scan local repos", "source", source.Name, "error", err)
			continue
		}

		ui.Info("found repos to pull", "source", source.Name, "count", len(localRepos))

		// Add jobs for each repo
		for repoName := range localRepos {
			repoPath := filepath.Join(source.LocalPath, repoName)
			allJobs = append(allJobs, pullJob{
				path:       repoPath,
				name:       repoName,
				privateKey: source.GetPrivateKey(),
			})
		}
	}

	if len(allJobs) == 0 {
		ui.Info("no repos to pull")
		return result, nil
	}

	// Pull repos in parallel
	updated, failed := pullReposParallel(allJobs, opts.Jobs)
	result.Updated = updated
	result.Failed = failed

	return result, nil
}

func pullReposParallel(jobs []pullJob, numWorkers int) (int, int) {
	if numWorkers <= 0 {
		numWorkers = 4
	}

	// Don't use more workers than jobs
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}

	jobsChan := make(chan pullJob, len(jobs))
	results := make(chan pullResult, len(jobs))

	// Start progress spinner
	progress := ui.NewProgress(len(jobs), "Pulling repos")

	// Start workers
	var wg gosync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go pullWorker(jobsChan, results, &wg)
	}

	// Send jobs
	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan)

	// Wait for workers to finish, then close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	updated := 0
	failed := 0
	var errors []pullResult
	for res := range results {
		progress.Increment()
		if res.success {
			updated++
		} else {
			failed++
			errors = append(errors, res)
		}
	}

	// Stop spinner before printing results
	progress.Finish()

	// Print errors
	for _, res := range errors {
		ui.Error("failed to pull", "repo", res.name, "error", res.err)
	}
	if updated > 0 {
		ui.Info("pulled repos", "count", updated)
	}

	return updated, failed
}

func pullWorker(jobs <-chan pullJob, results chan<- pullResult, wg *gosync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		err := git.Pull(git.PullOptions{
			Path:       job.path,
			PrivateKey: job.privateKey,
		})
		results <- pullResult{
			name:    job.name,
			success: err == nil,
			err:     err,
		}
	}
}
