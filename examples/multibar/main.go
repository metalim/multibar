package main

import (
	"time"

	"github.com/metalim/multibar"
)

func main() {
	mb := multibar.New()
	b1 := mb.NewBar(1000, "Files")
	b2 := mb.NewBar(100, "File")
	b3 := mb.NewBar(multibar.Undefined, "Working")

	mb.Start()

	for i := range 1010 {
		b1.Add(1)
		b2.Add(1)
		b3.Add(1)
		if i%101 == 0 {
			b2.Reset()
		}
		time.Sleep(100 * time.Millisecond)
	}
	mb.FinishAll()
}
