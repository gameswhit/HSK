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
	requestsPerBurst      = 900 // Number of requests in the burst
	maxConcurrentRequests = 900 // Limit to prevent overload
	maxRetries            = 3   // Max retry attempts
	initialBackoff        = 2 * time.Second
	maxBackoff            = 5 * time.Second
)

// API Endpoints (Menambahkan lebih dari 1 URL)
var urls = []string{
	"https://api.tea-fi.com/wallet/check-in?address=0x6833c0295a917a9897e6fe87ffc5e6306dc1901a",
	"https://api.tea-fi.com/wallet/check-in?address=0xcd69973e251e3e57ea37fc5b27c63ea995274ffb",
	"https://api.tea-fi.com/wallet/check-in?address=0xf0d710cfe518f24110b92dbcf68c033d043e0bba",
	"https://api.tea-fi.com/wallet/check-in?address=0x3752a88b483df3837dd092f0f28d02eca77718f7",
	"https://api.tea-fi.com/wallet/check-in?address=0xec145fbb08ea3c769f3d4693cdb00d82f4005531",
	"https://api.tea-fi.com/wallet/check-in?address=0x0e856dffd837d88caffbb1dede1d890429f69710",
	"https://api.tea-fi.com/wallet/check-in?address=0xe0a4f8fc24cbe0cb2533b192c5fe2f6974f7b880",
}

// Headers for the request
var headers = map[string]string{
	"Accept":             "application/json, text/plain, */*",
	"Accept-Language":    "en-US,en;q=0.9",
	"Cache-Control":      "no-cache",
	"Origin":             "https://app.tea-fi.com",
	"Pragma":             "no-cache",
	"Priority":           "u=1, i",
	"Referer":            "https://app.tea-fi.com/",
	"Sec-CH-UA":          `"Not A(Brand)";v="8", "Chromium";v="132", "Google Chrome";v="132"`,
	"Sec-CH-UA-Mobile":   "?0",
	"Sec-CH-UA-Platform": `"Windows"`,
	"Sec-Fetch-Dest":     "empty",
	"Sec-Fetch-Mode":     "cors",
	"Sec-Fetch-Site":     "same-site",
	"User-Agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
}


// Function to send POST request with retries
func sendPostRequestWithRetry(client *http.Client, url string, requestID int) {
	var resp *http.Response
	var err error

	backoff := initialBackoff

	// Retry logic
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte{}))
		if err != nil {
			fmt.Printf("[Request %d | %s] Error creating request: %v\n", requestID, url, err)
			return
		}

		// Add headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err = client.Do(req)

		// Handle response properly
		if err == nil && resp.StatusCode == 200 {
			break
		}

		// Retry on 403 Forbidden or network errors
		if err != nil || (resp != nil && resp.StatusCode == 403) {
			fmt.Printf("[Request %d | %s] Attempt %d failed (HTTP %d): %v. Retrying in %v...\n",
				requestID, url, attempt, resp.StatusCode, err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
	}

	// If request still fails after retries
	if err != nil || resp == nil || resp.StatusCode != 200 {
		fmt.Printf("[Request %d | %s] Failed after %d attempts. HTTP %d\n",
			requestID, url, maxRetries, resp.StatusCode)
		return
	}
	defer resp.Body.Close()

	// Read and print the full response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[Request %d | %s] Error reading response: %v\n", requestID, url, err)
		return
	}

	fmt.Printf("[Request %d | %s] Response:\n%s\n\n", requestID, url, string(body))
}

func main() {
	client := &http.Client{Timeout: 10 * time.Second}
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentRequests)

	// Loop untuk menjalankan requests untuk kedua URL
	for _, url := range urls {
		for i := 0; i < requestsPerBurst; i++ {
			wg.Add(1)
			sem <- struct{}{}

			go func(i int, url string) {
				defer wg.Done()
				defer func() { <-sem }()

				sendPostRequestWithRetry(client, url, i)
			}(i, url)
		}
	}

	wg.Wait()
	fmt.Println("All burst POST requests completed.")
}
