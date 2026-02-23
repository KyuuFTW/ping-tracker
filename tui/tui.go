package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"ping-tracker/tracker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			PaddingLeft(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Background(lipgloss.Color("236"))

	rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57"))

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(1)

	goodPing = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))  // green
	okPing   = lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // yellow
	badPing  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // red

	dirIn  = lipgloss.NewStyle().Foreground(lipgloss.Color("87"))
	dirOut = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

type tickMsg time.Time

// SortField defines which column to sort by.
type SortField int

const (
	SortApp SortField = iota
	SortPing
	SortLoss
	SortTxRate
	SortRxRate
	SortState
)

// Model is the bubbletea model for the TUI.
type Model struct {
	tracker     *tracker.Tracker
	connections []*tracker.Connection
	filter      string
	searching   bool
	cursor      int
	offset      int // scroll offset for viewport
	width       int
	height      int
	sortField   SortField
	sortAsc     bool
	paused      bool
	showHelp    bool
}

// NewModel creates a new TUI model.
func NewModel(t *tracker.Tracker) Model {
	return Model{
		tracker:   t,
		sortField: SortApp,
		sortAsc:   true,
		width:     120,
		height:    30,
	}
}

// SetFilter sets the initial app name filter.
func (m *Model) SetFilter(f string) {
	m.filter = f
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		if !m.paused {
			m.refresh()
		}
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

func (m *Model) refresh() {
	if m.filter != "" {
		m.connections = m.tracker.Search(m.filter)
	} else {
		m.connections = m.tracker.Snapshot()
	}
	m.sortConnections()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "/":
		m.searching = true
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}

	case "down", "j":
		if m.cursor < len(m.connections)-1 {
			m.cursor++
			maxVisible := m.visibleRows()
			if m.cursor >= m.offset+maxVisible {
				m.offset = m.cursor - maxVisible + 1
			}
		}

	case "home", "g":
		m.cursor = 0
		m.offset = 0

	case "end", "G":
		m.cursor = maxInt(0, len(m.connections)-1)
		maxVisible := m.visibleRows()
		if m.cursor >= maxVisible {
			m.offset = m.cursor - maxVisible + 1
		}

	case "1":
		m.toggleSort(SortApp)
	case "2":
		m.toggleSort(SortPing)
	case "3":
		m.toggleSort(SortLoss)
	case "4":
		m.toggleSort(SortTxRate)
	case "5":
		m.toggleSort(SortRxRate)
	case "6":
		m.toggleSort(SortState)

	case "p":
		m.paused = !m.paused

	case "r":
		m.refresh()

	case "c":
		m.filter = ""
		m.cursor = 0
		m.offset = 0
		m.refresh()

	case "?":
		m.showHelp = !m.showHelp
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searching = false
		m.cursor = 0
		m.offset = 0
		m.refresh()

	case "esc":
		m.searching = false

	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.filter += msg.String()
		}
	}

	return m, nil
}

func (m *Model) toggleSort(field SortField) {
	if m.sortField == field {
		m.sortAsc = !m.sortAsc
	} else {
		m.sortField = field
		m.sortAsc = true
	}
	m.sortConnections()
}

func (m *Model) sortConnections() {
	sort.SliceStable(m.connections, func(i, j int) bool {
		a, b := m.connections[i], m.connections[j]

		// Primary sort by selected field
		cmp := 0
		switch m.sortField {
		case SortApp:
			cmp = strings.Compare(strings.ToLower(a.AppName), strings.ToLower(b.AppName))
		case SortPing:
			cmp = compareDuration(a.Ping, b.Ping)
		case SortLoss:
			cmp = compareFloat(a.Loss, b.Loss)
		case SortTxRate:
			cmp = compareFloat(a.TxRate, b.TxRate)
		case SortRxRate:
			cmp = compareFloat(a.RxRate, b.RxRate)
		case SortState:
			cmp = strings.Compare(string(a.State), string(b.State))
		}
		if !m.sortAsc {
			cmp = -cmp
		}
		if cmp != 0 {
			return cmp < 0
		}

		// Secondary: OUT before IN
		if a.Direction != b.Direction {
			return a.Direction == tracker.Outbound
		}

		return false
	})
}

func compareDuration(a, b time.Duration) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func (m Model) visibleRows() int {
	// height minus: title(1) + header(1) + status(2) + search(1) + padding(1)
	return maxInt(1, m.height-6)
}

