package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner represents a progress spinner
type Spinner struct {
	mu         sync.Mutex
	writer     io.Writer
	frames     []string
	frameIndex int
	message    string
	running    bool
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewSpinner creates a new spinner with default frames
func NewSpinner(message string) *Spinner {
	return &Spinner{
		writer:  os.Stdout,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		message: message,
	}
}

// SetWriter sets a custom writer for the spinner
func (s *Spinner) SetWriter(w io.Writer) {
	s.writer = w
}

// SetFrames sets custom spinner frames
func (s *Spinner) SetFrames(frames []string) {
	s.frames = frames
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go s.animate()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	s.wg.Wait()
	// Clear the line
	fmt.Fprint(s.writer, "\r\033[K")
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

func (s *Spinner) animate() {
	defer s.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.running {
				s.mu.Unlock()
				return
			}
			frame := s.frames[s.frameIndex%len(s.frames)]
			message := s.message
			s.frameIndex++
			s.mu.Unlock()

			fmt.Fprintf(s.writer, "\r%s %s", frame, message)
		}
	}
}

// ProgressBar represents a simple progress bar
type ProgressBar struct {
	mu       sync.Mutex
	writer   io.Writer
	width    int
	current  int
	total    int
	message  string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, message string) *ProgressBar {
	return &ProgressBar{
		writer:  os.Stdout,
		width:   40,
		total:   total,
		message: message,
	}
}

// SetWriter sets a custom writer
func (pb *ProgressBar) SetWriter(w io.Writer) {
	pb.writer = w
}

// SetWidth sets the bar width
func (pb *ProgressBar) SetWidth(width int) {
	pb.width = width
}

// Update updates the progress bar
func (pb *ProgressBar) Update(current int) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.current = current
	if pb.total <= 0 {
		return
	}

	percent := float64(current) / float64(pb.total)
	filled := int(percent * float64(pb.width))
	if filled > pb.width {
		filled = pb.width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", pb.width-filled)
	fmt.Fprintf(pb.writer, "\r%s [%s] %3.0f%% (%d/%d)", pb.message, bar, percent*100, current, pb.total)
}

// Increment increments the progress by 1
func (pb *ProgressBar) Increment() {
	pb.mu.Lock()
	current := pb.current + 1
	pb.mu.Unlock()
	pb.Update(current)
}

// Finish marks the progress as complete
func (pb *ProgressBar) Finish() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.current = pb.total
	bar := strings.Repeat("█", pb.width)
	fmt.Fprintf(pb.writer, "\r%s [%s] 100%% (%d/%d)\n", pb.message, bar, pb.total, pb.total)
}

// SimpleSpinner is a helper function for simple spinner use cases
func SimpleSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()
	err := fn()
	spinner.Stop()
	return err
}

// WithSpinner wraps a function with a spinner
func WithSpinner(message string, fn func() error) error {
	return SimpleSpinner(message, fn)
}
