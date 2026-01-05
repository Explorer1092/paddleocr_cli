// PaddleOCR CLI - A command-line tool for OCR using PaddleOCR AI Studio API.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Explorer1092/paddleocr_cli/internal/config"
	"github.com/Explorer1092/paddleocr_cli/internal/ocr"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "paddleocr-cli",
	Short: "OCR documents using PaddleOCR AI Studio API",
	Long: `PaddleOCR CLI - A command-line tool for OCR using PaddleOCR AI Studio API.

Examples:
  paddleocr-cli resume.pdf                    # OCR and print to stdout
  paddleocr-cli resume.pdf -o output.md       # OCR and save to file
  paddleocr-cli resume.pdf --json             # Output as JSON
  paddleocr-cli configure                     # Configure credentials
  paddleocr-cli configure --show              # Show current config
  paddleocr-cli configure --test              # Test connection`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		runOCR(cmd, args)
	},
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure PaddleOCR credentials",
	Long:  "Configure or view PaddleOCR API credentials",
	Run:   runConfigure,
}

// OCR flags
var (
	outputFile    string
	jsonOutput    bool
	pageNum       int
	noSeparator   bool
	timeout       int
	orientation   bool
	unwarp        bool
	chart         bool
	quiet         bool
	configFile    string
)

// Configure flags
var (
	token      string
	serverURL  string
	showConfig bool
	testConn   bool
	locations  bool
	scope      string
)

func init() {
	// OCR flags (on root command)
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON instead of markdown")
	rootCmd.Flags().IntVar(&pageNum, "page", -1, "Extract only page N (0-indexed)")
	rootCmd.Flags().BoolVar(&noSeparator, "no-separator", false, "Don't add page separators in markdown output")
	rootCmd.Flags().IntVar(&timeout, "timeout", 120, "Request timeout in seconds")
	rootCmd.Flags().BoolVar(&orientation, "orientation", false, "Enable document orientation classification")
	rootCmd.Flags().BoolVar(&unwarp, "unwarp", false, "Enable document unwarping")
	rootCmd.Flags().BoolVar(&chart, "chart", false, "Enable chart recognition")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress messages")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Path to config file")

	// Configure flags
	configureCmd.Flags().StringVar(&token, "token", "", "Set the access token")
	configureCmd.Flags().StringVar(&serverURL, "server-url", "", "Set the server URL")
	configureCmd.Flags().BoolVar(&showConfig, "show", false, "Show current configuration")
	configureCmd.Flags().BoolVar(&testConn, "test", false, "Test connection to the server")
	configureCmd.Flags().BoolVar(&locations, "locations", false, "Show config file search locations")
	configureCmd.Flags().StringVarP(&scope, "scope", "s", "user", "Installation scope: user, project, or local")

	rootCmd.AddCommand(configureCmd)
}

func runOCR(cmd *cobra.Command, args []string) {
	filePath := args[0]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File not found: %s\n", filePath)
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	client := ocr.NewClient(cfg)

	if !client.IsConfigured() {
		fmt.Fprintln(os.Stderr, "Error: PaddleOCR is not configured.")
		fmt.Fprintln(os.Stderr, "Run 'paddleocr-cli configure' to set up credentials.")
		os.Exit(1)
	}

	// Perform OCR
	if !quiet {
		fmt.Fprintf(os.Stderr, "Processing: %s\n", filePath)
	}

	opts := ocr.OCROptions{
		UseDocOrientationClassify: orientation,
		UseDocUnwarping:           unwarp,
		UseChartRecognition:       chart,
		Timeout:                   time.Duration(timeout) * time.Second,
	}

	result := client.OCRFile(filePath, opts)

	if !result.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.ErrorMessage)
		os.Exit(1)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "OCR completed: %d page(s)\n", len(result.Pages))
	}

	// Format output
	var output string
	if jsonOutput {
		outputData := map[string]interface{}{
			"success": true,
			"pages":   result.Pages,
			"log_id":  result.LogID,
		}
		jsonBytes, err := json.MarshalIndent(outputData, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to marshal JSON: %v\n", err)
			os.Exit(1)
		}
		output = string(jsonBytes)
	} else {
		// Markdown output
		if pageNum >= 0 {
			if pageNum < len(result.Pages) {
				output = result.Pages[pageNum].Markdown
			} else {
				fmt.Fprintf(os.Stderr, "Error: Page %d not found (document has %d pages)\n", pageNum, len(result.Pages))
				os.Exit(1)
			}
		} else if noSeparator {
			var parts []string
			for _, page := range result.Pages {
				parts = append(parts, page.Markdown)
			}
			output = strings.Join(parts, "\n\n")
		} else {
			output = result.FullMarkdown()
		}
	}

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write output: %v\n", err)
			os.Exit(1)
		}
		if !quiet {
			fmt.Fprintf(os.Stderr, "Output saved to: %s\n", outputFile)
		}
	} else {
		fmt.Println(output)
	}
}

