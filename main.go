package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Request represents a parsed HTTP request
type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Data    string
}

func main() {
	curlFlag := flag.Bool("c", false, "Output in curl format")
	httpFlag := flag.Bool("h", false, "Output in httpie format")
	flag.Parse()

	if *curlFlag && *httpFlag {
		*httpFlag = true
	}

	fmt.Println("Paste your curl command, then press Ctrl+D:")

	// Read input
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	curlCmd := strings.Join(lines, " ")
	// Remove backslashes used for line continuation
	curlCmd = strings.ReplaceAll(curlCmd, "\\", "")
	curlCmd = strings.Join(strings.Fields(curlCmd), " ")

	// Parse curl command into structured format
	req := parseCurlCommand(curlCmd)

	// Output based on flag
	if *httpFlag {
		httpie := toHTTPie(req)
		fmt.Printf("\n✅ HTTPie Format:\n")
		fmt.Println(httpie)
	} else {
		cleanCurl := toCurl(req)
		fmt.Printf("\n✅ Cleaned Curl:\n")
		fmt.Println(cleanCurl)
	}
}

// parseCurlCommand extracts request details from a curl command
func parseCurlCommand(curl string) Request {
	req := Request{
		Method:  "GET",
		Headers: make(map[string]string),
	}

	// Extract URL
	req.URL = extractURL(curl)

	// Extract method
	req.Method = extractMethod(curl)

	// Extract headers (only keep Authorization)
	req.Headers = extractHeaders(curl)

	// Extract data
	req.Data = extractData(curl)

	return req
}

// extractURL finds the URL in the curl command
func extractURL(curl string) string {
	// Try to find URL after 'curl' command
	parts := strings.Fields(curl)
	for i, p := range parts {
		if p == "curl" && i+1 < len(parts) {
			url := parts[i+1]
			// Remove quotes if present
			url = strings.Trim(url, "'\"")
			return url
		}
	}
	
	// Fallback: look for http/https URL
	urlRegex := regexp.MustCompile(`(https?://[^\s'\"]+)`)
	matches := urlRegex.FindStringSubmatch(curl)
	if len(matches) > 0 {
		return matches[1]
	}
	
	return ""
}

// extractMethod finds the HTTP method in the curl command
func extractMethod(curl string) string {
	methodRegex := regexp.MustCompile(`-X\s+(\w+)`)
	matches := methodRegex.FindStringSubmatch(curl)
	if len(matches) == 2 {
		return strings.ToUpper(matches[1])
	}
	return "GET"
}

// extractHeaders finds all headers but only keeps Authorization
func extractHeaders(curl string) map[string]string {
	headers := make(map[string]string)
	// Support both single and double quotes
	headerRegex := regexp.MustCompile(`-H\s+['"]([^'"]+)['"]`)
	matches := headerRegex.FindAllStringSubmatch(curl, -1)

	for _, match := range matches {
		if len(match) > 1 {
			header := match[1]
			parts := strings.SplitN(header, ": ", 2)
			if len(parts) == 2 {
				headerName := strings.ToLower(parts[0])
				if headerName == "authorization" {
					headers[parts[0]] = parts[1]
				}
			}
		}
	}

	return headers
}

// extractData finds JSON data in the curl command
func extractData(curl string) string {
	// Find --data or --data-raw flag
	dataIdx := strings.Index(curl, "--data-raw")
	if dataIdx == -1 {
		dataIdx = strings.Index(curl, "--data")
	}
	if dataIdx == -1 {
		return ""
	}
	
	// Extract everything after --data/--data-raw
	remaining := curl[dataIdx:]
	
	// Find the opening quote
	quoteIdx := -1
	quoteChar := byte(0)
	for i := 0; i < len(remaining); i++ {
		if remaining[i] == '\'' || remaining[i] == '"' {
			quoteIdx = i
			quoteChar = remaining[i]
			break
		}
	}
	
	if quoteIdx == -1 {
		return ""
	}
	
	// Find the matching closing quote
	for i := quoteIdx + 1; i < len(remaining); i++ {
		if remaining[i] == quoteChar {
			return remaining[quoteIdx+1 : i]
		}
	}
	
	return ""
}

// toCurl converts Request to clean curl format
func toCurl(req Request) string {
	parts := []string{"curl", req.URL}

	if req.Method != "GET" {
		parts = append(parts, "-X", req.Method)
	}

	for name, value := range req.Headers {
		parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", name, value))
	}

	if req.Data != "" {
		parts = append(parts, "--data-raw", fmt.Sprintf("'%s'", req.Data))
	}

	return strings.Join(parts, " ")
}

// toHTTPie converts Request to HTTPie format
func toHTTPie(req Request) string {
	parts := []string{"http", req.Method, req.URL}

	for name, value := range req.Headers {
		if strings.ContainsAny(value, " .") {
			parts = append(parts, fmt.Sprintf("%s:'%s'", name, value))
		} else {
			parts = append(parts, fmt.Sprintf("%s:%s", name, value))
		}
	}

	if req.Data != "" {
		httpieData := jsonToHTTPie(req.Data)
		parts = append(parts, httpieData)
	}

	return strings.Join(parts, " ")
}

// jsonToHTTPie converts JSON data to HTTPie format (flattens first level)
func jsonToHTTPie(data string) string {
	data = strings.TrimSpace(data)

	if !strings.HasPrefix(data, "{") {
		return data
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return data
	}

	var parts []string
	for key, value := range m {
		parts = append(parts, formatHTTPieValue(key, value))
	}
	return strings.Join(parts, " ")
}

// formatHTTPieValue formats a key-value pair for HTTPie
func formatHTTPieValue(key string, value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%s=%s", key, v)
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%s:=%d", key, int(v))
		}
		return fmt.Sprintf("%s:=%v", key, v)
	case bool:
		return fmt.Sprintf("%s:=%v", key, v)
	case nil:
		return fmt.Sprintf("%s:=null", key)
	case map[string]any, []any:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%s:=%v", key, v)
		}
		return fmt.Sprintf("%s:='%s'", key, string(jsonBytes))
	default:
		return fmt.Sprintf("%s:=%v", key, v)
	}
}