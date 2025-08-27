package multibar

import "time"

type Bar struct {
	mb                   *MultiBar
	value, max           int64
	startedAt, updatedAt time.Time
	description          string
	finished             bool
}

func (b *Bar) label() string {
	if b.description == "" {
		return "Working"
	}
	return b.description
}

func (b *Bar) Reset() {
	b.value = 0
	b.startedAt = time.Now()
	b.updatedAt = b.startedAt
	b.mb.render()
}

func (b *Bar) SetDescription(description string) {
	b.description = description
	b.mb.updateMaxLabelLength()
	b.mb.render()
}

func (b *Bar) SetValue(value int64) {
	b.value = value
	b.updatedAt = time.Now()
	b.mb.render()
}

func (b *Bar) SetMax(max int64) {
	b.max = max
	b.mb.render()
}

func (b *Bar) Add(n int64) {
	b.value += n
	b.finished = b.value == b.max && b.max != Undefined
	b.updatedAt = time.Now()
	b.mb.render()
}

func (b *Bar) Finish() {
	if b.finished {
		return
	}
	b.finished = true
	b.updatedAt = time.Now()
	b.mb.render()
}

func (b *Bar) Value() int64 {
	return b.value
}

func (b *Bar) Max() int64 {
	return b.max
}
