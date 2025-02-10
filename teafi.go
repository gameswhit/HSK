package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	requestsPerBurst      = 500 // Number of requests in the burst
	maxConcurrentRequests = 500 // Limit to prevent overload
	maxRetries            = 3   // Max retry attempts
	initialBackoff        = 2 * time.Second
	maxBackoff            = 5 * time.Second
)

// API Endpoint
var url = "https://api.tea-fi.com/wallet/check-in?address=0x4d677Ad1a5FC5eE853E4d0D9cd30A88d6306a025"

// Headers for the request
var headers = map[string]string{
	"Accept":             "application/json, text/plain, */*",
	"Accept-Language":    "en-US,en;q=0.9",
	"Cache-Control":      "no-cache",
	"Origin":             "https://app.tea-fi.com",
	"Pragma":             "no-cache",
	"Priority":           "u=1, i",
	"Referer":            "https://app.tea-fi.com/",
	"Sec-CH-UA":          `"Not A(Brand";v="8", "Chromium";v="132", "Google Chrome";v="132"`,
	"Sec-CH-UA-Mobile":   "?0",
	"Sec-CH-UA-Platform": `"Windows"`,
	"Sec-Fetch-Dest":     "empty",
	"Sec-Fetch-Mode":     "cors",
	"Sec-Fetch-Site":     "same-site",
	"User-Agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
}

// Function to send POST request with retries
func sendPostRequestWithRetry(client *http.Client, i int) {
	var resp *http.Response
	var err error

	backoff := initialBackoff

	// Retry logic
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte{}))
		if err != nil {
			fmt.Printf("[Request %d] Error creating request: %v\n", i+1, err)
			return
		}

		// Add headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			break
		}

		// Retry on 403 Forbidden or network errors
		if err != nil || resp.StatusCode == 403 {
			fmt.Printf("[Request %d] Attempt %d failed (HTTP %d): %v. Retrying in %v...\n", i+1, attempt, resp.StatusCode, err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
	}

	// If request still fails after retries
	if err != nil || resp.StatusCode != 200 {
		fmt.Printf("[Request %d] Failed after %d attempts. HTTP %d\n", i+1, maxRetries, resp.StatusCode)
		return
	}
	defer resp.Body.Close()

	// Read and print the full response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[Request %d] Error reading response: %v\n", i+1, err)
		return
	}

	fmt.Printf("[Request %d] Response:\n%s\n\n", i+1, string(body))
}

func main() {
	client := &http.Client{Timeout: 10 * time.Second}
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentRequests)

	for i := 0; i < requestsPerBurst; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			sendPostRequestWithRetry(client, i)
		}(i)
	}

	wg.Wait()
	fmt.Println("All burst POST requests completed.")
}
