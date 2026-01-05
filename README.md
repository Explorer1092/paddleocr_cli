# PaddleOCR CLI

A command-line tool for OCR using PaddleOCR AI Studio API.

## Installation

### Using Poetry (Development)

```bash
cd .claude/skills/recruitment-analyzer/scripts/
poetry install
```

### Using pip

```bash
pip install paddleocr-cli
```

### Using uv

```bash
uv pip install paddleocr-cli
```

## Quick Start

### 1. Configure credentials

```bash
# Configure (default saves to user directory)
paddleocr_cli configure --server-url https://your-server.com --token YOUR_TOKEN

# Configure with scope
paddleocr_cli configure --server-url URL --token TOKEN -s project  # project root
paddleocr_cli configure --server-url URL --token TOKEN -s local    # script directory
paddleocr_cli configure --server-url URL --token TOKEN -s user     # ~/.config/ (default)

# Verify configuration
paddleocr_cli configure --show

# Test connection
paddleocr_cli configure --test
```

### 2. Run OCR

```bash
# Output to stdout
paddleocr_cli resume.pdf

# Output to file
paddleocr_cli resume.pdf -o output.md

# Output as JSON
paddleocr_cli resume.pdf --json
```

## Usage

```
paddleocr_cli [OPTIONS] FILE

positional arguments:
  FILE                  Path to PDF or image file

options:
  -o, --output FILE     Output file path (default: stdout)
  --json                Output as JSON instead of markdown
  --page N              Extract only page N (0-indexed)
  --no-separator        Don't add page separators in markdown output
  --timeout SECONDS     Request timeout in seconds (default: 120)
  --orientation         Enable document orientation classification
  --unwarp              Enable document unwarping
  --chart               Enable chart recognition
  -q, --quiet           Suppress progress messages
  --config FILE         Path to config file
```

### Configure subcommand

```
paddleocr_cli configure [OPTIONS]

options:
  --server-url URL      Set the server URL (required)
  --token TOKEN         Set the access token (required)
  -s, --scope SCOPE     Installation scope (default: user)
                        user    - ~/.config/paddleocr_cli/
                        project - project root (alongside .claude/)
                        local   - script directory
  --show                Show current configuration
  --test                Test connection to the server
  --locations           Show config file search locations
```

## Configuration

Configuration is stored in YAML format (`.paddleocr_cli.yaml`). The tool searches for config files in this order:

1. **Script directory**: `./.paddleocr_cli.yaml`
2. **Project root**: `<project>/.paddleocr_cli.yaml` (alongside `.claude/` directory)
3. **User directory**: `~/.config/paddleocr_cli/config.yaml`

### Config file format

```yaml
paddleocr:
  server_url: "https://your-server.aistudio-app.com"
  access_token: "your_token_here"
```

## Running as a module

```bash
# Using python -m
python -m paddleocr_cli resume.pdf

# Using poetry
poetry run paddleocr_cli resume.pdf

# Using uv
uv run paddleocr_cli resume.pdf
```

## API

You can also use the library programmatically:

```python
from paddleocr_cli import PaddleOCRClient, Config, PaddleOCRConfig

# Use default config (from config files)
client = PaddleOCRClient()

# Or provide custom config
config = Config(
    paddleocr=PaddleOCRConfig(
        server_url="https://your-server.com",
        access_token="your_token",
    )
)
client = PaddleOCRClient(config)

# Perform OCR
result = client.ocr_file("document.pdf")

if result.success:
    print(result.full_markdown)
    for page in result.pages:
        print(f"Page {page.page_index}: {len(page.markdown)} chars")
else:
    print(f"Error: {result.error_message}")
```

## License

MIT License
