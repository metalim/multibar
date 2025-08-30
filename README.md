# Multibar

> Note: This entire codebase was vibe-coded.

Lightweight terminal progress bars with multi-bar support and thread-safe updates.

## Features
- Multiple progress bars rendered together (multi-bar)
- Smooth partial block characters (▏▎▍▌▋▊▉█)
- Spinner while bars are running
- Percentage, elapsed time, and ETA
- ETA is hidden when a bar is finished
- Safe to update bars from multiple goroutines

## Install
```bash
go get github.com/metalim/multibar
```

## Quick start
```go
mb := multibar.New()
bar := mb.NewBar(100, "Downloading")
mb.Start()
for i := 0; i < 100; i++ {
    bar.Add(1)
    time.Sleep(10 * time.Millisecond)
}
// If max is undefined (multibar.Undefined), finish explicitly:
// bar.Finish()
```

## Multi-bar example
See `examples/multibar/main.go`.
```go
mb := multibar.New()
workBar := mb.NewBar(multibar.Undefined, "Working")
workersBar := mb.NewBar(len(files), fmt.Sprintf("Workers (0/%d)", len(files)))
bytesBar := mb.NewBar64(totalSize, "Total bytes")
mb.Start()
var wg sync.WaitGroup
for _, f := range files {
    fileBar := mb.NewBar64(f.Size, f.Name)
    wg.Add(1)
    go func(b *multibar.Bar, f File) {
        for j := int64(0); j < f.Size; j++ {
            workBar.Add(1)
            bytesBar.Add(1)
            b.Add(1)
            time.Sleep(10 * time.Millisecond)
        }
        workersBar.Add(1)
        workersBar.SetDescription(fmt.Sprintf("Workers (%d/%d)", workersBar.Value(), workersBar.Max()))
        wg.Done()
    }(fileBar, f)
}
wg.Wait()
workBar.Finish() // for the undefined bar
```

## API (essentials)
- `multibar.New(opts ...Option) *MultiBar`
  - `WithWriter(w io.Writer)` — redirect output (default `os.Stdout`)
- `(*MultiBar).NewBar(max int, desc string) *Bar`
- `(*MultiBar).NewBar64(max int64, desc string) *Bar`
- `(*MultiBar).Start()` — start rendering
- `(*Bar).Add(n int64)` — add progress
- `(*Bar).SetValue(v int64)` — set current value
- `(*Bar).SetMax(max int64)` — set max
- `(*Bar).SetDescription(desc string)` — change description
- `(*Bar).Finish()` — finish the bar (required for `Undefined`)
- `(*Bar).Value()`, `(*Bar).Max()` — getters
- Constant: `multibar.Undefined` — bar with unknown max

## Time behavior
- Elapsed freezes when the bar is finished
- ETA is hidden for finished bars
- Layout: percentage, elapsed (yellow), ETA (cyan)

## Thread-safety
- Bar mutations guarded by an internal `sync.Mutex`
- `MultiBar.render()` is serialized by a dedicated `renderMu` to avoid interleaved lines
- Access to `MultiBar` internals guarded by `mu`; render snapshots state under lock and prints without it

## Output format
```
⠴ <label padded> ███▊..................  69% 0:01:09 0:01:40
```
- For finished bars the spinner becomes a space, the bar turns green, and ETA is hidden

## License
MIT

