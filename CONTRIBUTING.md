# Contributing to Open Tool Speex

Thank you for your interest in contributing to Open Tool Speex! This document provides guidelines and information for contributors.

## Development Setup

### Prerequisites

- **Go 1.22+** (recommended: latest version)
- **SpeexDSP** library
- **Task** (task runner) - install from [taskfile.dev](https://taskfile.dev/installation/)

### Platform-specific Setup

#### Linux (Ubuntu/Debian)
```bash
sudo apt update
sudo apt install libspeexdsp-dev pkg-config golang-go
```

#### macOS
```bash
brew install speexdsp pkg-config go
```

#### Windows
- Install MinGW-w64/MSYS2
- Install SpeexDSP through package manager
- Or use GitHub Actions for building

### Getting Started

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/your-username/open_tool_speex.git
   cd open_tool_speex
   ```

2. **Install Task**
   ```bash
   # macOS/Linux
   sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
   
   # Or via package manager
   brew install go-task/tap/go-task  # macOS
   ```

3. **Install dependencies**
   ```bash
   task deps
   ```

4. **Run tests**
   ```bash
   task test
   ```

5. **Build the project**
   ```bash
   task build
   ```

## Development Workflow

### Available Tasks

Run `task --list` to see all available tasks:

- `task build` - Build the binary
- `task test` - Run tests
- `task test-coverage` - Run tests with coverage
- `task lint` - Run linting
- `task fmt` - Format code
- `task dev` - Development mode (build + test + run example)
- `task clean` - Clean build artifacts

### Code Structure

```
open_tool_speex/
├── cmd/open_tool_speex/     # CLI application
├── internal/
│   ├── audio/               # A-law codec
│   ├── speex/               # SpeexDSP wrappers
│   ├── processor/           # Audio processing logic
│   └── config/              # Configuration management
├── pkg/types/               # Shared types
├── testdata/                # Test audio files
└── .github/workflows/       # CI/CD pipelines
```

### Testing

#### Unit Tests
```bash
task test
```

#### Integration Tests
```bash
task test-integration
```

#### Coverage
```bash
task test-coverage
# Opens coverage.html in browser
```

#### Test Data
```bash
task testdata  # Generate test audio files
```

### Code Quality

#### Formatting
```bash
task fmt
```

#### Linting
```bash
task lint
```

#### Static Analysis
```bash
task vet
```

### Building

#### Single Platform
```bash
task build
```

#### All Platforms
```bash
task build-all
```

## Contributing Guidelines

### Pull Request Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Follow Go coding standards
   - Add tests for new functionality
   - Update documentation as needed

3. **Run quality checks**
   ```bash
   task fmt
   task lint
   task test
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

5. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

### Commit Message Format

We follow conventional commits:

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `style:` - Code style changes
- `refactor:` - Code refactoring
- `test:` - Test additions/changes
- `chore:` - Build process or auxiliary tool changes

### Code Standards

- **Go**: Follow standard Go formatting (`gofmt`)
- **Comments**: Document exported functions and types
- **Tests**: Aim for >80% test coverage
- **Error Handling**: Always handle errors appropriately
- **Naming**: Use descriptive names, follow Go conventions

### Testing Requirements

- All new features must have tests
- Bug fixes must include regression tests
- Integration tests for audio processing features
- Performance tests for critical paths

## CI/CD

### GitHub Actions

The project uses GitHub Actions for:

- **CI Pipeline** (`.github/workflows/ci.yml`):
  - Runs on every push and PR
  - Tests on multiple platforms
  - Code quality checks
  - Security scanning

- **Release Pipeline** (`.github/workflows/release.yml`):
  - Builds binaries for all platforms
  - Creates GitHub releases
  - Generates changelog

### Pre-commit Hooks

Consider setting up pre-commit hooks:

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install
```

## Audio Processing

### Supported Formats

- **Input/Output**: Raw A-law PCM (16 kHz, mono)
- **Future**: WAV, FLAC, MP3 support planned

### Processing Modes

1. **AEC+NS** (default): Echo cancellation + Noise suppression
2. **NS→AEC**: Noise suppression first, then echo cancellation
3. **NS-only**: Only noise suppression
4. **AEC-only**: Only echo cancellation
5. **Bypass**: No processing (for testing)

### Adding New Features

#### New Audio Formats
1. Add format detection in `internal/audio/`
2. Implement codec in separate file
3. Add tests
4. Update processor to handle new format

#### New Processing Algorithms
1. Add algorithm in `internal/speex/`
2. Implement Go wrapper for C library
3. Add configuration options
4. Update processor integration
5. Add comprehensive tests

## Debugging

### Common Issues

1. **SpeexDSP not found**
   ```bash
   # Linux
   sudo apt install libspeexdsp-dev pkg-config
   
   # macOS
   brew install speexdsp pkg-config
   ```

2. **CGO issues**
   ```bash
   export CGO_ENABLED=1
   export PKG_CONFIG_PATH=/usr/lib/pkgconfig
   ```

3. **Test failures**
   ```bash
   # Run specific test
   go test -v ./internal/audio -run TestAlaw
   
   # Run with race detection
   go test -race ./...
   ```

### Debug Mode

```bash
# Build with debug info
go build -gcflags="all=-N -l" -o open_tool_speex ./cmd/open_tool_speex

# Run with debug logging
GODEBUG=gctrace=1 ./open_tool_speex -mic input.alaw -speaker ref.alaw
```

## Release Process

### Creating a Release

1. **Update version**
   ```bash
   # Update version in go.mod if needed
   go mod edit -module=open_tool_speex/v2
   ```

2. **Create tag**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

3. **GitHub Actions will automatically**:
   - Build binaries for all platforms
   - Create GitHub release
   - Upload artifacts

### Manual Release

```bash
task release-prepare
# Upload build/ directory contents manually
```

## Support

- **Issues**: [GitHub Issues](https://github.com/your-username/open_tool_speex/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-username/open_tool_speex/discussions)
- **Documentation**: [Wiki](https://github.com/your-username/open_tool_speex/wiki)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

Thank you for contributing to Open Tool Speex! 🎵
