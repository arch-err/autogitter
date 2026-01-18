package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
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
