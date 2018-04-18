package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	"github.com/alex-slynko/demoshell/shell"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <filename>", os.Args[0])
		os.Exit(1)
	}
	go func() {
		sigchan := make(chan os.Signal, 10)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan
		fmt.Println("Interrupted")
		os.Exit(10)
	}()
	bytes, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error when reading the file %v", err)
		os.Exit(1)
	}
	player := &shell.LivePlayer{Out: os.Stdout, In: os.Stdin}
	player.Run(bytes)
}
