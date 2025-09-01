package multibar

import (
	"fmt"
	"io"
	"os"
	"sync"
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
	upN          = "\033[%dA"
	cursorOff    = "\033[?25l"
	cursorOn     = "\033[?25h"
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
	spinnerUpdate  time.Time
	maxLabelLength int
	renderedLines  int
	writer         io.Writer
	mu             sync.Mutex
	renderMu       sync.Mutex
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
	m.mu.Lock()
	m.bars = append(m.bars, b)
	m.mu.Unlock()

	// Update max label length for alignment
	m.updateMaxLabelLength(description)

	return b
}

// updateMaxLabelLength recalculates the maximum label length for proper alignment
func (m *MultiBar) updateMaxLabelLength(description string) {
	descLength := utf8.RuneCountInString(description)
	m.mu.Lock()
	if descLength > m.maxLabelLength {
		m.maxLabelLength = descLength
	}
	m.mu.Unlock()
}

// Start should be called after creating all bars to initialize rendering
func (m *MultiBar) Start() {
	m.render()
}

/*
	Output format:
	<spinner> <description> <bar> <percent> <elapsed> <estimated_total>

	Example:
	⠴ Downloading █████████████▉        69% 0:01:09 0:01:40

	Bar:
	- Working: default terminal color with partial-cell precision
	- Finished: green
	- Symbols:  ▏▎▍▌▋▊▉█

	Spinner:
	⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏
*/

const (
	spinnerRenderInterval = 100 * time.Millisecond
	barRenderInterval     = 10 * time.Millisecond
)

func (m *MultiBar) render(force ...bool) {
	// Serialize whole render to avoid interleaved output
	m.renderMu.Lock()
	defer m.renderMu.Unlock()

	m.mu.Lock()
	now := time.Now()
	if len(force) > 0 && !force[0] && !m.lastRender.IsZero() && now.Sub(m.lastRender) < barRenderInterval {
		m.mu.Unlock()
		return
	}
	if m.spinnerUpdate.IsZero() || now.Sub(m.spinnerUpdate) >= spinnerRenderInterval {
		m.spinnerIndex = (m.spinnerIndex + 1) % len(spinners)
		m.spinnerUpdate = now
	}
	m.lastRender = now
	moveUp := m.renderedLines > 0
	upLines := m.renderedLines
	writer := m.writer
	spinnerChar := spinners[m.spinnerIndex]
	maxLabel := m.maxLabelLength
	barsCopy := make([]*Bar, len(m.bars))
	copy(barsCopy, m.bars)
	m.renderedLines = len(barsCopy)
	m.mu.Unlock()

	fmt.Fprint(m.writer, cursorOff)
	if moveUp {
		fmt.Fprintf(writer, upN, upLines)
	}

	for _, bar := range barsCopy {
		bar.render(writer, spinnerChar, maxLabel)
		fmt.Fprintln(writer)
	}
	fmt.Fprint(m.writer, cursorOn)
}
