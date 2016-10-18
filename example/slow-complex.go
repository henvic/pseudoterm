package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/henvic/pseudoterm"
)

func main() {
	var term = &pseudoterm.Terminal{
		Command:    exec.Command("bash", "-c", "./slow-complex.sh"),
		EchoStream: os.Stdout,
	}

	var story = &pseudoterm.QueueStory{
		Timeout: 5 * time.Second,
	}

	var numFromStep string

	story.Add(
		pseudoterm.Step{
			Read:      "Starting",
			SkipWrite: true,
		},
		pseudoterm.Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		pseudoterm.Step{
			Read:    "Your age:",
			Write:   "10",
			Timeout: 200 * time.Millisecond,
		},
		pseudoterm.Step{
			ReadRegex: regexp.MustCompile("p([a-z]+)ch"),
			Write:     "ok",
		},
		pseudoterm.Step{
			ReadFunc: func(in string) bool {
				numFromStep = strings.TrimPrefix(in, "Random: ")
				return strings.HasPrefix(in, "Random: ")
			},
			Write: "ack",
		})

	if err := term.Run(story); err != nil {
		println(err.Error())
	}

	fmt.Fprintf(os.Stdout, "\nStory executed successfully: %v\n", story.Success())
}
