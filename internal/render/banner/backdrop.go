package banner

import (
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// ApplyAuroraBackdrop paints aurora background fills behind non-banner frame gaps.
func ApplyAuroraBackdrop(frame string, width int, now time.Time, cfg core.Config) string {
	trimmed := strings.TrimRight(frame, "\n")
	if trimmed == "" {
		return frame
	}
	lines := strings.Split(trimmed, "\n")
	canvasHeight := len(AuroraBannerCanvas())
	for idx, line := range lines {
		if idx < canvasHeight {
			continue
		}
		lines[idx] = AuroraBackdropLine(line, width, idx, now, cfg)
	}
	result := strings.Join(lines, "\n")
	if strings.HasSuffix(frame, "\n") {
		result += "\n"
	}

	return result
}

// AuroraBackdropLine applies aurora background fills to spaces in one rendered line.
func AuroraBackdropLine(line string, width, row int, now time.Time, cfg core.Config) string {
	if line == "" {
		return auroraBackdropSegment(0, row, width, now, cfg)
	}
	plainLine := ansi.StripANSI(line)
	var b strings.Builder
	col := 0
	hasBackground := false
	for line != "" {
		if line[0] == '\x1b' {
			if escape, ok := ansi.LeadingEscape(line); ok {
				b.WriteString(escape)
				hasBackground = updateANSIBackgroundState(hasBackground, escape)
				line = line[len(escape):]

				continue
			}
		}
		r, size := utf8.DecodeRuneInString(line)
		if r == ' ' && !hasBackground {
			run := 0
			remaining := line
			for remaining != "" {
				if remaining[0] == '\x1b' {
					break
				}
				next, nextSize := utf8.DecodeRuneInString(remaining)
				if next != ' ' {
					break
				}
				run++
				remaining = remaining[nextSize:]
			}
			for run > 0 {
				segmentWidth := run
				if spaceRunShouldUsePanelBackground(plainLine, col, segmentWidth) {
					b.WriteString(ansi.StyledText("", ansi.PanelBg, strings.Repeat(" ", segmentWidth)))
					col += segmentWidth
					run -= segmentWidth

					continue
				}
				b.WriteString(auroraBackdropSegment(col, row, segmentWidth, now, cfg))
				col += segmentWidth
				run -= segmentWidth
			}
			line = remaining

			continue
		}
		line = line[size:]
		b.WriteRune(r)
		col++
	}

	return b.String()
}

func auroraBackdropSegment(startCol, row, width int, now time.Time, cfg core.Config) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	currentCol := startCol
	endCol := startCol + width
	for currentCol < endCol {
		b.WriteString(auroraBackdropCell(currentCol, row, now, cfg))
		currentCol++
	}

	return b.String()
}

func auroraBackdropCell(col, row int, now time.Time, cfg core.Config) string {
	upper := auroraBackdropColor(col, float64(row), now)
	lower := auroraBackdropColor(col, float64(row)+auroraBackdropHalfStep, now)

	return ansi.StyledText(bannerColorEscape(upper, cfg), bannerBackgroundEscape(lower, cfg), auroraBackdropHalfBlock)
}

// AuroraBackdropBandColor returns the quantized aurora backdrop color for one band.
func AuroraBackdropBandColor(col, row int, now time.Time) Color {
	sampleCol := col - positiveMod(col, AuroraBackdropBandWidth)

	return auroraBackdropColor(sampleCol, float64(row), now)
}

func updateANSIBackgroundState(current bool, escape string) bool {
	if !strings.HasPrefix(escape, "\x1b[") || !strings.HasSuffix(escape, "m") {
		return current
	}
	body := escape[2 : len(escape)-1]
	if body == "" {
		return false
	}
	hasBackground := current
	for part := range strings.SplitSeq(body, ";") {
		switch part {
		case "0", "00", "39":
			if part == "0" || part == "00" {
				hasBackground = false
			}
		case "48":
			hasBackground = true
		case "49":
			hasBackground = false
		}
	}

	return hasBackground
}

