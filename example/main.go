package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/henvic/pseudoterm"
)

func main() {
	var term = &pseudoterm.Terminal{
		Command:    exec.Command("bash", "-c", "./example.sh"),
		EchoStream: os.Stdout,
	}

	var story = &pseudoterm.QueueStory{
		Timeout: 5 * time.Second,
	}

	story.Add(pseudoterm.Step{
		Read:      "Starting",
		SkipWrite: true,
	},
		pseudoterm.Step{
			Read:  "Your name:",
			Write: "Henrique",
		},
		pseudoterm.Step{
			Read:  "Your age:",
			Write: "10",
		})

	if err := term.Run(story); err != nil {
		println(err.Error())
	}

	fmt.Println(term.Wait())

	// Output:
	// Starting
	// Your name: Henrique
	// Your name is Henrique
	// Your age: 10
	// Your age is 10
	// Bye!
	// exit status 0
}
