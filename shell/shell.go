package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	basher "github.com/progrium/go-basher"
)

type LivePlayer struct {
	Out     io.Writer
	In      io.Reader
	stderrR *os.File
	stderrW *os.File
	outC    chan string
}

func (l *LivePlayer) Run(script []byte) error {
	username := os.Getenv("USER")
	lines := bytes.Split(script, []byte("\n"))
	inReader := bufio.NewReader(l.In)
	var command []byte
	for _, line := range lines {
		if lineIsEmpty(line) {
			continue
		}
		if bytes.HasPrefix(line, []byte("#doitlive")) {
		} else if bytes.HasPrefix(line, []byte("#!")) {
		} else if bytes.HasPrefix(line, []byte("#")) {
			l.Out.Write(bytes.TrimLeft(line, "#"))
		} else {
			l.Out.Write([]byte(fmt.Sprintf("%s $ %s\n", username, line)))
			inReader.ReadString('\n')
			command = append(command, line...)
			command = append(command, []byte("\n")...)

			if !bytes.HasSuffix(line, []byte("\\")) {
				bash, err := l.setupBasher()
				if err != nil {
					return fmt.Errorf("bash could not be loaded: %s", err.Error())
				}

				bash.Source("", func(string) ([]byte, error) {
					return []byte(`set +e
(>&2 set -o posix; >&2 set)
>&2 echo "END VARIABLES BEFORE"`), nil
				})
				bash.Source("", func(string) ([]byte, error) { return []byte(command), nil })
				bash.Source("", func(string) ([]byte, error) {
					return []byte(`>&2 echo "START VARIABLES AFTER"
(>&2 set -o posix; >&2 set)`), nil
				})
				bash.Run("", []string{})

				l.updateEnvWithNew()
				command = []byte{}
			}
		}
	}
	return nil
}

func (l *LivePlayer) setupBasher() (*basher.Context, error) {
	bash, err := basher.NewContext("/bin/bash", true)
	if err != nil {
		return bash, err
	}
	bash.CopyEnv()
	bash.Stdout = l.Out
	bash.Stdin = l.In
	r, w, err := os.Pipe()
	l.stderrR = r
	l.outC = make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		l.outC <- buf.String()
	}()
	bash.Stderr = w
	return bash, err
}

func (l *LivePlayer) updateEnvWithNew() {
	l.stderrW.Close()
	l.stderrR.Close()
	out := <-l.outC

	variables := strings.Split(out, "\n")
	originalOnes := []string{}
	var commandEnd int
	for i, variable := range variables {
		if variable == "END VARIABLES BEFORE" {
			commandEnd = i
			break
		}
		originalOnes = append(originalOnes, variable)
	}
	for i, variable := range variables[commandEnd+1:] {
		if variable == "START VARIABLES AFTER" {
			commandEnd = i + commandEnd + 1
			break
		}
		fmt.Fprintln(os.Stderr, variable)
	}
	for _, variable := range variables[commandEnd+1:] {
		if !includesElement(originalOnes, variable) {
			pair := strings.SplitN(variable, "=", 2)
			if len(pair) == 2 {
				os.Setenv(pair[0], trimQuotes(pair[1]))
			}
		}
	}
}

func trimQuotes(quotedString string) string {
	if strings.HasPrefix(quotedString, "'") {
		last := len(quotedString) - 1
		fmt.Println(quotedString[1:last])
		fmt.Println(quotedString)
		return quotedString[1:last]
	}
	return quotedString
}

func lineIsEmpty(line []byte) bool {
	for _, character := range line {
		if character != ' ' && character != '\n' {
			return false
		}
	}
	return true
}

func includesElement(slices []string, element string) bool {
	for _, e := range slices {
		if e == element {
			return true
		}
	}
	return false
}
