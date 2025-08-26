package main

import (
	"fmt"
	"time"

	"github.com/metalim/multibar"
)

type File struct {
	Name string
	Size int64
}

var demoFiles = []File{
	{"file1.zip", 100},
	{"file2.zip", 200},
	{"file3.zip", 300},
	{"file4.zip", 400},
	{"file5.zip", 500},
}

func main() {
	mb := multibar.New()
	workBar := mb.NewBar(multibar.Undefined, "Working")
	totalBar := mb.NewBar(int64(len(demoFiles)), fmt.Sprintf("Files (0/%d)", len(demoFiles)))

	mb.Start()

	for _, file := range demoFiles {
		fileBar := mb.NewBar(file.Size, file.Name)
		for j := 0; j < int(file.Size); j++ {
			workBar.Add(1)
			fileBar.Add(1)
			time.Sleep(10 * time.Millisecond)
		}
		totalBar.Add(1)
		totalBar.SetDescription(fmt.Sprintf("Files (%d/%d)", totalBar.Value(), totalBar.Max()))
	}
	workBar.Finish()
}
