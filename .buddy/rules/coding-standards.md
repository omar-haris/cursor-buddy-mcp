# Coding Standards
- category: coding
- priority: critical

## Overview
Core coding standards and best practices for the project.

## Code Quality
- Use meaningful variable names
- Handle all errors properly
- Add comments for complex logic
- Keep functions under 50 lines
- Write tests for all features

## Go-Specific Standards
- Follow Go naming conventions (camelCase, PascalCase)
- Use `gofmt` for code formatting
- Use `go vet` and `golint` for static analysis
- Handle errors explicitly, don't ignore them
- Use interfaces for abstraction
- Prefer composition over inheritance

## Error Handling
- Always check and handle errors
- Use structured error types
- Wrap errors with context using `fmt.Errorf`
- Log errors at appropriate levels
- Return meaningful error messages

## Testing
- Write unit tests for all public functions
- Use table-driven tests for multiple test cases
- Mock external dependencies
- Achieve minimum 80% code coverage
- Use descriptive test names

## Documentation
- Write clear and concise comments
- Document public APIs with examples
- Update README files when adding features
- Use godoc format for Go documentation
- Include usage examples in documentation

## Performance
- Avoid premature optimization
- Use profiling tools to identify bottlenecks
- Minimize memory allocations in hot paths
- Use appropriate data structures
- Cache expensive operations when beneficial

## Security
- Validate all user inputs
- Use parameterized queries for database operations
- Implement proper authentication and authorization
- Follow OWASP security guidelines
- Sanitize output to prevent XSS attacks

## Code Organization
- Group related functionality in packages
- Keep package interfaces small and focused
- Use meaningful package names
- Organize imports: standard library, third-party, local
- Maintain consistent project structure 