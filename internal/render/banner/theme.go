package banner

import core "github.com/lmliam/remote-monitor/internal/core"

// Color stores one RGB color used by animated banner themes.
type Color struct {
	RedValue   int
	GreenValue int
	BlueValue  int
}

func bannerColorRGB(redValue, greenValue, blueValue int) Color {
	return Color{
		RedValue:   redValue,
		GreenValue: greenValue,
		BlueValue:  blueValue,
	}
}

type bannerAnimation string

const (
	bannerAnimationBasic  bannerAnimation = core.ThemeBasic
	bannerAnimationAurora bannerAnimation = core.ThemeAurora
)

type bannerTheme struct {
	lines       []string
	palette     []Color
	phaseMillis int64
	animation   bannerAnimation
}

// CellKind identifies how a banner cell should be rendered.
type CellKind int

const (
	// CellEmpty marks a transparent banner cell.
	CellEmpty CellKind = iota
	// CellShadow marks a shadow cell around banner face art.
	CellShadow
	// CellFace marks a visible banner glyph.
	CellFace
)

// Cell stores one glyph and its role in the aurora banner canvas.
type Cell struct {
	Glyph rune
	Kind  CellKind
}

const (
	basicBannerPhaseMillis  = 90
	auroraBannerPhaseMillis = 70
	// AuroraBackdropBandWidth controls the color band quantization used behind the aurora banner.
	AuroraBackdropBandWidth = 3
	auroraBackdropHalfStep  = 0.5
	auroraBackdropHalfBlock = "▀"
	panelInteriorMinSpace   = 2
	titleStatusPadding      = 2
	titleStatusMinLeftWidth = 8
	bannerRowWaveDivisor    = 2
	auroraWaveAmplitude     = 7
	auroraWavePhaseDivisor  = 5.0
	auroraWaveRowScale      = 0.8
	auroraPhaseStride       = 2
	auroraRowColorStride    = 5
	rgbChannelMin           = 0
	rgbChannelMax           = 255
	hsvDefaultSaturation    = 0.78
	hsvDefaultValue         = 1.0
	hsvSectorCount          = 6
	hsvSectorTwo            = 2
	hsvSectorThree          = 3
	hsvSectorFour           = 4
	ansiCubeChannelMax      = 5
	ansiCubeColorBase       = 16
	ansiCubeRedWeight       = 36
	ansiCubeGreenWeight     = 6
	noiseMidpoint           = 0.5
	noiseWaveAXScale        = 0.21
	noiseWaveATScale        = 0.92
	noiseWaveASeedScale     = 1.31
	noiseWaveBXScale        = 0.07
	noiseWaveBTScale        = 1.18
	noiseWaveBSeedScale     = 2.07
	noiseWaveCXScale        = 0.12
	noiseWaveCTScale        = 0.48
	noiseWaveCSeedScale     = 0.73
	noiseWeightA            = 0.46
	noiseWeightB            = 0.34
	noiseWeightC            = 0.20
	backdropTimeScale       = 820.0
	backdropWaveADivisor    = 7.0
	backdropWaveARowScale   = 0.28
	backdropWaveBDivisor    = 15.0
	backdropWaveBTimeScale  = 0.65
	backdropWaveBRowScale   = 0.52
	backdropWaveCRowScale   = 0.9
	backdropWaveCTimeScale  = 0.35
	backdropCenterDivisor   = 2
	backdropWaveAWeight     = 46
	backdropWaveBWeight     = 35
	backdropWaveCWeight     = 22
	faceColOffset           = 12
	faceRowStride           = 2
	faceDelayMillis         = 180
	faceHighlightRed        = 236
	faceHighlightGreen      = 255
	faceHighlightBlue       = 246
	faceMixBase             = 0.32
	faceMixGain             = 0.18
	faceMixWaveBase         = 0.5
	faceMixColDivisor       = 10.0
	faceMixRowScale         = 0.75
	faceMixTimeScale        = 620.0
	centerTextDivisor       = 2

	auroraFieldMillisScale = 1000.0
	fieldBaseRed           = 4
	fieldBaseGreen         = 9
	fieldBaseBlue          = 24

	ribbonGreenAnchor = 18
	ribbonGreenSlope  = 1.12
	ribbonGreenSway   = 18
	ribbonGreenSpeed  = 0.41
	ribbonGreenSpread = 10.5
	ribbonGreenGain   = 0.92
	ribbonGreenSeed   = 0.7
	ribbonGreenRed    = 52
	ribbonGreenGreen  = 236
	ribbonGreenBlue   = 138

	ribbonCyanAnchor = 56
	ribbonCyanSlope  = -0.58
	ribbonCyanSway   = 23
	ribbonCyanSpeed  = 0.29
	ribbonCyanSpread = 13.5
	ribbonCyanGain   = 0.78
	ribbonCyanSeed   = 1.9
	ribbonCyanRed    = 84
	ribbonCyanGreen  = 244
	ribbonCyanBlue   = 206

	ribbonPurpleAnchor = 96
	ribbonPurpleSlope  = 0.16
	ribbonPurpleSway   = 19
	ribbonPurpleSpeed  = 0.36
	ribbonPurpleSpread = 14.5
	ribbonPurpleGain   = 0.82
	ribbonPurpleSeed   = 3.4
	ribbonPurpleRed    = 146
	ribbonPurpleGreen  = 118
	ribbonPurpleBlue   = 255

	ribbonMagentaAnchor = 132
	ribbonMagentaSlope  = -0.26
	ribbonMagentaSway   = 25
	ribbonMagentaSpeed  = 0.27
	ribbonMagentaSpread = 16.5
	ribbonMagentaGain   = 0.76
	ribbonMagentaSeed   = 4.8
	ribbonMagentaRed    = 232
	ribbonMagentaGreen  = 102
	ribbonMagentaBlue   = 210

	ribbonCoralAnchor = 166
	ribbonCoralSlope  = 0.48
	ribbonCoralSway   = 20
	ribbonCoralSpeed  = 0.33
	ribbonCoralSpread = 15.5
	ribbonCoralGain   = 0.72
	ribbonCoralSeed   = 6.1
	ribbonCoralRed    = 255
	ribbonCoralGreen  = 94
	ribbonCoralBlue   = 124

	ribbonDriftRowScale  = 0.52
	ribbonDriftTimeScale = 0.61
	ribbonDriftSpeed     = 0.28
	ribbonSwayRowScale   = 0.08
	ribbonDriftCenter    = 0.5
	ribbonDriftAmplitude = 30
	gaussianSpreadFactor = 2
	pulseBase            = 0.52
	pulseGain            = 0.58
	pulseColScale        = 0.07
	pulseRowScale        = 0.11
	pulseTimeScale       = 0.66
	pulseSeedOffset      = 4.7

	washColDivisor        = 2
	washColAmplitude      = 10
	washColRowScale       = 0.11
	washColTimeScale      = 0.24
	washRowAmplitude      = 3
	washRowTimeScale      = 0.31
	washRowColScale       = 0.02
	washDelayMillis       = 220
	washMixBase           = 0.09
	washMixGain           = 0.08
	washMixColScale       = 0.05
	washMixRowScale       = 0.09
	washMixTimeScale      = 0.43
	washMixSeed           = 9.3
	hazeBase              = 0.07
	hazeGain              = 0.06
	hazeColScale          = 0.08
	hazeRowScale          = 0.13
	hazeTimeScale         = 0.37
	hazeSeed              = 12.8
	hazeRedContribution   = 26
	hazeGreenContribution = 44
	hazeBlueContribution  = 72
)

