package requests_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"onion.api/infrastrucre/common"
)

func TestRetailClient_SingleRequest(testing *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("hello"))
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 2,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/test"},
	})
	if error != nil {
		testing.Fatal(error)
	}
	if len(results) != 1 {
		testing.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsOk() {
		testing.Fatalf("expected ok, got error: %v", results[0].Error)
	}
	if string(results[0].Response.Body) != "hello" {
		testing.Fatalf("expected 'hello', got '%s'", string(results[0].Response.Body))
	}
}

func TestRetailClient_MultipleRequests(testing *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&count, 1)
		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("ok"))
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 4,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	requestList := make([]common.RetailClientRequest, 10)
	for index := range requestList {
		requestList[index] = common.RetailClientRequest{Method: "GET", Path: "/item"}
	}

	results, error := client.Execute(testing.Context(), requestList)
	if error != nil {
		testing.Fatal(error)
	}
	for index, result := range results {
		if !result.IsOk() {
			testing.Fatalf("request %d: expected ok, got error: %v", index, result.Error)
		}
	}
	if atomic.LoadInt32(&count) != 10 {
		testing.Fatalf("expected 10 requests, got %d", atomic.LoadInt32(&count))
	}
}

func TestRetailClient_RetryOn500(testing *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		number := atomic.AddInt32(&count, 1)
		if number < 3 {
			responseWriter.WriteHeader(500)
			return
		}
		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("ok"))
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        3,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/retry"},
	})
	if error != nil {
		testing.Fatal(error)
	}
	if !results[0].IsOk() {
		testing.Fatalf("expected ok after retries, got error: %v", results[0].Error)
	}
	if results[0].Attempts != 3 {
		testing.Fatalf("expected 3 attempts, got %d", results[0].Attempts)
	}
}

func TestRetailClient_ExhaustsRetries(testing *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(500)
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        2,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/fail"},
	})
	if error == nil {
		testing.Fatal("expected error after exhausting retries")
	}
	if results[0].Attempts != 3 {
		testing.Fatalf("expected 3 attempts, got %d", results[0].Attempts)
	}
}

func TestRetailClient_NonRetryableStatus(testing *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(404)
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        3,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/notfound"},
	})
	if error == nil {
		testing.Fatal("expected error for 404")
	}
	if results[0].Attempts != 1 {
		testing.Fatalf("expected 1 attempt for non-retryable, got %d", results[0].Attempts)
	}
}

func TestRetailClient_AbsoluteUrl(testing *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("absolute"))
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		DegreeOfParallelism: 1,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: server.URL + "/abs"},
	})
	if error != nil {
		testing.Fatal(error)
	}
	if !results[0].IsOk() {
		testing.Fatalf("expected ok, got error: %v", results[0].Error)
	}
}

func TestRetailClient_DisposeFailsPending(testing *testing.T) {
	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "http://localhost:1",
		DegreeOfParallelism: 1,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             100 * time.Millisecond,
	}, nil)

	client.Dispose()

	_, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/"},
	})
	if error == nil {
		testing.Fatal("expected error after dispose")
	}
}

func TestRetailClient_PreservesHeaders(testing *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		receivedHeaders = request.Header.Clone()
		responseWriter.WriteHeader(200)
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
		DefaultHeaders: map[string][]string{
			"X-Custom": {"value1"},
		},
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/headers"},
	})
	if error != nil {
		testing.Fatal(error)
	}
	if !results[0].IsOk() {
		testing.Fatal("expected ok")
	}
	if receivedHeaders.Get("X-Custom") != "value1" {
		testing.Fatalf("expected X-Custom header, got %v", receivedHeaders.Get("X-Custom"))
	}
	if receivedHeaders.Get("User-Agent") == "" {
		testing.Fatal("expected User-Agent header")
	}
}

func TestRetailClient_SessionMinting(testing *testing.T) {
	var mintCount int32
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/mint" {
			atomic.AddInt32(&mintCount, 1)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.Write([]byte(`{"token":"abc123"}`))
			return
		}
		authorization := request.Header.Get("Authorization")
		if authorization != "Bearer abc123" {
			responseWriter.WriteHeader(401)
			return
		}
		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("ok"))
	}))
	defer server.Close()

	session := &testSession{
		token:   "",
		headers: nil,
	}

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
		Session:             session,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/data"},
	})
	if error != nil {
		testing.Fatal(error)
	}
	if !results[0].IsOk() {
		testing.Fatalf("expected ok, got error: %v", results[0].Error)
	}
	if atomic.LoadInt32(&mintCount) != 1 {
		testing.Fatalf("expected 1 mint, got %d", atomic.LoadInt32(&mintCount))
	}
}

