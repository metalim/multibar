package multibar

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

const (
	Undefined = -1
)

// Cell set for empty block, 7-step partials and a full block
var partialBlocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}
var spinners = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ANSI color codes (sorted by SGR code)
const (
	colorReset   = "\033[0m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	invertOn     = "\033[7m"
	invertOff    = "\033[27m"
)

func New() *MultiBar {
	return &MultiBar{}
}

type MultiBar struct {
	bars           []*Bar
	spinnerIndex   int
	lastRender     time.Time
	maxLabelLength int
	renderedLines  int
}

func (m *MultiBar) NewBar(max int64, description string) *Bar {
	b := &Bar{
		mb:          m,
		description: description,
		startedAt:   time.Now(),
	}
	b.max.Store(max)
	m.bars = append(m.bars, b)

	// Update max label length for alignment
	m.updateMaxLabelLength()

	return b
}

// updateMaxLabelLength recalculates the maximum label length for proper alignment
func (m *MultiBar) updateMaxLabelLength() {
	m.maxLabelLength = 0
	for _, b := range m.bars {
		desc := b.description
		if desc == "" {
			desc = "Working"
		}
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

func (m *MultiBar) render() {
	// Update spinner index based on time
	now := time.Now()
	if m.lastRender.IsZero() || now.Sub(m.lastRender) >= 100*time.Millisecond {
		m.spinnerIndex = (m.spinnerIndex + 1) % len(spinners)
		m.lastRender = now
	}

	// Move cursor up by number of lines previously rendered (skip on first render)
	if m.renderedLines > 0 {
		fmt.Printf("\033[%dA", m.renderedLines)
	}

	// Render each bar
	for _, bar := range m.bars {
		m.renderBar(bar)
		fmt.Println()
	}

	// Update rendered lines count
	m.renderedLines = len(m.bars)
}

func (m *MultiBar) renderBar(b *Bar) {
	value := b.value.Load()
	max := b.max.Load()

	// Spinner/label prep
	isGreenBar := b.finished || (max > 0 && value >= max)

	// Description - natural length, no fixed width
	description := b.description
	if description == "" {
		description = "Working"
	}

	// Calculate percentage - fixed width 4 characters
	var percentStr string
	if max > 0 {
		percent := int((value * 100) / max)
		percentStr = fmt.Sprintf("%3d%%", percent) // Fixed width: 3 digits + %
	} else {
		percentStr = "    " // Empty space for undefined progress (4 spaces)
	}

	// Calculate times
	elapsed := time.Since(b.startedAt)
	var estimatedStr string
	if max > 0 && value > 0 {
		// Estimated total time = elapsed * max / value
		estimated := time.Duration(float64(elapsed) * float64(max) / float64(value))
		estimatedStr = formatDuration(estimated)
	} else {
		estimatedStr = "       " // 7 spaces for H:MM:SS placeholder
	}

	// Ensure minimal width (7 characters like "0:00:00")
	if len(estimatedStr) < 7 && max > 0 {
		estimatedStr = " " + estimatedStr
	}

	// Build progress bar
	barWidth := 30 // Width of the progress bar
	barStr := m.buildProgressBar(value, max, barWidth, b.finished)

	// Format output with proper alignment based on max label length
	// Build fixed-width label area (description only), spinner printed separately
	spinner := " "
	if !isGreenBar {
		spinner = spinners[m.spinnerIndex]
	}
	descLen := utf8.RuneCountInString(description)
	if descLen > m.maxLabelLength {
		m.maxLabelLength = descLen
	}
	pad := m.maxLabelLength - descLen
	if pad < 0 {
		pad = 0
	}
	labelOut := description + strings.Repeat(" ", pad)

	// Print line: spinner, space, label, bar, percent, estimated, elapsed
	spinnerOut := spinner
	if !isGreenBar {
		spinnerOut = colorGreen + spinner + colorReset
	}
	fmt.Printf("%s %s %s %s %s",
		spinnerOut, // spinner (or space)
		labelOut,   // fixed-width description
		barStr,     // bar
		colorMagenta+percentStr+colorReset,
		colorCyan+estimatedStr+colorReset,
	)
	fmt.Printf(" %s", colorYellow+formatDuration(elapsed)+colorReset)
}

func (m *MultiBar) buildProgressBar(value, max int64, width int, finished bool) string {
	if max <= 0 {
		if finished {
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
	filledUnits := int((value * int64(totalUnits)) / max)
	if filledUnits > totalUnits {
		filledUnits = totalUnits
	}

	// Check if bar is completed (100%)
	isCompleted := finished || (max > 0 && value >= max)

	var barStr string
	if isCompleted {
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

		// Full filled characters
		filledStr = strings.Repeat(string(partialBlocks[8]), fullChars)

		// Partial character
		filledStr += string(partialBlocks[remainder])

		// Empty characters
		emptyChars := width - fullChars - 1
		emptyStr = strings.Repeat(string(partialBlocks[0]), emptyChars)

		return filledStr + emptyStr
	}
}

func formatDuration(d time.Duration) string {
	totalSeconds := int64(d.Seconds())
	if totalSeconds < 0 {
		totalSeconds = 0
	}
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

type Bar struct {
	mb          *MultiBar
	value, max  atomic.Int64
	startedAt   time.Time
	description string
	finished    bool
}

func (b *Bar) Reset() {
	b.value.Store(0)
	b.startedAt = time.Now()
	b.mb.render()
}

func (b *Bar) SetDescription(description string) {
	b.description = description
	b.mb.render()
}

func (b *Bar) SetValue(value int64) {
	b.value.Store(value)
	b.mb.render()
}

func (b *Bar) SetMax(max int64) {
	b.max.Store(max)
	b.mb.render()
}

func (b *Bar) Add(n int64) {
	b.value.Add(n)
	b.mb.render()
}

func (b *Bar) Finish() {
	b.value.Store(b.max.Load())
	b.finished = true
	b.mb.render()
}
