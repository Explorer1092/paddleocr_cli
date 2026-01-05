// Package ocr provides the PaddleOCR API client.
package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Explorer1092/paddleocr_cli/internal/config"
)

const (
	LayoutParsingEndpoint = "/layout-parsing"
	HealthEndpoint        = "/health"
)

// FileType represents the type of file being processed.
type FileType int

const (
	FileTypePDF   FileType = 0
	FileTypeImage FileType = 1
)

// OCRResult represents the OCR result for a single page.
type OCRResult struct {
	PageIndex int               `json:"page_index"`
	Markdown  string            `json:"markdown"`
	Images    map[string]string `json:"images"`
}

// DocumentOCRResult represents the OCR result for an entire document.
type DocumentOCRResult struct {
	Success      bool        `json:"success"`
	Pages        []OCRResult `json:"pages"`
	ErrorMessage string      `json:"error_message,omitempty"`
	LogID        string      `json:"log_id,omitempty"`
}

// FullMarkdown returns combined markdown from all pages.
func (r *DocumentOCRResult) FullMarkdown() string {
	var parts []string
	for _, page := range r.Pages {
		parts = append(parts, page.Markdown)
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// Client is the PaddleOCR API client.
type Client struct {
	config     *config.Config
	httpClient *http.Client
}

// NewClient creates a new OCR client.
func NewClient(cfg *config.Config) *Client {
	if cfg == nil {
		cfg, _ = config.Load("")
	}
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// IsConfigured checks if the client is properly configured.
func (c *Client) IsConfigured() bool {
	return c.config.IsConfigured()
}

// ServerURL returns the configured server URL.
func (c *Client) ServerURL() string {
	return strings.TrimRight(c.config.PaddleOCR.ServerURL, "/")
}

// getFileType determines file type from extension.
func getFileType(filePath string) FileType {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return FileTypePDF
	case ".png", ".jpg", ".jpeg", ".bmp", ".tiff", ".tif", ".webp":
		return FileTypeImage
	default:
		return FileTypeImage
	}
}

// encodeFile encodes a file to base64.
func encodeFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// OCROptions holds options for OCR processing.
type OCROptions struct {
	UseDocOrientationClassify bool
	UseDocUnwarping           bool
	UseChartRecognition       bool
	Timeout                   time.Duration
}

// DefaultOCROptions returns default OCR options.
func DefaultOCROptions() OCROptions {
	return OCROptions{
		Timeout: 120 * time.Second,
	}
}

// OCRFile performs OCR on a file.
func (c *Client) OCRFile(filePath string, opts OCROptions) *DocumentOCRResult {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("File not found: %s", filePath),
		}
	}

	// Check if configured
	if !c.IsConfigured() {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: "PaddleOCR is not configured. Run 'paddleocr-cli configure' first.",
		}
	}

	// Encode file
	fileData, err := encodeFile(filePath)
	if err != nil {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("Failed to read file: %v", err),
		}
	}

	// Prepare request payload
	payload := map[string]interface{}{
		"file":                     fileData,
		"fileType":                 int(getFileType(filePath)),
		"useDocOrientationClassify": opts.UseDocOrientationClassify,
		"useDocUnwarping":          opts.UseDocUnwarping,
		"useChartRecognition":      opts.UseChartRecognition,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("Failed to marshal payload: %v", err),
		}
	}

	// Create request
	url := c.ServerURL() + LayoutParsingEndpoint
	req, err := http.NewRequest("POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	req.Header.Set("Authorization", "token "+c.config.PaddleOCR.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Set timeout
	client := c.httpClient
	if opts.Timeout > 0 {
		client = &http.Client{Timeout: opts.Timeout}
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("Request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("Failed to read response: %v", err),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("HTTP %d: %s\n%s", resp.StatusCode, resp.Status, string(body)),
		}
	}

	// Parse response
	var response struct {
		LogID     string `json:"logId"`
		ErrorCode int    `json:"errorCode"`
		ErrorMsg  string `json:"errorMsg"`
		Result    struct {
			LayoutParsingResults []struct {
				Markdown struct {
					Text   string            `json:"text"`
					Images map[string]string `json:"images"`
				} `json:"markdown"`
			} `json:"layoutParsingResults"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("Invalid JSON response: %v", err),
		}
	}

	if response.ErrorCode != 0 {
		return &DocumentOCRResult{
			Success:      false,
			Pages:        []OCRResult{},
			ErrorMessage: fmt.Sprintf("API error (%d): %s", response.ErrorCode, response.ErrorMsg),
			LogID:        response.LogID,
		}
	}

	// Build result
	var pages []OCRResult
	for i, layoutResult := range response.Result.LayoutParsingResults {
		images := layoutResult.Markdown.Images
		if images == nil {
			images = make(map[string]string)
		}
		pages = append(pages, OCRResult{
			PageIndex: i,
			Markdown:  layoutResult.Markdown.Text,
			Images:    images,
		})
	}

	return &DocumentOCRResult{
		Success: true,
		Pages:   pages,
		LogID:   response.LogID,
	}
}

// TestConnection tests the connection to the OCR server.
func (c *Client) TestConnection() (bool, string) {
	if c.config.PaddleOCR.AccessToken == "" {
		return false, "Access token not configured"
	}

	url := c.ServerURL() + HealthEndpoint
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "token "+c.config.PaddleOCR.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("Failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var response struct {
		ErrorCode int    `json:"errorCode"`
		ErrorMsg  string `json:"errorMsg"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return false, fmt.Sprintf("Invalid JSON response: %v", err)
	}

	if response.ErrorCode == 0 {
		return true, "Connection successful"
	}

	return false, fmt.Sprintf("Server error: %s", response.ErrorMsg)
}
