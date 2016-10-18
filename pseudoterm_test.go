// +build !windows

package pseudoterm

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kylelemons/godebug/diff"
)

type faultyMock struct{}

var errFaultyMock = errors.New("Faulty mock")

func (m *faultyMock) Setup() (ctx context.Context, err error) {
	return context.Background(), nil
}

func (m *faultyMock) Teardown() {}

func (m *faultyMock) TickHandler() (err error) {
	return nil
}

func (m *faultyMock) HandleLine(s string) (in string, err error) {
	return "", errFaultyMock
}

func (m *faultyMock) Success() bool {
	return false
}

func TestTerminalWithCat(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("cat"),
		EchoStream: echoStream,
	}

	if err := term.Start(); err != nil {
		t.Errorf("Expected no error during start, got %v instead", err)
	}

	if _, err := term.WriteString("Starting... "); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	if _, err := term.WriteLine("one"); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	if _, err := term.WriteLine("two"); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	if _, err := term.WriteLine("three"); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	var wg sync.WaitGroup

	wg.Add(1)

	var ps *os.ProcessState

	go func() {
		ps = term.Wait()
		wg.Done()
	}()

	if _, err := term.Write(EOT); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	wg.Wait()

	if ps == nil {
		t.Errorf("Expected ps not to be nil")
	}

	if !ps.Exited() {
		t.Errorf("Expected program to be exited already")
	}

	if err := term.Stop(); err != nil {
		t.Errorf("Expected no error during stop, got %v instead", err)
	}

	var log = `Starting... one
two
three
Starting... one
two
three
`
	assertSimilar(t, log, echoStream.String())

	if !term.processState.Exited() {
		t.Errorf("Expected process to have exited")
	}

	if !term.processState.Success() {
		t.Errorf("Expected process to have terminated successfully")
	}
}

func TestTerminalRunWithFaultyMock(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("mocks/mock.sh"),
		EchoStream: echoStream,
	}

	var story = &faultyMock{}

	if err := term.Run(story); err == nil {
		t.Errorf("Expected faulty mock %v, got %v instead", errFaultyMock, err)
	}
}

func TestTerminalWithStory(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("mocks/mock.sh"),
		EchoStream: echoStream,
	}

	var story = &QueueStory{
		Timeout: 5 * time.Second,
	}

	story.Add(Step{
		Read:      "Starting",
		SkipWrite: true,
	},
		Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		Step{
			Read:  "Your age:",
			Write: "10",
		})

	if err := term.Run(story); err != nil {
		t.Errorf("Expected no error during run, got %v instead", err)
	}

	var log = `Starting
Your name: Henrique
Your name is Henrique
Your age: 10
Your age is 10
Bye!`

	assertSimilar(t, log, echoStream.String())

	if !story.Success() {
		t.Errorf("Story didn't success.")
	}

	if !term.processState.Exited() {
		t.Errorf("Expected process to have exited")
	}

	if !term.processState.Success() {
		t.Errorf("Expected process to have terminated successfully")
	}
}

func TestTerminalWithComplexStory(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("mocks/mock-complex.sh"),
		EchoStream: echoStream,
	}

	var story = &QueueStory{
		Timeout: 5 * time.Second,
	}

	var numFromStep string

	story.Add(
		Step{
			Read:      "Starting",
			SkipWrite: true,
		},
		Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		Step{
			Read:  "Your age:",
			Write: "10",
		},
		Step{
			ReadRegex: regexp.MustCompile("p([a-z]+)ch"),
			Write:     "ok",
		},
		Step{
			ReadFunc: func(in string) bool {
				numFromStep = strings.TrimPrefix(in, "Random: ")
				return strings.HasPrefix(in, "Random: ")
			},
			Write: "ack",
		})

	if err := term.Run(story); err != nil {
		t.Errorf("Expected no error during run, got %v instead", err)
	}

	var log = `Starting
Your name: Henrique
Your name is Henrique
Your age: 10
Your age is 10
Do you want a peach? ok
peach: ok
Random: ` + numFromStep + `ack
num: ack
Bye!`

	assertSimilar(t, log, echoStream.String())

	if !story.Success() {
		t.Errorf("Story didn't success.")
	}

	if !term.processState.Exited() {
		t.Errorf("Expected process to have exited")
	}

	if !term.processState.Success() {
		t.Errorf("Expected process to have terminated successfully")
	}
}

