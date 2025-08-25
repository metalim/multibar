package main

import (
	"fmt"
	"time"

	"github.com/metalim/multibar"
)

func main() {
	mb := multibar.New()
	b1 := mb.NewBar(1010, "Files")
	b2 := mb.NewBar(multibar.Undefined, "Working")
	var b3 *multibar.Bar

	mb.Start()

	for i := range 1010 {
		if i%101 == 0 {
			b3 = mb.NewBar(101, fmt.Sprintf("File %d", i/101+1))
		}
		b3.Add(1)
		b1.Add(1)
		b2.Add(1)
		time.Sleep(10 * time.Millisecond)
	}
	b2.Finish()
}
