# Contributing to kube-event-generator

Thank you for your interest in contributing to kube-event-generator! 
This document provides guidelines and instructions for contributing to the project.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/yourusername/kube-event-generator.git
   cd kube-event-generator
   ```
3. Add the upstream repository as a remote:
   ```bash
   git remote add upstream https://github.com/maczg/kube-event-generator.git
   ```
4. Create a new branch for your work:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Process

### Prerequisites

- Go 1.21 or later
- Docker (for local testing environment)
- Make
- golangci-lint (for code linting)

### Building and Testing

```bash
# Build the project
make build

# Run unit tests
make test-unit

# Run all tests (including integration tests)
make test

# Run linting
make lint

# Run security checks
make security

# Generate test coverage report
make test-coverage
```

### Code Style

We follow standard Go code conventions:

- Code must be formatted with `gofmt`
- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use meaningful variable and function names
- Add comments for exported functions, types, and packages
- Keep functions focused and reasonably sized

### Commit Messages

We follow the conventional commits specification:

- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `style:` Code style changes (formatting, missing semicolons, etc.)
- `refactor:` Code refactoring
- `test:` Adding or updating tests
- `chore:` Maintenance tasks

Example:
```
feat: add support for normal distribution in event generation

- Implement NormalDistribution type
- Add configuration parsing for normal distribution parameters
- Include unit tests for edge cases
```

## Submitting Changes

### Pull Request Process

1. Ensure your code follows the project's coding standards
2. Update documentation if you're changing behavior
3. Add tests for new functionality
4. Ensure all tests pass locally
5. Update the CHANGELOG.md with your changes (if applicable)
6. Push your changes to your fork
7. Submit a pull request to the main repository

### Pull Request Guidelines

- PRs should be focused on a single feature or bug fix
- Include a clear description of the changes
- Reference any related issues
- Ensure CI checks pass
- Be responsive to code review feedback

### Code Review

All submissions require review. We use GitHub pull requests for this purpose. Some tips for a successful review:

- Write clear, self-documenting code
- Include unit tests that demonstrate the feature works
- Update documentation as needed
- Respond to reviewer feedback promptly
- Be open to suggestions and constructive criticism

## Testing

### Unit Tests

- Place unit tests in the same package as the code being tested
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for high code coverage (minimum 70% for new code)

Example:
```go
func TestEventScheduler_AddEvent(t *testing.T) {
    tests := []struct {
        name    string
        event   Event
        wantErr bool
    }{
        {
            name:    "valid event",
            event:   Event{Name: "test", Time: time.Now()},
            wantErr: false,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests

- Place integration tests in separate files with `// +build integration` tag
- Test interactions with Kubernetes API
- Use test clusters or simulators (KWOK, kube-scheduler-simulator)

## Documentation

- Update README.md if you change user-facing functionality
- Add or update godoc comments for exported types and functions
- Include examples in documentation where helpful
- Keep CLAUDE.md updated with any architectural changes

## Reporting Issues

When reporting issues, please include:

- keg version (`keg version`)
- Go version (`go version`)
- Kubernetes version
- Operating system and version
- Steps to reproduce the issue
- Expected vs actual behavior
- Any relevant logs or error messages

## Feature Requests

Feature requests are welcome! Please:

- Check existing issues first to avoid duplicates
- Clearly describe the use case
- Explain why existing features don't meet your needs
- If possible, suggest an implementation approach

## Questions?

If you have questions about contributing, feel free to:

- Open an issue with the question label

Thank you for contributing to kube-event-generator!