// BasicBannerLines returns the block-letter art for the basic banner theme.
func BasicBannerLines() []string {
	return []string{
		`██████╗ ███████╗███╗   ███╗ ██████╗ ████████╗███████╗    ███╗   ███╗ ██████╗ ███╗   ██╗██╗████████╗ ██████╗ ██████╗`,
		`██╔══██╗██╔════╝████╗ ████║██╔═══██╗╚══██╔══╝██╔════╝    ████╗ ████║██╔═══██╗████╗  ██║██║╚══██╔══╝██╔═══██╗██╔══██╗`,
		`██████╔╝█████╗  ██╔████╔██║██║   ██║   ██║   █████╗      ██╔████╔██║██║   ██║██╔██╗ ██║██║   ██║   ██║   ██║██████╔╝`,
		`██╔══██╗██╔══╝  ██║╚██╔╝██║██║   ██║   ██║   ██╔══╝      ██║╚██╔╝██║██║   ██║██║╚██╗██║██║   ██║   ██║   ██║██╔══██╗`,
		`██║  ██║███████╗██║ ╚═╝ ██║╚██████╔╝   ██║   ███████╗    ██║ ╚═╝ ██║╚██████╔╝██║ ╚████║██║   ██║   ╚██████╔╝██║  ██║`,
		`╚═╝  ╚═╝╚══════╝╚═╝     ╚═╝ ╚═════╝    ╚═╝   ╚══════╝    ╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝`,
	}
}