func TestRetailClient_SessionRefresh(testing *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/mint" && request.Method == "POST" {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.Write([]byte(`{"token":"newtoken"}`))
			return
		}
		authorization := request.Header.Get("Authorization")
		if authorization != "Bearer newtoken" {
			responseWriter.WriteHeader(401)
			return
		}
		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("ok"))
	}))
	defer server.Close()

	session := &testSession{
		token:   "oldtoken",
		headers: map[string][]string{"Authorization": {"Bearer oldtoken"}},
	}

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        1,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
		Session:             session,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/data"},
	})
	if error != nil {
		testing.Fatal(error)
	}
	if !results[0].IsOk() {
		testing.Fatalf("expected ok, got error: %v", results[0].Error)
	}
}

func TestRetailClient_BodyRequest(testing *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		receivedBody, _ = io.ReadAll(request.Body)
		responseWriter.WriteHeader(200)
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "POST", Path: "/post", Body: []byte(`{"key":"value"}`)},
	})
	if error != nil {
		testing.Fatal(error)
	}
	if !results[0].IsOk() {
		testing.Fatalf("expected ok, got error: %v", results[0].Error)
	}
	if string(receivedBody) != `{"key":"value"}` {
		testing.Fatalf("expected body, got '%s'", string(receivedBody))
	}
}

func TestRetailClient_Label(testing *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(404)
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/test", Label: "custom label"},
	})
	if error == nil {
		testing.Fatal("expected error from all-failed batch")
	}
	if results[0].Error == nil {
		testing.Fatal("expected error")
	}
	if results[0].Error.Error() != "custom label: status 404: unsuccessful http status" {
		testing.Fatalf("unexpected error: %v", results[0].Error)
	}
}

func TestRetailClient_RetryAfter(testing *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Retry-After", "1")
		responseWriter.WriteHeader(429)
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 1,
		NumOfRetries:        1,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	startedAt := time.Now()
	results, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/throttle"},
	})
	elapsed := time.Since(startedAt)

	if error == nil {
		testing.Fatal("expected error after retries")
	}
	if results[0].IsOk() {
		testing.Fatal("expected error after retries")
	}
	if elapsed < 1*time.Second {
		testing.Fatalf("expected at least 1s delay from Retry-After, got %v", elapsed)
	}
}

func TestRetailClient_EmptyRequests(testing *testing.T) {
	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "http://localhost:1",
		DegreeOfParallelism: 1,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), nil)
	if error != nil {
		testing.Fatal(error)
	}
	if len(results) != 0 {
		testing.Fatalf("expected 0 results, got %d", len(results))
	}
}

// testSession is a simple test implementation of RetailSession.
type testSession struct {
	mutex    sync.Mutex
	token    string
	headers  map[string][]string
	minted   bool
	expired  bool
}

func (session *testSession) Prime() error { return nil }

func (session *testSession) MintRequest() common.RetailClientRequest {
	return common.RetailClientRequest{
		Method: "POST",
		Path:   "/mint",
		Label:  "test mint",
	}
}

func (session *testSession) TryAccept(response common.RetailClientResponse) bool {
	var parsed struct {
		Token string `json:"token"`
	}
	if error := json.Unmarshal(response.Body, &parsed); error != nil {
		return false
	}
	if parsed.Token == "" {
		return false
	}
	session.mutex.Lock()
	session.token = parsed.Token
	session.headers = map[string][]string{"Authorization": {"Bearer " + parsed.Token}}
	session.minted = true
	session.mutex.Unlock()
	return true
}

func (session *testSession) Headers(currentTime time.Time) map[string][]string {
	session.mutex.Lock()
	defer session.mutex.Unlock()
	if session.token == "" {
		return nil
	}
	return session.headers
}

func (session *testSession) IsExpired(response common.RetailClientResponse) bool {
	return response.StatusCode == 401 || response.StatusCode == 403
}

func (session *testSession) Invalidate() {
	session.mutex.Lock()
	session.token = ""
	session.headers = nil
	session.expired = true
	session.mutex.Unlock()
}

