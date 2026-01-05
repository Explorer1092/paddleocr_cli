"""
Configuration management for PaddleOCR CLI.

Config file search order:
1. Script directory (./paddleocr_cli.yaml)
2. Project root (alongside .claude/ directory)
3. User config directory (~/.config/paddleocr_cli/config.yaml)
"""

import os
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

try:
    import yaml
except ImportError:
    yaml = None  # type: ignore

CONFIG_FILENAME = ".paddleocr_cli.yaml"
USER_CONFIG_DIR = Path.home() / ".config" / "paddleocr_cli"


@dataclass
class PaddleOCRConfig:
    """PaddleOCR API configuration."""
    server_url: str = ""
    access_token: str = ""


@dataclass
class Config:
    """Main configuration class."""
    paddleocr: PaddleOCRConfig = field(default_factory=PaddleOCRConfig)

    @classmethod
    def from_dict(cls, data: dict) -> "Config":
        """Create Config from dictionary."""
        paddleocr_data = data.get("paddleocr", {})
        return cls(
            paddleocr=PaddleOCRConfig(
                server_url=paddleocr_data.get("server_url", ""),
                access_token=paddleocr_data.get("access_token", ""),
            )
        )

    def to_dict(self) -> dict:
        """Convert Config to dictionary."""
        return {
            "paddleocr": {
                "server_url": self.paddleocr.server_url,
                "access_token": self.paddleocr.access_token,
            }
        }


def get_script_dir() -> Path:
    """Get the directory where the script/package is located."""
    return Path(__file__).parent


def get_project_root() -> Optional[Path]:
    """
    Find the project root by looking for .claude/ directory.
    Returns the directory containing .claude/, or None if not found.
    """
    current = Path.cwd()

    # Search upward from current directory
    for parent in [current] + list(current.parents):
        if (parent / ".claude").is_dir():
            return parent
        # Stop at home directory or root
        if parent == Path.home() or parent == parent.parent:
            break

    return None


def find_config() -> Optional[Path]:
    """
    Find the configuration file in the following order:
    1. Script directory
    2. Project root (alongside .claude/)
    3. User config directory

    Returns the path to the config file, or None if not found.
    """
    search_paths = []

    # 1. Script directory
    script_dir = get_script_dir()
    search_paths.append(script_dir / CONFIG_FILENAME)

    # 2. Project root
    project_root = get_project_root()
    if project_root:
        search_paths.append(project_root / CONFIG_FILENAME)

    # 3. User config directory
    search_paths.append(USER_CONFIG_DIR / "config.yaml")

    for path in search_paths:
        if path.exists():
            return path

    return None


def load_config(config_path: Optional[Path] = None) -> Config:
    """
    Load configuration from file.

    Args:
        config_path: Explicit path to config file. If None, searches default locations.

    Returns:
        Config object (with defaults if no config file found).
    """
    if config_path is None:
        config_path = find_config()

    if config_path is None or not config_path.exists():
        return Config()

    if yaml is None:
        # Fallback: simple YAML parsing for basic cases
        return _load_config_simple(config_path)

    with open(config_path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f) or {}

    return Config.from_dict(data)


def _load_config_simple(config_path: Path) -> Config:
    """Simple config loading without PyYAML dependency."""
    config = Config()

    with open(config_path, "r", encoding="utf-8") as f:
        content = f.read()

    # Very basic YAML parsing
    for line in content.split("\n"):
        line = line.strip()
        if ":" not in line or line.startswith("#"):
            continue

        key, _, value = line.partition(":")
        key = key.strip()
        value = value.strip().strip('"').strip("'")

        if key == "server_url":
            config.paddleocr.server_url = value
        elif key == "access_token":
            config.paddleocr.access_token = value

    return config


def save_config(config: Config, config_path: Optional[Path] = None) -> Path:
    """
    Save configuration to file.

    Args:
        config: Config object to save.
        config_path: Path to save to. If None, saves to user config directory.

    Returns:
        Path where config was saved.
    """
    if config_path is None:
        config_path = USER_CONFIG_DIR / "config.yaml"

    # Ensure directory exists
    config_path.parent.mkdir(parents=True, exist_ok=True)

    if yaml is not None:
        with open(config_path, "w", encoding="utf-8") as f:
            yaml.dump(config.to_dict(), f, default_flow_style=False, allow_unicode=True)
    else:
        # Fallback: manual YAML writing
        with open(config_path, "w", encoding="utf-8") as f:
            f.write("# PaddleOCR CLI Configuration\n")
            f.write("paddleocr:\n")
            f.write(f'  server_url: "{config.paddleocr.server_url}"\n')
            f.write(f'  access_token: "{config.paddleocr.access_token}"\n')

    return config_path


def get_config_locations() -> list[tuple[str, Path, bool]]:
    """
    Get all possible config locations with their status.

    Returns:
        List of (description, path, exists) tuples.
    """
    locations = []

    # 1. Script directory
    script_dir = get_script_dir()
    script_config = script_dir / CONFIG_FILENAME
    locations.append(("Script directory", script_config, script_config.exists()))

    # 2. Project root
    project_root = get_project_root()
    if project_root:
        project_config = project_root / CONFIG_FILENAME
        locations.append(("Project root", project_config, project_config.exists()))
    else:
        locations.append(("Project root", Path("(not found)"), False))

    # 3. User config
    user_config = USER_CONFIG_DIR / "config.yaml"
    locations.append(("User config", user_config, user_config.exists()))

    return locations
