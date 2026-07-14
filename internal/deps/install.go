package deps

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/beargruug/runx/internal/detect"
	"github.com/beargruug/runx/internal/ui"
)

// Install installs dependencies for the given project.
func Install(project detect.Project) error {
	switch project.Stack {
	case detect.StackNode:
		return installNode(project)
	case detect.StackPython:
		return installPython(project)
	case detect.StackRust:
		return installRust(project)
	case detect.StackGo:
		return installGo(project)
	case detect.StackRuby:
		return installRuby(project)
	}
	return nil
}

func installNode(project detect.Project) error {
	pm := string(project.PackageManager)
	return ui.Spinner(fmt.Sprintf("Installing dependencies (%s install)...", pm), func() error {
		cmd := exec.Command(pm, "install")
		cmd.Dir = project.Dir
		cmd.Env = append(os.Environ(), "CI=true") // suppress interactive prompts
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s install failed: %s\n%s", pm, err, string(output))
		}
		return nil
	})
}

func installPython(project detect.Project) error {
	switch project.PackageManager {
	case detect.PMUv:
		return ui.Spinner("Installing dependencies (uv sync)...", func() error {
			return runInDir(project.Dir, "uv", "sync")
		})
	case detect.PMPoetry:
		return ui.Spinner("Installing dependencies (poetry install)...", func() error {
			return runInDir(project.Dir, "poetry", "install")
		})
	case detect.PMPipenv:
		return ui.Spinner("Installing dependencies (pipenv install)...", func() error {
			return runInDir(project.Dir, "pipenv", "install")
		})
	default:
		// pip with requirements.txt
		reqFile := project.Dir + "/requirements.txt"
		if _, err := os.Stat(reqFile); err == nil {
			return ui.Spinner("Installing dependencies (pip install)...", func() error {
				return runInDir(project.Dir, "pip", "install", "-r", "requirements.txt")
			})
		}
	}
	return nil
}

func installRust(project detect.Project) error {
	return ui.Spinner("Building project (cargo build)...", func() error {
		return runInDir(project.Dir, "cargo", "build")
	})
}

func installGo(project detect.Project) error {
	return ui.Spinner("Downloading dependencies (go mod download)...", func() error {
		return runInDir(project.Dir, "go", "mod", "download")
	})
}

func installRuby(project detect.Project) error {
	return ui.Spinner("Installing dependencies (bundle install)...", func() error {
		return runInDir(project.Dir, "bundle", "install")
	})
}

func runInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s\n%s", name, err, string(output))
	}
	return nil
}
