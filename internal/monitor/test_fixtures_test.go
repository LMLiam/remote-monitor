package monitor_test

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/input"

	"time"
)

func testConfig(overrides ...func(*core.Config)) core.Config {
	cfg := core.Config{
		Host:               "",
		Interval:           0,
		ProcessSort:        "",
		ProcessFilter:      "",
		ProcessCount:       0,
		HistoryLimit:       0,
		StaleAfter:         0,
		ReconnectBaseDelay: 0,
		RenderFPS:          0,
		Compact:            false,
		NoBanner:           false,
		ShowVersion:        false,
		OutputMode:         "",
		OutputPath:         "",
		Theme:              "",
		DisableTrueColor:   false,
		SSHConnectTimeout:  0,
		SSHAliveInterval:   0,
		SSHAliveCountMax:   0,
		SSHControlPersist:  0,
		SSHControlPath:     "",
	}
	for _, override := range overrides {
		override(&cfg)
	}

	return cfg
}

func testState(overrides ...func(*core.AppState)) core.AppState {
	state := core.AppState{
		Cfg:                testConfig(),
		RuntimeState:       "",
		RuntimeDetail:      "",
		LastTransport:      "",
		SampleCount:        0,
		ReconnectCount:     0,
		ReconnectAttempts:  0,
		NextRetry:          time.Time{},
		LastRx:             time.Time{},
		StreamAlive:        false,
		Current:            testSample(),
		HasSample:          false,
		ScrollOffset:       0,
		ScrollMax:          0,
		NetCeilings:        map[string]int64{},
		CPUHistory:         nil,
		CPUFreqHistory:     nil,
		CPUTempHistory:     nil,
		RAMHistory:         nil,
		RAMAvailHistory:    nil,
		DiskHistory:        nil,
		DiskLatencyHistory: nil,
		GPUHistory:         nil,
		VRAMHistory:        nil,
		TempHistory:        nil,
		PowerHistory:       nil,
		NetRXHistory:       nil,
		NetTXHistory:       nil,
		NetIssueHistory:    nil,
	}
	for _, override := range overrides {
		override(&state)
	}

	return state
}

func testSample(overrides ...func(*core.Sample)) core.Sample {
	smp := core.EmptySample()
	for _, override := range overrides {
		override(&smp)
	}

	return smp
}

func testNetStat(overrides ...func(*core.NetStat)) core.NetStat {
	net := core.NetStat{
		Iface:      "",
		RXBps:      0,
		TXBps:      0,
		RXPps:      0,
		TXPps:      0,
		SpeedMbps:  0,
		RXDrops:    0,
		RXErrors:   0,
		RXOverruns: 0,
		TXDrops:    0,
		TXErrors:   0,
		TXOverruns: 0,
	}
	for _, override := range overrides {
		override(&net)
	}

	return net
}

func testGPUStat(overrides ...func(*core.GPUStat)) core.GPUStat {
	gpu := core.GPUStat{
		Index:            0,
		UUID:             "",
		Name:             "",
		Util:             0,
		MemUtil:          0,
		EncoderUtil:      0,
		DecoderUtil:      0,
		MemUsed:          0,
		MemTotal:         0,
		Temp:             0,
		PowerDraw:        0,
		PowerLimit:       0,
		Fan:              0,
		SMClock:          0,
		MaxSMClock:       0,
		MemClock:         0,
		MaxMemClock:      0,
		GraphicsClock:    0,
		VideoClock:       0,
		PCIeGenCurrent:   0,
		PCIeGenMax:       0,
		PCIeWidthCurrent: 0,
		PCIeWidthMax:     0,
		ThrottleReasons:  "",
		PState:           "",
	}
	for _, override := range overrides {
		override(&gpu)
	}

	return gpu
}

func testStreamEvent(overrides ...func(*core.StreamEvent)) core.StreamEvent {
	ev := core.StreamEvent{
		State:          "",
		Detail:         "",
		ReconnectCount: 0,
		Attempts:       0,
		StreamAlive:    false,
		NextRetry:      time.Time{},
		At:             time.Time{},
	}
	for _, override := range overrides {
		override(&ev)
	}

	return ev
}

func testTTYCommand(overrides ...func(*input.TTYCommand)) input.TTYCommand {
	cmd := input.TTYCommand{
		LineDelta: 0,
		PageDelta: 0,
		ToTop:     false,
		ToBottom:  false,
		Quit:      false,
	}
	for _, override := range overrides {
		override(&cmd)
	}

	return cmd
}
