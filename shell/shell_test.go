package shell_test

import (
	"os"
	"strings"

	"github.com/alex-slynko/demoshell/shell"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Shell", func() {
	var (
		stdinReadPipe  *os.File
		stdinWritePipe *os.File
		out            *gbytes.Buffer
		oldStdout      *os.File
		player         *shell.LivePlayer
		username       string
	)

	BeforeEach(func() {
		var err error
		stdinReadPipe, stdinWritePipe, err = os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		out = gbytes.NewBuffer()
		player = &shell.LivePlayer{Out: out, In: stdinReadPipe}
		username = os.Getenv("USER")
		oldStdout = os.Stdout
	})

	AfterEach(func() {
		os.Setenv("USER", username)
		stdinReadPipe.Close()
		stdinWritePipe.Close()
		os.Stdout = oldStdout
	})

	Context("when user clicks enter", func() {
		BeforeEach(func() {
			stdinWritePipe.Write([]byte("\n"))
		})

		It("outputs contents of the file", func() {
			err := player.Run([]byte(`echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(gbytes.Say(`echo "Hello"`))
		})

		It("adds username to the output", func() {
			os.Setenv("USER", "testUser")
			err := player.Run([]byte(`echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(gbytes.Say(`testUser \$ echo "Hello"`))
		})

		It("outputs command result", func() {
			err := player.Run([]byte(`echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Split(string(out.Contents()), "\n")).To(ContainElement("Hello"))
		})

		It("does output but do not run comments", func() {
			err := player.Run([]byte(`#echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out.Contents())).To(Equal(`echo "Hello"`))
		})

		It("allows to run multiple commands", func() {
			stdinWritePipe.Write([]byte("\n"))
			err := player.Run([]byte(`echo "Hello"
echo "World"`))
			Expect(err).NotTo(HaveOccurred())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(out).To(gbytes.Say(`echo "Hello"`))
			Expect(out).To(gbytes.Say(`echo "World"`))
			Expect(output).To(ContainElement("Hello"))
			Expect(output).To(ContainElement("World"))
		})

		It("does respect multiline commands", func() {})

		It("skips empty lines", func() {
			ch := make(chan bool)
			go func() {
				defer GinkgoRecover()
				err := player.Run([]byte("\n\n\necho Hi"))
				Expect(err).NotTo(HaveOccurred())
				ch <- true
			}()
			Eventually(ch).Should(Receive())
		})
	})

	It("is interruptable by ctrl-c", func() {
	})

	It("waits for enter to execute command", func() {
		ch := make(chan bool)
		go func() {
			defer GinkgoRecover()
			err := player.Run([]byte("echo Hi"))
			Expect(err).NotTo(HaveOccurred())
			ch <- true
		}()
		Consistently(ch).ShouldNot(Receive())
		stdinWritePipe.Write([]byte("\n"))
		Eventually(ch).Should(Receive())
	})
})
