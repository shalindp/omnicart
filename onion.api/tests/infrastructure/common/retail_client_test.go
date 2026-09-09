package requests_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"onion.api/infrastrucre/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(testing, error)
	require.Len(testing, results, 1)
	require.True(testing, results[0].IsOk(), "expected ok, got error: %v", results[0].Error)
	assert.Equal(testing, "hello", string(results[0].Response.Body))
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
	require.NoError(testing, error)
	for index, result := range results {
		assert.True(testing, result.IsOk(), "request %d: expected ok, got error: %v", index, result.Error)
	}
	assert.Equal(testing, int32(10), atomic.LoadInt32(&count))
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
	require.NoError(testing, error)
	assert.True(testing, results[0].IsOk(), "expected ok after retries, got error: %v", results[0].Error)
	assert.Equal(testing, 3, results[0].Attempts)
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
	assert.Error(testing, error)
	assert.Equal(testing, 3, results[0].Attempts)
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
	assert.Error(testing, error)
	assert.Equal(testing, 1, results[0].Attempts)
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
	require.NoError(testing, error)
	assert.True(testing, results[0].IsOk())
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
	assert.Error(testing, error)
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
	require.NoError(testing, error)
	require.True(testing, results[0].IsOk())
	assert.Equal(testing, "value1", receivedHeaders.Get("X-Custom"))
	assert.NotEmpty(testing, receivedHeaders.Get("User-Agent"))
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
	require.NoError(testing, error)
	require.True(testing, results[0].IsOk())
	assert.Equal(testing, `{"key":"value"}`, string(receivedBody))
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
	assert.Error(testing, error)
	require.NotNil(testing, results[0].Error)
	assert.Equal(testing, "custom label: status 404: unsuccessful http status", results[0].Error.Error())
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

	assert.Error(testing, error)
	assert.False(testing, results[0].IsOk())
	assert.GreaterOrEqual(testing, elapsed, 1*time.Second)
}

func TestRetailClient_EmptyRequests(testing *testing.T) {
	client := common.NewBaseRetailer(common.RetailClientConfig{
		BaseUrl:             "http://localhost:1",
		DegreeOfParallelism: 1,
	}, nil)
	defer client.Dispose()

	results, error := client.Execute(testing.Context(), nil)
	require.NoError(testing, error)
	assert.Empty(testing, results)
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
	assert.Error(testing, error)
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
	require.NoError(testing, error)

	okCount := 0
	for _, result := range results {
		if result.IsOk() {
			okCount++
		}
	}
	assert.Equal(testing, 50, okCount)
	assert.Equal(testing, int32(50), atomic.LoadInt32(&count))
}

func TestProgressTicker(testing *testing.T) {
	ticker := common.NewProgressTicker(10 * time.Millisecond)

	assert.False(testing, ticker.ShouldReport(), "first report should be false")
	time.Sleep(15 * time.Millisecond)
	assert.True(testing, ticker.ShouldReport(), "should report after interval")

	rate := ticker.RatePerSecond(100)
	assert.Positive(testing, rate)
}

func TestTerminalError(testing *testing.T) {
	request := common.RetailClientRequest{Label: "test"}
	jobItem := &common.Job{Request: request, RequestLabel: "test"}

	error := common.TerminalError(jobItem, nil, fmt.Errorf("network error"), false)
	require.NotNil(testing, error)
	assert.Equal(testing, "test: network error", error.Error())

	error = common.TerminalError(jobItem, &common.RetailClientResponse{StatusCode: 500}, nil, true)
	require.NotNil(testing, error)
	assert.Equal(testing, "test: gave up after 0 attempt(s), last status 500: unsuccessful http status", error.Error())

	error = common.TerminalError(jobItem, &common.RetailClientResponse{StatusCode: 404}, nil, false)
	require.NotNil(testing, error)
	assert.Equal(testing, "test: status 404: unsuccessful http status", error.Error())
}

func TestRetryAfterParsing(testing *testing.T) {
	response := &common.RetailClientResponse{Headers: map[string][]string{"Retry-After": {"5"}}}
	duration := common.RetryAfter(response)
	require.NotNil(testing, duration)
	assert.Equal(testing, 5*time.Second, *duration)

	response = &common.RetailClientResponse{Headers: map[string][]string{"Retry-After": {"Wed, 09 Apr 2025 10:00:00 GMT"}}}
	duration = common.RetryAfter(response)
	assert.NotNil(testing, duration)

	response = &common.RetailClientResponse{Headers: map[string][]string{}}
	duration = common.RetryAfter(response)
	assert.Nil(testing, duration)
}
