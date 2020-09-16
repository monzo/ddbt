package utils

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	leftEdgeRune  = '▕'
	rightEdgeRune = '▏'
	filledRune    = '▇'
	blankRune     = '-'

	labelColumnWidth = 35
	rightColumnWidth = 30
	refreshRate      = 16 * time.Millisecond // 60 fps!
)

type ProgressBar struct {
	label           string
	completedItems  uint32
	numberItems     uint32
	output          io.Writer
	startTime       time.Time
	lastIncremented time.Time

	started         bool
	startMutex      sync.Mutex
	finishTicking   chan struct{}
	tickingFinished chan struct{}
	ticker          *time.Ticker

	statusRowMutex       sync.Mutex
	statusRowsLastRender int
	statusRows           []*StatusRow
}

func NewProgressBar(label string, numberItems int) *ProgressBar {
	pb := &ProgressBar{
		label:           label,
		completedItems:  0,
		numberItems:     uint32(numberItems),
		output:          os.Stdout,
		startTime:       time.Now(),
		lastIncremented: time.Now(),

		finishTicking:   make(chan struct{}),
		tickingFinished: make(chan struct{}),

		statusRows: make([]*StatusRow, 0),
	}

	pb.Start()

	return pb
}

func (pb *ProgressBar) Increment() {
	atomic.AddUint32(&pb.completedItems, 1)
	pb.lastIncremented = time.Now() // don't care about raising sets here, it's only for a rough guess of how fast we are
}

func (pb *ProgressBar) Width() int {
	width, err := TerminalWidth()

	if err != nil {
		// default width
		width = 400
	}

	return width
}

func (pb *ProgressBar) draw(includeStatusRows bool) {
	pb.statusRowMutex.Lock()
	defer pb.statusRowMutex.Unlock()

	termWidth := pb.Width()

	for i := 0; i < pb.statusRowsLastRender; i++ {
		// Clear the line
		_, _ = pb.output.Write([]byte("\r"))
		_, _ = pb.output.Write([]byte(strings.Repeat(" ", termWidth)))

		// Move up a line
		_, _ = pb.output.Write([]byte("\x1b[1A\x1b[2K"))
	}

	_, _ = pb.output.Write([]byte("\r"))
	_, _ = pb.output.Write([]byte(pb.String(termWidth)))

	if includeStatusRows {
		for _, row := range pb.statusRows {
			_, _ = pb.output.Write([]byte(row.String(termWidth)))
		}

		pb.statusRowsLastRender = len(pb.statusRows)
	} else {
		pb.statusRowsLastRender = 0
	}
}

func (pb *ProgressBar) Start() {
	pb.startMutex.Lock()

	if pb.started {
		pb.startMutex.Unlock()
		return
	}
	pb.started = true
	pb.startMutex.Unlock()

	pb.ticker = time.NewTicker(refreshRate)

	// Write the initial process of the bar
	_, _ = pb.output.Write([]byte(pb.String(pb.Width())))

	go pb.tick()
}

func (pb *ProgressBar) Stop() {
	pb.startMutex.Lock()
	defer pb.startMutex.Unlock()

	if !pb.started {
		return
	}

	pb.finishTicking <- struct{}{}
	<-pb.tickingFinished

	pb.started = false
}

func (pb *ProgressBar) tick() {
	for {
		select {
		case <-pb.ticker.C:
			// Draw an update in progress
			pb.draw(true)

		case <-pb.finishTicking:
			pb.ticker.Stop()

			// Draw the final update
			pb.draw(false)
			_, _ = pb.output.Write([]byte("\n"))

			pb.tickingFinished <- struct{}{}
			return
		}
	}
}

func (pb *ProgressBar) lastUpdateTime() time.Time {
	if pb.started {
		return time.Now()
	} else {
		return pb.lastIncremented
	}
}

