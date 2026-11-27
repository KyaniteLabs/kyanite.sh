package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// caseTimeout bounds one CLI invocation. A command that hangs fails the test
// instead of stalling the suite; every documented command finishes well inside
// it (the slowest, "inspire", waits on the AI fallback for a few seconds).
const caseTimeout = 20 * time.Second

type storedTask struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// TestCLICommandsTerminate is the CLI-level smoke test for kanban t_8b25250e.
// It builds the real focus binary, drives the empty-state instruction the UI
// advertises, and executes every documented command with a bounded timeout so
// a deadlock (such as the DI container mutex re-entry) fails the suite.
func TestCLICommandsTerminate(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the focus binary")
	}

	bin, home := buildFocus(t)
	env := smokeEnv(home)

	// 1. The empty state must advertise the documented first command.
	empty := runCLI(t, bin, home, env, "list")
	if !strings.Contains(empty, `focus add "task description"`) {
		t.Fatalf("empty-state list output does not advertise `focus add \"task description\"`:\n%s", empty)
	}

	// 2. That exact instruction must create and persist a task.
	runCLI(t, bin, home, env, "add", "task description")
	tasks := readStore(t, home)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 persisted task after `focus add`, got %d", len(tasks))
	}
	if tasks[0].Description != "task description" {
		t.Fatalf("persisted description = %q, want %q", tasks[0].Description, "task description")
	}

	// 3. The engine path that used to deadlock must also complete: done <id>.
	runCLI(t, bin, home, env, "done", tasks[0].ID)
	tasks = readStore(t, home)
	if tasks[0].Status != "completed" {
		t.Fatalf("status after `focus done` = %q, want completed", tasks[0].Status)
	}

	// 4. Every documented top-level command runs and exits 0 within the bound.
	cases := []struct {
		name string
		args []string
	}{
		{"add", []string{"add", "smoke probe task"}},
		{"bulk", []string{"bulk"}},
		{"calendar", []string{"calendar"}},
		{"chat", []string{"chat"}},
		{"completion", []string{"completion", "bash"}},
		{"config", []string{"config", "list"}},
		{"done", []string{"done", tasks[0].ID}},
		{"edit", []string{"edit", tasks[0].ID}},
		{"edit-wizard", []string{"edit-wizard", tasks[0].ID}},
		{"enhanced-config", []string{"enhanced-config"}},
		{"export", []string{"export"}},
		{"filter", []string{"filter"}},
		{"help", []string{"help"}},
		{"inspire", []string{"inspire"}},
		{"interactive", []string{"interactive"}},
		{"journal", []string{"journal"}},
		{"list", []string{"list"}},
		{"mcp-server", []string{"mcp-server"}},
		{"notes", []string{"notes", tasks[0].ID}},
		{"recur", []string{"recur", "daily standup", "--pattern=daily"}},
		{"remove", []string{"remove", tasks[0].ID}},
		{"search", []string{"search", "smoke"}},
		{"simple-interactive", []string{"simple-interactive"}},
		{"subtask", []string{"subtask", tasks[0].ID, "smoke subtask"}},
		{"template", []string{"template", "list"}},
		{"test-gum", []string{"test-gum"}},
		{"theme", []string{"theme", "amber-night"}},
		{"unified", []string{"unified"}},
		{"wizard", []string{"wizard"}},
	}

	covered := make(map[string]bool, len(cases))
	for _, tc := range cases {
		covered[tc.args[0]] = true
		runCLI(t, bin, home, env, tc.args...)
	}

	for _, name := range documentedCommands(t, bin, home, env) {
		if !covered[name] {
			t.Errorf("documented command %q has no smoke case; add one so a hang in it fails the suite", name)
		}
	}
}

// buildFocus builds the real focus binary into a temp dir and returns its path
// plus an isolated HOME for the run.
func buildFocus(t *testing.T) (bin, home string) {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the smoke test source file")
	}
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(file)))

	outDir := t.TempDir()
	bin = filepath.Join(outDir, "focus")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/focus")
	cmd.Dir = moduleRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/focus failed: %v\n%s", err, out)
	}
	return bin, t.TempDir()
}

// smokeEnv isolates the run: private HOME, no real AI backend, no editor, no
// TTY effects, and a guard bound far above the per-case timeout.
func smokeEnv(home string) []string {
	return append(os.Environ(),
		"HOME="+home,
		"KYANITE_BRAIN_OLLAMA_URL=http://127.0.0.1:1",
		"EDITOR=true",
		"TERM=dumb",
		"NO_COLOR=1",
		"FOCUS_COMMAND_TIMEOUT=2m",
	)
}

// runCLI executes one command with a bounded timeout and fails the test if the
// command hangs or exits non-zero.
func runCLI(t *testing.T, bin, home string, env []string, args ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), caseTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = env
	cmd.Stdin = strings.NewReader("")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("`focus %s` did not finish within %s: it hangs. Output:\n%s",
			strings.Join(args, " "), caseTimeout, tail(out.String()))
	}

	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		t.Fatalf("`focus %s` exited %d, want 0. Output:\n%s",
			strings.Join(args, " "), exitErr.ExitCode(), tail(out.String()))
	default:
		t.Fatalf("`focus %s` could not run: %v", strings.Join(args, " "), err)
	}
	return out.String()
}

// documentedCommands returns the top-level commands that `focus --help` lists,
// so a new command cannot be added without a bounded smoke case.
func documentedCommands(t *testing.T, bin, home string, env []string) []string {
	t.Helper()

	help := runCLI(t, bin, home, env, "--help")
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	help = ansi.ReplaceAllString(help, "")

	var names []string
	inBlock := false
	for _, line := range strings.Split(help, "\n") {
		if strings.Contains(line, "Available Commands:") {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			names = append(names, fields[0])
		}
	}
	if len(names) == 0 {
		t.Fatalf("could not parse the documented command list from help:\n%s", help)
	}
	return names
}

// readStore loads the isolated HOME's task store.
func readStore(t *testing.T, home string) []storedTask {
	t.Helper()

	path := filepath.Join(home, ".focus", "tasks.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading task store %s: %v", path, err)
	}
	var tasks []storedTask
	if err := json.Unmarshal(raw, &tasks); err != nil {
		t.Fatalf("parsing task store %s: %v", path, err)
	}
	return tasks
}

func tail(s string) string {
	const max = 2000
	if len(s) <= max {
		return s
	}
	return fmt.Sprintf("...(%d bytes omitted)...\n%s", len(s)-max, s[len(s)-max:])
}
