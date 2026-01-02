# Intervals De-Duper

A sophisticated de-duplication tool for [Intervals.icu](https://intervals.icu) that evaluates the quality of duplicate activities to determine which one to keep.

## Inspiration

This project is a Go rewrite of the original [intervals-dedupe](https://github.com/plastonick/intervals-dedupe) PHP script. It extends the original logic with heuristic evaluation to handle complex scenarios where multiple devices record the same activity with varying levels of data richness.

## Features

- **Heuristic Scoring**: Evaluates activities based on GPS availability, heart rate source, power meter data, and sampling frequency.
- **Metadata Adoption**: Automatically migrates descriptive names, Feel scores, and RPE from duplicates to the "Winner" activity.
- **Mismatch Safety**: Automatically detects and skips activities with significant distance or time differences to protect segments or failed starts.
- **Offline Analysis**: Export all activity data to JSON via `--dump` for local querying.
- **Configurable Opinions**: All prioritization logic is externalized in `config.yml`.
- **Interactive Mode**: Confirm deletions and name adoptions manually.

## Usage

The easiest way to use the de-duper is via **Docker** or by downloading a **pre-compiled binary** from the [Releases](https://github.com/kwv/intervals-deduper/releases) page.

1.  **Configure**: Create your `config.yml` from the example:
    ```bash
    cp config.example.yml config.yml
    ```
2.  **Run**: Choose one of the following methods:

### Method 1: Docker (Recommended)

Use `-it` for interactive prompts:

```bash
docker run -it -v $(pwd)/config.yml:/app/config.yml kwv4/intervals-deduper --interactive
```

### Method 2: Pre-compiled Binary

1.  Download the latest release for your OS (Windows, macOS, or Linux).
2.  Run the executable:
    ```bash
    ./intervals-deduper --dry-run
    ```

### Method 3: From Source (Development)

If you have Go installed:

```bash
go run . --dry-run
```

### CLI Arguments

- `--dry-run`: Preview deletions without making changes.
- `--interactive`: Prompt for confirmation before each deletion.
- `--days N`: Number of days to look back (overrides config).
- `--start YYYY-MM-DD`: Start date for scanning.
- `--end YYYY-MM-DD`: End date for scanning.
- `--verbose`: Show all scanned activities, even non-duplicates.
- `--dump filename.json`: Export all fetched activity details to a local JSON file.
- `--version`: Show version and exit.

## Configuration

The `config.yml` file allows you to define:
- **Weights**: Importance of different data streams.
- **Device Priorities**: Which hardware you trust more.

### API Documentation

This tool is built against the official [Intervals.icu API](https://intervals.icu/api-docs.html).

### Discovering your Devices & Uploaders

If you aren't sure what strings to use for `uploader_penalties` or `device_priority`, use the built-in discovery tools:

1.  **Console Discovery**: Run `go run . --verbose --days 30`. This will list all your activities and show the system name in brackets like `[Device / Uploader]`.
2.  **Data Export**: Run `go run . --dump my_data.json`. This creates a local file where you can see the raw `device_name` and `oauth_client_name` fields for every activity.

## License

MIT
