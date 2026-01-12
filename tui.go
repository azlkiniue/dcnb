package main

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/docker/docker/api/types/container"
)

type cleanupState int

const (
	stateIdle cleanupState = iota
	stateDeleting
	stateDone
	stateCancelled
)

type cleanupResultMsg struct {
	removed []string
	err     error
}

type cleanupModel struct {
	ctx        context.Context
	cli        DockerClient
	candidates []container.Summary
	removed    []string
	err        error
	state      cleanupState
	width      int
	height     int
	scrollPos  int
}

func NewCleanupModel(ctx context.Context, cli DockerClient, candidates []container.Summary) cleanupModel {
	return cleanupModel{
		ctx:        ctx,
		cli:        cli,
		candidates: candidates,
		state:      stateIdle,
	}
}

func (m cleanupModel) Init() tea.Cmd { // no async work needed at start
	return nil
}

func (m cleanupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg.String())
	case cleanupResultMsg:
		m.state = stateDone
		m.removed = msg.removed
		m.err = msg.err
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m cleanupModel) handleKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		m.state = stateCancelled
		return m, tea.Quit
	case "n":
		m.state = stateCancelled
		return m, tea.Quit
	case "up":
		if m.scrollPos > 0 {
			m.scrollPos--
		}
		return m, nil
	case "down":
		// Calculate maximum scroll position
		// + 7 accounts for header (1) + border (3) + scroll status (1) + help text (2) [1312]
		maxScroll := len(m.candidates) - m.height + 7
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.scrollPos < maxScroll {
			m.scrollPos++
		}
		return m, nil
	case "y", "enter":
		if m.state != stateIdle {
			return m, nil
		}
		m.state = stateDeleting
		return m, runCleanupCmd(m.ctx, m.cli)
	default:
		return m, nil
	}
}

func runCleanupCmd(ctx context.Context, cli DockerClient) tea.Cmd {
	return func() tea.Msg {
		removed, err := CleanAutoNamedContainers(ctx, cli)
		return cleanupResultMsg{removed: removed, err: err}
	}
}

func color(s string) lipgloss.Color {
	return lipgloss.Color(s)
}

func (m cleanupModel) View() string {
	var (
		b            strings.Builder
		headerStyle  = lipgloss.NewStyle().Foreground(color("51")).Bold(true).Align(lipgloss.Center)
		cellStyle    = lipgloss.NewStyle().Padding(0, 1)
		oddRowStyle  = cellStyle.Foreground(color("244"))
		evenRowStyle = cellStyle.Foreground(color("250"))
	)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(color("51"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			case row%2 == 0:
				return evenRowStyle
			default:
				return oddRowStyle
			}
		}).
		Headers("#", "Name", "Image")

	// Calculate how many rows can fit in the terminal
	// height - header (1) - border (3) - scroll status (1) - help text (2) [1312]
	availableHeight := m.height - 7
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Determine visible range
	endIdx := m.scrollPos + availableHeight
	if endIdx > len(m.candidates) {
		endIdx = len(m.candidates)
	}

	// Render only visible rows
	for i := m.scrollPos; i < endIdx; i++ {
		c := m.candidates[i]
		name := primaryContainerName(c.Names)
		image := c.Image
		if len(image) > 40 {
			image = image[:37] + "..."
		}
		t.Row(fmt.Sprintf("%d", i+1), name, image)
	}

	b.WriteString(t.Render())

	b.WriteString("\n")

	// Show scroll indicator if there are more items
	if len(m.candidates) > availableHeight {
		scrollInfo := fmt.Sprintf("(%d-%d of %d) Use ↑↓ to scroll", m.scrollPos+1, endIdx, len(m.candidates))
		scrollStyle := lipgloss.NewStyle().Foreground(color("243"))
		b.WriteString(scrollStyle.Render(scrollInfo))
		b.WriteString("\n")
	}

	switch m.state {
	case stateIdle:
		helpStyle := lipgloss.NewStyle().Foreground(color("243"))
		b.WriteString(helpStyle.Render("Press y/Enter to delete, n/q to cancel.\n"))
	case stateDeleting:
		spinStyle := lipgloss.NewStyle().Foreground(color("3"))
		b.WriteString(spinStyle.Render("⏳ Deleting containers...\n"))
	case stateDone:
		if m.err != nil {
			errorStyle := lipgloss.NewStyle().Foreground(color("1"))
			fmt.Fprintf(&b, "Removed %d container(s).\n", len(m.removed))
			b.WriteString(errorStyle.Render("Completed with errors: " + m.err.Error() + "\n"))
		} else {
			successStyle := lipgloss.NewStyle().Foreground(color("2"))
			b.WriteString(successStyle.Render(fmt.Sprintf("✓ Removed %d container(s) successfully.\n", len(m.removed))))
		}
	case stateCancelled:
		cancelStyle := lipgloss.NewStyle().Foreground(color("8"))
		b.WriteString(cancelStyle.Render("Operation cancelled.\n"))
	}

	return b.String()
}
