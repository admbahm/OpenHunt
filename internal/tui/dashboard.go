package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle          = lipgloss.NewStyle().Margin(1, 2)
	focusedStyle      = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205"))
	unfocusedStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

type item string

func (i item) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := func(s ...string) string {
		return lipgloss.NewStyle().PaddingLeft(2).Render(s...)
	}
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type Model struct {
	categories  list.Model
	locations   list.Model
	focused     int // 0 for categories, 1 for locations
	SelectedCat string
	SelectedLoc string
	Quitting    bool
	Submitted   bool
}

func NewModel(cats, locs []string) Model {
	catItems := make([]list.Item, len(cats))
	for i, c := range cats {
		catItems[i] = item(c)
	}

	locItems := make([]list.Item, len(locs))
	for i, l := range locs {
		locItems[i] = item(l)
	}

	cList := list.New(catItems, itemDelegate{}, 20, 10)
	cList.Title = "Functional Category"
	cList.SetShowHelp(false)
	cList.SetShowStatusBar(false)
	cList.SetFilteringEnabled(false)

	lList := list.New(locItems, itemDelegate{}, 20, 10)
	lList.Title = "Geographic Location"
	lList.SetShowHelp(false)
	lList.SetShowStatusBar(false)
	lList.SetFilteringEnabled(false)

	return Model{
		categories: cList,
		locations:  lList,
		focused:    0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		case "tab":
			m.focused = (m.focused + 1) % 2
			return m, nil
		case "enter":
			m.SelectedCat = string(m.categories.SelectedItem().(item))
			m.SelectedLoc = string(m.locations.SelectedItem().(item))
			m.Submitted = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.categories.SetSize(msg.Width/2-h, msg.Height-v)
		m.locations.SetSize(msg.Width/2-h, msg.Height-v)
	}

	var cmd tea.Cmd
	if m.focused == 0 {
		m.categories, cmd = m.categories.Update(msg)
	} else {
		m.locations, cmd = m.locations.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.Quitting {
		return "Exiting...\n"
	}
	if m.Submitted {
		return fmt.Sprintf("Selected Category: %s\nSelected Location: %s\nRunning crawl...\n", m.SelectedCat, m.SelectedLoc)
	}

	catStyle := unfocusedStyle
	locStyle := unfocusedStyle

	if m.focused == 0 {
		catStyle = focusedStyle
	} else {
		locStyle = focusedStyle
	}

	return docStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			catStyle.Render(m.categories.View()),
			locStyle.Render(m.locations.View()),
		),
	)
}