// AuroraFaceLines returns the face glyph art layered over the aurora backdrop.
func AuroraFaceLines() []string {
	return []string{
		` ██▀███  ▓█████  ███▄ ▄███▓ ▒█████  ▄▄▄█████▓▓█████     ███▄ ▄███▓ ▒█████   ███▄    █  ██▓▄▄▄█████▓ ▒█████   ██▀███  `,
		`▓██ ▒ ██▒▓█   ▀ ▓██▒▀█▀ ██▒▒██▒  ██▒▓  ██▒ ▓▒▓█   ▀    ▓██▒▀█▀ ██▒▒██▒  ██▒ ██ ▀█   █ ▓██▒▓  ██▒ ▓▒▒██▒  ██▒▓██ ▒ ██▒`,
		`▓██ ░▄█ ▒▒███   ▓██    ▓██░▒██░  ██▒▒ ▓██░ ▒░▒███      ▓██    ▓██░▒██░  ██▒▓██  ▀█ ██▒▒██▒▒ ▓██░ ▒░▒██░  ██▒▓██ ░▄█ ▒`,
		`▒██▀▀█▄  ▒▓█  ▄ ▒██    ▒██ ▒██   ██░░ ▓██▓ ░ ▒▓█  ▄    ▒██    ▒██ ▒██   ██░▓██▒  ▐▌██▒░██░░ ▓██▓ ░ ▒██   ██░▒██▀▀█▄  `,
		`░██▓ ▒██▒░▒████▒▒██▒   ░██▒░ ████▓▒░  ▒██▒ ░ ░▒████▒   ▒██▒   ░██▒░ ████▓▒░▒██░   ▓██░░██░  ▒██▒ ░ ░ ████▓▒░░██▓ ▒██▒`,
		`░ ▒▓ ░▒▓░░░ ▒░ ░░ ▒░   ░  ░░ ▒░▒░▒░   ▒ ░░   ░░ ▒░ ░   ░ ▒░   ░  ░░ ▒░▒░▒░ ░ ▒░   ▒ ▒ ░▓    ▒ ░░   ░ ▒░▒░▒░ ░ ▒▓ ░▒▓░`,
		`  ░▒ ░ ▒░ ░ ░  ░░  ░      ░  ░ ▒ ▒░     ░     ░ ░  ░   ░  ░      ░  ░ ▒ ▒░ ░ ░░   ░ ▒░ ▒ ░    ░      ░ ▒ ▒░   ░▒ ░ ▒░`,
		`  ░░   ░    ░   ░      ░   ░ ░ ░ ▒    ░         ░      ░      ░   ░ ░ ░ ▒     ░   ░ ░  ▒ ░  ░      ░ ░ ░ ▒    ░░   ░ `,
		`   ░        ░  ░       ░       ░ ░              ░  ░          ░       ░ ░           ░  ░               ░ ░     ░`,
	}
}

