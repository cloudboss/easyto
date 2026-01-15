# Claude Code Project Configuration

This file contains project-specific conventions, patterns, and guidelines for the easyto project.

## Project Overview

easyto is a tool that converts container images to AWS EC2 AMIs. It's written in Go 1.25+ and focuses on creating bootable disk images from container images.

## Code Style and Conventions

### YAGNI Principle (You Aren't Gonna Need It)

**Always follow YAGNI when writing code.** Only implement what is currently needed:

- **Don't create "utility" functions until they're needed in at least one place**
- **Don't add features, options, or abstractions for hypothetical future use**
- **Don't write helper functions that are only used in their own tests**
- If you find yourself writing code "that might be useful later", stop and remove it
- When implementing functionality, implement the minimal solution that works
- Add complexity only when current requirements demand it

**Examples:**
```go
// Bad - creating helpers for future use
func CreateTestFSWithDirs(dirs []string, files map[string]string) afero.Fs {
    // Only used in its own test, not needed elsewhere
}

// Good - create helpers only when reused
func CreateTarArchive(files map[string]string) ([]byte, error) {
    // Used in multiple tests across ctr2disk package
}
```

When reviewing code or creating test utilities:
1. Is this function used in production code or multiple tests? Keep it.
2. Is this function only tested but never used? Remove it.
3. Are you adding this "just in case"? Don't add it yet.

### Testing Standards

#### Filesystem Abstraction
- **Always use `afero.Fs` for filesystem operations** that need to be testable
- Pass `afero.Fs` as the **first parameter** to functions that need filesystem access
- In production code, use `afero.NewOsFs()` for real filesystem operations
- In tests, use `afero.NewMemMapFs()` for in-memory testing (faster, isolated, parallel-safe)
- For error testing, use `afero.NewReadOnlyFs()` to trigger write errors

#### Test Structure
- Use table-driven tests with `testCases` for multiple scenarios
- Use `testify/require` for assertions that should stop the test on failure
- Use `testify/assert` for assertions that should continue even on failure
- Keep test descriptions concise and clear (e.g., "Valid kernel archive", "Unknown service")

#### Comment Guidelines
- **Avoid obvious comments** in tests (e.g., "Use in-memory filesystem", "Create test data")
- Only add comments when the code's purpose or approach is non-obvious
- Comments should explain **why**, not **what**
- Prefer descriptive variable/function names over comments

### Dependency Injection Pattern

Functions that perform I/O should accept dependencies as parameters:
```go
// Good - testable
func processFile(fs afero.Fs, path string) error {
    f, err := fs.Open(path)
    // ...
}

// Bad - uses global state, hard to test
func processFile(path string) error {
    f, err := os.Open(path)
    // ...
}
```

### Error Handling

- Use `fmt.Errorf` with `%w` to wrap errors (Go 1.13+ error wrapping)
- Provide context in error messages that includes the operation and relevant parameters
- Example: `fmt.Errorf("unable to open %s for reading: %w", srcFile, err)`

## Project Structure

### Key Packages

- `pkg/ctr2disk/` - Core container-to-disk conversion logic
- `pkg/login/` - User and group management for VM images
- `pkg/testutil/` - Shared testing utilities and helpers
- `cmd/ctr2disk/` - CLI entry point
- `cmd/easyto/` - Main easyto CLI with tree of subcommands

### Testing Approach

1. **Unit Tests** - Test individual functions with mocked dependencies
   - Target: 80%+ coverage for utility functions
   - Use `afero.MemMapFs` for filesystem operations
   - Mock external services and APIs

2. **Integration Tests** - Test complete workflows (future work)
   - Require real filesystem, disk operations, or privileged access
   - Should be in separate `_integration_test.go` files
   - May require root privileges or specific environments

### Test Organization

- Keep test files colocated with source: `foo.go` → `foo_test.go`
- Group related test helpers in `pkg/testutil/`
- Skip tests that require privileges with:
  ```go
  if os.Geteuid() != 0 {
      t.Skip("Test requires root privileges")
  }
  ```

## Build and Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/ctr2disk

# Run specific test
go test ./pkg/ctr2disk -run TestCopyFile

# Build main binary
go build ./cmd/easyto
```

## Dependencies

### Core Libraries
- `github.com/spf13/afero` - Filesystem abstraction for testing
- `github.com/spf13/cobra` - CLI framework
- `github.com/google/go-containerregistry` - Container image handling
- `github.com/stretchr/testify` - Testing assertions and helpers

### Testing Philosophy
- Tests should be fast (use in-memory filesystem)
- Tests should be isolated (no shared state)
- Tests should be parallel-safe (pass dependencies, avoid globals)
- Tests should not make network calls (mock external services)

## Git Workflow

- Main branch: Feature branches are merged back to the default branch
- Commits should be concise and focused
- Always include `Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>` in commits made by Claude Code

## Coverage Targets

- **Utility functions**: 80%+ coverage
- **Integration code**: May have lower coverage (requires real infrastructure)
- **Current status** (as of last update):
  - `pkg/login`: 90.1%
  - `pkg/testutil`: 78%
  - `pkg/ctr2disk`: 31.8% (many functions require disk/mount operations)

## Patterns to Follow

### Builder Pattern
The project uses functional options for configuration:
```go
builder, err := ctr2disk.NewBuilder(
    afero.NewOsFs(),
    ctr2disk.WithAssetDir("/assets"),
    ctr2disk.WithCTRImageName("myimage:latest"),
)
```

### Error Types
Custom error construction for tar extraction failures:
```go
err = newErrExtract(tar.TypeReg, err)
```

## Common Operations

### Adding New Tests
1. Identify the function to test
2. Determine if it needs filesystem abstraction
3. Create table-driven test with multiple scenarios
4. Test both success and error cases
5. Use `testutil` helpers for common operations

### Refactoring for Testability
1. Identify functions using `os` package directly
2. Add `fs afero.Fs` as first parameter
3. Update all call sites to pass filesystem
4. Update tests to use `afero.NewMemMapFs()`
5. Run tests to verify behavior unchanged
