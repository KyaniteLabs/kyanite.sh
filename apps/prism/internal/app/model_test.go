package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyanite/prism/internal/ui"
)

// key builds the tea.KeyMsg a terminal would deliver for a printable rune.
func key(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// navigateScreen runs cmd and reports the screen it navigates to, or -1 when
// the command is not a navigation.
func navigateScreen(t *testing.T, m Model, cmd tea.Cmd) int {
	t.Helper()
	if cmd == nil {
		return -1
	}
	switch msg := cmd().(type) {
	case ui.NavigateMsg:
		return msg.Screen
	default:
		return -1
	}
}

// TestRootForwardsKeysToActiveScreen is the regression gate for the defect
// where the root model swallowed every key, leaving the menu and all screens
// unreachable by keyboard.
func TestRootForwardsKeysToActiveScreen(t *testing.T) {
	// Down must move the menu selection: the menus own cursor, not the root.
	m := NewModel(nil)
	down := tea.KeyMsg{Type: tea.KeyDown}
	next, cmd := m.Update(down)
	if cmd != nil {
		t.Fatalf("down emitted a command; want the key forwarded to the menu")
	}
	model := next.(Model)

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd = model.Update(enter)
	if got := navigateScreen(t, model, cmd); got != 2 {
		t.Fatalf("down+enter navigated to screen %d, want 2 (Generate Palette)", got)
	}
}

// TestMenuShortcutsReachTheMenu pins every documented ready-menu shortcut.
func TestMenuShortcutsReachTheMenu(t *testing.T) {
	cases := []struct {
		key  rune
		want int
	}{
		{'c', 1}, // Color Wheel
		{'g', 2}, // Generate Palette
		{'s', 6}, // Search Colors
		{'l', 3}, // Learn Color Theory
		{'a', 4}, // Check Accessibility (WCAG)
		{'m', 5}, // Manage Palettes
	}

	for _, tc := range cases {
		m := NewModel(nil)
		next, cmd := m.Update(key(tc.key))
		if got := navigateScreen(t, next.(Model), cmd); got != tc.want {
			t.Errorf("key %q navigated to screen %d, want %d", tc.key, got, tc.want)
		}
	}
}

// TestGlobalShortcutsStayGlobal makes sure forwarding did not leak global keys
// into the active screen.
func TestGlobalShortcutsStayGlobal(t *testing.T) {
	m := NewModel(nil)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	if cmd == nil {
		t.Fatal("ctrl+q no longer quits")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("ctrl+q no longer returns tea.Quit")
	}

	// ctrl+h opens the overlay and the overlay swallows screen keys.
	next, _ = next.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	model := next.(Model)
	if !model.showHelp {
		t.Fatal("ctrl+h did not open help")
	}
	_, cmd = model.Update(key('g'))
	if cmd != nil {
		t.Fatal("a menu shortcut fired while the help overlay was open")
	}
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(Model).showHelp {
		t.Fatal("esc did not close help")
	}
}
