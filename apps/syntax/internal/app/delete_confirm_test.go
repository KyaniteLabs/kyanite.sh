package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyanite/syntax/internal/backup"
	"github.com/kyanite/syntax/internal/location"
	"github.com/kyanite/syntax/internal/scene"
	"github.com/kyanite/syntax/internal/storage"
	"github.com/kyanite/syntax/internal/story"
)

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

var enterKey = tea.KeyMsg{Type: tea.KeyEnter}

// scenesTestModel returns a model on the scenes screen with one saved scene.
func scenesTestModel(t *testing.T) (Model, *scene.Scene, string) {
	t.Helper()
	dir := t.TempDir()
	// SaveScene writes into <dir>/scenes/, which a real project creates up
	// front; create the same layout here.
	if err := os.MkdirAll(filepath.Join(dir, "scenes"), 0o755); err != nil {
		t.Fatalf("mkdir scenes/ failed: %v", err)
	}
	sc := &scene.Scene{Chapter: 1, SceneNumber: 1, Name: "Opening", Status: "draft"}
	if err := storage.SaveScene(dir, sc); err != nil {
		t.Fatalf("SaveScene() failed: %v", err)
	}

	m := NewModel()
	m.CurrentScreen = ScreenScenes
	m.SelectedIndex = 0
	m.CurrentProject = &story.Project{
		Title:       "Test Novel",
		Directory:   dir,
		Scenes:      map[string]*scene.Scene{sc.ID: sc},
		TotalScenes: 1,
	}
	return m, sc, dir
}

func TestScenesDeleteAsksBeforeDeleting(t *testing.T) {
	m, sc, _ := scenesTestModel(t)

	res, _ := m.handleScenesKeys(runeKey("d"))
	m2 := res.(Model)

	if m2.PendingDelete == nil || m2.PendingDelete.Kind != "scene" || m2.PendingDelete.ID != sc.ID {
		t.Fatalf("pressing d must arm a scene delete confirmation, got %+v", m2.PendingDelete)
	}
	if !m2.InputMode {
		t.Fatal("pressing d must enter confirmation input mode")
	}
	if _, still := m2.CurrentProject.Scenes[sc.ID]; !still {
		t.Fatal("scene was deleted before any confirmation")
	}
	if !strings.Contains(m2.viewScenes(), "(y/N)") {
		t.Fatal("scenes view must show the y/N confirmation prompt")
	}
}

func TestScenesDeleteCancelsOnEmptyInput(t *testing.T) {
	m, sc, _ := scenesTestModel(t)

	res, _ := m.handleScenesKeys(runeKey("d"))
	res, _ = res.(Model).handleScenesKeys(enterKey)
	m3 := res.(Model)

	if _, still := m3.CurrentProject.Scenes[sc.ID]; !still {
		t.Fatal("scene was deleted on empty confirmation input")
	}
	if m3.PendingDelete != nil || m3.InputMode {
		t.Fatal("confirmation state was not cleared after empty input")
	}
	if m3.Message != "Deletion cancelled" {
		t.Fatalf("expected a cancellation message, got %q", m3.Message)
	}
}

func TestScenesDeleteCancelsOnEsc(t *testing.T) {
	m, sc, _ := scenesTestModel(t)

	res, _ := m.handleScenesKeys(runeKey("d"))
	res, _ = res.(Model).handleScenesKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m3 := res.(Model)

	if _, still := m3.CurrentProject.Scenes[sc.ID]; !still {
		t.Fatal("scene was deleted after escaping the confirmation")
	}
	if m3.PendingDelete != nil || m3.InputMode {
		t.Fatal("confirmation state was not cleared after esc")
	}
}

func TestScenesDeleteConfirmedByYes(t *testing.T) {
	m, sc, dir := scenesTestModel(t)

	res, _ := m.handleScenesKeys(runeKey("d"))
	res, _ = res.(Model).handleScenesKeys(runeKey("y"))
	res, _ = res.(Model).handleScenesKeys(enterKey)
	m3 := res.(Model)

	if _, still := m3.CurrentProject.Scenes[sc.ID]; still {
		t.Fatal("scene still present after y confirmation")
	}
	if m3.CurrentProject.TotalScenes != 0 {
		t.Fatalf("TotalScenes = %d, want 0", m3.CurrentProject.TotalScenes)
	}
	if _, err := storage.LoadScene(dir, sc.ID); err == nil {
		t.Fatal("scene file still on disk after confirmed delete")
	}
}

func TestScenesDeleteRejectedByPlainNo(t *testing.T) {
	m, sc, _ := scenesTestModel(t)

	res, _ := m.handleScenesKeys(runeKey("d"))
	res, _ = res.(Model).handleScenesKeys(runeKey("n"))
	res, _ = res.(Model).handleScenesKeys(enterKey)
	m3 := res.(Model)

	if _, still := m3.CurrentProject.Scenes[sc.ID]; !still {
		t.Fatal("scene was deleted after answering n")
	}
}

