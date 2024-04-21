package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gen2brain/beeep"
)

type ProgressStatus string

const (
	Idle    ProgressStatus = "idle"
	Paused  ProgressStatus = "paused"
	Running ProgressStatus = "running"
)

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	specialColor      = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Copy().Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(2, 0).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
)

type model struct {
	Tabs                     []string
	ActiveTab                int
	ProgressMode             int
	ProgressPomodoro         progress.Model
	ProgressShort            progress.Model
	ProgressLong             progress.Model
	ProgressStatus           ProgressStatus
	ProgressPomodoroDuration time.Duration
	ProgressShortDuration    time.Duration
	ProgressLongDuration     time.Duration
	ProgressCurrentTime      time.Duration
	ProgressPercent          float64
}

type tickMsg struct{}
type progressDoneMsg struct{}

func initialModel() model {
	return model{
		Tabs:                     []string{"Pomodoro", "Short break", "Long break"},
		ActiveTab:                0, // Tabs index
		ProgressMode:             0, // Tabs index
		ProgressPomodoro:         progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage()),
		ProgressShort:            progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage()),
		ProgressLong:             progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage()),
		ProgressStatus:           Idle,
		ProgressPomodoroDuration: 5 * time.Second,
		ProgressShortDuration:    120 * time.Second,
		ProgressLongDuration:     180 * time.Second,
		ProgressCurrentTime:      0,
		ProgressPercent:          0.0,
	}
}

func (m *model) resetProgress() {
	m.ProgressCurrentTime = 0
	m.ProgressPercent = 0.0
	m.ProgressStatus = Idle
}

func (m model) getDurationByIndex(index int) time.Duration {
	switch index {
	case 0:
		return m.ProgressPomodoroDuration
	case 1:
		return m.ProgressShortDuration
	case 2:
		return m.ProgressLongDuration
	default:
		panic("")
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func progressDone() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		beeep.Alert("Pomodoro done", "", "assets/pomodoro.png")
		return progressDoneMsg{}
	})
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			m.resetProgress()
			return m, nil
		case "right", "d", "tab":
			m.ActiveTab = min(m.ActiveTab+1, len(m.Tabs)-1)
			return m, nil
		case "left", "a":
			m.ActiveTab = max(m.ActiveTab-1, 0)
			return m, nil
		case " ":
			if m.ProgressStatus == Idle {
				m.ProgressMode = m.ActiveTab
				m.ProgressStatus = Running
				return m, tick()
			}

			if m.ProgressMode == m.ActiveTab {
				if m.ProgressStatus == Running {
					m.ProgressStatus = Paused
					return m, tick()
				}

				if m.ProgressStatus == Paused {
					m.ProgressStatus = Running
					return m, tick()
				}
			}

			return m, nil

		}
	case tickMsg:
		if m.ProgressPercent >= 1.0 {
			m.ProgressPercent = 1.0
			return m, progressDone()
		}

		if m.ProgressStatus == Running {
			m.ProgressCurrentTime += 1 * time.Second
			m.ProgressPercent += 1.0 / float64(m.getDurationByIndex(m.ProgressMode).Seconds())
			return m, tick()
		}

		return m, nil

	case progressDoneMsg:
		m.resetProgress()
		return m, nil
	}

	return m, nil
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

func chosenView(m model) string {
	progressPercent := 0.0
	viewDuration := m.getDurationByIndex(m.ActiveTab)

	if m.ActiveTab == m.ProgressMode {
		progressPercent = m.ProgressPercent
		viewDuration = viewDuration - m.ProgressCurrentTime
	}

	msg := fmt.Sprintf("%s %s", m.ProgressLong.ViewAs(progressPercent), viewDuration.String())

	return msg
}

func (m model) View() string {
	doc := strings.Builder{}

	var renderedTabs []string

	for i, t := range m.Tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.ActiveTab

		if isActive {
			style = activeTabStyle.Copy()
		} else {
			style = inactiveTabStyle.Copy()
		}

		border, _, _, _, _ := style.GetBorder()

		if isFirst && isActive {
			border.BottomLeft = "|"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "|"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}

		style = style.Border(border).Padding(0, 5)
		if m.ProgressStatus == Running && m.ProgressMode == i {
			style.Bold(true).Foreground(specialColor)
		}

		renderedTabs = append(renderedTabs, style.Render(t))

	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	doc.WriteString(row)
	doc.WriteString("\n")
	doc.WriteString(windowStyle.Width((lipgloss.Width(row) - windowStyle.GetHorizontalFrameSize())).Render(chosenView(m)))
	return docStyle.Render(doc.String())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
