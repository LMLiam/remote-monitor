package banner

import (
	"fmt"
	core "github.com/lmliam/remote-monitor/internal/core"
	"github.com/lmliam/remote-monitor/internal/render/ansi"
	"math"
)

func ansi256Color(index int) string {
	return fmt.Sprintf("\x1b[38;5;%dm", clampFromZero(index, rgbChannelMax))
}

func ansi256Background(index int) string {
	return fmt.Sprintf("\x1b[48;5;%dm", clampFromZero(index, rgbChannelMax))
}

func ansiTrueColor(redValue, greenValue, blueValue int) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", clampFromZero(redValue, rgbChannelMax), clampFromZero(greenValue, rgbChannelMax), clampFromZero(blueValue, rgbChannelMax))
}

func ansiTrueColorBackground(redValue, greenValue, blueValue int) string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", clampFromZero(redValue, rgbChannelMax), clampFromZero(greenValue, rgbChannelMax), clampFromZero(blueValue, rgbChannelMax))
}

func bannerColorEscape(color Color, cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansiTrueColor(color.RedValue, color.GreenValue, color.BlueValue)
	}

	return ansi256Color(rgbToANSI256(color.RedValue, color.GreenValue, color.BlueValue))
}

func bannerBackgroundEscape(color Color, cfg core.Config) string {
	if ansi.SupportsTrueColor(cfg.DisableTrueColor) {
		return ansiTrueColorBackground(color.RedValue, color.GreenValue, color.BlueValue)
	}

	return ansi256Background(rgbToANSI256(color.RedValue, color.GreenValue, color.BlueValue))
}

func buildBannerPalette(steps int) []Color {
	if steps < 1 {
		steps = 1
	}
	palette := make([]Color, 0, steps)
	for i := range steps {
		hue := float64(i) / float64(steps)
		redValue, greenValue, blueValue := hsvToRGB(hue, hsvDefaultSaturation, hsvDefaultValue)
		palette = append(palette, Color{
			RedValue:   redValue,
			GreenValue: greenValue,
			BlueValue:  blueValue,
		})
	}

	return palette
}

func buildGradientBannerPalette(steps int, anchors []Color) []Color {
	if len(anchors) == 0 {
		return buildBannerPalette(steps)
	}
	if len(anchors) == 1 {
		palette := make([]Color, max(steps, 1))
		for i := range palette {
			palette[i] = anchors[0]
		}

		return palette
	}
	if steps < 1 {
		steps = 1
	}
	palette := make([]Color, 0, steps)
	segments := len(anchors)
	for i := range steps {
		position := float64(i) / float64(steps)
		scaled := position * float64(segments)
		segment := int(math.Floor(scaled)) % segments
		next := (segment + 1) % segments
		fraction := scaled - math.Floor(scaled)
		palette = append(palette, interpolateBannerColor(anchors[segment], anchors[next], fraction))
	}

	return palette
}

func interpolateBannerColor(from, to Color, fraction float64) Color {
	return Color{
		RedValue:   interpolateChannel(from.RedValue, to.RedValue, fraction),
		GreenValue: interpolateChannel(from.GreenValue, to.GreenValue, fraction),
		BlueValue:  interpolateChannel(from.BlueValue, to.BlueValue, fraction),
	}
}

func interpolateChannel(from, to int, fraction float64) int {
	return clampFromZero(int(math.Round(float64(from)+(float64(to)-float64(from))*fraction)), rgbChannelMax)
}

func hsvToRGB(hue, saturation, value float64) (redValue, greenValue, blueValue int) {
	if saturation <= 0 {
		scaled := clampFromZero(int(math.Round(value*rgbChannelMax)), rgbChannelMax)

		return scaled, scaled, scaled
	}
	scaled := hue * hsvSectorCount
	sector := math.Floor(scaled)
	fraction := scaled - sector
	p := value * (1 - saturation)
	q := value * (1 - saturation*fraction)
	t := value * (1 - saturation*(1-fraction))

	switch int(sector) % hsvSectorCount {
	case 0:
		return scaleRGB(value), scaleRGB(t), scaleRGB(p)
	case 1:
		return scaleRGB(q), scaleRGB(value), scaleRGB(p)
	case hsvSectorTwo:
		return scaleRGB(p), scaleRGB(value), scaleRGB(t)
	case hsvSectorThree:
		return scaleRGB(p), scaleRGB(q), scaleRGB(value)
	case hsvSectorFour:
		return scaleRGB(t), scaleRGB(p), scaleRGB(value)
	default:
		return scaleRGB(value), scaleRGB(p), scaleRGB(q)
	}
}

func scaleRGB(value float64) int {
	return clampFromZero(int(math.Round(value*rgbChannelMax)), rgbChannelMax)
}

func rgbToANSI256(redValue, greenValue, blueValue int) int {
	redIndex := clampFromZero(int(math.Round(float64(redValue)/rgbChannelMax*ansiCubeChannelMax)), ansiCubeChannelMax)
	greenIndex := clampFromZero(int(math.Round(float64(greenValue)/rgbChannelMax*ansiCubeChannelMax)), ansiCubeChannelMax)
	blueIndex := clampFromZero(int(math.Round(float64(blueValue)/rgbChannelMax*ansiCubeChannelMax)), ansiCubeChannelMax)

	return ansiCubeColorBase + (ansiCubeRedWeight * redIndex) + (ansiCubeGreenWeight * greenIndex) + blueIndex
}
