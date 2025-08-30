package multibar

import (
	"fmt"
	"io"
	"os"
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

func (m *MultiBar) NewBar(maxValue int, description string) *Bar {
	return m.NewBar64(int64(maxValue), description)
}

func (m *MultiBar) NewBar64(maxValue int64, description string) *Bar {
	b := &Bar{
		mb:          m,
		max:         maxValue,
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

// FinishAll marks all bars as finished. But why?
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
		bar.render(m.writer, spinners[m.spinnerIndex], m.maxLabelLength)
		fmt.Fprintln(m.writer)
	}

	// Update rendered lines count
	m.renderedLines = len(m.bars)
}
