package jinja

// TestFunc defines the signature for a Jinja test function.
type TestFunc func(input interface{}, args ...interface{}) (bool, error)

// GlobalTests stores the registered Jinja test functions.
var GlobalTests map[string]TestFunc

// definedTest implements the 'defined' Jinja test.
func definedTest(input interface{}, args ...interface{}) (bool, error) {
	return input != Undefined, nil
}

func init() {
	GlobalTests = map[string]TestFunc{
		"defined": definedTest,
	}
}
