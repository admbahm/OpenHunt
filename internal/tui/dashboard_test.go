package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testModel() Model {
	return NewModel(
		[]string{"All", "Engineering"},
		[]string{"All", "Ireland"},
		[]string{"All", "Remote"},
	)
}

func updateModel(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := model.Update(msg)
	result, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", updated)
	}
	return result
}

func TestModelCyclesFocusedPanel(t *testing.T) {
	model := testModel()
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	if model.focused != 1 {
		t.Fatalf("focused panel = %d, want 1", model.focused)
	}

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	if model.focused != 0 {
		t.Fatalf("focused panel = %d, want wrapped value 0", model.focused)
	}
}

func TestModelSubmitsSelections(t *testing.T) {
	model := testModel()
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if !model.Submitted {
		t.Fatal("selection was not submitted")
	}
	if model.SelectedCat != "Engineering" ||
		model.SelectedCountry != "Ireland" ||
		model.SelectedLoc != "Remote" {
		t.Fatalf(
			"selection = %q, %q, %q",
			model.SelectedCat,
			model.SelectedCountry,
			model.SelectedLoc,
		)
	}
}

func TestModelRendersPolishedSelector(t *testing.T) {
	model := testModel()
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 30})

	view := model.View()
	for _, expected := range []string{
		"OpenHunt",
		`/ _ \ _ __`,
		"Functional Category",
		"Geographic Location",
		"Selection",
		"start crawl",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("view does not contain %q", expected)
		}
	}
}