func TestTerminalWithStoryShouldNotBlock(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("mocks/mock-mixed.sh"),
		EchoStream: echoStream,
	}

	var story = &QueueStory{
		Timeout: 6 * time.Second,
	}

	story.Add(Step{
		Read:      "Starting",
		SkipWrite: true,
	},
		Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		Step{
			Read:  "Your age:",
			Write: "10",
		})

	if err := term.Start(); err != nil {
		t.Errorf("Expected no error during start, got %v instead", err)
	}

	var w sync.WaitGroup

	w.Add(1)
	time.Sleep(250 * time.Millisecond)
	if _, err := term.WriteLine("yes"); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	go func() {
		err := term.Watch(story)

		if err != nil {
			t.Errorf("Expected no error during watch, got %v instead", err)
		}

		w.Done()
	}()

	time.Sleep(250 * time.Millisecond)
	if _, err := term.WriteLine("yes"); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	w.Wait()

	if err := term.Stop(); err != nil {
		t.Errorf("Expected no error during stop, got %v instead", err)
	}

	var log = `Continue? [no]: yes
Starting
Your name: Henrique
Your name is Henrique
Your age: 10
Your age is 10
Avoid killing itself? [no]: yes
Bye!`

	assertSimilar(t, log, echoStream.String())

	if !story.Success() {
		t.Errorf("Story didn't success.")
	}

	if !term.processState.Exited() {
		t.Errorf("Expected process to have exited")
	}

	if !term.processState.Success() {
		t.Errorf("Expected process to have terminated successfully")
	}
}

func TestTerminalWithAlreadyStartedStory(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("mocks/mock.sh"),
		EchoStream: echoStream,
	}

	var story = &QueueStory{
		Timeout: 5 * time.Second,
	}

	story.Add(Step{
		Read:      "Starting",
		SkipWrite: true,
	},
		Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		Step{
			Read:  "Your age:",
			Write: "10",
		})

	_, _ = story.Setup()

	if err := term.Start(); err != nil {
		t.Errorf("Expected no error during start, got %v instead", err)
	}

	if err := term.Watch(story); err != errAlreadyInitialized {
		t.Errorf("Expected error %v during run, got %v instead", errAlreadyInitialized, err)
	}
}

func TestTerminalWithStoryAndNoOutput(t *testing.T) {
	var term = &Terminal{
		Command: exec.Command("mocks/mock.sh"),
	}

	var story = &QueueStory{
		Timeout: 5 * time.Second,
	}

	story.Add(Step{
		Read:      "Starting",
		SkipWrite: true,
	},
		Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		Step{
			Read:  "Your age:",
			Write: "10",
		})

	if err := term.Run(story); err != nil {
		t.Errorf("Expected no error during run, got %v instead", err)
	}

	if term.EchoStream != nil {
		t.Errorf("Expected echo stream to be nil")
	}

	if !story.Success() {
		t.Errorf("Story didn't success.")
	}
}

func TestTerminalWithReadOnlyStory(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("mocks/read-only-mock.sh"),
		EchoStream: echoStream,
	}

	var story = &QueueStory{
		Timeout: 5 * time.Second,
	}

	if err := term.Run(story); err != nil {
		t.Errorf("Expected no error during run, got %v instead", err)
	}

	var log = `Hi!
Wait...
Bye!`

	assertSimilar(t, log, echoStream.String())

	if !story.Success() {
		t.Errorf("Story didn't success.")
	}
}

