package multibar

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const Undefined = -1

// ANSI color codes (sorted by SGR code)
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	invertOn     = "\033[7m"
	invertOff    = "\033[27m"
)

var (
	partialBlocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}
	spinners      = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

type Option func(*MultiBar)

func WithWriter(w io.Writer) Option {
	return func(m *MultiBar) {
		m.writer = w
	}
}

func New(opts ...Option) *MultiBar {
	m := &MultiBar{
		writer: os.Stdout,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type MultiBar struct {
	bars           []*Bar
	spinnerIndex   int
	lastRender     time.Time
	maxLabelLength int
	renderedLines  int
	writer         io.Writer
}

func (m *MultiBar) NewBar(max int64, description string) *Bar {
	b := &Bar{
		mb:          m,
		max:         max,
		description: description,
		startedAt:   time.Now(),
	}
	m.bars = append(m.bars, b)

	// Update max label length for alignment
	m.updateMaxLabelLength()

	return b
}

// updateMaxLabelLength recalculates the maximum label length for proper alignment
func (m *MultiBar) updateMaxLabelLength() {
	m.maxLabelLength = 0
	for _, b := range m.bars {
		desc := b.label()
		// Calculate max description width (no ANSI codes)
		labelLength := utf8.RuneCountInString(desc)
		if labelLength > m.maxLabelLength {
			m.maxLabelLength = labelLength
		}
	}
}

// Start should be called after creating all bars to initialize rendering
func (m *MultiBar) Start() {
	m.render()
}

// FinishAll marks all bars as finished
func (m *MultiBar) FinishAll() {
	for _, bar := range m.bars {
		bar.Finish()
	}
}

/*
	Output format:
	<spinner> <description> <bar> <percent> <estimated_total> <elapsed>

	Example:
	⠴ Downloading █████████████▉        69% 0:01:40 0:01:09

	Bar:
	- Working: default terminal color with partial-cell precision
	- Finished: green

	Spinner:
	⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏
*/

const (
	spinnerRenderInterval = 100 * time.Millisecond
)

func (m *MultiBar) render() {
	// Update spinner index based on time
	now := time.Now()
	if m.lastRender.IsZero() || now.Sub(m.lastRender) >= spinnerRenderInterval {
		m.spinnerIndex = (m.spinnerIndex + 1) % len(spinners)
		m.lastRender = now
	}

	// Move cursor up by number of lines previously rendered (skip on first render)
	if m.renderedLines > 0 {
		fmt.Fprintf(m.writer, "\033[%dA", m.renderedLines)
	}

	// Render each bar
	for _, bar := range m.bars {
		m.renderBar(bar)
		fmt.Fprintln(m.writer)
	}

	// Update rendered lines count
	m.renderedLines = len(m.bars)
}

func (m *MultiBar) renderBar(b *Bar) {
	isError := b.max != Undefined && b.value > b.max

	description := b.label()

	// Calculate percentage - fixed width 4 characters
	var percentStr string
	if b.finished && b.max != Undefined {
		percentStr = "100%"
	} else if b.max != Undefined {
		percent := int((b.value * 100) / b.max)
		percentStr = fmt.Sprintf("%3d%%", percent) // Fixed width: 3 digits + %
	} else {
		percentStr = "    " // Empty space for undefined progress (4 spaces)
	}

	// Calculate times
	elapsed := time.Since(b.startedAt)
	var estimatedStr string
	if b.finished && b.max != Undefined {
		estimatedStr = formatDuration(elapsed)
	} else if b.max != Undefined && b.value > 0 {
		// Estimated total time = elapsed * max / value
		estimated := time.Duration(float64(elapsed) * float64(b.max) / float64(b.value))
		estimatedStr = formatDuration(estimated)
	} else {
		estimatedStr = "       " // 7 spaces for H:MM:SS placeholder
	}

	// Ensure minimal width (7 characters like "0:00:00")
	if len(estimatedStr) < 7 && b.max > 0 {
		estimatedStr = " " + estimatedStr
	}

	// Build progress bar
	barWidth := 30 // Width of the progress bar
	barStr := m.buildProgressBar(b.value, b.max, barWidth, b.finished, isError)

	// Format output with proper alignment based on max label length
	// Build fixed-width label area (description only), spinner printed separately
	spinner := " "
	if !b.finished {
		spinner = spinners[m.spinnerIndex]
	}

	descLen := utf8.RuneCountInString(description)
	pad := m.maxLabelLength - descLen
	if pad < 0 {
		pad = 0
	}
	labelOut := description + strings.Repeat(" ", pad)

	// Print line: spinner, space, label, bar, percent, estimated, elapsed
	var spinnerOut string
	switch {
	case isError:
		spinnerOut = colorRed + spinner + colorReset
	case b.finished:
		spinnerOut = colorGreen + spinner + colorReset
	default:
		spinnerOut = spinner
	}

	fmt.Fprintf(m.writer, "%s %s %s %s %s %s",
		spinnerOut, // spinner (or space)
		labelOut,   // fixed-width description
		barStr,     // bar
		colorMagenta+percentStr+colorReset,
		colorCyan+estimatedStr+colorReset,
		colorYellow+formatDuration(elapsed)+colorReset,
	)
}

func (m *MultiBar) buildProgressBar(value, maxVal int64, width int, isFinished bool, isError bool) string {
	if maxVal == Undefined {
		if isFinished {
			// Finished undefined: full green bar
			barStr := strings.Repeat(string(partialBlocks[8]), width)
			return colorGreen + barStr + colorReset
		}
		// Indeterminate progress: tri-symbol marker advances by 1 gradation per unit
		totalUnits := width * 8
		if totalUnits <= 0 {
			return ""
		}
		u := int(value % int64(totalUnits))
		if u < 0 {
			u += totalUnits
		}
		center := u / 8
		rem := u % 8 // 0..7

		var b strings.Builder
		for i := 0; i < width; i++ {
			switch {
			case i == center-1:
				// Left partial inverted
				b.WriteString(invertOn)
				b.WriteRune(partialBlocks[rem])
				b.WriteString(invertOff)
			case i == center:
				// Full block
				b.WriteRune(partialBlocks[8])
			case i == center+1:
				// Right partial normal
				b.WriteRune(partialBlocks[rem])
			default:
				b.WriteRune(' ')
			}
		}

		return b.String()
	}

	// Calculate filled portion in terms of total units (width * 8) using integer math
	totalUnits := width * 8
	filledUnits := int((value * int64(totalUnits)) / maxVal)

	var barStr string
	if isFinished {
		// Completed bar - green
		barStr = strings.Repeat(string(partialBlocks[8]), width)
		return colorGreen + barStr + colorReset
	} else {
		// Working bar - default terminal color
		filledStr := ""
		emptyStr := ""

		// Calculate how many characters are fully filled and the remainder
		fullChars := filledUnits / 8
		remainder := filledUnits % 8

		if fullChars >= width {
			fullChars = width
			remainder = 0
		}

		// Full filled characters
		filledStr = strings.Repeat(string(partialBlocks[8]), fullChars)

		// Partial character only if there is room
		extra := 0
		if remainder > 0 && fullChars < width {
			filledStr += string(partialBlocks[remainder])
			extra = 1
		}

		// Empty characters
		emptyChars := width - fullChars - extra
		if emptyChars < 0 {
			emptyChars = 0
		}
		emptyStr = strings.Repeat(string(partialBlocks[0]), emptyChars)

		if isError {
			return colorRed + filledStr + emptyStr + colorReset
		}
		return filledStr + emptyStr
	}
}

func formatDuration(d time.Duration) string {
	totalSeconds := int64(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}
