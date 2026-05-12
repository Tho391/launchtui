package ui

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// logLineMsg is sent into the bubbletea program every time a tailer reads a
// new line from a log file. It includes a stream label ("out"/"err") so the
// detail view can colour them differently.
type logLineMsg struct {
	Label  string // job label this line belongs to — used to ignore stale tailers
	Stream string
	Line   string
}

// logStoppedMsg signals that a tailer has been cancelled or hit a fatal
// error — never emitted on plain EOF (we keep polling on EOF).
type logStoppedMsg struct {
	Label string
	Err   error
}

// logTailer streams the tail of a single file into a tea.Program until ctx
// is cancelled. It survives truncation (e.g. logrotate replacing the file
// in place) by re-stat'ing every poll interval.
type logTailer struct {
	cancel context.CancelFunc
}

func (t *logTailer) stop() {
	if t != nil && t.cancel != nil {
		t.cancel()
	}
}

// startLogTailer spawns a goroutine that emits one logLineMsg per new line.
// `send` is the bubbletea Program.Send callback — passed in from update.go so
// this file does not need to know about the global program.
func startLogTailer(label, stream, path string, send func(tea.Msg)) *logTailer {
	ctx, cancel := context.WithCancel(context.Background())
	t := &logTailer{cancel: cancel}
	go tail(ctx, label, stream, path, send)
	return t
}

func tail(ctx context.Context, label, stream, path string, send func(tea.Msg)) {
	const pollEvery = 500 * time.Millisecond

	var (
		f      *os.File
		reader *bufio.Reader
		offset int64
		size   int64
	)
	defer func() {
		if f != nil {
			_ = f.Close()
		}
	}()

	open := func() bool {
		fi, err := os.Stat(path)
		if err != nil {
			return false
		}
		nf, err := os.Open(path)
		if err != nil {
			return false
		}
		// Seek to end on first open so we only show *new* lines —
		// otherwise opening a multi-GB log would dump it into the TUI.
		if f == nil {
			if _, err := nf.Seek(0, io.SeekEnd); err == nil {
				offset, _ = nf.Seek(0, io.SeekCurrent)
			}
		} else {
			_ = f.Close()
			// Reopened after a truncate/rename — start from byte 0.
			offset = 0
		}
		f = nf
		reader = bufio.NewReader(f)
		size = fi.Size()
		return true
	}

	if !open() {
		// File does not exist yet — keep polling. Real services often haven't
		// produced a log on first run.
	}

	tick := time.NewTicker(pollEvery)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			send(logStoppedMsg{Label: label, Err: ctx.Err()})
			return
		case <-tick.C:
		}

		if f == nil {
			if !open() {
				continue
			}
		}

		fi, err := os.Stat(path)
		if err != nil {
			_ = f.Close()
			f = nil
			continue
		}
		// File replaced or truncated — reopen.
		if fi.Size() < size {
			if !open() {
				continue
			}
		}
		size = fi.Size()

		for {
			line, err := reader.ReadString('\n')
			if line != "" {
				send(logLineMsg{Label: label, Stream: stream, Line: trimRight(line)})
				offset += int64(len(line))
			}
			if err != nil {
				break
			}
		}
	}
}

// trimRight drops a single trailing \r\n / \n so the UI does not render an
// empty line between entries.
func trimRight(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
