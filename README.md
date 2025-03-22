# Buildx Telemetry

A tool for converting Docker Buildx logs to OpenTelemetry traces.

## Overview

This application consumes Docker Buildx log output in the `rawjson` format and exports the build steps as OpenTelemetry traces. This allows you to view your Docker build steps in your favorite OpenTelemetry tracing visualization tool.

## Installation

```bash
go install github.com/sakajunquality/buildx-telemetry/cmd@latest
```

## Usage

Pipe the output of a Docker build command with `--progress=rawjson` to the application:

```bash
docker buildx build --progress=rawjson . 2>&1 | buildx-telemetry
```

Or use a pre-recorded log file:

```bash
buildx-telemetry --input=build-log.json
```

### Options

- `--otlp-endpoint`: OpenTelemetry endpoint (default: "localhost:4317")
- `--service-name`: Service name for telemetry (default: "docker-build-telemetry")
- `--debug`: Enable debug mode to print detailed step information
- `--input`: Input file (defaults to stdin)
- `--log-level`: Set the logging level (debug, info, warn, error) (default: "info")
- `--exit-code-on-error`: Exit code to use when an error occurs (default: 1)

## Project Structure

```
├── cmd/
│   └── main.go             # Application entry point
├── internal/
│   ├── buildx/             # Buildx log parsing and models
│   │   └── parser.go
│   ├── logger/             # Structured logging
│   │   └── logger.go
│   └── telemetry/          # OpenTelemetry integration
│       └── tracer.go
```

## Development

Clone the repository:

```bash
git clone https://github.com/sakajunquality/buildx-telemetry.git
cd buildx-telemetry
```

Build the application:

```bash
go build -o buildx-telemetry ./cmd
```

## Logging

The application uses structured logging with the Zap library. By default, logs are output in JSON format for production use, but in development mode (--debug), they are output in a more human-readable format.

You can control the log level with the `--log-level` flag:

```bash
buildx-telemetry --log-level=debug
```

## Error Handling

When an error occurs, the application logs the error details and exits with the specified exit code. You can control the exit code using the `--exit-code-on-error` flag:

```bash
buildx-telemetry --exit-code-on-error=2
```

Setting `--exit-code-on-error=0` will make the application continue with exit code 0 even when errors occur, which can be useful in CI/CD pipelines where you want to avoid failing the pipeline.

## Example

1. Start a local OpenTelemetry collector (e.g., Jaeger)
2. Run a Docker build with the `rawjson` output:

```bash
docker buildx build --progress=rawjson . 2>&1 | buildx-telemetry
```

3. Open your tracing UI (e.g., Jaeger UI) to view the build steps as traces.

## License

[MIT](LICENSE) 