func auroraBackdropColor(col int, row float64, now time.Time) Color {
	t := float64(now.UnixMilli()) / auroraFieldMillisScale
	base := bannerColorRGB(fieldBaseRed, fieldBaseGreen, fieldBaseBlue)
	ribbons := []struct {
		anchor float64
		slope  float64
		sway   float64
		speed  float64
		spread float64
		gain   float64
		seed   float64
		color  Color
	}{
		{
			anchor: ribbonGreenAnchor,
			slope:  ribbonGreenSlope,
			sway:   ribbonGreenSway,
			speed:  ribbonGreenSpeed,
			spread: ribbonGreenSpread,
			gain:   ribbonGreenGain,
			seed:   ribbonGreenSeed,
			color:  bannerColorRGB(ribbonGreenRed, ribbonGreenGreen, ribbonGreenBlue),
		},
		{
			anchor: ribbonCyanAnchor,
			slope:  ribbonCyanSlope,
			sway:   ribbonCyanSway,
			speed:  ribbonCyanSpeed,
			spread: ribbonCyanSpread,
			gain:   ribbonCyanGain,
			seed:   ribbonCyanSeed,
			color:  bannerColorRGB(ribbonCyanRed, ribbonCyanGreen, ribbonCyanBlue),
		},
		{
			anchor: ribbonPurpleAnchor,
			slope:  ribbonPurpleSlope,
			sway:   ribbonPurpleSway,
			speed:  ribbonPurpleSpeed,
			spread: ribbonPurpleSpread,
			gain:   ribbonPurpleGain,
			seed:   ribbonPurpleSeed,
			color:  bannerColorRGB(ribbonPurpleRed, ribbonPurpleGreen, ribbonPurpleBlue),
		},
		{
			anchor: ribbonMagentaAnchor,
			slope:  ribbonMagentaSlope,
			sway:   ribbonMagentaSway,
			speed:  ribbonMagentaSpeed,
			spread: ribbonMagentaSpread,
			gain:   ribbonMagentaGain,
			seed:   ribbonMagentaSeed,
			color:  bannerColorRGB(ribbonMagentaRed, ribbonMagentaGreen, ribbonMagentaBlue),
		},
		{
			anchor: ribbonCoralAnchor,
			slope:  ribbonCoralSlope,
			sway:   ribbonCoralSway,
			speed:  ribbonCoralSpeed,
			spread: ribbonCoralSpread,
			gain:   ribbonCoralGain,
			seed:   ribbonCoralSeed,
			color:  bannerColorRGB(ribbonCoralRed, ribbonCoralGreen, ribbonCoralBlue),
		},
	}
	redValue := float64(base.RedValue)
	greenValue := float64(base.GreenValue)
	blueValue := float64(base.BlueValue)
	weight := 1.0
	for _, ribbon := range ribbons {
		centerDrift := auroraFieldNoise(row*ribbonDriftRowScale+t*ribbonDriftTimeScale, t*ribbonDriftSpeed, ribbon.seed)
		center := ribbon.anchor +
			row*ribbon.slope +
			ribbon.sway*math.Sin(t*ribbon.speed+row*ribbonSwayRowScale+ribbon.seed) +
			(centerDrift-ribbonDriftCenter)*ribbonDriftAmplitude
		delta := float64(col) - center
		intensity := math.Exp(-(delta * delta) / (gaussianSpreadFactor * ribbon.spread * ribbon.spread))
		pulse := pulseBase + pulseGain*auroraFieldNoise(float64(col)*pulseColScale+row*pulseRowScale, t*pulseTimeScale+ribbon.seed, ribbon.seed+pulseSeedOffset)
		contribution := intensity * ribbon.gain * pulse
		redValue += float64(ribbon.color.RedValue) * contribution
		greenValue += float64(ribbon.color.GreenValue) * contribution
		blueValue += float64(ribbon.color.BlueValue) * contribution
		weight += contribution
	}
	backdropPalette := auroraBackdropPalette()
	washIndex := auroraBackdropIndex(
		col/washColDivisor+int(math.Round(washColAmplitude*math.Sin(float64(row)*washColRowScale+t*washColTimeScale))),
		int(math.Round(row+washRowAmplitude*math.Sin(t*washRowTimeScale+float64(col)*washRowColScale))),
		now.Add(washDelayMillis*time.Millisecond),
		len(backdropPalette),
	)
	wash := backdropPalette[washIndex]
	washMix := washMixBase + washMixGain*auroraFieldNoise(float64(col)*washMixColScale+row*washMixRowScale, t*washMixTimeScale, washMixSeed)
	redValue += float64(wash.RedValue) * washMix
	greenValue += float64(wash.GreenValue) * washMix
	blueValue += float64(wash.BlueValue) * washMix
	weight += washMix

	haze := hazeBase + hazeGain*auroraFieldNoise(float64(col)*hazeColScale+row*hazeRowScale, t*hazeTimeScale, hazeSeed)
	redValue += hazeRedContribution * haze
	greenValue += hazeGreenContribution * haze
	blueValue += hazeBlueContribution * haze
	weight += haze

	return Color{
		RedValue:   clampFromZero(int(math.Round(redValue/weight)), rgbChannelMax),
		GreenValue: clampFromZero(int(math.Round(greenValue/weight)), rgbChannelMax),
		BlueValue:  clampFromZero(int(math.Round(blueValue/weight)), rgbChannelMax),
	}
}

