package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyanite/syntax/internal/location"
	"github.com/kyanite/syntax/internal/storage"
	"github.com/kyanite/syntax/internal/utils"
)

func (m Model) viewLocations() string {
	if m.CurrentProject == nil {
		return "No project loaded"
	}

	var b strings.Builder

	// Title bar
	titleBar := m.Styles.StatusBar.Render(fmt.Sprintf(" %s - Locations ", m.CurrentProject.Title))
	b.WriteString(titleBar)
	b.WriteString("\n\n")

	b.WriteString(m.Styles.Heading.Render("🗺️  Locations"))
	b.WriteString("\n\n")

	if len(m.CurrentProject.Locations) == 0 {
		b.WriteString(m.Styles.Text.Render("No locations yet. Press 'n' to create one."))
	} else {
		// Convert map to sorted slice for consistent display
		locations := make([]*location.Location, 0, len(m.CurrentProject.Locations))
		for _, loc := range m.CurrentProject.Locations {
			locations = append(locations, loc)
		}

		for i, loc := range locations {
			prefix := "  "
			if i == m.SelectedIndex {
				prefix = "> "
			}

			style := m.Styles.Text
			if i == m.SelectedIndex {
				style = m.Styles.Accent
			}

			// Display location name and type
			locType := loc.Type
			if locType == "" {
				locType = "unknown"
			}

			line := fmt.Sprintf("%s%s (%s)", prefix, loc.Name, locType)
			b.WriteString(style.Render(line))
			b.WriteString("\n")

			// Show description preview if selected
			if i == m.SelectedIndex && loc.Description != "" {
				preview := loc.Description
				if len(preview) > 80 {
					preview = preview[:77] + "..."
				}
				b.WriteString(m.Styles.Text.Faint(true).Render("    " + preview))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")

	// Show input prompt if in input mode
	if m.InputMode {
		b.WriteString(m.Styles.Accent.Render(fmt.Sprintf("%s: %s█", m.InputPrompt, m.InputValue)))
		b.WriteString("\n")
	} else {
		b.WriteString(m.Styles.Text.Render("n - New Location  |  Enter - View/Edit  |  d - Delete  |  Esc - Back"))
	}

	// Footer
	if m.Message != "" {
		b.WriteString("\n\n")
		b.WriteString(m.Styles.Success.Render(m.Message))
	}

	if m.Error != nil {
		b.WriteString("\n\n")
		b.WriteString(m.Styles.Error.Render(fmt.Sprintf("Error: %v", m.Error)))
	}

	return b.String()
}

func (m Model) handleLocationsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	locCount := len(m.CurrentProject.Locations)

	switch msg.String() {
	case "up", "k":
		if m.SelectedIndex > 0 {
			m.SelectedIndex--
		}
		return m, nil

	case "down", "j":
		if locCount > 0 && m.SelectedIndex < locCount-1 {
			m.SelectedIndex++
		}
		return m, nil

	case "n":
		// Create new location
		m.InputMode = true
		m.InputValue = ""
		m.InputPrompt = "Enter location name"
		m.Message = ""
		return m, nil

	case "enter":
		if m.InputMode {
			// Resolve an armed delete confirmation first: only y/yes
			// deletes, empty input or any other answer cancels.
			if m.PendingDelete != nil {
				pd, ok := m.takeDeleteConfirm()
				if !ok {
					m.Message = "Deletion cancelled"
					return m, nil
				}

				loc := m.CurrentProject.Locations[pd.ID]
				if err := storage.DeleteLocation(m.CurrentProject.Directory, pd.ID); err != nil {
					m.Error = err
				} else {
					delete(m.CurrentProject.Locations, pd.ID)
					m.CurrentProject.TotalLocations--
					if loc != nil {
						m.Message = fmt.Sprintf("Deleted location: %s", loc.Name)
					}

					// Save updated project metadata
					if err := storage.SaveProjectMetadata(m.CurrentProject); err != nil {
						m.Error = fmt.Errorf("failed to save project: %w", err)
					}

					// Adjust selected index
					if m.SelectedIndex >= len(m.CurrentProject.Locations) {
						m.SelectedIndex = len(m.CurrentProject.Locations) - 1
					}
					if m.SelectedIndex < 0 {
						m.SelectedIndex = 0
					}
				}
				return m, nil
			}

			// Finish creating location
			if m.InputValue != "" {
				newLoc := &location.Location{
					Name: m.InputValue,
					Type: "place",
				}
				if err := storage.SaveLocation(m.CurrentProject.Directory, newLoc); err != nil {
					m.Error = err
				} else {
					m.CurrentProject.Locations[newLoc.ID] = newLoc
					m.CurrentProject.TotalLocations++
					m.Message = fmt.Sprintf("Created location: %s", newLoc.Name)

					// Save updated project metadata
					if err := storage.SaveProjectMetadata(m.CurrentProject); err != nil {
						m.Error = fmt.Errorf("failed to save project: %w", err)
					}
				}
			}
			m.InputMode = false
			m.InputValue = ""
			return m, nil
		}

		// Navigate to location editor
		loc := utils.GetLocationAtIndex(m.CurrentProject.Locations, m.SelectedIndex)
		if loc != nil {
			m.CurrentLocation = loc
			m.PreviousScreen = ScreenLocations
			m.CurrentScreen = ScreenLocationEditor
			m.SelectedIndex = 0 // Reset for field navigation
		}
		return m, nil

	case "d":
		if !m.InputMode && locCount > 0 {
			// Arm delete confirmation for the selected location; nothing
			// is deleted until the user types y/yes and presses Enter.
			locID := utils.FindLocationIDAtIndex(m.CurrentProject.Locations, m.SelectedIndex)
			if locID != "" {
				loc := m.CurrentProject.Locations[locID]
				m.armDeleteConfirm(
					PendingDelete{Kind: "location", ID: locID, Name: loc.Name},
					fmt.Sprintf("Delete location '%s'? (y/N)", loc.Name),
				)
			}
		}
		return m, nil

	case "esc":
		if m.InputMode {
			m.InputMode = false
			m.InputValue = ""
			m.InputPrompt = ""
			m.PendingDelete = nil
			m.Message = ""
			return m, nil
		}
		m.SelectedIndex = 0
		m.Message = ""
		m.Error = nil
		m.CurrentScreen = ScreenEditor
		return m, nil

	default:
		if m.InputMode {
			// Handle text input
			switch msg.String() {
			case "backspace":
				if len(m.InputValue) > 0 {
					m.InputValue = m.InputValue[:len(m.InputValue)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.InputValue += msg.String()
				}
			}
		}
	}

	return m, nil
}
