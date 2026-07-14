package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/beargruug/runx/internal/ui"
)

// Known default values for common env vars.
var knownDefaults = map[string]string{
	"PORT":          "3000",
	"HOST":          "localhost",
	"NODE_ENV":      "development",
	"RAILS_ENV":     "development",
	"FLASK_ENV":     "development",
	"DATABASE_URL":  "postgres://localhost:5432/myapp",
	"REDIS_URL":     "redis://localhost:6379",
	"MONGO_URL":     "mongodb://localhost:27017/myapp",
	"MONGODB_URI":   "mongodb://localhost:27017/myapp",
	"DB_HOST":       "localhost",
	"DB_PORT":       "5432",
	"DB_USER":       "root",
	"DB_PASSWORD":   "",
	"DB_NAME":       "myapp",
	"LOG_LEVEL":     "debug",
	"DEBUG":         "true",
	"SECRET_KEY":    "dev-secret-change-in-production",
	"JWT_SECRET":    "dev-secret-change-in-production",
	"SESSION_SECRET": "dev-secret-change-in-production",
}

// Placeholder values that indicate the user should fill in a real value.
var placeholders = []string{
	"your_", "YOUR_", "xxx", "XXX", "changeme", "CHANGEME",
	"replace", "REPLACE", "todo", "TODO", "fixme", "FIXME",
	"<", ">", "enter_", "ENTER_", "put_", "PUT_",
	"sk-", "pk-", "sk_test", "pk_test",
}

// Setup checks for .env template files and creates .env if needed.
func Setup(dir string) error {
	envPath := filepath.Join(dir, ".env")

	// If .env already exists, skip
	if _, err := os.Stat(envPath); err == nil {
		ui.StepDone(".env already exists")
		return nil
	}

	// Look for template files
	templates := []string{".env.example", ".env.sample", ".env.template", ".env.development"}
	var templatePath string
	for _, t := range templates {
		path := filepath.Join(dir, t)
		if _, err := os.Stat(path); err == nil {
			templatePath = path
			break
		}
	}

	if templatePath == "" {
		return nil // No template found, nothing to do
	}

	ui.Step(fmt.Sprintf("Found %s, creating .env...", filepath.Base(templatePath)))

	// Parse the template
	entries, err := parseEnvFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filepath.Base(templatePath), err)
	}

	// Process entries: fill in defaults or prompt for values
	var outputLines []string
	promptCount := 0

	for _, entry := range entries {
		if entry.isComment {
			outputLines = append(outputLines, entry.raw)
			continue
		}

		value := entry.value

		// Check if value is a placeholder that needs user input
		if isPlaceholder(value) {
			// Check if we have a known default
			if defaultVal, ok := knownDefaults[entry.key]; ok {
				value = defaultVal
				ui.Info(fmt.Sprintf("%s → %s (auto-filled)", entry.key, value))
			} else {
				// Prompt user
				promptCount++
				fmt.Printf("  ? %s", entry.key)
				if value != "" {
					fmt.Printf(" (current: %s)", value)
				}
				fmt.Print(": ")

				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					input := strings.TrimSpace(scanner.Text())
					if input != "" {
						value = input
					}
				}
			}
		} else if value == "" {
			// Empty value — try known defaults
			if defaultVal, ok := knownDefaults[entry.key]; ok {
				value = defaultVal
				ui.Info(fmt.Sprintf("%s → %s (auto-filled)", entry.key, value))
			}
		}

		outputLines = append(outputLines, fmt.Sprintf("%s=%s", entry.key, value))
	}

	// Write .env file
	content := strings.Join(outputLines, "\n") + "\n"
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .env: %w", err)
	}

	ui.StepDone(fmt.Sprintf(".env created (%d variables)", len(entries)-countComments(entries)))
	return nil
}

type envEntry struct {
	key       string
	value     string
	raw       string
	isComment bool
}

func parseEnvFile(path string) ([]envEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []envEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Empty line or comment
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			entries = append(entries, envEntry{raw: line, isComment: true})
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(trimmed, "=", 2)
		key := parts[0]
		value := ""
		if len(parts) > 1 {
			value = parts[1]
			// Remove surrounding quotes
			value = strings.Trim(value, "\"'")
		}

		entries = append(entries, envEntry{key: key, value: value, raw: line})
	}

	return entries, scanner.Err()
}

func isPlaceholder(value string) bool {
	lower := strings.ToLower(value)
	for _, p := range placeholders {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func countComments(entries []envEntry) int {
	n := 0
	for _, e := range entries {
		if e.isComment {
			n++
		}
	}
	return n
}
