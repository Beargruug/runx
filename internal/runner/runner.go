package runner

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/beargruug/runx/internal/detect"
	"github.com/beargruug/runx/internal/ui"
)

// Run starts the project.
func Run(project detect.Project) error {
	switch project.Stack {
	case detect.StackNode:
		return runNode(project)
	case detect.StackPython:
		return runPython(project)
	case detect.StackRust:
		return runRust(project)
	case detect.StackGo:
		return runGo(project)
	case detect.StackRuby:
		return runRuby(project)
	case detect.StackDocker:
		return runDocker(project)
	case detect.StackMakefile:
		return runMakefile(project)
	}
	return fmt.Errorf("unsupported stack: %s", project.Stack)
}

// runInteractive runs a command with full stdio attached (for dev servers, etc.)
func runInteractive(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward signals to the child process
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()
	defer signal.Stop(sigCh)

	return cmd.Run()
}

func runNode(project detect.Project) error {
	pm := string(project.PackageManager)
	script := project.RunCommand
	if script == "" {
		return fmt.Errorf("no 'dev' or 'start' script found in package.json")
	}

	ui.Header(fmt.Sprintf("Running %s %s", pm, script))
	fmt.Println()

	if pm == "npm" {
		return runInteractive(project.Dir, pm, "run", script)
	}
	// pnpm, yarn, bun support direct script names
	return runInteractive(project.Dir, pm, script)
}

func runPython(project detect.Project) error {
	entry := project.EntryPoint

	switch project.PackageManager {
	case detect.PMUv:
		if entry != "" {
			ui.Header(fmt.Sprintf("Running uv run %s", entry))
			fmt.Println()
			return runInteractive(project.Dir, "uv", "run", entry)
		}
		// Try manage.py for Django
		if entry == "manage.py" {
			ui.Header("Running Django dev server")
			fmt.Println()
			return runInteractive(project.Dir, "uv", "run", "python", "manage.py", "runserver")
		}
	case detect.PMPoetry:
		if entry != "" {
			ui.Header(fmt.Sprintf("Running poetry run python %s", entry))
			fmt.Println()
			return runInteractive(project.Dir, "poetry", "run", "python", entry)
		}
	case detect.PMPipenv:
		if entry != "" {
			ui.Header(fmt.Sprintf("Running pipenv run python %s", entry))
			fmt.Println()
			return runInteractive(project.Dir, "pipenv", "run", "python", entry)
		}
	default:
		if entry == "manage.py" {
			ui.Header("Running Django dev server")
			fmt.Println()
			return runInteractive(project.Dir, "python3", "manage.py", "runserver")
		}
		if entry != "" {
			ui.Header(fmt.Sprintf("Running python3 %s", entry))
			fmt.Println()
			return runInteractive(project.Dir, "python3", entry)
		}
	}

	return fmt.Errorf("no Python entrypoint found (looked for main.py, app.py, manage.py, run.py)")
}

func runRust(project detect.Project) error {
	ui.Header("Running cargo run")
	fmt.Println()
	return runInteractive(project.Dir, "cargo", "run")
}

func runGo(project detect.Project) error {
	target := "."
	if project.EntryPoint != "" && project.EntryPoint != "main.go" {
		target = "./" + project.EntryPoint
	}

	ui.Header(fmt.Sprintf("Running go run %s", target))
	fmt.Println()
	return runInteractive(project.Dir, "go", "run", target)
}

func runRuby(project detect.Project) error {
	if project.RunCommand == "rails server" {
		ui.Header("Running Rails server")
		fmt.Println()
		return runInteractive(project.Dir, "bundle", "exec", "rails", "server")
	}

	// Try common entrypoints
	for _, entry := range []string{"app.rb", "main.rb", "server.rb"} {
		path := project.Dir + "/" + entry
		if _, err := os.Stat(path); err == nil {
			ui.Header(fmt.Sprintf("Running ruby %s", entry))
			fmt.Println()
			return runInteractive(project.Dir, "ruby", entry)
		}
	}

	// Try Rakefile
	if _, err := os.Stat(project.Dir + "/Rakefile"); err == nil {
		ui.Header("Running rake")
		fmt.Println()
		return runInteractive(project.Dir, "bundle", "exec", "rake")
	}

	return fmt.Errorf("no Ruby entrypoint found")
}

func runDocker(project detect.Project) error {
	ui.Header("Running docker compose up")
	fmt.Println()
	return runInteractive(project.Dir, "docker", "compose", "up")
}

func runMakefile(project detect.Project) error {
	ui.Header("Running make")
	fmt.Println()
	return runInteractive(project.Dir, "make")
}