func auroraFieldNoise(x, t, seed float64) float64 {
	waveA := noiseMidpoint + noiseMidpoint*math.Sin(x*noiseWaveAXScale+t*noiseWaveATScale+seed*noiseWaveASeedScale)
	waveB := noiseMidpoint + noiseMidpoint*math.Sin(x*noiseWaveBXScale-t*noiseWaveBTScale+seed*noiseWaveBSeedScale)
	waveC := noiseMidpoint + noiseMidpoint*math.Sin(x*noiseWaveCXScale+t*noiseWaveCTScale+seed*noiseWaveCSeedScale)

	return math.Max(0, math.Min(1, waveA*noiseWeightA+waveB*noiseWeightB+waveC*noiseWeightC))
}

func auroraBackdropIndex(col, row int, now time.Time, paletteLen int) int {
	if paletteLen < 1 {
		return 0
	}
	t := float64(now.UnixMilli()) / backdropTimeScale
	waveA := math.Sin(float64(col)/backdropWaveADivisor - t + float64(row)*backdropWaveARowScale)
	waveB := math.Sin(float64(col)/backdropWaveBDivisor + t*backdropWaveBTimeScale - float64(row)*backdropWaveBRowScale)
	waveC := math.Sin(float64(row)*backdropWaveCRowScale + t*backdropWaveCTimeScale)
	base := int(math.Round(float64(paletteLen/backdropCenterDivisor) + waveA*backdropWaveAWeight + waveB*backdropWaveBWeight + waveC*backdropWaveCWeight))
	base %= paletteLen
	if base < 0 {
		base += paletteLen
	}

	return base
}

func auroraFaceColor(col, row int, now time.Time) Color {
	palette := auroraBannerPalette()
	index := auroraBackdropIndex(col+faceColOffset, row*faceRowStride, now.Add(faceDelayMillis*time.Millisecond), len(palette))
	base := palette[index%len(palette)]
	highlight := bannerColorRGB(faceHighlightRed, faceHighlightGreen, faceHighlightBlue)
	mix := faceMixBase + faceMixGain*(faceMixWaveBase+faceMixWaveBase*math.Sin(float64(col)/faceMixColDivisor+float64(row)*faceMixRowScale+float64(now.UnixMilli())/faceMixTimeScale))

	return interpolateBannerColor(base, highlight, mix)
}

func spaceRunShouldUsePanelBackground(plainLine string, start, width int) bool {
	if width <= 0 {
		return false
	}
	end := start + width - 1
	left, hasLeft := runeAtColumn(plainLine, start-1)
	right, hasRight := runeAtColumn(plainLine, end+1)
	if hasLeft && hasRight && left == '│' && right == '│' {
		return width > panelInteriorMinSpace
	}

	return hasVerticalBorderBefore(plainLine, start) && hasVerticalBorderAfter(plainLine, end)
}

func runeAtColumn(s string, target int) (rune, bool) {
	if target < 0 {
		return 0, false
	}
	col := 0
	for _, r := range s {
		if col == target {
			return r, true
		}
		col++
	}

	return 0, false
}

func hasVerticalBorderBefore(s string, target int) bool {
	col := 0
	for _, r := range s {
		if col >= target {
			return false
		}
		if r == '│' {
			return true
		}
		col++
	}

	return false
}

func hasVerticalBorderAfter(s string, target int) bool {
	col := 0
	for _, r := range s {
		if col > target && r == '│' {
			return true
		}
		col++
	}

	return false
}
