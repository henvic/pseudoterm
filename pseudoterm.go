package pseudoterm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kr/pty"
)

var (
	// EOT is the End Of Transmission character
	EOT = []byte{4}

	// LineReaderInterval is the time to sleep between checks for output
	// Too low or too high mess things up (lower performance or high CPU usage).
	LineReaderInterval = 10 * time.Millisecond

	// SkipWrite is used as a return value from Story HandleLine to indicate
	// that a line should not be written when reading a line on a given step.
	SkipWrite = errors.New("Skip writing line input")

	// SkipZeroMatches is used as a return value from Story HandleLine to indicate
	// that there are no more steps left to be dealt with.
	SkipZeroMatches = errors.New("Skip line input due to no match available")

	// ErrUnsupported is used to indicate there is
	ErrUnsupported = pty.ErrUnsupported
)

// Terminal is a pseudo terminal you can use to run commands
// on a pseudo tty programatically
type Terminal struct {
	Command         *exec.Cmd
	EchoStream      io.Writer
	CopyStreamError error
	processState    *os.ProcessState
	terminal        *os.File
	bfs             *bytes.Buffer
	end             chan empty
}

// Story is interface you can implement to handle commands
type Story interface {
	Setup() (ctx context.Context, err error)
	Teardown()
	TickHandler() (err error)
	HandleLine(s string) (in string, err error)
}

// ExecutionError indicates if any happened during the execution
type ExecutionError struct {
	RunError     error
	SigtermError error
}

func (e ExecutionError) Error() string {
	var msgs []string
	if e.RunError != nil {
		msgs = append(msgs, "Run error: "+e.RunError.Error())
	}

	if e.SigtermError != nil {
		msgs = append(msgs, "SIGTERM signal error: "+e.SigtermError.Error())
	}

	return strings.Join(msgs, "; ")
}

type empty struct{}

// Run starts the program and handle lines printed by it
func (t *Terminal) Run(story Story) (err error) {
	if err = t.Start(); err != nil {
		return err
	}

	err = t.Watch(story)
	var et = t.Stop()

	if err == nil && et == nil {
		return nil
	}

	return ExecutionError{
		RunError:     err,
		SigtermError: et,
	}
}

// Stop the program
func (t *Terminal) Stop() (err error) {
	if t.processState == nil {
		if _, err = t.Write(EOT); err != nil {
			return err
		}

		return t.terminal.Close()
	}

	return err
}

// Start the program
func (t *Terminal) Start() (err error) {
	if t.terminal != nil {
		return errors.New("Already started")
	}

	t.end = make(chan empty, 1)
	t.terminal, err = pty.Start(t.Command)

	if err == nil {
		t.copyStreamToBuffer()

		go func() {
			// we don't care if process was terminated correctly or not
			// as we only care about having an open connection to it or not
			t.processState, _ = t.Command.Process.Wait()
			t.end <- empty{}
		}()
	}

	return err
}

// Wait for process to end and return process state
func (t *Terminal) Wait() (ps *os.ProcessState) {
	<-t.end
	return t.processState
}

// Write bytes to the pseudo terminal
func (t *Terminal) Write(b []byte) (n int, err error) {
	return t.terminal.Write(b)
}

// WriteString to the pseudo terminal
func (t *Terminal) WriteString(s string) (n int, err error) {
	return t.terminal.WriteString(s)
}

// WriteLine to the pseudo terminal
func (t *Terminal) WriteLine(s string) (n int, err error) {
	return t.terminal.WriteString(s + "\n")
}

// Watch starts handling lines printed by the program
func (t *Terminal) Watch(s Story) error {
	defer s.Teardown()
	var ctx, err = s.Setup()

	if err != nil {
		return err
	}

	var ok bool
	var endReadLine = make(chan empty, 1)

	go func() {
		ok, err = t.readLine(s)
		endReadLine <- empty{}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-endReadLine:
			if ok || err != nil {
				return err
			}

			go func() {
				ok, err = t.readLine(s)
				endReadLine <- empty{}
			}()
		default:
			time.Sleep(LineReaderInterval)
		}
	}
}

