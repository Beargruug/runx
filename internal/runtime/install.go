package runtime

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/beargruug/runx/internal/detect"
	"github.com/beargruug/runx/internal/ui"
)

// EnsureRuntime checks if the required runtime is installed, and installs it if not.
func EnsureRuntime(stack detect.Stack) error {
	switch stack {
	case detect.StackNode:
		return ensureNode()
	case detect.StackPython:
		return ensurePython()
	case detect.StackRust:
		return ensureRust()
	case detect.StackGo:
		return ensureGo()
	case detect.StackRuby:
		return ensureRuby()
	case detect.StackDocker:
		return ensureDocker()
	case detect.StackMakefile:
		return ensureMake()
	}
	return nil
}

func ensureMise() error {
	if commandExists("mise") {
		return nil
	}

	ui.Step("Installing mise (runtime manager)...")

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" && commandExists("brew") {
		cmd = exec.Command("brew", "install", "mise")
	} else {
		// Use the official installer
		cmd = exec.Command("sh", "-c", "curl -fsSL https://mise.jdx.dev/install.sh | sh")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install mise: %s\n%s", err, string(output))
	}

	// Activate mise for current shell session
	misePath := findMiseBinary()
	if misePath == "" {
		return fmt.Errorf("mise installed but binary not found in PATH")
	}

	return nil
}

func findMiseBinary() string {
	// Check common locations
	locations := []string{
		"mise",
		"~/.local/bin/mise",
		"/usr/local/bin/mise",
	}
	for _, loc := range locations {
		if p, err := exec.LookPath(expandHome(loc)); err == nil {
			return p
		}
	}
	return ""
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := exec.Command("sh", "-c", "echo $HOME").Output()
		return strings.TrimSpace(string(home)) + path[1:]
	}
	return path
}

func ensureNode() error {
	if commandExists("node") {
		version, _ := getCommandOutput("node", "--version")
		ui.StepDone(fmt.Sprintf("Node.js %s found", strings.TrimSpace(version)))
		return nil
	}

	if err := ensureMise(); err != nil {
		return err
	}

	return ui.Spinner("Installing Node.js via mise...", func() error {
		return runCommand("mise", "install", "node@lts", "-y")
	})
}

func ensurePython() error {
	if commandExists("python3") || commandExists("python") {
		cmd := "python3"
		if !commandExists("python3") {
			cmd = "python"
		}
		version, _ := getCommandOutput(cmd, "--version")
		ui.StepDone(fmt.Sprintf("%s found", strings.TrimSpace(version)))
		return nil
	}

	if err := ensureMise(); err != nil {
		return err
	}

	return ui.Spinner("Installing Python via mise...", func() error {
		return runCommand("mise", "install", "python@latest", "-y")
	})
}

func ensureRust() error {
	if commandExists("cargo") {
		version, _ := getCommandOutput("rustc", "--version")
		ui.StepDone(fmt.Sprintf("%s found", strings.TrimSpace(version)))
		return nil
	}

	return ui.Spinner("Installing Rust via rustup...", func() error {
		return runCommand("sh", "-c", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y")
	})
}

func ensureGo() error {
	if commandExists("go") {
		version, _ := getCommandOutput("go", "version")
		ui.StepDone(fmt.Sprintf("%s found", strings.TrimSpace(version)))
		return nil
	}

	if err := ensureMise(); err != nil {
		return err
	}

	return ui.Spinner("Installing Go via mise...", func() error {
		return runCommand("mise", "install", "go@latest", "-y")
	})
}

func ensureRuby() error {
	if commandExists("ruby") {
		version, _ := getCommandOutput("ruby", "--version")
		ui.StepDone(fmt.Sprintf("Ruby %s found", strings.TrimSpace(version)))
		return nil
	}

	if err := ensureMise(); err != nil {
		return err
	}

	return ui.Spinner("Installing Ruby via mise...", func() error {
		return runCommand("mise", "install", "ruby@latest", "-y")
	})
}

func ensureDocker() error {
	if commandExists("docker") {
		ui.StepDone("Docker found")
		return nil
	}
	return fmt.Errorf("Docker is not installed. Please install Docker Desktop from https://docker.com/get-started")
}

func ensureMake() error {
	if commandExists("make") {
		ui.StepDone("make found")
		return nil
	}
	return fmt.Errorf("make is not installed. Install it via your system package manager (e.g., xcode-select --install on macOS)")
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func getCommandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}
