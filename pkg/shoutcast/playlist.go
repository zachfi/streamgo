package shoutcast

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// parsePLS parses a PLS playlist file and returns the first stream URL
func parsePLS(body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("failed to read playlist: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "File") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				url := strings.TrimSpace(parts[1])
				if url != "" {
					return url, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no stream URL found in PLS playlist")
}

// parseM3U parses an M3U playlist file and returns the first stream URL
func parseM3U(body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("failed to read playlist: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Check if it's a URL (starts with http:// or https://)
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			return line, nil
		}
	}

	return "", fmt.Errorf("no stream URL found in M3U playlist")
}

// resolvePlaylistURL checks if the URL is a playlist file and resolves it to a stream URL
func resolvePlaylistURL(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("accept", "*/*")
	req.Header.Add("user-agent", "iTunes/12.9.2 (Macintosh; OS X 10.14.3) AppleWebKit/606.4.5")

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{Dial: dialer.Dial}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	// Check if it's already a stream (has icy-metaint header)
	if resp.Header.Get("icy-metaint") != "" {
		// It's already a stream, return as-is
		return url, nil
	}

	// Read the body to check if it's a playlist
	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	content := string(bodyData)

	// Check if it's a playlist file by content type or content
	isPLS := strings.Contains(contentType, "audio/x-scpls") ||
		strings.Contains(contentType, "application/pls+xml") ||
		strings.HasSuffix(url, ".pls") ||
		strings.Contains(content, "[playlist]") ||
		strings.Contains(content, "File1=")

	isM3U := strings.Contains(contentType, "audio/mpegurl") ||
		strings.Contains(contentType, "application/vnd.apple.mpegurl") ||
		strings.HasSuffix(url, ".m3u") ||
		strings.HasSuffix(url, ".m3u8") ||
		strings.Contains(content, "#EXTM3U") ||
		(strings.HasPrefix(strings.TrimSpace(content), "http://") || strings.HasPrefix(strings.TrimSpace(content), "https://"))

	if isPLS {
		streamURL, err := parsePLS(strings.NewReader(content))
		if err != nil {
			return "", fmt.Errorf("failed to parse PLS playlist: %w", err)
		}
		return streamURL, nil
	} else if isM3U {
		streamURL, err := parseM3U(strings.NewReader(content))
		if err != nil {
			return "", fmt.Errorf("failed to parse M3U playlist: %w", err)
		}
		return streamURL, nil
	}

	return "", fmt.Errorf("URL does not appear to be a stream or playlist (Content-Type: %s)", contentType)
}
