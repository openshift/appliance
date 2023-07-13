package executer

import (
	"strings"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Executer", func() {
	var (
		executer Executer
	)

	BeforeEach(func() {
		executer = NewExecuter()
	})

	It("Fails if command line args are nil", func() {
		result, err := executer.Execute(Command{
			Args: nil,
		})
		Expect(err).To(HaveOccurred())
		msg := err.Error()
		Expect(msg).To(ContainSubstring("argument"))
		Expect(msg).To(ContainSubstring("required"))
		Expect(result).To(BeEmpty())
	})

	It("Fails if command line args are empty", func() {
		result, err := executer.Execute(Command{
			Args: []string{},
		})
		Expect(err).To(HaveOccurred())
		msg := err.Error()
		Expect(msg).To(ContainSubstring("argument"))
		Expect(msg).To(ContainSubstring("required"))
		Expect(result).To(BeEmpty())
	})

	It("Fails if the command doesn't exist", func() {
		result, err := executer.Execute(Command{
			Args: []string{
				"junk",
			},
		})
		Expect(err).To(HaveOccurred())
		msg := err.Error()
		Expect(msg).To(ContainSubstring("junk"))
		Expect(msg).To(ContainSubstring("executable"))
		Expect(msg).To(ContainSubstring("not"))
		Expect(msg).To(ContainSubstring("found"))
		Expect(result).To(BeEmpty())
	})

	It("Finds the path of the executable and runs it", func() {
		_, err := executer.Execute(Command{
			Args: []string{
				"echo",
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("Executes command with full path name", func() {
		_, err := executer.Execute(Command{
			Args: []string{
				"/usr/bin/echo",
			},
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("Returns whatever the command writes to the standard output", func() {
		result, err := executer.Execute(Command{
			Args: []string{
				"echo", "Hello!",
			},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(result))
	})

	It("Adds enviroment variable", func() {
		result, err := executer.Execute(Command{
			Env: map[string]any{
				"MYVAR": "myvalue",
			},
			Args: []string{
				"env",
			},
		})
		Expect(err).ToNot(HaveOccurred())
		lines := strings.Split(result, "\n")
		Expect(lines).To(ContainElement("MYVAR=myvalue"))
	})

	It("Removes enviroment variable", func() {
		result, err := executer.Execute(Command{
			Env: map[string]any{
				"HOME": nil,
			},
			Args: []string{
				"env",
			},
		})
		Expect(err).ToNot(HaveOccurred())
		lines := strings.Split(result, "\n")
		Expect(lines).ToNot(ContainElement(MatchRegexp("^HOME=.*$")))
	})

	It("Passes command line arguments", func() {
		result, err := executer.Execute(Command{
			Args: []string{
				"echo", "Hello", "world!",
			},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal("Hello world!"))
	})
})
