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

func main() {
	fromFlag := flag.String("from", "curl", "Input format: curl or httpie")
	toFlag := flag.String("to", "httpie", "Output format: curl or httpie")
	flag.Parse()

	inputFormat := strings.ToLower(*fromFlag)
	outputFormat := strings.ToLower(*toFlag)

	if inputFormat != "curl" && inputFormat != "httpie" {
		fmt.Println("Error: -from must be 'curl' or 'httpie'")
		os.Exit(1)
	}
	if outputFormat != "curl" && outputFormat != "httpie" {
		fmt.Println("Error: -to must be 'curl' or 'httpie'")
		os.Exit(1)
	}

	fmt.Printf("Paste your %s command, then press Ctrl+D:\n", inputFormat)

	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	input := strings.Join(lines, " ")
	input = strings.Join(strings.Fields(input), " ")

	var url, method string
	var headers []string
	var data string

	// Parse based on input format
	if inputFormat == "curl" {
		url, method, headers, data = parseCurl(input)
	} else {
		url, method, headers, data = parseHTTPie(input)
	}

	// Output based on output format
	if outputFormat == "httpie" {
		httpie := buildHTTPie(url, method, headers, data)
		fmt.Printf("\n✅ HTTPie Format:\n%s\n", httpie)
	} else {
		curl := buildCurl(url, method, headers, data)
		fmt.Printf("\n✅ Curl Format:\n%s\n", curl)
	}
}

func parseCurl(curl string) (url, method string, headers []string, data string) {
	// Extract headers
	headerRegex := regexp.MustCompile(`-H\s+'([^']+)'`)
	headerMatches := headerRegex.FindAllStringSubmatch(curl, -1)

	for _, h := range headerMatches {
		header := h[1]
		hLower := strings.ToLower(header)
		if strings.HasPrefix(hLower, "content-type:") || strings.HasPrefix(hLower, "authorization:") {
			headers = append(headers, header)
		}
	}

	cleaned := headerRegex.ReplaceAllString(curl, "")
	cleaned = strings.ReplaceAll(cleaned, "\\", "")
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	// Extract URL
	parts := strings.Fields(cleaned)
	for i, p := range parts {
		if p == "curl" && i+1 < len(parts) {
			url = strings.Trim(parts[i+1], "'")
			break
		}
	}

	// Detect HTTP method
	method = "GET"
	methodRegex := regexp.MustCompile(`-X\s+(\w+)`)
	matches := methodRegex.FindStringSubmatch(cleaned)
	if len(matches) == 2 {
		method = strings.ToUpper(matches[1])
	}

	// Extract JSON data
	dataRegex := regexp.MustCompile(`--data-raw\s+'([^']+)'|--data\s+'([^']+)'`)
	dataMatches := dataRegex.FindStringSubmatch(curl)
	if len(dataMatches) > 0 {
		if dataMatches[1] != "" {
			data = dataMatches[1]
		} else if dataMatches[2] != "" {
			data = dataMatches[2]
		}
	}

	return
}

func parseHTTPie(httpie string) (url, method string, headers []string, data string) {
	parts := strings.Fields(httpie)

	if len(parts) < 2 {
		return
	}

	// Skip "http" or "https" command
	start := 0
	if parts[0] == "http" || parts[0] == "https" {
		start = 1
	}

	// First arg after command is method or URL
	if start < len(parts) {
		if isHTTPMethod(parts[start]) {
			method = strings.ToUpper(parts[start])
			start++
		} else {
			method = "GET"
		}
	}

	// Next is URL
	if start < len(parts) {
		url = parts[start]
		start++
	}

	// Rest are headers and data
	jsonParts := make(map[string]interface{})

	for i := start; i < len(parts); i++ {
		part := parts[i]

		// Header (Header:value or Header:'value')
		if strings.Contains(part, ":") && !strings.Contains(part, "=") {
			headerParts := strings.SplitN(part, ":", 2)
			if len(headerParts) == 2 {
				name := headerParts[0]
				value := strings.Trim(headerParts[1], "'")
				headers = append(headers, fmt.Sprintf("%s: %s", name, value))
			}
		} else if strings.Contains(part, "=") {
			// Data field
			parseHTTPieDataField(part, jsonParts)
		}
	}

	// Convert collected data to JSON
	if len(jsonParts) > 0 {
		jsonBytes, _ := json.Marshal(jsonParts)
		data = string(jsonBytes)
	}

	return
}

func parseHTTPieDataField(field string, jsonParts map[string]interface{}) {
	if strings.Contains(field, ":=") {
		// Raw JSON field
		parts := strings.SplitN(field, ":=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Parse the value as JSON
			var v interface{}
			if err := json.Unmarshal([]byte(value), &v); err == nil {
				jsonParts[key] = v
			} else {
				jsonParts[key] = value
			}
		}
	} else if strings.Contains(field, "=") {
		// String field
		parts := strings.SplitN(field, "=", 2)
		if len(parts) == 2 {
			jsonParts[parts[0]] = parts[1]
		}
	}
}

func isHTTPMethod(s string) bool {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	upper := strings.ToUpper(s)
	for _, m := range methods {
		if m == upper {
			return true
		}
	}
	return false
}

func buildCurl(url, method string, headers []string, data string) string {
	cmd := "curl " + url

	if method != "GET" {
		cmd += " -X " + method
	}

	for _, h := range headers {
		cmd += fmt.Sprintf(" -H '%s'", h)
	}

	if data != "" {
		cmd += " --data-raw '" + data + "'"
	}

	return cmd
}

func buildHTTPie(url, method string, headers []string, data string) string {
	cmd := fmt.Sprintf("http %s %s", strings.ToUpper(method), url)

	for _, h := range headers {
		parts := strings.SplitN(h, ": ", 2)
		if len(parts) == 2 {
			headerName := parts[0]
			headerValue := parts[1]

			if strings.ContainsAny(headerValue, " .") {
				cmd += fmt.Sprintf(" %s:'%s'", headerName, headerValue)
			} else {
				cmd += fmt.Sprintf(" %s:%s", headerName, headerValue)
			}
		}
	}

	if data != "" {
		httpieData := jsonToHTTPie(data)
		cmd += " " + httpieData
	}

	return cmd
}

func jsonToHTTPie(data string) string {
	data = strings.TrimSpace(data)

	if !strings.HasPrefix(data, "{") {
		return data
	}

	var m map[string]interface{}
	err := json.Unmarshal([]byte(data), &m)
	if err != nil {
		return data
	}

	var parts []string
	flattenJSON("", m, &parts)
	return strings.Join(parts, " ")
}

func flattenJSON(prefix string, obj interface{}, parts *[]string) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "[" + key + "]"
			}
			flattenJSON(newPrefix, value, parts)
		}
	case []interface{}:
		for i, item := range v {
			newPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			flattenJSON(newPrefix, item, parts)
		}
	case string:
		*parts = append(*parts, fmt.Sprintf("%s=%s", prefix, v))
	case float64:
		if v == float64(int(v)) {
			*parts = append(*parts, fmt.Sprintf("%s:=%d", prefix, int(v)))
		} else {
			*parts = append(*parts, fmt.Sprintf("%s:=%v", prefix, v))
		}
	case bool:
		*parts = append(*parts, fmt.Sprintf("%s:=%v", prefix, v))
	case nil:
		*parts = append(*parts, fmt.Sprintf("%s:=null", prefix))
	default:
		*parts = append(*parts, fmt.Sprintf("%s:=%v", prefix, v))
	}
}
