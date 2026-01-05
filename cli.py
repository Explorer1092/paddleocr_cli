"""
Command-line interface for PaddleOCR CLI.
"""

import argparse
import sys
from pathlib import Path
from typing import Optional

from . import __version__
from .config import (
    Config,
    PaddleOCRConfig,
    find_config,
    get_config_locations,
    load_config,
    save_config,
)
from .ocr import PaddleOCRClient


def create_parser() -> argparse.ArgumentParser:
    """Create the argument parser."""
    parser = argparse.ArgumentParser(
        prog="paddleocr_cli",
        description="OCR documents using PaddleOCR AI Studio API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  paddleocr_cli resume.pdf                    # OCR and print to stdout
  paddleocr_cli resume.pdf -o output.md       # OCR and save to file
  paddleocr_cli resume.pdf --json             # Output as JSON
  paddleocr_cli configure                     # Configure credentials
  paddleocr_cli configure --show              # Show current config
  paddleocr_cli configure --test              # Test connection
        """,
    )

    parser.add_argument(
        "--version",
        action="version",
        version=f"%(prog)s {__version__}",
    )

    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # Configure subcommand
    configure_parser = subparsers.add_parser(
        "configure",
        help="Configure PaddleOCR credentials",
        description="Configure or view PaddleOCR API credentials",
    )
    configure_parser.add_argument(
        "--token",
        metavar="TOKEN",
        help="Set the access token",
    )
    configure_parser.add_argument(
        "--server-url",
        metavar="URL",
        help="Set the server URL (required)",
    )
    configure_parser.add_argument(
        "--show",
        action="store_true",
        help="Show current configuration",
    )
    configure_parser.add_argument(
        "--test",
        action="store_true",
        help="Test connection to the server",
    )
    configure_parser.add_argument(
        "--locations",
        action="store_true",
        help="Show config file search locations",
    )
    configure_parser.add_argument(
        "-s", "--scope",
        choices=["user", "project", "local"],
        default="user",
        help="Installation scope: user, project, or local (default: user)",
    )

    # OCR subcommand (default when file is provided)
    ocr_parser = subparsers.add_parser(
        "ocr",
        help="Perform OCR on a document",
        description="Perform OCR on a PDF or image file",
    )
    _add_ocr_arguments(ocr_parser)

    return parser


def _add_ocr_arguments(parser: argparse.ArgumentParser) -> None:
    """Add OCR-related arguments to a parser."""
    parser.add_argument(
        "file",
        nargs="?",
        type=Path,
        help="Path to PDF or image file",
    )
    parser.add_argument(
        "-o", "--output",
        metavar="FILE",
        type=Path,
        help="Output file path (default: stdout)",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output as JSON instead of markdown",
    )
    parser.add_argument(
        "--page",
        type=int,
        metavar="N",
        help="Extract only page N (0-indexed)",
    )
    parser.add_argument(
        "--no-separator",
        action="store_true",
        help="Don't add page separators in markdown output",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=120,
        metavar="SECONDS",
        help="Request timeout in seconds (default: 120)",
    )
    parser.add_argument(
        "--orientation",
        action="store_true",
        help="Enable document orientation classification",
    )
    parser.add_argument(
        "--unwarp",
        action="store_true",
        help="Enable document unwarping",
    )
    parser.add_argument(
        "--chart",
        action="store_true",
        help="Enable chart recognition",
    )
    parser.add_argument(
        "-q", "--quiet",
        action="store_true",
        help="Suppress progress messages",
    )
    parser.add_argument(
        "--config",
        type=Path,
        metavar="FILE",
        help="Path to config file",
    )


def cmd_configure(args: argparse.Namespace) -> int:
    """Handle the configure subcommand."""
    # Show config locations
    if args.locations:
        print("Configuration file search locations:\n")
        for desc, path, exists in get_config_locations():
            status = "[FOUND]" if exists else "[not found]"
            print(f"  {status:12} {desc}")
            print(f"             {path}\n")
        return 0

    # Load current config
    config_path = find_config()
    config = load_config(config_path)

    # Show current config
    if args.show:
        print("Current configuration:\n")
        if config_path:
            print(f"  Config file: {config_path}")
        else:
            print("  Config file: (none found)")
        print()
        server_display = config.paddleocr.server_url or "(not set)"
        print(f"  Server URL:   {server_display}")
        token_display = "***" + config.paddleocr.access_token[-8:] if len(config.paddleocr.access_token) > 8 else "(not set)"
        print(f"  Access token: {token_display}")
        return 0

    # Test connection
    if args.test:
        if not config.paddleocr.server_url or not config.paddleocr.access_token:
            print("Error: server_url and access_token must be configured first.", file=sys.stderr)
            print("Run: paddleocr_cli configure --server-url URL --token TOKEN", file=sys.stderr)
            return 1
        print("Testing connection to PaddleOCR server...")
        client = PaddleOCRClient(config)
        success, message = client.test_connection()
        if success:
            print(f"  [OK] {message}")
            return 0
        else:
            print(f"  [FAILED] {message}")
            return 1

    # Update config (non-interactive mode only)
    if not args.token and not args.server_url:
        print("Usage: paddleocr_cli configure --server-url URL --token TOKEN [-s SCOPE]", file=sys.stderr)
        print("\nOptions:", file=sys.stderr)
        print("  --server-url URL   Set the server URL (required)", file=sys.stderr)
        print("  --token TOKEN      Set the access token (required)", file=sys.stderr)
        print("  -s, --scope SCOPE  Installation scope (default: user)", file=sys.stderr)
        print("                     user    - ~/.config/paddleocr_cli/", file=sys.stderr)
        print("                     project - project root (alongside .claude/)", file=sys.stderr)
        print("                     local   - script directory", file=sys.stderr)
        print("  --show             Show current configuration", file=sys.stderr)
        print("  --test             Test connection", file=sys.stderr)
        return 1

    if args.token:
        config.paddleocr.access_token = args.token

    if args.server_url:
        config.paddleocr.server_url = args.server_url

    # Determine save path based on scope
    from .config import get_project_root, get_script_dir, CONFIG_FILENAME

    if args.scope == "local":
        # Script directory
        save_path = get_script_dir() / CONFIG_FILENAME
    elif args.scope == "project":
        # Project root (alongside .claude/)
        project_root = get_project_root()
        if project_root:
            save_path = project_root / CONFIG_FILENAME
        else:
            print("Error: No project root found (no .claude/ directory in parent paths)", file=sys.stderr)
            return 1
    else:  # user (default)
        save_path = None  # Will use default user config path

    saved_path = save_config(config, save_path)
    print(f"Configuration saved to: {saved_path}")

    return 0


def cmd_ocr(args: argparse.Namespace) -> int:
    """Handle OCR command."""
    if not args.file:
        print("Error: No file specified", file=sys.stderr)
        print("Usage: paddleocr_cli <file> [-o output.md]", file=sys.stderr)
        return 1

    if not args.file.exists():
        print(f"Error: File not found: {args.file}", file=sys.stderr)
        return 1

    # Load config
    config = load_config(args.config if hasattr(args, 'config') and args.config else None)
    client = PaddleOCRClient(config)

    if not client.is_configured:
        print("Error: PaddleOCR is not configured.", file=sys.stderr)
        print("Run 'paddleocr_cli configure' to set up credentials.", file=sys.stderr)
        return 1

    # Perform OCR
    if not args.quiet:
        print(f"Processing: {args.file}", file=sys.stderr)

    result = client.ocr_file(
        args.file,
        use_doc_orientation_classify=args.orientation,
        use_doc_unwarping=args.unwarp,
        use_chart_recognition=args.chart,
        timeout=args.timeout,
    )

    if not result.success:
        print(f"Error: {result.error_message}", file=sys.stderr)
        return 1

    if not args.quiet:
        print(f"OCR completed: {len(result.pages)} page(s)", file=sys.stderr)

    # Format output
    if args.json:
        import json
        output_data = {
            "success": True,
            "pages": [
                {
                    "page_index": p.page_index,
                    "markdown": p.markdown,
                    "images": p.images,
                }
                for p in result.pages
            ],
            "log_id": result.log_id,
        }
        output = json.dumps(output_data, ensure_ascii=False, indent=2)
    else:
        # Markdown output
        if args.page is not None:
            if 0 <= args.page < len(result.pages):
                output = result.pages[args.page].markdown
            else:
                print(f"Error: Page {args.page} not found (document has {len(result.pages)} pages)", file=sys.stderr)
                return 1
        elif args.no_separator:
            output = "\n\n".join(p.markdown for p in result.pages)
        else:
            output = result.full_markdown

    # Write output
    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(output)
        if not args.quiet:
            print(f"Output saved to: {args.output}", file=sys.stderr)
    else:
        print(output)

    return 0


def main(argv: Optional[list[str]] = None) -> int:
    """Main entry point."""
    if argv is None:
        argv = sys.argv[1:]

    # Pre-process: if first arg looks like a file (not a subcommand), insert 'ocr'
    known_commands = {"configure", "ocr", "-h", "--help", "--version"}
    if argv and argv[0] not in known_commands and not argv[0].startswith("-"):
        # Check if it looks like a file path
        if Path(argv[0]).suffix or "/" in argv[0] or "\\" in argv[0]:
            argv = ["ocr"] + argv

    parser = create_parser()
    args = parser.parse_args(argv)

    # Route to appropriate command
    if args.command == "configure":
        return cmd_configure(args)
    elif args.command == "ocr":
        return cmd_ocr(args)
    else:
        parser.print_help()
        return 0


if __name__ == "__main__":
    sys.exit(main())
