package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Stack represents a detected project stack.
type Stack string

const (
	StackNode     Stack = "node"
	StackPython   Stack = "python"
	StackRust     Stack = "rust"
	StackGo       Stack = "go"
	StackRuby     Stack = "ruby"
	StackDocker   Stack = "docker"
	StackMakefile Stack = "makefile"
)

// PackageManager represents a detected package manager.
type PackageManager string

const (
	PMNpm    PackageManager = "npm"
	PMYarn   PackageManager = "yarn"
	PMPnpm   PackageManager = "pnpm"
	PMBun    PackageManager = "bun"
	PMPip    PackageManager = "pip"
	PMPoetry PackageManager = "poetry"
	PMUv     PackageManager = "uv"
	PMPipenv PackageManager = "pipenv"
	PMCargo  PackageManager = "cargo"
	PMGoMod  PackageManager = "gomod"
	PMBundle PackageManager = "bundle"
)

// Project holds the detection results for a project.
type Project struct {
	Dir            string
	Stack          Stack
	PackageManager PackageManager
	Name           string
	RunCommand     string // e.g., "dev", "start" for node scripts
	EntryPoint     string // e.g., "main.py", "main.go"
}

// DetectAll detects the stack(s) in the given directory.
// Returns all detected projects (important for monorepos).
func DetectAll(dir string) []Project {
	var projects []Project

	// Check for monorepo first
	monoProjects := DetectMonorepo(dir)
	if len(monoProjects) > 0 {
		return monoProjects
	}

	// Single project detection
	if p, ok := detectSingle(dir); ok {
		projects = append(projects, p)
	}

	return projects
}

func detectSingle(dir string) (Project, bool) {
	// Order matters — more specific first
	checks := []struct {
		file  string
		stack Stack
		fn    func(string) Project
	}{
		{"package.json", StackNode, detectNode},
		{"Cargo.toml", StackRust, detectRust},
		{"go.mod", StackGo, detectGo},
		{"pyproject.toml", StackPython, detectPython},
		{"requirements.txt", StackPython, detectPythonRequirements},
		{"setup.py", StackPython, detectPythonSetup},
		{"Gemfile", StackRuby, detectRuby},
		{"docker-compose.yml", StackDocker, detectDocker},
		{"docker-compose.yaml", StackDocker, detectDocker},
		{"compose.yml", StackDocker, detectDocker},
		{"compose.yaml", StackDocker, detectDocker},
		{"Makefile", StackMakefile, detectMakefile},
	}

	for _, c := range checks {
		if fileExists(filepath.Join(dir, c.file)) {
			p := c.fn(dir)
			return p, true
		}
	}

	return Project{}, false
}

func detectNode(dir string) Project {
	p := Project{
		Dir:   dir,
		Stack: StackNode,
		Name:  filepath.Base(dir),
	}

	// Detect package manager from lockfile
	switch {
	case fileExists(filepath.Join(dir, "bun.lockb")) || fileExists(filepath.Join(dir, "bun.lock")):
		p.PackageManager = PMBun
	case fileExists(filepath.Join(dir, "pnpm-lock.yaml")):
		p.PackageManager = PMPnpm
	case fileExists(filepath.Join(dir, "yarn.lock")):
		p.PackageManager = PMYarn
	default:
		p.PackageManager = PMNpm
	}

	// Detect run command from scripts
	pkgPath := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err == nil {
		var pkg struct {
			Name    string            `json:"name"`
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			if pkg.Name != "" {
				p.Name = pkg.Name
			}
			// Prefer dev, then start
			if _, ok := pkg.Scripts["dev"]; ok {
				p.RunCommand = "dev"
			} else if _, ok := pkg.Scripts["start"]; ok {
				p.RunCommand = "start"
			}
		}
	}

	return p
}

func detectRust(dir string) Project {
	return Project{
		Dir:            dir,
		Stack:          StackRust,
		PackageManager: PMCargo,
		Name:           filepath.Base(dir),
	}
}

func detectGo(dir string) Project {
	p := Project{
		Dir:            dir,
		Stack:          StackGo,
		PackageManager: PMGoMod,
		Name:           filepath.Base(dir),
	}

	// Try to find main.go
	if fileExists(filepath.Join(dir, "main.go")) {
		p.EntryPoint = "main.go"
	} else if fileExists(filepath.Join(dir, "cmd")) {
		// Check for cmd/ directory pattern
		entries, _ := os.ReadDir(filepath.Join(dir, "cmd"))
		if len(entries) > 0 {
			p.EntryPoint = "cmd/" + entries[0].Name()
		}
	}

	return p
}

func detectPython(dir string) Project {
	p := Project{
		Dir:   dir,
		Stack: StackPython,
		Name:  filepath.Base(dir),
	}

	// Detect package manager
	switch {
	case fileExists(filepath.Join(dir, "uv.lock")):
		p.PackageManager = PMUv
	case fileExists(filepath.Join(dir, "poetry.lock")):
		p.PackageManager = PMPoetry
	case fileExists(filepath.Join(dir, "Pipfile")):
		p.PackageManager = PMPipenv
	default:
		p.PackageManager = PMPip
	}

	// Detect entrypoint
	for _, entry := range []string{"main.py", "app.py", "manage.py", "run.py"} {
		if fileExists(filepath.Join(dir, entry)) {
			p.EntryPoint = entry
			break
		}
	}

	return p
}

func detectPythonRequirements(dir string) Project {
	p := detectPython(dir)
	if p.PackageManager == "" {
		p.PackageManager = PMPip
	}
	return p
}

func detectPythonSetup(dir string) Project {
	p := detectPython(dir)
	if p.PackageManager == "" {
		p.PackageManager = PMPip
	}
	return p
}

func detectRuby(dir string) Project {
	p := Project{
		Dir:            dir,
		Stack:          StackRuby,
		PackageManager: PMBundle,
		Name:           filepath.Base(dir),
	}

	// Detect if it's a Rails project
	if fileExists(filepath.Join(dir, "config", "routes.rb")) {
		p.RunCommand = "rails server"
	}

	return p
}

func detectDocker(dir string) Project {
	return Project{
		Dir:   dir,
		Stack: StackDocker,
		Name:  filepath.Base(dir),
	}
}

func detectMakefile(dir string) Project {
	return Project{
		Dir:   dir,
		Stack: StackMakefile,
		Name:  filepath.Base(dir),
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
