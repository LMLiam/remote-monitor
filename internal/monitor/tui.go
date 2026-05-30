package monitor

import (
	"context"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	tuiMinRenderWidth  = 100
	tuiMinRenderHeight = 28
	tuiViewportMargin  = 2
)

type tuiTickMsg time.Time

type sampleUpdateMsg struct {
	Sample core.Sample
}

type streamEventMsg struct {
	event core.StreamEvent
}

// TUIModel owns the Bubble Tea viewport and monitor state for interactive mode.
type TUIModel struct {
	State    core.AppState
	Viewport viewport.Model
	Width    int
	Height   int
	Interval time.Duration
}

func runTUI(ctx context.Context, initial core.AppState, sampleCh <-chan core.Sample, eventCh <-chan core.StreamEvent) error {
	model := NewTUIModel(initial)
	program := tea.NewProgram(model)

	go forwardMonitorMessages(ctx, program, sampleCh, eventCh)
	go func() {
		<-ctx.Done()
		program.Quit()
	}()

	_, err := program.Run()

	return err
}

// NewTUIModel creates an initialized interactive terminal model.
func NewTUIModel(initial core.AppState) *TUIModel {
	width := 120
	height := 40
	vp := viewport.New(
		viewport.WithWidth(width),
		viewport.WithHeight(height),
	)
	vp.MouseWheelEnabled = true
	vp.FillHeight = true

	model := &TUIModel{
		State:    initial,
		Viewport: vp,
		Width:    width,
		Height:   height,
		Interval: RenderInterval(initial.Cfg, true),
	}
	model.RefreshViewport()

	return model
}

func forwardMonitorMessages(ctx context.Context, program *tea.Program, sampleCh <-chan core.Sample, eventCh <-chan core.StreamEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case smp := <-sampleCh:
			program.Send(sampleUpdateMsg{Sample: smp})
		case ev := <-eventCh:
			program.Send(streamEventMsg{event: ev})
		}
	}
}

func tickTUI(d time.Duration) tea.Cmd {
	if d <= 0 {
		d = time.Second
	}

	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tuiTickMsg(t)
	})
}

// Init starts the TUI refresh ticker.
func (m *TUIModel) Init() tea.Cmd {
	return tickTUI(m.Interval)
}

// Update handles terminal, sampler, stream, and viewport messages.
//
//nolint:ireturn // Bubble Tea's Model interface requires Update to return tea.Model.
func (m *TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = max(msg.Width, 1)
		m.Height = max(msg.Height, 1)
		m.RefreshViewport()

		return m, nil
	case sampleUpdateMsg:
		ApplySample(&m.State, msg.Sample)
		m.RefreshViewport()

		return m, nil
	case streamEventMsg:
		ApplyEvent(&m.State, msg.event)
		m.RefreshViewport()

		return m, nil
	case tuiTickMsg:
		m.RefreshViewport()

		return m, tickTUI(m.Interval)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	m.SyncScrollState()

	return m, cmd
}

// View renders the centered interactive viewport.
func (m *TUIModel) View() tea.View {
	v := tea.NewView(
		lipgloss.Place(
			max(m.Width, 1),
			max(m.Height, 1),
			lipgloss.Center,
			lipgloss.Center,
			m.Viewport.View(),
		),
	)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "remote-monitor"

	return v
}

// RefreshViewport rerenders dashboard content and preserves scroll position.
func (m *TUIModel) RefreshViewport() {
	terminalWidth := max(m.Width, 1)
	terminalHeight := max(m.Height, 1)
	renderWidth := max(terminalWidth, tuiMinRenderWidth)
	renderHeight := max(terminalHeight, tuiMinRenderHeight)
	content := render.FullFrame(m.State, renderWidth, renderHeight)
	viewportWidth, viewportHeight := centeredViewportSize(content, terminalWidth, terminalHeight)
	offset := max(m.State.ScrollOffset, m.Viewport.YOffset())

	m.Viewport.SetWidth(viewportWidth)
	m.Viewport.SetHeight(viewportHeight)
	m.Viewport.SetContent(content)
	maxOffset := max(0, m.Viewport.TotalLineCount()-m.Viewport.Height())
	m.Viewport.SetYOffset(min(max(offset, 0), maxOffset))
	m.SyncScrollState()
}

// SyncScrollState copies viewport scroll limits back into monitor state.
func (m *TUIModel) SyncScrollState() {
	m.State.ScrollOffset = m.Viewport.YOffset()
	m.State.ScrollMax = max(0, m.Viewport.TotalLineCount()-m.Viewport.Height())
}

func centeredViewportSize(content string, terminalWidth, terminalHeight int) (width, height int) {
	lines := render.SplitRenderedLines(content)
	if len(lines) == 0 {
		return max(terminalWidth, 1), max(terminalHeight, 1)
	}

	contentWidth := 1
	for _, line := range lines {
		contentWidth = max(contentWidth, ansi.VisibleLen(line))
	}

	width = min(centeredViewportExtent(terminalWidth), contentWidth)
	height = min(centeredViewportExtent(terminalHeight), len(lines))

	return max(width, 1), max(height, 1)
}

func centeredViewportExtent(terminalExtent int) int {
	if terminalExtent <= tuiViewportMargin {
		return max(terminalExtent, 1)
	}

	return terminalExtent - tuiViewportMargin
}
