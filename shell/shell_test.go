package shell_test

import (
	"os"
	"strings"

	"github.com/alex-slynko/demoshell/shell"
	. "github.com/onsi/ginkgo/v2"
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
		oldEnv         []string
		stderr         *gbytes.Buffer
	)

	BeforeEach(func() {
		var err error
		stderr = gbytes.NewBuffer()
		stdinReadPipe, stdinWritePipe, err = os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		out = gbytes.NewBuffer()
		player = &shell.LivePlayer{Out: out, In: stdinReadPipe, Err: stderr}
		oldStdout = os.Stdout
		oldEnv = os.Environ()
	})

	AfterEach(func() {
		stdinReadPipe.Close()
		stdinWritePipe.Close()
		os.Stdout = oldStdout
		for _, variable := range oldEnv {
			pair := strings.SplitN(variable, "=", 2)
			os.Setenv(pair[0], pair[1])
		}
	})

	Context("when user clicks enter", func() {
		BeforeEach(func() {
			stdinWritePipe.Write([]byte("\n"))
		})

		It("outputs to stderr if required", func() {
			Expect(player.Run([]byte(`time`))).To(Succeed())
			Expect(stderr.Contents()).NotTo(BeEmpty())
		})

		It("redirects streams", func() {
			Expect(player.Run([]byte(`>&2 echo Hello`))).To(Succeed())
			Expect(strings.Split(string(out.Contents()), "\n")).NotTo(ContainElement("Hello"))
			Expect(stderr.Contents()).NotTo(BeEmpty())
		})

		It("outputs contents of the file", func() {
			Expect(player.Run([]byte(`echo "Hello"`))).To(Succeed())
			Expect(out).To(gbytes.Say(`echo "Hello"`))
		})

		It("adds username to the output", func() {
			os.Setenv("DEMOUSER", "testUser")
			Expect(player.Run([]byte(`echo "Hello"`))).To(Succeed())
			Expect(out).To(gbytes.Say(`testUser.*\$ echo "Hello"`))
		})

		It("outputs command result", func() {
			Expect(player.Run([]byte(`echo "Hello"`))).To(Succeed())
			Expect(strings.Split(string(out.Contents()), "\n")).To(ContainElement("Hello"))
		})

		It("allows to run multiple commands", func() {
			stdinWritePipe.Write([]byte("\n"))
			Expect(player.Run([]byte(`echo "Hello"
echo "World"`))).To(Succeed())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(out).To(gbytes.Say(`echo "Hello"`))
			Expect(out).To(gbytes.Say(`echo "World"`))
			Expect(output).To(ContainElement("Hello"))
			Expect(output).To(ContainElement("World"))
		})

		It("does respect multiline commands", func() {
			stdinWritePipe.Write([]byte("\n"))
			Expect(player.Run([]byte(`echo "Hello \
World"`))).To(Succeed())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(out).To(gbytes.Say(`echo "Hello \\`))
			Expect(out).To(gbytes.Say(`World"`))
			Expect(output).To(ContainElement("Hello World"))
		})

		It("adds > to the multiline commands", func() {
			stdinWritePipe.Write([]byte("\n"))
			os.Setenv("DEMOUSER", "testUser")
			Expect(player.Run([]byte(`echo "Hello \
World"`))).To(Succeed())
			Expect(out).To(gbytes.Say(`testUser.*\$ echo "Hello \\`))
			Expect(out).To(gbytes.Say(`> World"`))
		})

		It("does export environment variables", func() {
			os.Setenv("DEMOSHELL_TEST_VAR", "test_secret")
			Expect(player.Run([]byte(`printenv DEMOSHELL_TEST_VAR`))).To(Succeed())
			Expect(out).To(gbytes.Say(`test_secret`))
		})

		It("skips empty lines", func() {
			ch := make(chan bool)
			go func() {
				defer GinkgoRecover()
				Expect(player.Run([]byte("\n\n\necho Hi"))).To(Succeed())
				ch <- true
			}()
			Eventually(ch).Should(Receive())
		})
	})

	Context("state", func() {
		BeforeEach(func() {
			stdinWritePipe.Write([]byte("\n"))
			stdinWritePipe.Write([]byte("\n"))
		})

		Context("environment variables", func() {
			It("works without environment variables", func() {
				os.Clearenv()
				Expect(player.Run([]byte(`DEMOSHELL_KEEP_ENV_VARIABLE="Hello World"
echo "$DEMOSHELL_KEEP_ENV_VARIABLE"`))).To(Succeed())
				output := strings.Split(string(out.Contents()), "\n")
				Expect(output).To(ContainElement("Hello World"))
			})

			XIt("does not delete random environment variable", func() {
				Expect(player.Run([]byte(`env
&>2 env`))).To(Succeed())
				output := strings.Split(string(out.Contents()), "\n")
				errOutput := strings.Split(string(stderr.Contents()), "\n")
				Expect(output).To(Equal(errOutput))
			})
			It("keeps environment variables that are exported in the script", func() {
				Expect(player.Run([]byte(`DEMOSHELL_KEEP_ENV_VARIABLE="Hello World"
echo "$DEMOSHELL_KEEP_ENV_VARIABLE"`))).To(Succeed())
				output := strings.Split(string(out.Contents()), "\n")
				Expect(output).To(ContainElement("Hello World"))
			})

			It("keeps number environment variable", func() {
				Expect(player.Run([]byte(`DEMOSHELL_KEEP_ENV_VARIABLE=1
echo "$DEMOSHELL_KEEP_ENV_VARIABLE"`))).To(Succeed())
				output := strings.Split(string(out.Contents()), "\n")
				Expect(output).To(ContainElement("1"))
			})

			It("deletes environment variable that was deleted", func() {
				stdinWritePipe.Write([]byte("\n"))
				Expect(player.Run([]byte(`DEMOSHELL_KEEP_ENV_VARIABLE="'Hello World'"
unset DEMOSHELL_KEEP_ENV_VARIABLE
echo "$DEMOSHELL_KEEP_ENV_VARIABLE"`))).To(Succeed())
				output := strings.Split(string(out.Contents()), "\n")
				Expect(output).NotTo(ContainElement("'Hello World'"))
			})

			It("keeps environment variable with single quotes", func() {
				Expect(player.Run([]byte(`DEMOSHELL_KEEP_ENV_VARIABLE="'Hello 'World'"
echo "$DEMOSHELL_KEEP_ENV_VARIABLE"`))).To(Succeed())
				output := strings.Split(string(out.Contents()), "\n")
				Expect(output).To(ContainElement("'Hello 'World'"))
			})
		})
	})

	Context("comments", func() {
		It("does output comments on seprate lines", func() {
			stdinWritePipe.Write([]byte("\n"))
			Expect(player.Run([]byte(`#Test comment
echo "Hello"`))).To(Succeed())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(output).To(ContainElement("Test comment"))
			Expect(output).To(ContainElement("Hello"))
		})

		It("does not output shebang", func() {
			Expect(player.Run([]byte(`#!echo "Hello"`))).To(Succeed())
			Expect(string(out.Contents())).To(BeEmpty())
		})

		It("does output but do not run comments", func() {
			Expect(player.Run([]byte(`#echo "Hello"`))).To(Succeed())
			Expect(string(out.Contents())).To(Equal(`echo "Hello"
`))
		})

		Context("doitlive", func() {
			It("does not output doitlive comments", func() {
				Expect(player.Run([]byte(`#doitlive speed=5`))).To(Succeed())
				Expect(string(out.Contents())).To(BeEmpty())
			})

			It("respects commentecho", func() {
				Expect(player.Run([]byte(`#doitlive commentecho: false
# This is secret!`))).To(Succeed())
				Expect(string(out.Contents())).To(BeEmpty())
			})

			It("respects commands with extra whitespaces", func() {
				Expect(player.Run([]byte(`#doitlive    commentecho: false
# This is secret!`))).To(Succeed())
				Expect(string(out.Contents())).To(BeEmpty())
			})
		})
	})

	It("waits for enter to execute command", func() {
		ch := make(chan bool)
		go func() {
			defer GinkgoRecover()
			Expect(player.Run([]byte("echo Hi"))).To(Succeed())
			ch <- true
		}()
		Consistently(ch).ShouldNot(Receive())
		stdinWritePipe.Write([]byte("\n"))
		Eventually(ch).Should(Receive())
	})
})
