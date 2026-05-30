package render

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"time"
)

const statusDetailMaxRunes = 28

func currentStatus(state core.AppState) string {
	if !state.HasSample {
		return state.RuntimeState
	}
	if state.StreamAlive && time.Since(state.LastRx) > state.Cfg.StaleAfter {
		return core.StatusStale
	}

	return state.RuntimeState
}

func statusValueText(state core.AppState) string {
	if !state.HasSample {
		return currentStatus(state)
	}

	return fmt.Sprintf("%s %s old", currentStatus(state), formatAge(time.Since(state.LastRx)))
}

func freshnessText(state core.AppState) string {
	if !state.HasSample {
		return "waiting"
	}

	return formatAge(time.Since(state.LastRx)) + " old"
}

// HealthText returns the short health message shown in the dashboard header.
func HealthText(state core.AppState) string {
	switch currentStatus(state) {
	case core.StatusLive:
		return core.DetailStreamHealthy
	case core.StatusStale:
		return "waiting for fresh Sample"
	}
	if state.RuntimeDetail != "" {
		return ansi.FitText(state.RuntimeDetail, statusDetailMaxRunes)
	}

	return "waiting"
}

func loadText(s core.Sample) string {
	if s.CPUCores > 0 {
		return fmt.Sprintf("%.2f / %d cores • %.2f • %.2f", s.Load1, s.CPUCores, s.Load5, s.Load15)
	}

	return fmt.Sprintf("%.2f • %.2f • %.2f", s.Load1, s.Load5, s.Load15)
}

func modeText(cfg core.Config) string {
	mode := "go • utf8 • 256c"
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		mode += "+tc"
	}

	return mode + " • " + core.CanonicalThemeName(cfg.Theme)
}