func TestTerminalWithStoryTimeout(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("mocks/mock-timeout.sh"),
		EchoStream: echoStream,
	}

	var story = &QueueStory{
		Timeout: 100 * time.Millisecond,
	}

	story.Add(Step{
		Read:      "Starting",
		SkipWrite: true,
	},
		Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		Step{
			Read:  "Your age:",
			Write: "10",
		})

	err := term.Run(story)

	switch err.(type) {
	case ExecutionError:
		var wantErr = "Run error: context deadline exceeded"
		if err.Error() != wantErr {
			t.Errorf("Wanted error to be %v, got %v instead", wantErr, err)
		}
	default:
		t.Errorf("Expected error to be of type ExecutionError, got error %v instead", err)
	}

	var log = `Starting
Your name: Henrique
Your name is Henrique
`

	assertSimilar(t, log, echoStream.String())

	var sequenceMissing = []Step{
		Step{
			Read:  "Your age:",
			Write: "10",
		},
	}

	if !reflect.DeepEqual(story.Sequence, sequenceMissing) {
		t.Errorf("Expected story sequence to contain only missing sequence, got %+v instead",
			story.Sequence)
	}

	if err := term.Start(); err == nil || err.Error() != "Already started" {
		t.Errorf(`Unexpected error %v, wanted "Already started" instead`, err)
	}

	if err := term.Run(story); err == nil || err.Error() != "Already started" {
		t.Errorf(`Unexpected error %v, wanted "Already started" instead`, err)
	}

	if story.Success() {
		t.Errorf("Story should have not succeeded.")
	}
}

func TestTerminalWithStepTimeout(t *testing.T) {
	var echoStream = &bytes.Buffer{}
	var term = &Terminal{
		Command:    exec.Command("mocks/mock-timeout.sh"),
		EchoStream: echoStream,
	}

	var story = &QueueStory{
		Timeout: 5 * time.Second,
	}

	story.Add(Step{
		Read:      "Starting",
		SkipWrite: true,
	},
		Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		Step{
			Read:    "Your age:",
			Write:   "10",
			Timeout: 1 * time.Millisecond,
		})

	err := term.Run(story)
	var wantErr = `Run error: Timed out while waiting for line "Your age:": timeout 1ms`

	if err == nil || err.Error() != wantErr {
		t.Errorf("Unexpected error: wanted %v, got %v instead", wantErr, err)
	}

	var log = `Starting
Your name: Henrique
Your name is Henrique
`

	assertSimilar(t, log, echoStream.String())

	if len(story.Sequence) != 1 || story.Sequence[0].Read != "Your age:" {
		t.Errorf("Expected story sequence to contain missing sequence, got %+v instead", story.Sequence)
	}

	if err := term.Start(); err == nil || err.Error() != "Already started" {
		t.Errorf(`Unexpected error %v, wanted "Already started" instead`, err)
	}

	if err := term.Run(story); err == nil || err.Error() != "Already started" {
		t.Errorf(`Unexpected error %v, wanted "Already started" instead`, err)
	}

	if story.Success() {
		t.Errorf("Story should have not succeeded.")
	}
}

func TestStoryTimeout(t *testing.T) {
	var story = &QueueStory{
		Timeout: 10 * time.Millisecond,
	}

	var sequence = []Step{
		Step{
			Read:  "Select from 1..2:",
			Write: "2",
		},
		Step{
			Read:    "Project:",
			Write:   "test",
			Timeout: time.Second,
		},
	}

	story.Add(sequence...)

	var ctx, err = story.Setup()

	if err != nil {
		t.Errorf("Expected story setup to be fine, got %v error instead", err)
	}

	if !reflect.DeepEqual(story.Sequence, sequence) {
		t.Errorf("Expected story sequence to be equal passed value")
	}

	time.Sleep(20 * time.Millisecond)

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected context error to be %v, got %v instead",
				context.DeadlineExceeded,
				ctx.Err())
		}
	default:
		t.Errorf("Expected context error due to story timeout, got %v instead", ctx.Err())
	}

	if _, err := story.Setup(); err != errAlreadyInitialized {
		t.Errorf("Wanted multiple initialization error to be %v, got %v instead",
			errAlreadyInitialized,
			err)
	}
}

func TestStepTimeout(t *testing.T) {
	var story = &QueueStory{}

	var sequence = []Step{
		Step{
			Read:    "Select from 1..2:",
			Write:   "2",
			Timeout: 10 * time.Millisecond,
		},
		Step{
			Read:  "Project:",
			Write: "test",
		},
	}

	story.Add(sequence...)

	var _, err = story.Setup()

	if err != nil {
		t.Errorf("Expected story setup to be fine, got %v error instead", err)
	}

	if !reflect.DeepEqual(story.Sequence, sequence) {
		t.Errorf("Expected story sequence to be equal passed value")
	}

	time.Sleep(20 * time.Millisecond)

	var wantErr = `Timed out while waiting for line "Select from 1..2:": timeout 10ms`

	if err := story.TickHandler(); err == nil ||
		err.Error() != wantErr {
		t.Errorf("Wanted err to be %v, got %v instead", wantErr, err)
	}

	select {
	case <-story.ctx.Done():
		if story.ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected context error to be %v, got %v instead",
				context.DeadlineExceeded,
				story.ctx.Err())
		}
	default:
		t.Errorf("Expected context error due to timeout, got %v instead", story.ctx.Err())
	}

	if _, err := story.Setup(); err != errAlreadyInitialized {
		t.Errorf("Wanted multiple initialization error to be %v, got %v instead",
			errAlreadyInitialized,
			err)
	}
}

