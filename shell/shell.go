package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	basher "github.com/progrium/go-basher"
)

type LivePlayer struct {
	Out io.Writer
	In  io.Reader
}

func (l *LivePlayer) Run(script []byte) error {
	username := os.Getenv("USER")
	lines := bytes.Split(script, []byte("\n"))
	inReader := bufio.NewReader(l.In)
	for _, line := range lines {
		if lineIsEmpty(line) {
			continue
		}
		if bytes.HasPrefix(line, []byte("#")) {
			l.Out.Write(bytes.TrimLeft(line, "#"))
		} else {
			l.Out.Write([]byte(fmt.Sprintf("%s $ %s\n", username, line)))
			inReader.ReadString('\n')
			bash, err := l.setupBasher()
			if err != nil {
				return fmt.Errorf("bash could not be loaded: %s", err.Error())
			}

			bash.Source("", func(string) ([]byte, error) { return []byte(line), nil })
			bash.Run("", []string{})
		}
	}
	return nil
}

func (l *LivePlayer) setupBasher() (*basher.Context, error) {
	bash, err := basher.NewContext("/bin/bash", false)
	bash.Stdout = l.Out
	bash.Stdin = l.In
	return bash, err
}

func lineIsEmpty(line []byte) bool {
	for _, character := range line {
		if character != ' ' && character != '\n' {
			return false
		}
	}
	return true
}
