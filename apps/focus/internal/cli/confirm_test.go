package cli

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyanite/focus/internal/engine"
	"github.com/kyanite/focus/internal/repository"
	"github.com/kyanite/focus/internal/templates"
	"github.com/kyanite/focus/pkg/models"
)

func TestConfirmYesNo(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty input cancels", "", false},
		{"blank line cancels", "\n", false},
		{"whitespace cancels", "   \n", false},
		{"n cancels", "n\n", false},
		{"no cancels", "no\n", false},
		{"other word cancels", "maybe\n", false},
		{"y confirms", "y\n", true},
		{"Y confirms", "Y\n", true},
		{"yes confirms", "yes\n", true},
		{"YES confirms", "YES\n", true},
		{"y without newline confirms", "y", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			got := confirmYesNo(strings.NewReader(tt.input), &out, "Delete this")
			if got != tt.want {
				t.Errorf("confirmYesNo(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if !strings.Contains(out.String(), "(y/N): ") {
				t.Errorf("prompt missing (y/N) marker, got %q", out.String())
			}
		})
	}
}

func TestReadTypedYes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty input cancels", "", false},
		{"blank line cancels", "\n", false},
		{"y alone is not the word yes", "y\n", false},
		{"no cancels", "no\n", false},
		{"yes confirms", "yes\n", true},
		{"YES confirms", "YES\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readTypedYes(strings.NewReader(tt.input)); got != tt.want {
				t.Errorf("readTypedYes(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// newTestEngine returns an engine over a throwaway store holding one task.
func newTestEngine(t *testing.T) (*engine.Engine, models.Task) {
	t.Helper()
	tempDir := t.TempDir()
	repo := repository.NewStoreRepository(filepath.Join(tempDir, "tasks.json"))
	eng := engine.New(repo)

	task, err := eng.AddTask(models.ParsedTask{
		Description: "Delete-me task",
		Priority:    "medium",
	})
	if err != nil {
		t.Fatalf("AddTask() failed: %v", err)
	}
	return eng, task
}

func TestRunDeleteTaskCancelsOnEmptyInput(t *testing.T) {
	eng, task := newTestEngine(t)

	var out strings.Builder
	runDeleteTask(eng, task.ID, strings.NewReader(""), &out)

	if _, err := eng.GetTask(task.ID); err != nil {
		t.Fatalf("task was deleted despite empty confirmation input: %v", err)
	}
	if !strings.Contains(out.String(), "(y/N): ") {
		t.Errorf("expected a y/N prompt before the delete, got %q", out.String())
	}
}

func TestRunDeleteTaskCancelledByNo(t *testing.T) {
	eng, task := newTestEngine(t)

	runDeleteTask(eng, task.ID, strings.NewReader("n\n"), io.Discard)

	if _, err := eng.GetTask(task.ID); err != nil {
		t.Fatalf("task was deleted after answering n: %v", err)
	}
}

func TestRunDeleteTaskConfirmedByYes(t *testing.T) {
	eng, task := newTestEngine(t)

	runDeleteTask(eng, task.ID, strings.NewReader("y\n"), io.Discard)

	if _, err := eng.GetTask(task.ID); err == nil {
		t.Fatal("task still exists after y confirmation")
	}
}

func TestRunTemplateDeleteCancelsOnEmptyInput(t *testing.T) {
	store := templates.NewStore(filepath.Join(t.TempDir(), "templates.json"))
	created, err := store.Add(models.TaskTemplate{
		Name:        "Standup",
		Description: "Daily standup prep",
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	var out strings.Builder
	runTemplateDelete(store, "Standup", strings.NewReader(""), &out)

	if _, err := store.Get(created.ID); err != nil {
		t.Fatalf("template was deleted despite empty confirmation input: %v", err)
	}
	if !strings.Contains(out.String(), "(y/N): ") {
		t.Errorf("expected a y/N prompt before the delete, got %q", out.String())
	}
}

func TestRunTemplateDeleteConfirmedByYes(t *testing.T) {
	store := templates.NewStore(filepath.Join(t.TempDir(), "templates.json"))
	created, err := store.Add(models.TaskTemplate{
		Name:        "Standup",
		Description: "Daily standup prep",
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	runTemplateDelete(store, "Standup", strings.NewReader("y\n"), io.Discard)

	if _, err := store.Get(created.ID); err == nil {
		t.Fatal("template still exists after y confirmation")
	}
}

func TestConfirmAndBulkDeleteCancelsOnEmptyInput(t *testing.T) {
	eng, task := newTestEngine(t)

	_, _, confirmed := confirmAndBulkDelete(eng, []models.Task{task}, strings.NewReader(""))

	if confirmed {
		t.Fatal("bulk delete reported confirmed on empty input")
	}
	if _, err := eng.GetTask(task.ID); err != nil {
		t.Fatalf("task was deleted despite empty confirmation input: %v", err)
	}
}

func TestConfirmAndBulkDeleteRejectsPlainY(t *testing.T) {
	eng, task := newTestEngine(t)

	_, _, confirmed := confirmAndBulkDelete(eng, []models.Task{task}, strings.NewReader("y\n"))

	if confirmed {
		t.Fatal("bulk delete confirmed on 'y'; the gate is the full word yes")
	}
	if _, err := eng.GetTask(task.ID); err != nil {
		t.Fatalf("task was deleted after answering y: %v", err)
	}
}

func TestConfirmAndBulkDeleteConfirmedByTypedYes(t *testing.T) {
	eng, task := newTestEngine(t)

	deleted, failed, confirmed := confirmAndBulkDelete(eng, []models.Task{task}, strings.NewReader("yes\n"))

	if !confirmed {
		t.Fatal("bulk delete not confirmed by typed yes")
	}
	if deleted != 1 || failed != 0 {
		t.Fatalf("deleted=%d failed=%d, want 1/0", deleted, failed)
	}
	if _, err := eng.GetTask(task.ID); err == nil {
		t.Fatal("task still exists after typed yes")
	}
}