func TestRetailClient_NoBaseUrlError(testing *testing.T) {
	client := common.NewBaseRetailer(common.RetailClientConfig{
		DegreeOfParallelism: 1,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	_, error := client.Execute(testing.Context(), []common.RetailClientRequest{
		{Method: "GET", Path: "/relative"},
	})
	if error == nil {
		testing.Fatal("expected error for relative path with empty BaseUrl")
	}
}

func TestRetailClient_ConcurrentRequests(testing *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&count, 1)
		time.Sleep(10 * time.Millisecond)
		responseWriter.WriteHeader(200)
	}))
	defer server.Close()

	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             server.URL,
		DegreeOfParallelism: 8,
		NumOfRetries:        0,
		DelayInMs:           10,
		Timeout:             5 * time.Second,
	}, nil)
	defer client.Dispose()

	requestList := make([]common.RetailClientRequest, 50)
	for index := range requestList {
		requestList[index] = common.RetailClientRequest{Method: "GET", Path: "/concurrent"}
	}

	results, error := client.Execute(testing.Context(), requestList)
	if error != nil {
		testing.Fatal(error)
	}

	okCount := 0
	for _, result := range results {
		if result.IsOk() {
			okCount++
		}
	}
	if okCount != 50 {
		testing.Fatalf("expected 50 ok results, got %d", okCount)
	}
	if atomic.LoadInt32(&count) != 50 {
		testing.Fatalf("expected 50 requests, got %d", atomic.LoadInt32(&count))
	}
}

func TestProgressTicker(testing *testing.T) {
	ticker := common.NewProgressTicker(10 * time.Millisecond)

	if ticker.ShouldReport() {
		testing.Fatal("expected first report to be false (nextAt = start + interval)")
	}
	time.Sleep(15 * time.Millisecond)
	if !ticker.ShouldReport() {
		testing.Fatal("expected report after interval")
	}

	rate := ticker.RatePerSecond(100)
	if rate <= 0 {
		testing.Fatalf("expected positive rate, got %f", rate)
	}
}

func TestTerminalError(testing *testing.T) {
	request := common.RetailClientRequest{Label: "test"}
	jobItem := &common.Job{Request: request, RequestLabel: "test"}

	error := common.TerminalError(jobItem, nil, fmt.Errorf("network error"), false)
	if error == nil {
		testing.Fatal("expected error")
	}
	if error.Error() != "test: network error" {
		testing.Fatalf("unexpected error: %v", error)
	}

	error = common.TerminalError(jobItem, &common.RetailClientResponse{StatusCode: 500}, nil, true)
	if error == nil {
		testing.Fatal("expected error")
	}
	if error.Error() != "test: gave up after 0 attempt(s), last status 500: unsuccessful http status" {
		testing.Fatalf("unexpected error: %v", error)
	}

	error = common.TerminalError(jobItem, &common.RetailClientResponse{StatusCode: 404}, nil, false)
	if error == nil {
		testing.Fatal("expected error")
	}
	if error.Error() != "test: status 404: unsuccessful http status" {
		testing.Fatalf("unexpected error: %v", error)
	}
}

func TestWithSession(testing *testing.T) {
	request := common.RetailClientRequest{
		Method:  "GET",
		Path:    "/test",
		Headers: map[string][]string{"X-Custom": {"value"}},
	}
	sessionHeaders := map[string][]string{"Authorization": {"Bearer token"}}

	merged := common.WithSession(request, sessionHeaders)
	if merged.Headers["Authorization"][0] != "Bearer token" {
		testing.Fatal("expected session header")
	}
	if merged.Headers["X-Custom"][0] != "value" {
		testing.Fatal("expected original header")
	}

	noSession := common.WithSession(request, nil)
	if noSession.Headers["X-Custom"][0] != "value" {
		testing.Fatal("expected original header when no session")
	}
}

func TestRetryAfterParsing(testing *testing.T) {
	response := &common.RetailClientResponse{Headers: map[string][]string{"Retry-After": {"5"}}}
	duration := common.RetryAfter(response)
	if duration == nil || *duration != 5*time.Second {
		testing.Fatalf("expected 5s, got %v", duration)
	}

	response = &common.RetailClientResponse{Headers: map[string][]string{"Retry-After": {"Wed, 09 Apr 2025 10:00:00 GMT"}}}
	duration = common.RetryAfter(response)
	if duration == nil {
		testing.Fatal("expected retry-after from date")
	}

	response = &common.RetailClientResponse{Headers: map[string][]string{}}
	duration = common.RetryAfter(response)
	if duration != nil {
		testing.Fatalf("expected nil, got %v", duration)
	}
}
