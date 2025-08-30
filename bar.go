package multibar

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	partialBlocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}
	spinners      = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

type Bar struct {
	mb                   multiBarInterface
	value, max           int64
	startedAt, updatedAt time.Time
	description          string
	finished             bool
	mu                   sync.Mutex
}

type multiBarInterface interface {
	updateMaxLabelLength()
	render()
}

func (b *Bar) label() string {
	b.mu.Lock()
	d := b.description
	b.mu.Unlock()
	if d == "" {
		return "Working"
	}
	return d
}

func (b *Bar) Reset() {
	b.mu.Lock()
	b.value = 0
	b.startedAt = time.Now()
	b.updatedAt = b.startedAt
	b.mu.Unlock()
	b.mb.render()
}

func (b *Bar) SetDescription(description string) {
	b.mu.Lock()
	b.description = description
	b.mu.Unlock()
	b.mb.updateMaxLabelLength()
	b.mb.render()
}

func (b *Bar) SetValue(value int64) {
	b.mu.Lock()
	b.value = value
	b.updatedAt = time.Now()
	b.mu.Unlock()
	b.mb.render()
}

func (b *Bar) SetMax(max int64) {
	b.mu.Lock()
	b.max = max
	b.mu.Unlock()
	b.mb.render()
}

func (b *Bar) Add(n int64) {
	b.mu.Lock()
	b.value += n
	b.finished = b.value == b.max && b.max != Undefined
	b.updatedAt = time.Now()
	b.mu.Unlock()
	b.mb.render()
}

func (b *Bar) Finish() {
	b.mu.Lock()
	if b.finished {
		b.mu.Unlock()
		return
	}
	b.updatedAt = time.Now()
	b.finished = true
	b.mu.Unlock()
	b.mb.render()
}

func (b *Bar) render(w io.Writer, spinner string, maxLabelLength int) {
	b.mu.Lock()
	isError := b.max != Undefined && b.value > b.max
	description := b.description
	value := b.value
	maxVal := b.max
	finished := b.finished
	startedAt := b.startedAt
	updatedAt := b.updatedAt
	b.mu.Unlock()

	if description == "" {
		description = "Working"
	}

	// Calculate percentage - fixed width 4 characters
	var percentStr string
	if finished && maxVal != Undefined {
		percentStr = "100%"
	} else if maxVal != Undefined {
		percent := int((value * 100) / maxVal)
		percentStr = fmt.Sprintf("%3d%%", percent) // Fixed width: 3 digits + %
	} else {
		percentStr = "    " // Empty space for undefined progress (4 spaces)
	}

	// Calculate times
	var elapsed time.Duration
	if finished {
		if !updatedAt.IsZero() {
			elapsed = updatedAt.Sub(startedAt)
		} else {
			elapsed = time.Since(startedAt)
		}
	} else {
		elapsed = time.Since(startedAt)
	}
	var estimatedStr string
	if finished {
		estimatedStr = "       "
	} else if maxVal != Undefined && value > 0 {
		// Estimated total time = elapsed * max / value
		estimated := time.Duration(float64(elapsed) * float64(maxVal) / float64(value))
		estimatedStr = formatDuration(estimated)
	} else {
		estimatedStr = "       " // 7 spaces for H:MM:SS placeholder
	}

	// Ensure minimal width (7 characters like "0:00:00")
	if len(estimatedStr) < 7 && maxVal > 0 {
		estimatedStr = " " + estimatedStr
	}

	// Build progress bar
	barWidth := 30 // Width of the progress bar
	barStr := b.buildProgressBar(value, maxVal, barWidth, finished, isError)

	// Format output with proper alignment based on max label length
	// Build fixed-width label area (description only), spinner printed separately
	if finished {
		spinner = " "
	}

	descLen := utf8.RuneCountInString(description)
	pad := maxLabelLength - descLen
	if pad < 0 {
		pad = 0
	}
	labelOut := description + strings.Repeat(" ", pad)

	// Print line: spinner, space, label, bar, percent, elapsed, estimated
	var spinnerOut string
	switch {
	case isError:
		spinnerOut = colorRed + spinner + colorReset
	case finished:
		spinnerOut = colorGreen + spinner + colorReset
	default:
		spinnerOut = spinner
	}

	fmt.Fprintf(w, "%s %s %s %s %s %s",
		spinnerOut, // spinner (or space)
		labelOut,   // fixed-width description
		barStr,     // bar
		colorMagenta+percentStr+colorReset,
		colorYellow+formatDuration(elapsed)+colorReset,
		colorCyan+estimatedStr+colorReset,
	)
}

func (b *Bar) buildProgressBar(value, maxVal int64, width int, isFinished bool, isError bool) string {
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

		var sb strings.Builder
		for i := 0; i < width; i++ {
			switch {
			case i == center-1:
				// Left partial inverted
				sb.WriteString(invertOn)
				sb.WriteRune(partialBlocks[rem])
				sb.WriteString(invertOff)
			case i == center:
				// Full block
				sb.WriteRune(partialBlocks[8])
			case i == center+1:
				// Right partial normal
				sb.WriteRune(partialBlocks[rem])
			default:
				sb.WriteRune(' ')
			}
		}

		return sb.String()
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

func (b *Bar) Value() int64 {
	b.mu.Lock()
	v := b.value
	b.mu.Unlock()
	return v
}

func (b *Bar) Max() int64 {
	b.mu.Lock()
	m := b.max
	b.mu.Unlock()
	return m
}

func formatDuration(d time.Duration) string {
	totalSeconds := int64(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}
