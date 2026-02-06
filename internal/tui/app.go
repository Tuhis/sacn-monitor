package tui

import (
	"fmt"
	"sort"
	"time"

	"sacn-monitor/internal/stats"
	"sacn-monitor/internal/universe"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Colors
var (
	cyanColor = lipgloss.Color("#00FFFF")
	grayColor = lipgloss.Color("#666666")

	whiteColor  = lipgloss.Color("#FFFFFF")
	yellowColor = lipgloss.Color("#FFFF00")
	redColor    = lipgloss.Color("#FF6666")
)

// Styles
var (
	activeCardStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(cyanColor).
			Width(4)

	inactiveCardStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(grayColor).
				Width(4)

	statsStyle = lipgloss.NewStyle().
			Foreground(whiteColor)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(whiteColor).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 2)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyanColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cyanColor).
			Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(grayColor).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(grayColor).
				Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(grayColor)
)

// KeyMap defines keybindings
type KeyMap struct {
	Left  key.Binding
	Right key.Binding
	Up    key.Binding
	Down  key.Binding
	Tab   key.Binding
	Quit  key.Binding
}

var keys = KeyMap{
	Left:  key.NewBinding(key.WithKeys("left", "h")),
	Right: key.NewBinding(key.WithKeys("right", "l")),
	Up:    key.NewBinding(key.WithKeys("up", "k")),
	Down:  key.NewBinding(key.WithKeys("down", "j")),
	Tab:   key.NewBinding(key.WithKeys("tab")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
}

// Model is the main TUI model
type Model struct {
	universeManager  *universe.Manager
	statsTracker     *stats.Tracker
	selectedUniverse uint16
	universeList     []uint16
	scrollOffset     int
	width            int
	height           int
	columnsPerRow    int
}

// NewModel creates a new TUI model
func NewModel(um *universe.Manager, st *stats.Tracker) Model {
	return Model{
		universeManager: um,
		statsTracker:    st,
		columnsPerRow:   16, // Default, will adjust based on terminal width
	}
}

// TickMsg is a message for periodic updates
type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Tab):
			// Cycle to next universe
			if len(m.universeList) > 1 {
				for i, id := range m.universeList {
					if id == m.selectedUniverse {
						m.selectedUniverse = m.universeList[(i+1)%len(m.universeList)]
						break
					}
				}
			}
		case key.Matches(msg, keys.Down):
			m.scrollOffset += m.columnsPerRow
		case key.Matches(msg, keys.Up):
			if m.scrollOffset >= m.columnsPerRow {
				m.scrollOffset -= m.columnsPerRow
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Calculate columns: each card is ~6 chars wide (4 + border)
		m.columnsPerRow = max(1, (m.width-2)/6)

	case TickMsg:
		// Update universe list
		m.updateUniverseList()
		return m, tickCmd()
	}

	return m, nil
}

func (m *Model) updateUniverseList() {
	universes := m.universeManager.GetAll()
	m.universeList = make([]uint16, len(universes))
	for i, u := range universes {
		m.universeList[i] = u.ID
	}
	sort.Slice(m.universeList, func(i, j int) bool {
		return m.universeList[i] < m.universeList[j]
	})

	// Select first universe if none selected or selected no longer exists
	if len(m.universeList) > 0 {
		found := false
		for _, id := range m.universeList {
			if id == m.selectedUniverse {
				found = true
				break
			}
		}
		if !found {
			m.selectedUniverse = m.universeList[0]
		}
	}
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var s string

	// Title
	s += titleStyle.Render("sACN Monitor") + "\n\n"

	// Universe tabs
	if len(m.universeList) > 0 {
		tabs := ""
		for _, id := range m.universeList {
			tabText := fmt.Sprintf("Universe %d", id)
			if id == m.selectedUniverse {
				tabs += tabActiveStyle.Render(tabText) + " "
			} else {
				tabs += tabInactiveStyle.Render(tabText) + " "
			}
		}
		s += tabs + "\n\n"

		// Stats for selected universe
		s += m.renderStats() + "\n\n"

		// Channel grid
		s += m.renderChannelGrid() + "\n"
	} else {
		s += helpStyle.Render("Waiting for sACN data...") + "\n\n"
		s += helpStyle.Render("Listening on UDP port 5568 for multicast/unicast/broadcast traffic.") + "\n"
	}

	// Help
	s += "\n" + helpStyle.Render("Tab: switch universe | ↑↓: scroll | q: quit")

	return s
}

func (m Model) renderStats() string {
	u := m.universeManager.Get(m.selectedUniverse)
	if u == nil {
		return ""
	}

	info := u.GetInfo()
	rate := m.statsTracker.GetPacketRate(m.selectedUniverse)
	loss := m.statsTracker.GetLossPercentage(m.selectedUniverse)
	activeCount := u.ActiveChannelCount()

	// Format loss with color
	lossStr := fmt.Sprintf("%.1f%%", loss)
	if loss > 1 {
		lossStr = lipgloss.NewStyle().Foreground(redColor).Render(lossStr)
	} else if loss > 0 {
		lossStr = lipgloss.NewStyle().Foreground(yellowColor).Render(lossStr)
	}

	stats := fmt.Sprintf(
		"Source: %s | Rate: %.1f pps | Loss: %s | Active: %d/512",
		info.SourceName,
		rate,
		lossStr,
		activeCount,
	)

	return statsStyle.Render(stats)
}

func (m Model) renderChannelGrid() string {
	u := m.universeManager.Get(m.selectedUniverse)
	if u == nil {
		return ""
	}

	channels := u.GetAllChannels()

	var rows []string
	channelsPerRow := m.columnsPerRow
	if channelsPerRow < 1 {
		channelsPerRow = 16
	}

	// Calculate visible rows based on height
	// Reserve space for: title(2) + tabs(3) + stats(2) + help(2) = 9 lines
	availableHeight := m.height - 9
	if availableHeight < 4 {
		availableHeight = 4
	}
	// Each card row is 4 lines tall (border + 2 content + border)
	rowsPerScreen := availableHeight / 4
	if rowsPerScreen < 1 {
		rowsPerScreen = 1
	}

	startChannel := m.scrollOffset
	if startChannel >= 512 {
		startChannel = 512 - channelsPerRow
	}
	if startChannel < 0 {
		startChannel = 0
	}

	endChannel := min(512, startChannel+(rowsPerScreen*channelsPerRow))

	for i := startChannel; i < endChannel; i += channelsPerRow {
		var cards []string
		for j := 0; j < channelsPerRow && (i+j) < 512; j++ {
			ch := channels[i+j]
			channelNum := i + j + 1 // 1-based channel number

			var cardStyle lipgloss.Style
			var valueStr string

			if ch.Active {
				cardStyle = activeCardStyle
				valueStr = fmt.Sprintf("%3d", ch.Value)
			} else {
				cardStyle = inactiveCardStyle
				valueStr = " . "
			}

			cardContent := fmt.Sprintf("%3d\n%s", channelNum, valueStr)
			cards = append(cards, cardStyle.Render(cardContent))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
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
