# Shine `devenv`

A Go project with bleeding-edge development environment setup using devenv + Nix.

## Development Environment

This project uses [devenv](https://devenv.sh) for a reproducible development environment.

### Prerequisites

- Nix with flakes enabled
- devenv installed
- direnv (optional but recommended)

### Getting Started

1. **Allow direnv** (if using direnv):

   ```bash
   direnv allow
   ```

2. **Enter the development shell**:

   ```bash
   devenv shell
   ```

3. **Available commands**:
   - `dev` - Start development server with hot reload (Air)
   - `build` - Build the project
   - `test` - Run tests
   - `lint` - Run linters
   - `tidy` - Tidy Go modules

### Project Structure

```
shine/
├── cmd/
│   └── shine/          # Main application entry point
├── pkg/                # Public packages
├── internal/           # Private application code
├── devenv.nix          # Development environment configuration
├── .air.toml           # Hot reload configuration
└── go.mod              # Go module definition
```

### Development Workflow

**Hot reload development**:

```bash
dev
```

**Build the project**:

```bash
build
# or manually:
go build -o bin/shine ./cmd/shine
```

**Run tests**:

```bash
test
# or manually:
go test -v ./...
```

**Run linters**:

```bash
lint
# or manually:
golangci-lint run ./...
```

### Environment Details

The devenv setup includes:

- **Go 1.23** - Latest stable Go version
- **gomod2nix** - Reproducible Go module builds
- **Air** - Hot reload for rapid development
- **golangci-lint** - Comprehensive linting
- **Delve** - Go debugger
- **Pre-commit hooks** - Automatic code quality checks

### Tools Included

- `gofmt` - Code formatting
- `govet` - Static analysis
- `golangci-lint` - Multiple linters in one
- `goimports` - Import management
- `delve` - Debugging

## License

[License information here]