func (t *Terminal) copyStreamToBuffer() {
	t.bfs = &bytes.Buffer{}

	go func() {
		if t.EchoStream == nil {
			_, t.CopyStreamError = io.Copy(t.bfs, t.terminal)
		} else {
			var tee = io.TeeReader(t.terminal, t.EchoStream)
			_, t.CopyStreamError = io.Copy(t.bfs, tee)
		}
	}()
}

func (t *Terminal) readLine(s Story) (end bool, err error) {
	if t.processState != nil {
		return true, nil
	}

	line, err := t.bfs.ReadString('\n')

	if err != nil && err != io.EOF {
		return false, err
	}

	if err := s.TickHandler(); err != nil {
		return false, err
	}

	return t.handleLine(s)
}

func (t *Terminal) handleLine(s Story) (end bool, err error) {
	if len(line) != 0 {
		in, err := s.HandleLine(line)

		switch {
		case err == SkipWrite || err == SkipZeroMatches:
		case err == nil:
			if _, e := t.WriteLine(in); e != nil {
				return false, e
			}
		default:
			return false, err
		}
	}

	return false, nil
}

// QueueStory is a command execution story with sequential steps that must be fulfilled
type QueueStory struct {
	Sequence      []Step
	StepTimeout   time.Duration
	pastStepTime  time.Time
	ctx           context.Context
	ctxCancelFunc context.CancelFunc
}

// Step is like a route rule to handle lines
type Step struct {
	Read       string
	Write      string
	SkipWrite  bool
	Timeout    time.Duration
	timeoutCtx context.Context
}

var errAlreadyInitialized = errors.New("Story has already initialized")

// Add steps to a QueueStory
func (q *QueueStory) Add(args ...Step) {
	q.Sequence = append(q.Sequence, args...)
}

// Setup executed by Terminal on Watch()
func (q *QueueStory) Setup() (ctx context.Context, err error) {
	if q.ctx != nil {
		return nil, errAlreadyInitialized
	}

	q.pastStepTime = time.Now()
	q.ctx, q.ctxCancelFunc = context.WithCancel(context.Background())

	if q.StepTimeout != time.Duration(0) {
		q.ctx, q.ctxCancelFunc = context.WithTimeout(q.ctx, q.StepTimeout)
	}

	return q.ctx, nil
}

// Cancel Story
func (q *QueueStory) Cancel() {
	q.ctxCancelFunc()
}

// Teardown executed by Terminal during Watch() teardown
func (q *QueueStory) Teardown() {
	if q.ctxCancelFunc != nil {
		q.ctxCancelFunc()
	}
}

// TickHandler is called on terminal Watch between LineReaderInterval
// regardless if there are changes or not, before HandleLine
func (q *QueueStory) TickHandler() error {
	if len(q.Sequence) == 0 {
		return nil
	}

	var step = q.Sequence[0]

	if step.Timeout == time.Duration(0) {
		return nil
	}

	if time.Now().Before(q.pastStepTime.Add(step.Timeout)) {
		return nil
	}

	q.ctx, q.ctxCancelFunc = context.WithDeadline(q.ctx, time.Time{})

	return fmt.Errorf("Timed out while waiting for line \"%v\": timeout %v",
		q.Sequence[0].Read,
		q.Sequence[0].Timeout)
}

// HandleLine handles a QueueStory line the program prints
func (q *QueueStory) HandleLine(s string) (in string, err error) {
	if len(q.Sequence) == 0 {
		return "", SkipZeroMatches
	}

	if similar(s, q.Sequence[0].Read) {
		var step = q.shift()
		q.pastStepTime = time.Now()

		if step.SkipWrite {
			return "", SkipWrite
		}

		return step.Write, nil
	}

	return "", SkipWrite
}

// Success tells if all steps are executed and there is none left
func (q *QueueStory) Success() bool {
	return q.ctx != nil && len(q.Sequence) == 0
}

func (q *QueueStory) shift() Step {
	var step Step

	if len(q.Sequence) != 0 {
		step = q.Sequence[0]
		q.Sequence = q.Sequence[1:]
	} else {
		q.Sequence = []Step{}
	}

	return step
}

func similar(s, ref string) bool {
	return strings.TrimSpace(s) == strings.TrimSpace(ref)
}
