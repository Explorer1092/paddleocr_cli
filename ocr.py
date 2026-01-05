"""
PaddleOCR API client for document OCR.
"""

import base64
import json
import urllib.request
import urllib.error
from dataclasses import dataclass
from pathlib import Path
from typing import Optional, Union

from .config import Config, load_config


@dataclass
class OCRResult:
    """OCR result for a single page."""
    page_index: int
    markdown: str
    images: dict[str, str]  # image_path -> image_url


@dataclass
class DocumentOCRResult:
    """OCR result for entire document."""
    success: bool
    pages: list[OCRResult]
    error_message: Optional[str] = None
    log_id: Optional[str] = None

    @property
    def full_markdown(self) -> str:
        """Get combined markdown from all pages."""
        return "\n\n---\n\n".join(page.markdown for page in self.pages)


class PaddleOCRClient:
    """Client for PaddleOCR AI Studio API."""

    LAYOUT_PARSING_ENDPOINT = "/layout-parsing"

    def __init__(self, config: Optional[Config] = None):
        """
        Initialize the OCR client.

        Args:
            config: Configuration object. If None, loads from default locations.
        """
        self.config = config or load_config()

    @property
    def server_url(self) -> str:
        return self.config.paddleocr.server_url.rstrip("/")

    @property
    def access_token(self) -> str:
        return self.config.paddleocr.access_token

    @property
    def is_configured(self) -> bool:
        """Check if the client is properly configured."""
        return bool(self.access_token) and bool(self.server_url)

    def _get_file_type(self, file_path: Path) -> int:
        """
        Determine file type from extension.

        Returns:
            0 for PDF, 1 for images.
        """
        suffix = file_path.suffix.lower()
        if suffix == ".pdf":
            return 0
        elif suffix in (".png", ".jpg", ".jpeg", ".bmp", ".tiff", ".tif", ".webp"):
            return 1
        else:
            # Default to image for unknown types
            return 1

    def _encode_file(self, file_path: Path) -> str:
        """Encode file to base64."""
        with open(file_path, "rb") as f:
            return base64.b64encode(f.read()).decode("ascii")

    def ocr_file(
        self,
        file_path: Union[str, Path],
        use_doc_orientation_classify: bool = False,
        use_doc_unwarping: bool = False,
        use_chart_recognition: bool = False,
        timeout: int = 120,
    ) -> DocumentOCRResult:
        """
        Perform OCR on a file.

        Args:
            file_path: Path to the PDF or image file.
            use_doc_orientation_classify: Enable document orientation classification.
            use_doc_unwarping: Enable document unwarping.
            use_chart_recognition: Enable chart recognition.
            timeout: Request timeout in seconds.

        Returns:
            DocumentOCRResult with OCR results or error information.
        """
        file_path = Path(file_path)

        if not file_path.exists():
            return DocumentOCRResult(
                success=False,
                pages=[],
                error_message=f"File not found: {file_path}",
            )

        if not self.is_configured:
            return DocumentOCRResult(
                success=False,
                pages=[],
                error_message="PaddleOCR is not configured. Run 'paddleocr_cli configure' first.",
            )

        # Prepare request
        url = f"{self.server_url}{self.LAYOUT_PARSING_ENDPOINT}"
        headers = {
            "Authorization": f"token {self.access_token}",
            "Content-Type": "application/json",
        }

        payload = {
            "file": self._encode_file(file_path),
            "fileType": self._get_file_type(file_path),
            "useDocOrientationClassify": use_doc_orientation_classify,
            "useDocUnwarping": use_doc_unwarping,
            "useChartRecognition": use_chart_recognition,
        }

        try:
            req = urllib.request.Request(
                url,
                data=json.dumps(payload).encode("utf-8"),
                headers=headers,
                method="POST",
            )

            with urllib.request.urlopen(req, timeout=timeout) as resp:
                response_data = json.loads(resp.read().decode("utf-8"))

        except urllib.error.HTTPError as e:
            error_body = e.read().decode("utf-8", errors="replace")
            return DocumentOCRResult(
                success=False,
                pages=[],
                error_message=f"HTTP {e.code}: {e.reason}\n{error_body}",
            )
        except urllib.error.URLError as e:
            return DocumentOCRResult(
                success=False,
                pages=[],
                error_message=f"Connection error: {e.reason}",
            )
        except TimeoutError:
            return DocumentOCRResult(
                success=False,
                pages=[],
                error_message=f"Request timed out after {timeout} seconds",
            )
        except json.JSONDecodeError as e:
            return DocumentOCRResult(
                success=False,
                pages=[],
                error_message=f"Invalid JSON response: {e}",
            )

        # Parse response
        log_id = response_data.get("logId")
        error_code = response_data.get("errorCode", -1)
        error_msg = response_data.get("errorMsg", "Unknown error")

        if error_code != 0:
            return DocumentOCRResult(
                success=False,
                pages=[],
                error_message=f"API error ({error_code}): {error_msg}",
                log_id=log_id,
            )

        result = response_data.get("result", {})
        layout_results = result.get("layoutParsingResults", [])

        pages = []
        for i, page_result in enumerate(layout_results):
            markdown_data = page_result.get("markdown", {})
            pages.append(
                OCRResult(
                    page_index=i,
                    markdown=markdown_data.get("text", ""),
                    images=markdown_data.get("images", {}),
                )
            )

        return DocumentOCRResult(
            success=True,
            pages=pages,
            log_id=log_id,
        )

    def test_connection(self, timeout: int = 10) -> tuple[bool, str]:
        """
        Test connection to the OCR server.

        Returns:
            Tuple of (success, message).
        """
        if not self.access_token:
            return False, "Access token not configured"

        url = f"{self.server_url}/health"
        headers = {
            "Authorization": f"token {self.access_token}",
        }

        try:
            req = urllib.request.Request(url, headers=headers)
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                if data.get("errorCode") == 0:
                    return True, "Connection successful"
                else:
                    return False, f"Server error: {data.get('errorMsg', 'Unknown')}"
        except urllib.error.HTTPError as e:
            return False, f"HTTP {e.code}: {e.reason}"
        except urllib.error.URLError as e:
            return False, f"Connection failed: {e.reason}"
        except Exception as e:
            return False, f"Error: {e}"
