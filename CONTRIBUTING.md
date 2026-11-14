# Contributing to Secrethor

Thank you for your interest in contributing to Secrethor! Contributions of every size are welcome: bug reports, documentation fixes, new checks, tests and ideas.

Not sure where to start? The [roadmap](README.md#-roadmap) lists what we would like to build next, and opening an issue to discuss an idea is always a good first step.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Style](#code-style)
- [Testing](#testing)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)
- [Release Process](#release-process)

## Code of Conduct

Be respectful and constructive. We want Secrethor to be a project where everyone feels welcome to take part.

## Getting Started

1. **Fork the Repository**
   ```bash
   git clone https://github.com/miltlima/secrethor.git
   cd secrethor
   ```

2. **Set Up Development Environment**
   - Install Go 1.23 or later
   - Install Docker
   - Install kubectl
   - Optional: [kind](https://kind.sigs.k8s.io/) and Helm, to try your change on a local cluster

3. **Build and Run the Tests**
   ```bash
   make build
   make test
   ```

## Development Workflow

1. **Create a Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Your Changes**
   - Follow the code style guidelines
   - Write tests for new functionality
   - Update documentation

3. **Commit Your Changes**
   ```bash
   git commit -s -m "feat: add new feature"
   ```
   Use conventional commit messages:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation
   - `style:` for formatting changes
   - `refactor:` for code refactoring
   - `test:` for test-related changes
   - `chore:` for maintenance tasks

4. **Push Your Changes**
   ```bash
   git push origin feature/your-feature-name
   ```

## Code Style

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Run `make lint` and `go vet ./...` before submitting PRs
- Keep functions small and focused
- Write clear and concise comments

## Testing

1. **Run Unit Tests**
   ```bash
   make test
   ```

2. **Run E2E Tests**
   ```bash
   make test-e2e
   ```

3. **Code Coverage**
   - Write the test first and watch it fail before changing the code
   - `make test` writes the coverage profile to `cover.out`:
     ```bash
     go tool cover -func cover.out
     ```

## Documentation

1. **Code Documentation**
   - Document all exported functions and types
   - Use clear and concise comments
   - Follow Go documentation conventions

2. **User Documentation**
   - Update README.md for significant changes
   - Add or update examples in the `samples` directory
   - Keep API documentation up to date

## Pull Request Process

1. **Create a Pull Request**
   - Describe your changes clearly
   - Reference related issues

2. **PR Checklist**
   - [ ] Tests added/updated
   - [ ] Documentation updated
   - [ ] Code follows style guidelines
   - [ ] All tests pass
   - [ ] Branch is up to date with main

3. **Review Process**
   - Address reviewer comments
   - Keep PR focused and small
   - Update PR as needed

## Release Process

Releases are automatic. Every push to `main` runs the release workflow, which:

- calculates the next version from the commit messages (`fix:` and others bump the patch, `feat:` the minor, a `BREAKING CHANGE:` footer the major)
- updates `CHANGELOG.md` and the chart version
- creates the tag and publishes the Docker image and the Helm chart

So you never bump versions or edit the changelog by hand. What matters is the commit message: use the right type, and add a `BREAKING CHANGE:` footer when users have to change something to upgrade.

## Need Help?

- Open an [issue](https://github.com/miltlima/secrethor/issues)
- Check the [README](README.md)

Thank you for contributing to Secrethor! 🚀 