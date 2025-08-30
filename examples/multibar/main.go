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
	filesBar := mb.NewBar(len(demoFiles), fmt.Sprintf("Files (0/%d)", len(demoFiles)))
	var totalSize int64
	for _, file := range demoFiles {
		totalSize += file.Size
	}
	bytesBar := mb.NewBar64(totalSize, "Total bytes")

	mb.Start()

	for _, file := range demoFiles {
		fileBar := mb.NewBar64(file.Size, file.Name)
		for j := 0; j < int(file.Size); j++ {
			workBar.Add(1)
			bytesBar.Add(1)
			fileBar.Add(1)
			time.Sleep(10 * time.Millisecond)
		}
		// bars automatically finish when max is reached
		filesBar.Add(1)
		filesBar.SetDescription(fmt.Sprintf("Files (%d/%d)", filesBar.Value(), filesBar.Max()))
	}
	// undefined bar has to be finished manually
	workBar.Finish()
}
