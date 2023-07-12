package jq

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Query tool", func() {

	It("Can't be created without a logger", func() {
		tool, err := NewQueryTool().Build()
		Expect(err).To(HaveOccurred())
		msg := err.Error()
		Expect(msg).To(ContainSubstring("logger"))
		Expect(msg).To(ContainSubstring("mandatory"))
		Expect(tool).To(BeNil())
	})

	It("Accepts primitive input", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it accepts the input:
		var x int
		err = tool.Query(`.`, 42, &x)
		Expect(err).ToNot(HaveOccurred())
		Expect(x).To(Equal(42))
	})

	It("Accepts struct input", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it accepts the input:
		type Point struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		p := &Point{
			X: 42,
			Y: 24,
		}
		var x int
		err = tool.Query(`.x`, p, &x)
		Expect(err).ToNot(HaveOccurred())
		Expect(x).To(Equal(42))
	})

	It("Accepts map input", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it accepts the input:
		m := map[string]int{
			"x": 42,
			"y": 24,
		}
		var x int
		err = tool.Query(`.x`, m, &x)
		Expect(err).ToNot(HaveOccurred())
		Expect(x).To(Equal(42))
	})

	It("Accepts slice input", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it accepts the input:
		s := []int{42, 24}
		var x int
		err = tool.Query(`.[0]`, s, &x)
		Expect(err).ToNot(HaveOccurred())
		Expect(x).To(Equal(42))
	})

	It("Gets all values if output is slice", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it accepts the input:
		s := []int{42, 24}
		var t []int
		err = tool.Query(`.[]`, s, &t)
		Expect(err).ToNot(HaveOccurred())
		Expect(t).To(ConsistOf(42, 24))
	})

	It("Gets first value if there is only one and output is not slice", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it accepts the input:
		s := []int{42}
		var x int
		err = tool.Query(`.[]`, s, &x)
		Expect(err).ToNot(HaveOccurred())
		Expect(x).To(Equal(42))
	})

	It("Gets first value if there is only one and output is slice", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it accepts the input:
		s := []int{42}
		var t []int
		err = tool.Query(`.[]`, s, &t)
		Expect(err).ToNot(HaveOccurred())
		Expect(t).To(ConsistOf(42))
	})

	It("Fails if there are multiple results and output isn't slice", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it fails:
		s := []int{42, 24}
		var x int
		err = tool.Query(`.[]`, s, &x)
		Expect(err).To(HaveOccurred())
		msg := err.Error()
		Expect(msg).To(ContainSubstring("cannot"))
		Expect(msg).To(ContainSubstring("unmarshal"))
		Expect(msg).To(ContainSubstring("int"))
	})

	It("Fails if output is not compatible with input", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it fails:
		var x int
		err = tool.Query(`.`, "mytext", &x)
		Expect(err).To(HaveOccurred())
		msg := err.Error()
		Expect(msg).To(ContainSubstring("cannot"))
		Expect(msg).To(ContainSubstring("unmarshal"))
		Expect(msg).To(ContainSubstring("int"))
	})

	It("Rejects output that isn't a pointer", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it rejects the ouptut:
		var x int
		err = tool.Query(`.`, 42, x)
		Expect(err).To(HaveOccurred())
		msg := err.Error()
		Expect(msg).To(ContainSubstring("pointer"))
		Expect(msg).To(ContainSubstring("int"))
	})

	It("Can read from a string", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it can read from a string:
		var x int
		err = tool.QueryString(
			`.x`,
			`{
				"x": 42,
				"y": 24
			}`,
			&x,
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(x).To(Equal(42))
	})

	It("Can read from an array of bytes", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it can read from an array of bytes:
		var x int
		err = tool.QueryBytes(
			`.x`,
			[]byte(`{
				"x": 42,
				"y": 24
			}`),
			&x,
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(x).To(Equal(42))
	})

	It("Accepts struct output", func() {
		// Create the tool:
		tool, err := NewQueryTool().
			SetLogger(logger).
			Build()
		Expect(err).ToNot(HaveOccurred())

		// Check that it writes into a struct option:
		type Point struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		var p Point
		err = tool.QueryString(
			`{
				"x": .x,
				"y": .y
			}`,
			`{
				"x": 42,
				"y": 24
			}`,
			&p,
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(p.X).To(Equal(42))
		Expect(p.Y).To(Equal(24))
	})
})
