# Contributing to Secrethor

Thank you for your interest in contributing to Secrethor! This document provides guidelines and instructions for contributing to the project.

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

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before contributing.

## Getting Started

1. **Fork the Repository**
   ```bash
   git clone https://github.com/miltlima/secrethor.git
   cd secrethor
   ```

2. **Set Up Development Environment**
   - Install Go 1.22 or later
   - Install Docker
   - Install kubectl
   - Install operator-sdk

3. **Install Dependencies**
   ```bash
   make install
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
- Run `golint` and `go vet` before submitting PRs
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
   - Maintain at least 80% test coverage
   - Run coverage analysis:
     ```bash
     make coverage
     ```

## Documentation

1. **Code Documentation**
   - Document all exported functions and types
   - Use clear and concise comments
   - Follow Go documentation conventions

2. **User Documentation**
   - Update README.md for significant changes
   - Add or update examples in the docs directory
   - Keep API documentation up to date

## Pull Request Process

1. **Create a Pull Request**
   - Use the PR template
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

1. **Version Bumping**
   - Follow semantic versioning
   - Update version in relevant files
   - Update CHANGELOG.md

2. **Release Checklist**
   - [ ] All tests pass
   - [ ] Documentation updated
   - [ ] Version bumped
   - [ ] CHANGELOG.md updated
   - [ ] Release notes prepared

## Community

- Join our [Slack channel](https://slack.secrethor.dev)
- Participate in discussions
- Help other contributors
- Share your use cases

## Need Help?

- Open an issue
- Join our Slack channel
- Check the documentation
- Contact the maintainers

Thank you for contributing to Secrethor! 🚀 