func TestStoryCancel(t *testing.T) {
	var story = &QueueStory{
		Timeout: 10 * time.Millisecond,
	}

	var sequence = []Step{
		Step{
			Read:  "Select from 1..2:",
			Write: "2",
		},
	}

	story.Add(sequence...)

	var ctx, err = story.Setup()

	if err != nil {
		t.Errorf("Expected story setup to be fine, got %v error instead", err)
	}

	if !reflect.DeepEqual(story.Sequence, sequence) {
		t.Errorf("Expected story sequence to be equal passed value")
	}

	story.Cancel()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context error to be %v, got %v instead",
				context.DeadlineExceeded,
				ctx.Err())
		}
	default:
		t.Errorf("Expected context error due to story cancel, got %v instead", ctx.Err())
	}
}

func TestStoryHandleLineAndTeardown(t *testing.T) {
	var story = &QueueStory{
		Timeout: 4 * LineReaderInterval,
	}

	var sequence = []Step{
		Step{
			Read:  "Select from 1..2:",
			Write: "2",
		},
	}

	story.Add(sequence...)

	var ctx, err = story.Setup()

	if err != nil {
		t.Errorf("Expected story setup to be fine, got %v error instead", err)
	}

	if !reflect.DeepEqual(story.Sequence, sequence) {
		t.Errorf("Expected story sequence to be equal passed value")
	}

	if err := story.TickHandler(); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	if in, err := story.HandleLine("Select from 1..2:"); in != "2" || err != nil {
		t.Errorf("Expected values doens't match: (%v, %v)", in, err)
	}

	time.Sleep(LineReaderInterval)

	if err := story.TickHandler(); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	story.Teardown()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context error to be %v, got %v instead",
				context.DeadlineExceeded,
				ctx.Err())
		}
	default:
		t.Errorf("Expected context error due to story cancel, got %v instead", ctx.Err())
	}

	if !story.Success() {
		t.Errorf("Story didn't success.")
	}
}

func TestStoryAddAndInternalShift(t *testing.T) {
	var story = &QueueStory{}
	var addStep = Step{
		Read: "foo",
	}

	if _, err := story.Setup(); err != nil {
		t.Errorf("Error trying to setup story: %v", err)
	}

	story.Add(addStep)

	var step = story.shift()

	if len(story.Sequence) != 0 {
		t.Errorf("Expected sequence to have length 0, got %v instead", len(story.Sequence))
	}

	if step.Read != "foo" {
		t.Errorf("Wrong value on step")
	}

	var stepDummy = story.shift()

	if stepDummy.Read != "" {
		t.Errorf("Expected step to be dummy, got %+v instead", stepDummy)
	}

	if len(story.Sequence) != 0 {
		t.Errorf("Expected sequence to have length 0, got %v instead", len(story.Sequence))
	}

	if !story.Success() {
		t.Errorf("Story didn't success.")
	}
}

func assertSimilar(t *testing.T, want string, got string) {
	if w, g := normalize(want), normalize(got); w != g {
		t.Errorf(
			"Strings doesn't match after normalization:\n%s",
			diff.Diff(w, g))
	}
}

// Normalize string breaking lines with \n and removing extra spacing
// on the beginning and end of strings
func normalize(s string) string {
	s = strings.Replace(s, "^D", "", -1)
	var parts = strings.Split(s, "\n")
	var final = make([]string, 10*len(parts))

	var c = 0

	for p := range parts {
		var tp = strings.TrimSpace(parts[p])

		if tp != "" {
			final[c] = "\n"
			c++
		}

		final[c] = tp
		c++
	}

	return strings.TrimSpace(strings.Join(final, ""))
}
