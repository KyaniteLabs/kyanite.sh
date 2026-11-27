package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// commandTimeoutEnv overrides defaultCommandTimeout with a Go duration string,
// for example FOCUS_COMMAND_TIMEOUT=30s.
const commandTimeoutEnv = "FOCUS_COMMAND_TIMEOUT"

// defaultCommandTimeout bounds a single non-interactive CLI command. AI calls
// carry their own 30s bound (aiCallTimeout); this guard catches stalls outside
// them, such as a mutex deadlock, that would otherwise hang forever.
const defaultCommandTimeout = 2 * time.Minute

// interactiveCommands may legitimately wait for a human for an unbounded time
// (TUI dashboards, form wizards, gum prompts, editor sessions and confirmation
// prompts). They are deliberately exempt from the wall-clock bound.
var interactiveCommands = map[string]bool{
	"focus":                    true, // TUI dashboard
	"focus bulk delete":        true, // typed "yes" confirmation
	"focus chat":               true, // stdin chat loop
	"focus config":             true, // gum configuration prompts
	"focus edit":               true, // huh edit form
	"focus edit-wizard":        true, // huh edit form
	"focus enhanced-config":    true, // huh configuration form
	"focus filter":             true, // gum task filter
	"focus interactive":        true, // gum task creation
	"focus journal new":        true, // line-by-line entry input
	"focus notes":              true, // opens $EDITOR
	"focus remove":             true, // y/N confirmation
	"focus simple-interactive": true, // stdin task creation
	"focus unified":            true, // TUI dashboard
	"focus wizard":             true, // huh task form
}

// executeCommand runs the CLI behind a wall-clock guard so that a stalled
// command reports an error and exits non-zero instead of blocking forever.
func executeCommand(root *cobra.Command) error {
	target, _, err := root.Find(os.Args[1:])
	if err != nil || target == nil || interactiveCommands[target.CommandPath()] {
		// Find errors are cobra's to report; interactive commands have no bound.
		return root.Execute()
	}
	return executeWithTimeout(root.Execute, commandTimeout(), target.CommandPath())
}

// executeWithTimeout runs run and reports a stall if it has not returned
// within timeout. It is separated from executeCommand so the bound can be
// tested without running the real command tree.
func executeWithTimeout(run func() error, timeout time.Duration, name string) error {
	done := make(chan error, 1)
	go func() { done <- run() }()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		return err
	case <-timer.C:
		return fmt.Errorf("command %q did not finish within %s and appears stalled; exiting (set %s to change the bound)",
			name, timeout, commandTimeoutEnv)
	}
}

// commandTimeout returns the bound for one command, honouring
// FOCUS_COMMAND_TIMEOUT when it holds a positive duration.
func commandTimeout() time.Duration {
	if raw := os.Getenv(commandTimeoutEnv); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return defaultCommandTimeout
}
