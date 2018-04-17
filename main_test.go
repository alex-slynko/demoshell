package main_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	var pathToCLI string
	BeforeSuite(func() {
		var err error
		pathToCLI, err = gexec.Build("github.com/alex-slynko/demoshell")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	It("requires an argument", func() {
		command := exec.Command(pathToCLI)
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(1))
	})

	It("checks that file exists", func() {
		command := exec.Command(pathToCLI, "fixtures/missing.session")
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(1))
	})

	It("outputs all commands from the file", func() {
		command := exec.Command(pathToCLI, "fixtures/basic.session")
		inPipe, err := command.StdinPipe()
		Expect(err).NotTo(HaveOccurred())
		inPipe.Write([]byte("\n"))
		command.Env = append(command.Env, "USER=demo")
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say(`demo.*\$ echo "Hello"`))
	})

})
