package llm

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// findEnvFile searches for .env file in current directory and parent directories
func findEnvFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		envPath := dir + "/.env"
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}

		// Move to parent directory
		parentDir := dir + "/.."
		absParent, err := absPath(parentDir)
		if err != nil {
			return "", err
		}

		// If we haven't moved, we've reached the root
		if absParent == dir {
			break
		}
		dir = absParent
	}

	return "", fmt.Errorf(".env file not found")
}

// absPath returns the absolute path of a path
func absPath(path string) (string, error) {
	if strings.HasPrefix(path, "/") {
		return path, nil
	}
	return filepath.Abs(path)
}

// loadEnv loads environment variables from .env file
// This is a simple implementation that reads KEY=VALUE pairs
// It searches for .env in the current directory and parent directories
func loadEnv() error {
	// Find .env file by searching upward from current directory
	envPath, err := findEnvFile()
	if err != nil {
		return fmt.Errorf("failed to find .env file: %w", err)
	}

	file, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = strings.Trim(value, "'")
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestGenerateStream tests the streaming generation functionality
func TestGenerateStream(t *testing.T) {
	// Load .env file
	if err := loadEnv(); err != nil {
		t.Skipf("Skipping test: %v (ensure .env file exists in project root)", err)
		return
	}

	// Get configuration from environment
	apiKey := getEnv("OPENAI_API_KEY", "")
	if apiKey == "" {
		t.Skip("Skipping test: OPENAI_API_KEY not set in .env file")
		return
	}

	baseURL := getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1")

	// Create OpenAI client
	client := NewOpenAI(apiKey, baseURL).(*OpenAI)

	// Test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test prompt
	testPrompt := "Explain what C programming language is in 2-3 sentences."

	t.Log("Starting streaming generation test...")
	t.Logf("Using base URL: %s", baseURL)
	t.Logf("Test prompt: %s", testPrompt)

	// Start streaming
	streamChan := client.GenerateStream(ctx, testPrompt)

	// Collect streamed content
	var fullContent strings.Builder
	chunkCount := 0

	for chunk := range streamChan {
		chunkCount++
		fullContent.WriteString(chunk)
		t.Logf("Chunk #%d: %q", chunkCount, chunk)
	}

	// Verify results
	t.Logf("\nTest completed:")
	t.Logf("- Total chunks received: %d", chunkCount)
	t.Logf("- Total content length: %d characters", fullContent.Len())
	t.Logf("- Full content: %s", fullContent.String())

	if chunkCount == 0 {
		t.Error("Expected to receive at least one chunk, but got none")
	}

	if fullContent.Len() == 0 {
		t.Error("Expected to receive some content, but got empty string")
	}
}

// TestGenerateResponse tests the Q&A response streaming functionality
func TestGenerateResponse(t *testing.T) {
	// Load .env file
	if err := loadEnv(); err != nil {
		t.Skipf("Skipping test: %v (ensure .env file exists in project root)", err)
		return
	}

	// Get configuration from environment
	apiKey := getEnv("OPENAI_API_KEY", "")
	if apiKey == "" {
		t.Skip("Skipping test: OPENAI_API_KEY not set in .env file")
		return
	}

	baseURL := getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1")

	// Create OpenAI client
	client := NewOpenAI(apiKey, baseURL).(*OpenAI)

	// Test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test question and context
	question := "What is the main topic?"
	context := "This podcast is about the history of computer programming languages."

	t.Log("Starting Q&A response streaming test...")
	t.Logf("Question: %s", question)
	t.Logf("Context: %s", context)

	// Start streaming
	streamChan := client.GenerateResponse(ctx, question, context)

	// Collect streamed content
	var fullContent strings.Builder
	chunkCount := 0

	for chunk := range streamChan {
		chunkCount++
		fullContent.WriteString(chunk)
		t.Logf("Chunk #%d: %q", chunkCount, chunk)
	}

	// Verify results
	t.Logf("\nTest completed:")
	t.Logf("- Total chunks received: %d", chunkCount)
	t.Logf("- Total content length: %d characters", fullContent.Len())
	t.Logf("- Full response: %s", fullContent.String())

	if chunkCount == 0 {
		t.Error("Expected to receive at least one chunk, but got none")
	}

	if fullContent.Len() == 0 {
		t.Error("Expected to receive some content, but got empty string")
	}
}

// TestLoadEnv tests the .env loading functionality
func TestLoadEnv(t *testing.T) {
	// Create a temporary .env file for testing
	tmpFile, err := os.CreateTemp("", ".env-test-*.tmp")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test content
	testContent := `# This is a comment
TEST_KEY_1=value1
TEST_KEY_2="quoted value"
TEST_KEY_3='single quoted'

# Another comment
TEST_KEY_4=value with spaces
`
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Load the file (we'll modify loadEnv to accept a filename for this test)
	// For now, just verify the parsing logic works
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open temp file: %v", err)
	}
	defer file.Close()

	// Parse and set env vars
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = strings.Trim(value, "'")
		}

		os.Setenv(key, value)
	}

	// Verify parsed values
	tests := []struct {
		key      string
		expected string
	}{
		{"TEST_KEY_1", "value1"},
		{"TEST_KEY_2", "quoted value"},
		{"TEST_KEY_3", "single quoted"},
		{"TEST_KEY_4", "value with spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := os.Getenv(tt.key); got != tt.expected {
				t.Errorf("Expected %s=%q, got %q", tt.key, tt.expected, got)
			} else {
				t.Logf("✓ %s=%q", tt.key, got)
			}
		})
	}
}
