package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	brandStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
)

func Banner() {
	fmt.Println(brandStyle.Render("  runx") + mutedStyle.Render(" — universal project runner"))
	fmt.Println()
}

func Step(msg string) {
	fmt.Printf("  %s %s\n", infoStyle.Render("→"), msg)
}

func StepDone(msg string) {
	fmt.Printf("  %s %s\n", successStyle.Render("✓"), msg)
}

func StepFail(msg string) {
	fmt.Printf("  %s %s\n", errorStyle.Render("✗"), msg)
}

func Warn(msg string) {
	fmt.Printf("  %s %s\n", warnStyle.Render("!"), msg)
}

func Info(msg string) {
	fmt.Printf("  %s %s\n", mutedStyle.Render("·"), msg)
}

func Header(msg string) {
	fmt.Printf("\n  %s\n", headerStyle.Render(msg))
}

func Fatal(msg string) {
	StepFail(msg)
	os.Exit(1)
}

// Spinner runs a function with a spinner animation.
func Spinner(msg string, fn func() error) error {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan error, 1)

	go func() {
		done <- fn()
	}()

	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			// Clear the spinner line
			fmt.Printf("\r%s\r", strings.Repeat(" ", len(msg)+10))
			if err != nil {
				StepFail(msg)
			} else {
				StepDone(msg)
			}
			return err
		case <-ticker.C:
			frame := infoStyle.Render(frames[i%len(frames)])
			fmt.Printf("\r  %s %s", frame, msg)
			i++
		}
	}
}
