# Contributing to DNSMonster

First off, thank you for considering contributing to DNSMonster! It's people like you that make DNSMonster such a great tool.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Enhancements](#suggesting-enhancements)

## Code of Conduct

This project and everyone participating in it is governed by respect and professionalism. Be kind to others.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Create a new branch for your contribution
4. Make your changes
5. Push to your fork and submit a pull request

## Development Setup

### Prerequisites

- Go 1.24 or higher
- Git
- Make (optional, for build automation)

### Setup Steps

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/dnsmonster.git
cd dnsmonster

# Add upstream remote
git remote add upstream https://github.com/mosajjal/dnsmonster.git

# Install dependencies
go mod download

# Build the project
go build ./cmd/dnsmonster

# Run tests
go test ./...
```

## How to Contribute

### Types of Contributions

We welcome many types of contributions:

- **Bug fixes**: Fix issues found in the code
- **New features**: Add new functionality
- **Documentation**: Improve or add documentation
- **Tests**: Add or improve test coverage
- **Performance improvements**: Optimize existing code
- **Code refactoring**: Improve code structure and readability

### Before You Start

- Check if there's already an issue for what you want to work on
- For large changes, please open an issue first to discuss
- Make sure you're working on the latest main branch

## Coding Standards

### Go Code Style

We follow standard Go conventions:

1. **Use `gofmt`**: All code must be formatted with `gofmt`

   ```bash
   gofmt -w .
   ```

2. **Use `goimports`**: Organize imports properly

   ```bash
   goimports -w .
   ```

3. **Run `go vet`**: Check for common mistakes

   ```bash
   go vet ./...
   ```

4. **Use `golangci-lint`**: Run comprehensive linting

   ```bash
   golangci-lint run
   ```

### Code Organization

- Keep functions small and focused (ideally < 50 lines)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Group related code together

### Error Handling

- **Always check errors**: Never ignore errors
- **Wrap errors** with context: Use `fmt.Errorf("context: %w", err)`
- **Don't panic**: Use proper error returns instead of panic()
- **Log errors appropriately**: Use appropriate log levels

```go
// Good
if err != nil {
    return fmt.Errorf("failed to open file %s: %w", filename, err)
}

// Bad
if err != nil {
    panic(err)
}
```

### Logging

Use structured logging with logrus:

```go
log.WithFields(log.Fields{
    "component": "output",
    "type": "clickhouse",
    "batch_size": batchSize,
}).Info("Batch processed successfully")
```

### Comments

- Add package-level documentation to all packages
- Document all exported functions, types, and constants
- Use complete sentences in comments
- Keep comments up-to-date with code changes

```go
// ProcessDNSPacket extracts DNS information from a raw packet.
// It returns a DNSResult containing parsed data or an error if parsing fails.
func ProcessDNSPacket(data []byte) (DNSResult, error) {
    // implementation
}
```

## Testing

### Writing Tests

- Write tests for all new code
- Aim for at least 70% code coverage for new features
- Use table-driven tests when appropriate
- Test edge cases and error conditions

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "expected",
            wantErr: false,
        },
        // more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests with verbose output
go test ./... -v

# Run specific test
go test ./internal/util -run TestMaskIPv4

# Run benchmarks
go test ./... -bench=.
```

### Benchmark Tests

Add benchmark tests for performance-critical code:

```go
func BenchmarkFunctionName(b *testing.B) {
    // setup
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        FunctionName()
    }
}
```

## Pull Request Process

### Before Submitting

1. **Update tests**: Ensure tests pass and add new tests for your changes

   ```bash
   go test ./...
   ```

2. **Run linters**: Fix any linting issues

   ```bash
   golangci-lint run
   ```

3. **Update documentation**: Update relevant documentation
   - Update README.md if adding features
   - Update inline code documentation
   - Update user-facing documentation if applicable

4. **Commit messages**: Write clear, descriptive commit messages

   ```
   Fix panic in sentinel output when proxy URL is invalid
   
   - Replace panic() with proper error handling
   - Add error logging with context
   - Return early on error instead of crashing
   
   Fixes #123
   ```

### PR Description

Your PR description should include:

- **What**: Brief description of changes
- **Why**: Motivation for the changes
- **How**: Technical approach taken
- **Testing**: How you tested the changes
- **Related Issues**: Link to related issues

Example:

```markdown
## What
Add proper error handling to output modules

## Why
Several output modules use panic() which can crash the entire application.
This PR replaces panic() calls with proper error handling and logging.

## How
- Modified sentinel.go, splunk.go, victorialogs.go
- Added error returns instead of panic
- Added detailed error logging

## Testing
- Added unit tests for error cases
- Manually tested with invalid configuration
- Verified graceful degradation

## Related Issues
Fixes #123
Related to #456
```

### Review Process

1. At least one maintainer must approve your PR
2. All CI checks must pass
3. Code coverage should not decrease
4. Address all review comments

## Reporting Bugs

### Before Submitting a Bug Report

- Check if the bug has already been reported
- Check if the bug exists in the latest version
- Collect relevant information (logs, configuration, etc.)

### How to Submit a Bug Report

Use the GitHub issue tracker and include:

1. **Title**: Clear, concise description
2. **Environment**:
   - DNSMonster version
   - Operating system and version
   - Go version
3. **Steps to Reproduce**: Detailed steps
4. **Expected Behavior**: What should happen
5. **Actual Behavior**: What actually happens
6. **Logs**: Relevant log output
7. **Configuration**: Relevant configuration settings
8. **Additional Context**: Any other relevant information

## Suggesting Enhancements

### Before Submitting an Enhancement

- Check if the enhancement has been suggested before
- Consider if it fits the project's scope
- Think about how it benefits other users

### How to Submit an Enhancement

Use the GitHub issue tracker and include:

1. **Title**: Clear description of the enhancement
2. **Problem**: What problem does this solve?
3. **Solution**: Proposed solution
4. **Alternatives**: Alternative solutions considered
5. **Additional Context**: Any other relevant information

## Questions?

Feel free to:

- Open a GitHub issue
- Join our community discussions
- Check existing documentation

## License

By contributing, you agree that your contributions will be licensed under the project's GPL-3.0 License.

---

Thank you for contributing to DNSMonster! 🎉