// locationsTestModel returns a model on the locations screen with one saved
// location.
func locationsTestModel(t *testing.T) (Model, *location.Location, string) {
	t.Helper()
	dir := t.TempDir()
	// SaveLocation writes into <dir>/locations/; create the layout a real
	// project would have.
	if err := os.MkdirAll(filepath.Join(dir, "locations"), 0o755); err != nil {
		t.Fatalf("mkdir locations/ failed: %v", err)
	}
	loc := &location.Location{Name: "The Tavern", Type: "place"}
	if err := storage.SaveLocation(dir, loc); err != nil {
		t.Fatalf("SaveLocation() failed: %v", err)
	}

	m := NewModel()
	m.CurrentScreen = ScreenLocations
	m.SelectedIndex = 0
	m.CurrentProject = &story.Project{
		Title:          "Test Novel",
		Directory:      dir,
		Locations:      map[string]*location.Location{loc.ID: loc},
		TotalLocations: 1,
	}
	return m, loc, dir
}

func TestLocationsDeleteCancelsOnEmptyInput(t *testing.T) {
	m, loc, _ := locationsTestModel(t)

	res, _ := m.handleLocationsKeys(runeKey("d"))
	m2 := res.(Model)

	if m2.PendingDelete == nil || m2.PendingDelete.Kind != "location" {
		t.Fatalf("pressing d must arm a location delete confirmation, got %+v", m2.PendingDelete)
	}
	if _, still := m2.CurrentProject.Locations[loc.ID]; !still {
		t.Fatal("location was deleted before any confirmation")
	}
	if !strings.Contains(m2.viewLocations(), "(y/N)") {
		t.Fatal("locations view must show the y/N confirmation prompt")
	}

	res, _ = m2.handleLocationsKeys(enterKey)
	m3 := res.(Model)

	if _, still := m3.CurrentProject.Locations[loc.ID]; !still {
		t.Fatal("location was deleted on empty confirmation input")
	}
	if m3.PendingDelete != nil || m3.InputMode {
		t.Fatal("confirmation state was not cleared after empty input")
	}
}

func TestLocationsDeleteConfirmedByYes(t *testing.T) {
	m, loc, _ := locationsTestModel(t)

	res, _ := m.handleLocationsKeys(runeKey("d"))
	res, _ = res.(Model).handleLocationsKeys(runeKey("y"))
	res, _ = res.(Model).handleLocationsKeys(enterKey)
	m3 := res.(Model)

	if _, still := m3.CurrentProject.Locations[loc.ID]; still {
		t.Fatal("location still present after y confirmation")
	}
	if m3.CurrentProject.TotalLocations != 0 {
		t.Fatalf("TotalLocations = %d, want 0", m3.CurrentProject.TotalLocations)
	}
}

// backupsTestModel returns a model on the backups screen with one backup.
func backupsTestModel(t *testing.T) (Model, string) {
	t.Helper()
	dir := t.TempDir()
	if err := backup.CreateBackup(dir); err != nil {
		t.Fatalf("CreateBackup() failed: %v", err)
	}

	m := NewModel()
	m.CurrentScreen = ScreenBackups
	m.SelectedIndex = 0
	m.CurrentProject = &story.Project{Title: "Test Novel", Directory: dir}
	return m, dir
}

func TestBackupsDeleteCancelsOnEmptyInput(t *testing.T) {
	m, dir := backupsTestModel(t)

	res, _ := m.handleBackupsKeys(runeKey("d"))
	m2 := res.(Model)

	if m2.PendingDelete == nil || m2.PendingDelete.Kind != "backup" {
		t.Fatalf("pressing d must arm a backup delete confirmation, got %+v", m2.PendingDelete)
	}
	if !strings.Contains(m2.viewBackups(), "(y/N)") {
		t.Fatal("backups view must show the y/N confirmation prompt")
	}

	res, _ = m2.handleBackupsKeys(enterKey)
	m3 := res.(Model)

	backups, err := backup.ListBackups(dir)
	if err != nil {
		t.Fatalf("ListBackups() failed: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup was deleted on empty confirmation input; %d remain", len(backups))
	}
	if m3.PendingDelete != nil || m3.InputMode {
		t.Fatal("confirmation state was not cleared after empty input")
	}
}

func TestBackupsDeleteConfirmedByYes(t *testing.T) {
	m, dir := backupsTestModel(t)

	res, _ := m.handleBackupsKeys(runeKey("d"))
	res, _ = res.(Model).handleBackupsKeys(runeKey("y"))
	res, _ = res.(Model).handleBackupsKeys(enterKey)

	backups, err := backup.ListBackups(dir)
	if err != nil {
		t.Fatalf("ListBackups() failed: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("backup still present after y confirmation; %d remain", len(backups))
	}
}
