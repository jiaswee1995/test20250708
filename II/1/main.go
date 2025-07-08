package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

func main() {
	urls := []string{
		"https://google.com",
		"https://baidu.com",
		"https://bbbbbbaidu.com",
		"https://gggggggoogle.com",
	}

	success, fail := FetchURLs(urls, 2, 3*time.Second)

	fmt.Println("成功结果:")
	for _, r := range success {
		fmt.Printf("- URL: %s | 状态码: %d | 耗时: %s\n", r.URL, r.StatusCode, r.Duration)
	}

	fmt.Println("失败结果:")
	for _, r := range fail {
		fmt.Printf("- URL: %s | 耗时: %s | 错误: %v\n", r.URL, r.Duration, r.Err)
	}
}

type RequestResult struct {
	URL        string
	StatusCode int
	Duration   time.Duration
	Err        error
	Body       string
}

func FetchURLs(urls []string, maxConcurrency int, timeoutPerRequest time.Duration) (successes []RequestResult, failures []RequestResult) {
	var (
		wg         sync.WaitGroup
		concurrent = make(chan struct{}, maxConcurrency)
		resultLock sync.Mutex
	)

	for _, url := range urls {
		url := url
		wg.Add(1)

		go func() {
			defer wg.Done()
			concurrent <- struct{}{}
			defer func() { <-concurrent }()

			ctx, cancel := context.WithTimeout(context.Background(), timeoutPerRequest)
			defer cancel()

			start := time.Now()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				resultLock.Lock()
				failures = append(failures, RequestResult{URL: url, Duration: time.Since(start), Err: err})
				resultLock.Unlock()
				return
			}

			resp, err := http.DefaultClient.Do(req)
			duration := time.Since(start)

			if err != nil {
				resultLock.Lock()
				failures = append(failures, RequestResult{URL: url, Duration: duration, Err: err})
				resultLock.Unlock()
				return
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			body := string(bodyBytes)

			result := RequestResult{
				URL:        url,
				StatusCode: resp.StatusCode,
				Duration:   duration,
				Err:        nil,
				Body:       body,
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				resultLock.Lock()
				successes = append(successes, result)
				resultLock.Unlock()
			} else {
				result.Err = fmt.Errorf("non-2xx status code: %d", resp.StatusCode)
				resultLock.Lock()
				failures = append(failures, result)
				resultLock.Unlock()
			}
		}()
	}

	wg.Wait()
	return
}
