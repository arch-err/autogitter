package ui

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"golang.design/x/clipboard"
	"golang.org/x/term"
)

var (
	// Styles for diff display
	AddedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	RemovedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	UnchangedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ABABAB"))
	HeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	SourceStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
)

var Logger *log.Logger

func init() {
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})
}

func SetDebug(enabled bool) {
	if enabled {
		Logger.SetLevel(log.DebugLevel)
	} else {
		Logger.SetLevel(log.InfoLevel)
	}
}

type DiffEntry struct {
	Name   string
	Status DiffStatus
}

type DiffStatus int

const (
	StatusAdded DiffStatus = iota
	StatusRemoved
	StatusUnchanged
)

func PrintDiff(sourceName string, entries []DiffEntry) {
	fmt.Println()
	fmt.Println(SourceStyle.Render(fmt.Sprintf("  %s", sourceName)))
	fmt.Println()

	for _, entry := range entries {
		var prefix string
		var style lipgloss.Style

		switch entry.Status {
		case StatusAdded:
			prefix = "  + "
			style = AddedStyle
		case StatusRemoved:
			prefix = "  - "
			style = RemovedStyle
		case StatusUnchanged:
			prefix = "    "
			style = UnchangedStyle
		}

		fmt.Println(style.Render(prefix + entry.Name))
	}
	fmt.Println()
}

// SourceDiff represents the diff for a single source
type SourceDiff struct {
	Name    string
	Entries []DiffEntry
}

// PrintUnifiedDiff prints a unified diff-style output comparing local vs config
func PrintUnifiedDiff(diffs []SourceDiff) {
	// Diff header style (cyan like git diff headers)
	diffHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))
	// Hunk header style (purple/magenta like @@ lines)
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))

	fmt.Println(diffHeaderStyle.Render("--- local"))
	fmt.Println(diffHeaderStyle.Render("+++ config"))

	for _, diff := range diffs {
		fmt.Println(hunkStyle.Render(fmt.Sprintf("@@ %s @@", diff.Name)))

		for _, entry := range diff.Entries {
			var line string
			var style lipgloss.Style

			switch entry.Status {
			case StatusAdded:
				line = "+ " + entry.Name
				style = AddedStyle
			case StatusRemoved:
				line = "- " + entry.Name
				style = RemovedStyle
			case StatusUnchanged:
				line = "  " + entry.Name
				style = UnchangedStyle
			}

			fmt.Println(style.Render(line))
		}
		fmt.Println()
	}
}

func ConfirmPrune(repos []string) (bool, error) {
	if len(repos) == 0 {
		return false, nil
	}

	var confirm bool
	err := huh.NewConfirm().
		Title("Delete these repos from disk?").
		Description(fmt.Sprintf("%d repo(s) not in config will be permanently deleted", len(repos))).
		Affirmative("Yes, delete").
		Negative("No, keep").
		Value(&confirm).
		Run()

	return confirm, err
}

func ConfirmAction() (string, error) {
	var action string
	err := huh.NewSelect[string]().
		Title("Repos found that are not in config. What would you like to do?").
		Options(
			huh.NewOption("Prune - Delete repos not in config", "prune"),
			huh.NewOption("Add - Add repos to config", "add"),
			huh.NewOption("Skip - Do nothing", "skip"),
		).
		Value(&action).
		Run()

	return action, err
}

func ConfirmSync(toClone int, toRemove int) (bool, error) {
	var confirm bool
	desc := fmt.Sprintf("Will clone %d repo(s)", toClone)
	if toRemove > 0 {
		desc += fmt.Sprintf(" and remove %d repo(s)", toRemove)
	}

	err := huh.NewConfirm().
		Title("Proceed with sync?").
		Description(desc).
		Affirmative("Yes").
		Negative("No").
		Value(&confirm).
		Run()

	return confirm, err
}

func ConfirmCreateDir(path string) (bool, error) {
	var confirm bool
	err := huh.NewConfirm().
		Title(fmt.Sprintf("Directory does not exist: %s", path)).
		Description("Would you like to create it?").
		Affirmative("Yes, create").
		Negative("No, skip").
		Value(&confirm).
		Run()

	return confirm, err
}

func PrintSummary(cloned, pruned, skipped int) {
	fmt.Println()
	fmt.Println(HeaderStyle.Render("Summary"))
	if cloned > 0 {
		fmt.Println(AddedStyle.Render(fmt.Sprintf("  Cloned: %d", cloned)))
	}
	if pruned > 0 {
		fmt.Println(RemovedStyle.Render(fmt.Sprintf("  Pruned: %d", pruned)))
	}
	if skipped > 0 {
		fmt.Println(UnchangedStyle.Render(fmt.Sprintf("  Skipped: %d", skipped)))
	}
	fmt.Println()
}

func Info(msg string, args ...interface{}) {
	Logger.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	Logger.Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	Logger.Error(msg, args...)
}

func Debug(msg string, args ...interface{}) {
	Logger.Debug(msg, args...)
}

// Progress tracks cloning progress with an animated spinner
type Progress struct {
	total     int
	completed int
	message   string
	mu        sync.Mutex
	done      chan struct{}
	isTTY     bool
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewProgress creates a new progress tracker
func NewProgress(total int, message string) *Progress {
	p := &Progress{
		total:   total,
		message: message,
		done:    make(chan struct{}),
		isTTY:   term.IsTerminal(int(os.Stdout.Fd())),
	}

	if p.isTTY {
		go p.animate()
	} else {
		fmt.Printf("%s (0/%d)\n", message, total)
	}

	return p
}

func (p *Progress) animate() {
	frame := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			// Clear the spinner line
			fmt.Print("\r\033[K")
			return
		case <-ticker.C:
			p.mu.Lock()
			spinner := spinnerFrames[frame%len(spinnerFrames)]
			fmt.Printf("\r%s %s (%d/%d)", spinner, p.message, p.completed, p.total)
			p.mu.Unlock()
			frame++
		}
	}
}

// Increment marks one more item as completed
func (p *Progress) Increment() {
	p.mu.Lock()
	p.completed++
	completed := p.completed
	total := p.total
	p.mu.Unlock()

	if !p.isTTY {
		fmt.Printf("%s (%d/%d)\n", p.message, completed, total)
	}
}

// Finish stops the progress animation
func (p *Progress) Finish() {
	if p.isTTY {
		close(p.done)
		// Small delay to ensure animation goroutine exits
		time.Sleep(100 * time.Millisecond)
	}
}

// IsTTY returns whether we're running in an interactive terminal
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// CopyToClipboard copies text to the system clipboard.
// It tries the clipboard package first, then falls back to OSC 52 escape sequences.
func CopyToClipboard(text string) bool {
	// Try clipboard package first
	if err := clipboard.Init(); err == nil {
		clipboard.Write(clipboard.FmtText, []byte(text))
		return true
	}

	// Fall back to OSC 52 escape sequence (works in most terminals including tmux)
	// OSC 52 format: \033]52;c;<base64-encoded-text>\007
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	fmt.Printf("\033]52;c;%s\007", encoded)
	return true
}