func runConfigure(cmd *cobra.Command, args []string) {
	// Show config locations
	if locations {
		fmt.Println("Configuration file search locations:\n")
		for _, loc := range config.GetConfigLocations() {
			status := "[not found]"
			if loc.Exists {
				status = "[FOUND]"
			}
			fmt.Printf("  %-12s %s\n", status, loc.Description)
			fmt.Printf("             %s\n\n", loc.Path)
		}
		return
	}

	// Load current config
	configPath := config.FindConfig()
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Show current config
	if showConfig {
		fmt.Println("Current configuration:\n")
		if configPath != "" {
			fmt.Printf("  Config file: %s\n", configPath)
		} else {
			fmt.Println("  Config file: (none found)")
		}
		fmt.Println()
		serverDisplay := cfg.PaddleOCR.ServerURL
		if serverDisplay == "" {
			serverDisplay = "(not set)"
		}
		fmt.Printf("  Server URL:   %s\n", serverDisplay)
		tokenDisplay := "(not set)"
		if len(cfg.PaddleOCR.AccessToken) > 8 {
			tokenDisplay = "***" + cfg.PaddleOCR.AccessToken[len(cfg.PaddleOCR.AccessToken)-8:]
		}
		fmt.Printf("  Access token: %s\n", tokenDisplay)
		return
	}

	// Test connection
	if testConn {
		if !cfg.IsConfigured() {
			fmt.Fprintln(os.Stderr, "Error: server_url and access_token must be configured first.")
			fmt.Fprintln(os.Stderr, "Run: paddleocr-cli configure --server-url URL --token TOKEN")
			os.Exit(1)
		}
		fmt.Println("Testing connection to PaddleOCR server...")
		client := ocr.NewClient(cfg)
		success, message := client.TestConnection()
		if success {
			fmt.Printf("  [OK] %s\n", message)
		} else {
			fmt.Printf("  [FAILED] %s\n", message)
			os.Exit(1)
		}
		return
	}

	// Update config
	if token == "" && serverURL == "" {
		fmt.Fprintln(os.Stderr, "Usage: paddleocr-cli configure --server-url URL --token TOKEN [-s SCOPE]")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		fmt.Fprintln(os.Stderr, "  --server-url URL   Set the server URL (required)")
		fmt.Fprintln(os.Stderr, "  --token TOKEN      Set the access token (required)")
		fmt.Fprintln(os.Stderr, "  -s, --scope SCOPE  Installation scope (default: user)")
		fmt.Fprintln(os.Stderr, "                     user    - ~/.config/paddleocr_cli/")
		fmt.Fprintln(os.Stderr, "                     project - project root (alongside .claude/)")
		fmt.Fprintln(os.Stderr, "                     local   - current directory")
		fmt.Fprintln(os.Stderr, "  --show             Show current configuration")
		fmt.Fprintln(os.Stderr, "  --test             Test connection")
		os.Exit(1)
	}

	if token != "" {
		cfg.PaddleOCR.AccessToken = token
	}

	if serverURL != "" {
		cfg.PaddleOCR.ServerURL = serverURL
	}

	// Determine save path based on scope
	savePath, err := config.GetSavePath(scope)
	if err != nil {
		if scope == "project" {
			fmt.Fprintln(os.Stderr, "Error: No project root found (no .claude/ directory in parent paths)")
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	if err := config.Save(cfg, savePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration saved to: %s\n", savePath)
}
