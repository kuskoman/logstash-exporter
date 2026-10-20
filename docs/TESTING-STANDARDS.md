# Testing Standards

This document outlines the standardized testing practices for the Logstash Exporter project.

## Test Structure

1. Use table-driven tests where possible
2. Use `t.Parallel()` for tests that can run concurrently
3. Use subtests with `t.Run()` for logical grouping
4. Keep tests independent and isolated

## Naming Conventions

### Test Functions

- Test function names should be in the format `Test<FunctionName>`
- Test table cases should use descriptive names in a consistent format
- Prefer standardized naming patterns:
  - "should_<expected_behavior>" (e.g., "should_return_error_for_invalid_input")
  - "with_<condition>" (e.g., "with_empty_config")
  - "<condition>" (e.g., "multiple_endpoints", "nil_input")

### Variables

- Use descriptive names for test variables
- Mock clients and services should be named `mock<Type>` (e.g., `mockClient`)
- Error-returning mocks should be named `errorMock<Type>` (e.g., `errorMockClient`)
- Test data files should be named `test<Type>` (e.g., `testConfig`)
- Channels for synchronization should be named descriptively (e.g., `listenerCalled`)

## Error Handling

- Use `t.Fatalf` only for setup failures that prevent the test from continuing
- Use `t.Errorf` for assertion failures
- Use the format "expected X, got Y" for error messages
- Check for presence of errors with `if err != nil` or `if err == nil` consistently

## Mocks and Fixtures

- Place fixtures in the `fixtures/` directory
- Define mock implementations at the beginning of the file
- Use table-driven tests with struct definitions for test cases
- Mock implementations should implement the full interface
- Use helper functions for common test setup and teardown

## Context and Timeouts

- Use `context.WithTimeout` for tests that need to time out
- Use a constant at the file level for timeout durations (e.g., `const testTimeout = 5 * time.Second`)
- Always use `defer cancel()` after creating a context with cancel function
- Include proper cleanup in tests to avoid resource leaks

## Example

```go
// TestSomeFunction demonstrates the standard testing pattern
func TestSomeFunction(t *testing.T) {
    t.Parallel()

    // Define test cases
    testCases := []struct {
        name     string
        input    string
        expected string
        err      error
    }{
        {
            name:     "with_valid_input",
            input:    "valid",
            expected: "result",
            err:      nil,
        },
        {
            name:     "with_invalid_input",
            input:    "invalid",
            expected: "",
            err:      errors.New("invalid input"),
        },
    }

    for _, tc := range testCases {
        tc := tc // Capture range variable
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            result, err := SomeFunction(tc.input)

            // Check error
            if tc.err == nil {
                if err != nil {
                    t.Errorf("expected no error, got %v", err)
                }
            } else {
                if err == nil {
                    t.Errorf("expected error %v, got nil", tc.err)
                }
            }

            // Check result
            if result != tc.expected {
                t.Errorf("expected %q, got %q", tc.expected, result)
            }
        })
    }
}
```

## Guidelines for Specific Types of Tests

### Unit Tests

- Focus on testing a single function or method
- Use mocks for dependencies
- Place in the same package as the code being tested

### Integration Tests

- Test interactions between components
- Use test fixtures rather than mocks where appropriate
- May require setup and teardown of test resources

### Concurrency Tests

- Use channels for synchronization
- Set appropriate timeouts to prevent hanging tests
- Use `t.Cleanup()` for proper cleanup

## Recommended Refactorings

When updating existing tests, consider:

1. Standardizing error message formats
2. Making test cases more consistent
3. Adding parallel execution where possible
4. Using table-driven tests for repetitive test logic
5. Improving mock implementations for consistency
