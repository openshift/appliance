package jq

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/itchyny/gojq"
)

// QueryToolBuilder contains the data needed to build a tool that knows how to run JQ queries. Don't
// create instances of this directly, use the NewQueryTool function instead.
type QueryToolBuilder struct {
	logger logr.Logger
}

// QueryTool knows how to run JQ queries. Don't create instances of this directly, use the
// NewQueryTool function instead.
type QueryTool struct {
	logger logr.Logger
}

// NewQueryTool creates a builder that can then be used to create a JQ tool.
func NewQueryTool() *QueryToolBuilder {
	return &QueryToolBuilder{}
}

// SetLogger sets the logger that the JQ tool will use to write the log. This is mandatory.
func (b *QueryToolBuilder) SetLogger(value logr.Logger) *QueryToolBuilder {
	b.logger = value
	return b
}

// Build uses the information stored in the builder to create a new JQ tool.
func (b *QueryToolBuilder) Build() (result *QueryTool, err error) {
	// Check parameters:
	if b.logger.GetSink() == nil {
		err = errors.New("logger is mandatory")
		return
	}

	// Create and populate the object:
	result = &QueryTool{
		logger: b.logger,
	}
	return
}

// Query the given query on the given input data and stores the result into the given output
// variable. An error will be returned if the query can't be parsed or if the data doesn't fit into
// the output variable.
func (t *QueryTool) Query(query string, input any, output any) error {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}
	return t.QueryBytes(query, inputBytes, output)
}

// QueryString is similar to Query, but it expects an input string containing JSON text.
func (t *QueryTool) QueryString(query string, input string, output any) error {
	return t.QueryBytes(query, []byte(input), output)
}

// QueryBytes is similar to Query, but it expects as input an array of bytes containing the JSON
// text.
func (t *QueryTool) QueryBytes(query string, input []byte, output any) error {
	// Check that the output is a pointer:
	outputValue := reflect.ValueOf(output)
	if outputValue.Kind() != reflect.Pointer {
		return fmt.Errorf("output must be a pointer, but it is of type %T", output)
	}

	// Parse the query:
	parsed, err := gojq.Parse(query)
	if err != nil {
		return fmt.Errorf("failed to parse query '%s': %w", query, err)
	}

	// Deserialize the input to ensure that we have a type that the JQ library supports.
	var inputObj any
	err = json.Unmarshal(input, &inputObj)
	if err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}

	// Run the query collecting the output. Note one of the outputs can be an error, and in that
	// case we just return it.
	var outputList []any
	outputIter := parsed.Run(inputObj)
	for {
		outputObj, ok := outputIter.Next()
		if !ok {
			break
		}
		err, ok = outputObj.(error)
		if ok {
			return err
		}
		outputList = append(outputList, outputObj)
	}

	// Marshal the output list and try to unmarshal it into the output variable. This is needed
	// to convert whatever types are returned by JQ into what the caller expects. If that fails
	// and there is only one result it may be that the caller passed a variable that can hold
	// only that result instead of an slice, so we try again with that single result.
	outputBytes, err := json.Marshal(outputList)
	if err != nil {
		return err
	}
	err = json.Unmarshal(outputBytes, output)
	if err != nil && len(outputList) == 1 {
		outputBytes, err = json.Marshal(outputList[0])
		if err != nil {
			return err
		}
		err = json.Unmarshal(outputBytes, output)
	}
	t.logger.V(1).Info(
		"Query result",
		"query", query,
		"input", string(input),
		"output", output,
	)
	return err
}