func (pb *ProgressBar) String(termWidth int) string {
	completed := pb.completedItems // Because this is atomically updated, grab a local reference
	percentage := float64(completed) / float64(pb.numberItems)

	var builder strings.Builder

	// Draw the right hand edge first, so we know how many columns it will be in size
	numItemsStr := strconv.Itoa(int(pb.numberItems))
	compeltedStr := strconv.Itoa(int(completed))
	for i := len(compeltedStr); i < len(numItemsStr); i++ {
		builder.WriteRune(' ')
	}
	builder.WriteString(compeltedStr)
	builder.WriteRune('/')
	builder.WriteString(numItemsStr)

	// Write the time it's taken
	duration := pb.lastUpdateTime().Sub(pb.startTime)
	builder.WriteString(fmt.Sprintf(
		" [%02.0f:%02d]",
		math.Floor(duration.Minutes()),
		int64(duration.Seconds())%60,
	))

	// Display our operations per second
	builder.WriteString(fmt.Sprintf(
		" %6.0f op/s",
		(float64(completed)*1e9)/float64(duration.Nanoseconds()),
	))

	rightEdge := builder.String()

	if rightColumnWidth > len(rightEdge) {
		rightEdge = strings.Repeat(" ", rightColumnWidth-len(rightEdge)) + rightEdge
	}
	builder.Reset()

	// Draw the left hand edge
	builder.WriteString(pb.label)

	if toFill := labelColumnWidth - builder.Len(); toFill > 0 {
		builder.WriteString(strings.Repeat(" ", toFill)) // Create a buffer so that all the labels align
	}

	builder.WriteString(fmt.Sprintf("%3.0f%%", percentage*100))

	// Calculate the Percentage & number of bars to fill
	spaceForProgressBar := termWidth - builder.Len() - 2 - len(rightEdge) // (left/right edge runes)
	barsToFill := int(math.Round(float64(spaceForProgressBar) * percentage))

	// Draw the actual progress bar itself
	builder.WriteRune(leftEdgeRune)
	for i := 0; i < spaceForProgressBar; i++ {
		if barsToFill > i {
			builder.WriteRune(filledRune)
		} else {
			builder.WriteRune(blankRune)
		}
	}
	builder.WriteRune(rightEdgeRune)

	// Add the right edge text
	builder.WriteString(rightEdge)

	return builder.String()
}

func (pb *ProgressBar) NewStatusRow() *StatusRow {
	pb.statusRowMutex.Lock()
	defer pb.statusRowMutex.Unlock()

	sr := &StatusRow{isIdle: true, changed: time.Now()}
	pb.statusRows = append(pb.statusRows, sr)

	return sr
}

type StatusRow struct {
	m sync.Mutex

	message string
	changed time.Time
	isIdle  bool
}

func (sr *StatusRow) Update(message string) {
	sr.m.Lock()
	defer sr.m.Unlock()

	sr.message = message
	sr.changed = time.Now()
	sr.isIdle = false
}

func (sr *StatusRow) SetIdle() {
	sr.m.Lock()
	defer sr.m.Unlock()

	sr.isIdle = true
	sr.changed = time.Now()
	sr.message = "Idle"
}

func (sr *StatusRow) String(termWidth int) string {
	sr.m.Lock()
	defer sr.m.Unlock()

	var builder strings.Builder

	builder.WriteString("\n   ↳ ")

	if !sr.isIdle {
		duration := time.Now().Sub(sr.changed)
		builder.WriteString(fmt.Sprintf(
			"[%02.0f:%02d]",
			math.Floor(duration.Minutes()),
			int64(duration.Seconds())%60,
		))
	} else {
		builder.WriteString("[--:--]")
	}
	builder.WriteString(" ")

	builder.WriteString(sr.message)

	// Overwrite any old characters from the previous render
	spaces := termWidth - builder.Len()
	if spaces > 0 {
		builder.WriteString(strings.Repeat(" ", spaces))
	}

	return builder.String()
}
