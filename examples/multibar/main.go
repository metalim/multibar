package main

import (
	"fmt"
	"sync"
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
	workersBar := mb.NewBar(len(demoFiles), fmt.Sprintf("Workers (0/%d)", len(demoFiles)))
	var totalSize int64
	for _, file := range demoFiles {
		totalSize += file.Size
	}
	bytesBar := mb.NewBar64(totalSize, "Total bytes")

	mb.Start()

	var wg sync.WaitGroup
	for _, file := range demoFiles {
		fileBar := mb.NewBar64(file.Size, file.Name)
		wg.Add(1)
		go func(b *multibar.Bar, f File) {
			for j := 0; j < int(file.Size); j++ {
				workBar.Add(1)
				bytesBar.Add(1)
				b.Add(1)
				time.Sleep(10 * time.Millisecond)
			}
			// bars automatically finish when max is reached
			workersBar.Add(1)
			workersBar.SetDescription(fmt.Sprintf("Workers (%d/%d)", workersBar.Value(), workersBar.Max()))
			wg.Done()
		}(fileBar, file)
	}
	wg.Wait()
	// undefined bar has to be finished manually
	workBar.Finish()
}
