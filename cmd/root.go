package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/beargruug/runx/internal/deps"
	"github.com/beargruug/runx/internal/detect"
	"github.com/beargruug/runx/internal/env"
	"github.com/beargruug/runx/internal/runner"
	"github.com/beargruug/runx/internal/runtime"
	"github.com/beargruug/runx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	dryRun      bool
	skipInstall bool
	runAll      bool
)

var rootCmd = &cobra.Command{
	Use:   "runx [package]",
	Short: "Universal project runner — clone, run, done.",
	Long:  "runx detects your project stack, installs runtimes & dependencies, sets up .env, and starts the dev server. Zero config.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  execute,
}

func init() {
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without doing it")
	rootCmd.Flags().BoolVar(&skipInstall, "skip-install", false, "Skip runtime and dependency installation")
	rootCmd.Flags().BoolVar(&runAll, "all", false, "Run all packages in a monorepo")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	ui.Banner()

	// Get working directory
	dir, err := os.Getwd()
	if err != nil {
		ui.Fatal("Failed to get working directory")
	}

	// Detect stack(s)
	ui.Step("Detecting project stack...")
	projects := detect.DetectAll(dir)

	if len(projects) == 0 {
		ui.Fatal("No supported project detected in " + dir)
	}

	var project detect.Project

	if len(projects) == 1 {
		project = projects[0]
		relDir, _ := filepath.Rel(dir, project.Dir)
		if relDir == "." {
			ui.StepDone(fmt.Sprintf("Detected %s (%s)", project.Stack, project.Name))
		} else {
			ui.StepDone(fmt.Sprintf("Detected %s (%s) in %s", project.Stack, project.Name, relDir))
		}
	} else {
		// Monorepo detected
		ui.StepDone(fmt.Sprintf("Detected monorepo with %d packages", len(projects)))

		if runAll {
			return runAllProjects(dir, projects)
		}

		// Check if a specific package was requested via args
		if len(args) > 0 {
			target := args[0]
			found := false
			for _, p := range projects {
				if p.Name == target || filepath.Base(p.Dir) == target {
					project = p
					found = true
					break
				}
			}
			if !found {
				ui.Fatal(fmt.Sprintf("Package '%s' not found in monorepo", target))
			}
		} else {
			// Interactive picker
			selected, err := pickProject(dir, projects)
			if err != nil {
				return err
			}
			project = selected
		}
	}

	return runProject(dir, project)
}

func runProject(rootDir string, project detect.Project) error {
	// Dry run mode
	if dryRun {
		ui.Header("Dry run — would execute:")
		ui.Info(fmt.Sprintf("Stack:    %s", project.Stack))
		ui.Info(fmt.Sprintf("Dir:      %s", project.Dir))
		ui.Info(fmt.Sprintf("Pkg mgr:  %s", project.PackageManager))
		ui.Info(fmt.Sprintf("Run cmd:  %s", project.RunCommand))
		return nil
	}

	if !skipInstall {
		// Install runtime
		ui.Header("Runtime")
		if err := runtime.EnsureRuntime(project.Stack); err != nil {
			ui.StepFail(err.Error())
			return err
		}

		// Install dependencies
		ui.Header("Dependencies")
		if err := deps.Install(project); err != nil {
			ui.StepFail(err.Error())
			return err
		}
	}

	// Setup .env
	ui.Header("Environment")
	if err := env.Setup(project.Dir); err != nil {
		ui.Warn(fmt.Sprintf(".env setup failed: %s", err))
		// Non-fatal, continue
	}

	// Run the project
	ui.Header("Starting")
	return runner.Run(project)
}

func runAllProjects(rootDir string, projects []detect.Project) error {
	// For --all, we run all projects in parallel
	// For simplicity in v1, we use docker compose if available,
	// otherwise run each in a goroutine
	errCh := make(chan error, len(projects))

	for _, p := range projects {
		go func(proj detect.Project) {
			errCh <- runProject(rootDir, proj)
		}(p)
	}

	// Wait for all or first error
	for range projects {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}

func pickProject(rootDir string, projects []detect.Project) (detect.Project, error) {
	var options []huh.Option[int]
	for i, p := range projects {
		relDir, _ := filepath.Rel(rootDir, p.Dir)
		label := fmt.Sprintf("%s (%s)", p.Name, p.Stack)
		if relDir != "." {
			label = fmt.Sprintf("%s — %s (%s)", relDir, p.Name, p.Stack)
		}
		options = append(options, huh.NewOption(label, i))
	}

	var selected int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Which package do you want to run?").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return detect.Project{}, err
	}

	return projects[selected], nil
}
