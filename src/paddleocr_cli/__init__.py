"""
PaddleOCR CLI - A command-line tool for OCR using PaddleOCR AI Studio API.

Usage:
    paddleocr_cli <file>              # Output markdown to stdout
    paddleocr_cli <file> -o out.md    # Output to file
    paddleocr_cli configure           # Configure credentials
"""

__version__ = "0.0.0"  # Managed by poetry-dynamic-versioning
__author__ = "PaddleOCR"

from .config import Config, PaddleOCRConfig, find_config, load_config, save_config
from .ocr import PaddleOCRClient

__all__ = [
    "Config",
    "PaddleOCRConfig",
    "PaddleOCRClient",
    "find_config",
    "load_config",
    "save_config",
]
