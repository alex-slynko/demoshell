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
		oldEnv         []string
	)

	BeforeEach(func() {
		var err error
		stdinReadPipe, stdinWritePipe, err = os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		out = gbytes.NewBuffer()
		player = &shell.LivePlayer{Out: out, In: stdinReadPipe}
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

		It("outputs contents of the file", func() {
			err := player.Run([]byte(`echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(gbytes.Say(`echo "Hello"`))
		})

		It("adds username to the output", func() {
			os.Setenv("DEMOUSER", "testUser")
			err := player.Run([]byte(`echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(gbytes.Say(`testUser.*\$ echo "Hello"`))
		})

		It("outputs command result", func() {
			err := player.Run([]byte(`echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.Split(string(out.Contents()), "\n")).To(ContainElement("Hello"))
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

		It("does respect multiline commands", func() {
			stdinWritePipe.Write([]byte("\n"))
			err := player.Run([]byte(`echo "Hello \
World"`))
			Expect(err).NotTo(HaveOccurred())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(out).To(gbytes.Say(`echo "Hello \\`))
			Expect(out).To(gbytes.Say(`World"`))
			Expect(output).To(ContainElement("Hello World"))
		})

		It("adds > to the multiline commands", func() {
			stdinWritePipe.Write([]byte("\n"))
			os.Setenv("DEMOUSER", "testUser")
			err := player.Run([]byte(`echo "Hello \
World"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(gbytes.Say(`testUser.*\$ echo "Hello \\`))
			Expect(out).To(gbytes.Say(`> World"`))
		})

		It("does export environment variables", func() {
			os.Setenv("DEMOSHELL_TEST_VAR", "test_secret")
			err := player.Run([]byte(`printenv DEMOSHELL_TEST_VAR`))
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(gbytes.Say(`test_secret`))
		})

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

	Context("state", func() {
		BeforeEach(func() {
			stdinWritePipe.Write([]byte("\n"))
		})

		It("keeps environment variables that are exported in the script", func() {
			stdinWritePipe.Write([]byte("\n"))
			err := player.Run([]byte(`DEMOSHELL_KEEP_ENV_VARIABLE="Hello World"
echo "$DEMOSHELL_KEEP_ENV_VARIABLE"`))
			Expect(err).NotTo(HaveOccurred())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(output).To(ContainElement("Hello World"))
		})

		It("keeps number environment variable", func() {
			stdinWritePipe.Write([]byte("\n"))
			err := player.Run([]byte(`DEMOSHELL_KEEP_ENV_VARIABLE=1
echo "$DEMOSHELL_KEEP_ENV_VARIABLE"`))
			Expect(err).NotTo(HaveOccurred())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(output).To(ContainElement("1"))
		})

		It("keeps environment variable in single quotes", func() {
			stdinWritePipe.Write([]byte("\n"))
			err := player.Run([]byte(`DEMOSHELL_KEEP_ENV_VARIABLE="'Hello World'"
echo "$DEMOSHELL_KEEP_ENV_VARIABLE"`))
			Expect(err).NotTo(HaveOccurred())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(output).To(ContainElement("'Hello World'"))
		})
	})

	Context("comments", func() {
		It("does output comments on seprate lines", func() {
			stdinWritePipe.Write([]byte("\n"))
			err := player.Run([]byte(`#Test comment
echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			output := strings.Split(string(out.Contents()), "\n")
			Expect(output).To(ContainElement("Test comment"))
			Expect(output).To(ContainElement("Hello"))
		})

		It("does not output shebang", func() {
			err := player.Run([]byte(`#!echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out.Contents())).To(BeEmpty())
		})

		It("does output but do not run comments", func() {
			err := player.Run([]byte(`#echo "Hello"`))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out.Contents())).To(Equal(`echo "Hello"
`))
		})

		Context("doitlive", func() {
			It("does not output doitlive comments", func() {
				err := player.Run([]byte(`#doitlive speed=5`))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(out.Contents())).To(BeEmpty())
			})

			FIt("respects commentecho", func() {
				Expect(player.Run([]byte(`#doitlive commentecho: false
# This is secret!`))).To(Succeed())
				Expect(string(out.Contents())).To(BeEmpty())
			})
		})
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
