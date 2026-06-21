package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const openHuntBanner = `  ___                   _   _             _
 / _ \ _ __   ___ _ __ | | | |_   _ _ __ | |_
| | | | '_ \ / _ \ '_ \| |_| | | | | '_ \| __|
| |_| | |_) |  __/ | | |  _  | |_| | | | | |_
 \___/| .__/ \___|_| |_|_| |_|\__,_|_| |_|\__|
      |_|`

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

	label := subtleStyle.Render("  " + string(i))
	if index == m.Index() {
		label = selectedItemStyle.Render("● " + string(i))
	}
	fmt.Fprint(w, label)
}

type Model struct {
	categories list.Model
	countries  list.Model
	locations  list.Model
	focused    int
	width      int
	height     int

	SelectedCat     string
	SelectedCountry string
	SelectedLoc     string
	Quitting        bool
	Submitted       bool
}

func NewModel(cats, countries, locs []string) Model {
	return Model{
		categories: newFilterList(cats),
		countries:  newFilterList(countries),
		locations:  newFilterList(locs),
		width:      100,
		height:     26,
	}
}

func newFilterList(values []string) list.Model {
	items := make([]list.Item, len(values))
	for i, value := range values {
		items[i] = item(value)
	}

	model := list.New(items, itemDelegate{}, 20, 8)
	model.SetShowTitle(false)
	model.SetShowHelp(false)
	model.SetShowStatusBar(false)
	model.SetShowPagination(false)
	model.SetFilteringEnabled(false)
	return model
}

func selectedValue(model list.Model) string {
	selected, ok := model.SelectedItem().(item)
	if !ok {
		return "None"
	}
	return string(selected)
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		case "tab":
			m.focused = (m.focused + 1) % 3
			return m, nil
		case "shift+tab":
			m.focused = (m.focused + 2) % 3
			return m, nil
		case "enter":
			m.SelectedCat = selectedValue(m.categories)
			m.SelectedCountry = selectedValue(m.countries)
			m.SelectedLoc = selectedValue(m.locations)
			m.Submitted = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeLists()
	}

	var cmd tea.Cmd
	switch m.focused {
	case 0:
		m.categories, cmd = m.categories.Update(msg)
	case 1:
		m.countries, cmd = m.countries.Update(msg)
	default:
		m.locations, cmd = m.locations.Update(msg)
	}
	return m, cmd
}

func (m *Model) resizeLists() {
	innerWidth := max(66, m.width-appStyle.GetHorizontalFrameSize())
	panelWidth := max(20, (innerWidth-4)/3)
	listHeight := max(6, min(10, m.height-18))
	m.categories.SetSize(panelWidth-4, listHeight)
	m.countries.SetSize(panelWidth-4, listHeight)
	m.locations.SetSize(panelWidth-4, listHeight)
}

func (m Model) View() string {
	if m.Quitting {
		return "Exiting...\n"
	}
	if m.Submitted {
		return fmt.Sprintf(
			"Starting hunt: %s • %s • %s\n",
			m.SelectedCat,
			m.SelectedCountry,
			m.SelectedLoc,
		)
	}

	width := max(72, m.width)
	innerWidth := width - appStyle.GetHorizontalFrameSize()
	panelWidth := max(20, (innerWidth-4)/3)
	panelHeight := max(8, min(12, m.height-17))

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		bannerStyle.Render(openHuntBanner),
		headerMarkStyle.Render("OpenHunt")+"  "+subtleStyle.Render("Choose filters for this crawl"),
	)

	categoryStyle, countryStyle, locationStyle := panelStyle, panelStyle, panelStyle
	switch m.focused {
	case 0:
		categoryStyle = focusedPanelStyle
	case 1:
		countryStyle = focusedPanelStyle
	case 2:
		locationStyle = focusedPanelStyle
	}

	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderPanel(categoryStyle, "01  Functional Category", m.categories.View(), panelWidth, panelHeight),
		"  ",
		renderPanel(countryStyle, "02  Country", m.countries.View(), panelWidth, panelHeight),
		"  ",
		renderPanel(locationStyle, "03  Geographic Location", m.locations.View(), panelWidth, panelHeight),
	)

	summary := summaryStyle.Width(innerWidth - 2).Render(
		subtleStyle.Render("Selection  ") +
			selectedItemStyle.Render(selectedValue(m.categories)) +
			subtleStyle.Render("  •  ") +
			selectedItemStyle.Render(selectedValue(m.countries)) +
			subtleStyle.Render("  •  ") +
			selectedItemStyle.Render(selectedValue(m.locations)),
	)

	help := keyStyle.Render("tab") + subtleStyle.Render(" next panel   ") +
		keyStyle.Render("shift+tab") + subtleStyle.Render(" previous   ") +
		keyStyle.Render("↑/↓") + subtleStyle.Render(" select   ") +
		keyStyle.Render("enter") + subtleStyle.Render(" start crawl   ") +
		keyStyle.Render("q") + subtleStyle.Render(" quit")

	return appStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, header, "", panels, "", summary, "", help),
	)
}

func renderPanel(style lipgloss.Style, title, body string, width, height int) string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Render(title),
		"",
		body,
	)
	return style.
		Width(width - style.GetHorizontalFrameSize()).
		Height(height).
		Render(content)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
