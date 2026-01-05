"""
PaddleOCR CLI - A command-line tool for OCR using PaddleOCR AI Studio API.

Usage:
    paddleocr_cli <file>              # Output markdown to stdout
    paddleocr_cli <file> -o out.md    # Output to file
    paddleocr_cli configure           # Configure credentials
"""

__version__ = "0.1.0"
__author__ = "AI Interview Assistant"

from .config import Config, find_config, load_config, save_config
from .ocr import PaddleOCRClient

__all__ = [
    "Config",
    "PaddleOCRClient",
    "find_config",
    "load_config",
    "save_config",
]
