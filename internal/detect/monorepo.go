package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MonorepoType represents the kind of monorepo detected.
type MonorepoType string

const (
	MonoNpm        MonorepoType = "npm-workspaces"
	MonoPnpm       MonorepoType = "pnpm-workspaces"
	MonoYarn       MonorepoType = "yarn-workspaces"
	MonoNx         MonorepoType = "nx"
	MonoTurborepo  MonorepoType = "turborepo"
	MonoRustCargo  MonorepoType = "cargo-workspace"
	MonoGoWork     MonorepoType = "go-workspace"
)

// DetectMonorepo checks if the directory is a monorepo and returns all sub-projects.
func DetectMonorepo(dir string) []Project {
	// Check for Go workspace
	if fileExists(filepath.Join(dir, "go.work")) {
		return detectGoWorkspace(dir)
	}

	// Check for Cargo workspace
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		if projects := detectCargoWorkspace(dir); len(projects) > 0 {
			return projects
		}
	}

	// Check for pnpm workspaces
	if fileExists(filepath.Join(dir, "pnpm-workspace.yaml")) {
		return detectPnpmWorkspace(dir)
	}

	// Check for package.json workspaces (npm/yarn)
	if fileExists(filepath.Join(dir, "package.json")) {
		if projects := detectJSWorkspace(dir); len(projects) > 0 {
			return projects
		}
	}

	return nil
}

func detectPnpmWorkspace(dir string) []Project {
	data, err := os.ReadFile(filepath.Join(dir, "pnpm-workspace.yaml"))
	if err != nil {
		return nil
	}

	var ws struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil
	}

	return resolveWorkspaceGlobs(dir, ws.Packages, PMPnpm)
}

func detectJSWorkspace(dir string) []Project {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}

	var pkg struct {
		Workspaces interface{} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Workspaces == nil {
		return nil
	}

	var patterns []string
	switch w := pkg.Workspaces.(type) {
	case []interface{}:
		for _, v := range w {
			if s, ok := v.(string); ok {
				patterns = append(patterns, s)
			}
		}
	case map[string]interface{}:
		// Yarn workspaces with {packages: [...]}
		if pkgs, ok := w["packages"].([]interface{}); ok {
			for _, v := range pkgs {
				if s, ok := v.(string); ok {
					patterns = append(patterns, s)
				}
			}
		}
	}

	if len(patterns) == 0 {
		return nil
	}

	// Detect PM
	pm := PMNpm
	if fileExists(filepath.Join(dir, "yarn.lock")) {
		pm = PMYarn
	} else if fileExists(filepath.Join(dir, "bun.lockb")) || fileExists(filepath.Join(dir, "bun.lock")) {
		pm = PMBun
	}

	return resolveWorkspaceGlobs(dir, patterns, pm)
}

func resolveWorkspaceGlobs(dir string, patterns []string, pm PackageManager) []Project {
	var projects []Project

	for _, pattern := range patterns {
		// Resolve glob pattern
		globPattern := filepath.Join(dir, pattern)
		// If pattern doesn't end with *, add package.json
		if !strings.HasSuffix(globPattern, "*") {
			// This is a direct path
			if p, ok := detectSingle(globPattern); ok {
				p.PackageManager = pm
				projects = append(projects, p)
			}
			continue
		}

		matches, err := filepath.Glob(globPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || !info.IsDir() {
				continue
			}
			if p, ok := detectSingle(match); ok {
				p.PackageManager = pm
				projects = append(projects, p)
			}
		}
	}

	return projects
}

func detectCargoWorkspace(dir string) []Project {
	data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		return nil
	}

	// Simple check for [workspace] section
	content := string(data)
	if !strings.Contains(content, "[workspace]") {
		return nil
	}

	// Parse members from workspace
	var projects []Project
	// Look for members = ["crate1", "crate2"]
	lines := strings.Split(content, "\n")
	inWorkspace := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "[workspace]" {
			inWorkspace = true
			continue
		}
		if strings.HasPrefix(line, "[") && line != "[workspace]" {
			inWorkspace = false
			continue
		}
		if inWorkspace && strings.HasPrefix(line, "members") {
			// Extract members array
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			membersStr := strings.TrimSpace(parts[1])
			membersStr = strings.Trim(membersStr, "[]")
			members := strings.Split(membersStr, ",")
			for _, m := range members {
				m = strings.TrimSpace(m)
				m = strings.Trim(m, "\"' ")
				if m == "" {
					continue
				}
				memberDir := filepath.Join(dir, m)
				if fileExists(filepath.Join(memberDir, "Cargo.toml")) {
					projects = append(projects, Project{
						Dir:            memberDir,
						Stack:          StackRust,
						PackageManager: PMCargo,
						Name:           filepath.Base(m),
					})
				}
			}
		}
	}

	return projects
}

func detectGoWorkspace(dir string) []Project {
	data, err := os.ReadFile(filepath.Join(dir, "go.work"))
	if err != nil {
		return nil
	}

	var projects []Project
	lines := strings.Split(string(data), "\n")
	inUse := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "use (" {
			inUse = true
			continue
		}
		if line == ")" {
			inUse = false
			continue
		}
		if inUse && line != "" {
			modDir := filepath.Join(dir, line)
			if p, ok := detectSingle(modDir); ok {
				projects = append(projects, p)
			}
		}
		// Single-line use directive
		if strings.HasPrefix(line, "use ") && !strings.Contains(line, "(") {
			modDir := filepath.Join(dir, strings.TrimPrefix(line, "use "))
			if p, ok := detectSingle(modDir); ok {
				projects = append(projects, p)
			}
		}
	}

	return projects
}