func (m Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	var b strings.Builder

	// Title
	pauseStr := ""
	if m.paused {
		pauseStr = " [PAUSED]"
	}
	title := titleStyle.Render(fmt.Sprintf("Ping Tracker - %d connections%s", len(m.connections), pauseStr))
	b.WriteString(title + "\n")

	// Search bar
	if m.searching {
		b.WriteString(searchStyle.Render("Search: ") + m.filter + "\u2588\n")
	} else if m.filter != "" {
		b.WriteString(searchStyle.Render("Filter: ") + m.filter + "\n")
	} else {
		b.WriteString("\n")
	}

	// Column widths
	colPID := 7
	colApp := 18
	colPing := 10
	colLoss := 7
	colDir := 4
	colProto := 6
	colLocal := 22
	colRemote := 22
	colState := 12
	colTx := 10
	colRx := 10

	// Header - use padRight for consistency with row rendering
	header := padRight("PID", colPID) + " " + padRight("[1]App", colApp) + " " +
		padRight("[2]Ping", colPing) + " " + padRight("[3]Loss", colLoss) + " " +
		padRight("Dir", colDir) + " " + padRight("Proto", colProto) + " " +
		padRight("Local", colLocal) + " " + padRight("Remote", colRemote) + " " +
		padRight("[6]State", colState) + " " + padRight("[4]TX", colTx) + " " +
		padRight("[5]RX", colRx)
	b.WriteString(headerStyle.Render(truncate(header, m.width)) + "\n")

	// Rows
	maxRows := m.visibleRows()
	end := minInt(m.offset+maxRows, len(m.connections))

	for i := m.offset; i < end; i++ {
		c := m.connections[i]
		row := m.renderRow(c, colPID, colApp, colPing, colLoss, colDir, colProto, colLocal, colRemote, colState, colTx, colRx)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(row) + "\n")
		} else {
			b.WriteString(rowStyle.Render(row) + "\n")
		}
	}

	// Pad empty rows
	for i := end - m.offset; i < maxRows; i++ {
		b.WriteString("\n")
	}

	// Status bar
	sortNames := []string{"App", "Ping", "Loss", "TX", "RX", "State"}
	sortDir := "asc"
	if !m.sortAsc {
		sortDir = "desc"
	}
	status := fmt.Sprintf(" Sort: %s (%s) | /:search  c:clear  p:pause  r:refresh  1-6:sort  ?:help  q:quit",
		sortNames[m.sortField], sortDir)
	b.WriteString(statusBarStyle.Render(truncate(status, m.width)))

	return b.String()
}

func (m Model) renderRow(c *tracker.Connection, colPID, colApp, colPing, colLoss, colDir, colProto, colLocal, colRemote, colState, colTx, colRx int) string {
	// Format local/remote
	local := fmt.Sprintf("%s:%d", c.LocalAddr, c.LocalPort)
	remote := fmt.Sprintf("%s:%d", c.RemoteAddr, c.RemotePort)

	// Format plain text for direction
	dirPlain := string(c.Direction)
	var dirStyle lipgloss.Style
	if c.Direction == tracker.Inbound {
		dirPlain = "IN"
		dirStyle = dirIn
	} else {
		dirPlain = "OUT"
		dirStyle = dirOut
	}

	// Format plain text for ping
	pingPlain := "-"
	var pingStyle lipgloss.Style
	if c.Ping > 0 {
		ms := float64(c.Ping.Microseconds()) / 1000.0
		pingPlain = fmt.Sprintf("%.1fms", ms)
		switch {
		case ms < 50:
			pingStyle = goodPing
		case ms < 150:
			pingStyle = okPing
		default:
			pingStyle = badPing
		}
	}

	// Format plain text for loss
	lossPlain := "-"
	var lossStyle lipgloss.Style
	if c.PingCount > 0 {
		lossPlain = fmt.Sprintf("%.0f%%", c.Loss)
		switch {
		case c.Loss < 1:
			lossStyle = goodPing
		case c.Loss < 10:
			lossStyle = okPing
		default:
			lossStyle = badPing
		}
	}

	// Build each cell as padded plain text, then apply color to content only.
	// This avoids ANSI escape codes breaking fmt.Sprintf alignment.
	pidCell := padRight(fmt.Sprintf("%d", c.PID), colPID)
	appCell := padRight(truncStr(c.AppName, colApp), colApp)
	pingCell := styledPadRight(pingPlain, pingStyle, colPing)
	lossCell := styledPadRight(lossPlain, lossStyle, colLoss)
	dirCell := styledPadRight(dirPlain, dirStyle, colDir)
	protoCell := padRight(c.Protocol, colProto)
	localCell := padRight(truncStr(local, colLocal), colLocal)
	remoteCell := padRight(truncStr(remote, colRemote), colRemote)
	stateCell := padRight(string(c.State), colState)
	txCell := padRight(tracker.FormatBytes(c.TxRate), colTx)
	rxCell := padRight(tracker.FormatBytes(c.RxRate), colRx)

	return pidCell + " " + appCell + " " + pingCell + " " + lossCell + " " +
		dirCell + " " + protoCell + " " + localCell + " " + remoteCell + " " +
		stateCell + " " + txCell + " " + rxCell
}

// padRight pads a plain string to the given width with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// styledPadRight applies a lipgloss style to the text content, then pads
// with plain spaces so the visible width is exactly `width` characters.
// If the style is zero-value (no styling), it falls back to plain padding.
func styledPadRight(text string, style lipgloss.Style, width int) string {
	visLen := len(text)
	if visLen >= width {
		text = text[:width]
		visLen = width
	}
	styled := style.Render(text)
	if visLen < width {
		styled += strings.Repeat(" ", width-visLen)
	}
	return styled
}

func (m Model) renderHelp() string {
	help := `
  Ping Tracker - Help
  ====================

  Navigation:
    j/k or Up/Down   Move cursor
    g / G             Jump to top / bottom

  Search:
    /                 Start search (filters by app name)
    Enter             Confirm search
    Esc               Cancel search
    c                 Clear filter

  Sorting:
    1                 Sort by App name
    2                 Sort by Ping latency
    3                 Sort by Packet loss
    4                 Sort by TX bandwidth
    5                 Sort by RX bandwidth
    6                 Sort by State

  Controls:
    p                 Pause/resume auto-refresh
    r                 Manual refresh
    ?                 Toggle this help
    q / Ctrl+C        Quit

  Press any key to close this help.
`
	return lipgloss.NewStyle().Padding(1, 2).Render(help)
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func truncStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