// Palette returns the color ramp used by the basic banner animation.
func Palette() []Color {
	const basicPaletteSteps = 120

	return buildBannerPalette(basicPaletteSteps)
}

func auroraBannerPalette() []Color {
	const (
		auroraBannerPaletteSteps = 160
		mintRed                  = 84
		mintGreen                = 250
		mintBlue                 = 220
		iceRed                   = 145
		iceGreen                 = 255
		iceBlue                  = 247
		skyRed                   = 117
		skyGreen                 = 201
		skyBlue                  = 255
		violetRed                = 169
		violetGreen              = 144
		violetBlue               = 255
		seafoamRed               = 96
		seafoamGreen             = 234
		seafoamBlue              = 188
	)

	return buildGradientBannerPalette(auroraBannerPaletteSteps, []Color{
		bannerColorRGB(mintRed, mintGreen, mintBlue),
		bannerColorRGB(iceRed, iceGreen, iceBlue),
		bannerColorRGB(skyRed, skyGreen, skyBlue),
		bannerColorRGB(violetRed, violetGreen, violetBlue),
		bannerColorRGB(seafoamRed, seafoamGreen, seafoamBlue),
	})
}

func auroraBackdropPalette() []Color {
	const (
		auroraBackdropPaletteSteps = 256
		nightRed                   = 5
		nightGreen                 = 9
		nightBlue                  = 26
		deepGreenRed               = 14
		deepGreenGreen             = 58
		deepGreenBlue              = 52
		ribbonGreenRed             = 48
		ribbonGreenGreen           = 224
		ribbonGreenBlue            = 146
		electricBlueRed            = 92
		electricBlueGreen          = 196
		electricBlueBlue           = 255
		purpleRed                  = 148
		purpleGreen                = 112
		purpleBlue                 = 255
		magentaRed                 = 225
		magentaGreen               = 98
		magentaBlue                = 216
		coralRed                   = 255
		coralGreen                 = 94
		coralBlue                  = 128
		forestRed                  = 28
		forestGreen                = 142
		forestBlue                 = 82
	)

	return buildGradientBannerPalette(auroraBackdropPaletteSteps, []Color{
		bannerColorRGB(nightRed, nightGreen, nightBlue),
		bannerColorRGB(deepGreenRed, deepGreenGreen, deepGreenBlue),
		bannerColorRGB(ribbonGreenRed, ribbonGreenGreen, ribbonGreenBlue),
		bannerColorRGB(electricBlueRed, electricBlueGreen, electricBlueBlue),
		bannerColorRGB(purpleRed, purpleGreen, purpleBlue),
		bannerColorRGB(magentaRed, magentaGreen, magentaBlue),
		bannerColorRGB(coralRed, coralGreen, coralBlue),
		bannerColorRGB(forestRed, forestGreen, forestBlue),
	})
}

// AuroraBannerCanvas returns the aurora banner face art as fixed-width cells.
func AuroraBannerCanvas() [][]Cell {
	return buildBannerCanvas(AuroraFaceLines())
}

func auroraBannerWidth() int {
	return bannerCanvasWidth(AuroraBannerCanvas())
}

func bannerThemeSpec(name string) bannerTheme {
	switch core.CanonicalThemeName(name) {
	case core.ThemeBasic:
		return bannerTheme{
			lines:       BasicBannerLines(),
			palette:     Palette(),
			phaseMillis: basicBannerPhaseMillis,
			animation:   bannerAnimationBasic,
		}
	default:
		return bannerTheme{
			lines:       AuroraFaceLines(),
			palette:     auroraBannerPalette(),
			phaseMillis: auroraBannerPhaseMillis,
			animation:   bannerAnimationAurora,
		}
	}
}
