package common

import (
	"bytes"
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"onion.api/infrastrucre/common/responses"
	"onion.api/persistence/entities"
)

// IRetailer is the interface that all retailer clients must implement.
type IRetailer interface {
	GetProducts(context context.Context) ([]responses.ScrapedProductResponse, error)
	GetStores(context context.Context) ([]responses.StoreResponse, error)
}

const (
	DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36"
	DefaultTimeout    = 30 * time.Second
	MaxBackoff        = 60 * time.Second
	HeartbeatInterval = 15 * time.Second
	JitterFraction    = 0.25
	MaxBodySize       = 10 * 1024 * 1024
)

// RetailClientRequest represents a single HTTP request to be executed.
// RetailClientRequest is a single HTTP request to be queued and executed.
type RetailClientRequest struct {
	Method  string
	Path    string
	Body    []byte
	Headers map[string][]string
	Label   string
	Meta    interface{}
}

// RetailClientResponse is the result of an HTTP request.
type RetailClientResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string][]string
}

// RetailClientResult is the outcome of a single request after retries.
type RetailClientResult struct {
	Request  RetailClientRequest
	Response *RetailClientResponse
	Error    error
	Attempts int
}

// IsOk returns true if the request succeeded.
func (result RetailClientResult) IsOk() bool {
	return result.Error == nil && result.Response != nil
}

// WasCancelled returns true if the request was cancelled by the caller.
func (result RetailClientResult) WasCancelled() bool {
	currentError := result.Error
	for currentError != nil {
		var timeout *RetailClientTimeoutError
		if errors.As(currentError, &timeout) {
			return false
		}
		if errors.Is(currentError, context.Canceled) {
			return true
		}
		currentError = errors.Unwrap(currentError)
	}
	return false
}

// RetailClientConfig configures the BaseRetailer.
type RetailClientConfig struct {
	BaseUrl             string
	DegreeOfParallelism int
	NumOfRetries        int
	DelayInMs           int
	DelayMaxInMs        int
	Timeout             time.Duration
	DefaultHeaders      map[string][]string
	Retryable           func(*RetailClientResponse, error) bool
	Session             RetailSession
	Name                string
	Logger              Logger
}

// Logger is a minimal logging interface.
type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

// RetailClientError is a general client error.
type RetailClientError struct {
	Message string
	Inner   error
}

// Error returns the error message.
func (error *RetailClientError) Error() string {
	return error.Message
}

// Unwrap returns the inner error.
func (error *RetailClientError) Unwrap() error { return error.Inner }

// RetailClientHTTPStatusError is an error from an HTTP status code.
type RetailClientHTTPStatusError struct {
	Message    string
	StatusCode int
}

// Error returns the error message.
func (error *RetailClientHTTPStatusError) Error() string { return error.Message }

// RetailClientTimeoutError is a timeout error.
type RetailClientTimeoutError struct {
	Message string
	Timeout time.Duration
	Inner   error
}

// Error returns the error message.
func (error *RetailClientTimeoutError) Error() string {
	return error.Message
}

// Unwrap returns the inner error.
func (error *RetailClientTimeoutError) Unwrap() error { return error.Inner }

// RetailClientSessionError is a session-related error.
type RetailClientSessionError struct {
	Message string
	Inner   error
}

// Error returns the error message.
func (error *RetailClientSessionError) Error() string {
	return error.Message
}

// Unwrap returns the inner error.
func (error *RetailClientSessionError) Unwrap() error { return error.Inner }

// BaseRetailer is one queue per retailer, drained in fixed-size batches by a single pump.
// It also holds the database queries for retailers that need persistence.
type BaseRetailer struct {
	httpClient     *http.Client
	baseUrl        string
	degree         int
	numRetries     int
	delayMin       time.Duration
	delayMax       time.Duration
	timeout        time.Duration
	defaultHeaders map[string][]string
	retryable      func(*RetailClientResponse, error) bool
	session        RetailSession
	name           string
	Logger         Logger
	Queries        *entities.Queries

	mutex       sync.Mutex
	queue       *list.List
	work        chan struct{}
	done        chan struct{}
	pumpOnce    sync.Once
	disposeOnce sync.Once

	nextBatchAt time.Time
	backoff     time.Duration

	succeeded int64
	rejected  int64
	failed    int64
	retries   int64

	heartbeat *progressTicker
}

// Job represents an internal unit of work in the client queue.
type Job struct {
	Request      RetailClientRequest
	token        context.Context
	cancel       context.CancelFunc
	completion   chan RetailClientResult
	attempts     int32
	reauthorised bool
	RequestLabel string
	settled      int32
}

// isSettled returns true if the job has already completed.
func (jobItem *Job) isSettled() bool {
	return atomic.LoadInt32(&jobItem.settled) == 1
}

// markSettled marks the job as settled. Returns true if this is the first call.
func (jobItem *Job) markSettled() bool {
	return atomic.CompareAndSwapInt32(&jobItem.settled, 0, 1)
}

type attempt struct {
	response *RetailClientResponse
	error    error
}

// NewBaseRetailer creates a new BaseRetailer.
func NewBaseRetailer(config RetailClientConfig, httpClient *http.Client) *BaseRetailer {
	degree := config.DegreeOfParallelism
	if degree < 1 {
		degree = 1
	}

	numRetries := config.NumOfRetries
	if numRetries < 0 {
		numRetries = 0
	}

	delayMin := time.Duration(maxInt(config.DelayInMs, 0)) * time.Millisecond

	delayMax := delayMin
	if config.DelayMaxInMs > config.DelayInMs {
		delayMax = time.Duration(config.DelayMaxInMs) * time.Millisecond
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	retryable := config.Retryable
	if retryable == nil {
		retryable = DefaultRetryable
	}

	name := config.Name
	if name == "" {
		name = hostOf(config.BaseUrl)
		if name == "" {
			name = "throttle"
		}
	}

	headers := make(map[string][]string)
	for key, values := range config.DefaultHeaders {
		headers[key] = append([]string{}, values...)
	}
	if _, exists := headers["User-Agent"]; !exists {
		headers["User-Agent"] = []string{DefaultUserAgent}
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 0}
	}

	return &BaseRetailer{
		httpClient:     httpClient,
		baseUrl:        config.BaseUrl,
		degree:         degree,
		numRetries:     numRetries,
		delayMin:       delayMin,
		delayMax:       delayMax,
		timeout:        timeout,
		defaultHeaders: headers,
		retryable:      retryable,
		session:        config.Session,
		name:           name,
		Logger:         config.Logger,
		queue:          list.New(),
		work:           make(chan struct{}, 1),
		done:           make(chan struct{}),
		heartbeat:      NewProgressTicker(HeartbeatInterval),
	}
}

// DefaultRetryable returns true for retryable status codes.
func DefaultRetryable(response *RetailClientResponse, error error) bool {
	if error != nil || response == nil {
		return true
	}
	return response.StatusCode == 429 || response.StatusCode >= 500
}

// Execute enqueues requests and waits for all results.
func (client *BaseRetailer) Execute(requestContext context.Context, requests []RetailClientRequest) ([]RetailClientResult, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	jobs := make([]*Job, len(requests))
	for index, request := range requests {
		jobContext, cancel := context.WithCancel(requestContext)
		jobs[index] = &Job{
			Request:      request,
			token:        jobContext,
			cancel:       cancel,
			completion:   make(chan RetailClientResult, 1),
			RequestLabel: label(request),
		}
	}

	go func() {
		<-requestContext.Done()
		for _, jobItem := range jobs {
			if !jobItem.isSettled() {
				jobItem.cancel()
				if jobItem.markSettled() {
					select {
					case jobItem.completion <- RetailClientResult{
						Request: jobItem.Request,
						Error:   requestContext.Err(),
					}:
					default:
					}
				}
			}
		}
	}()

	client.enqueue(jobs)

	results := make([]RetailClientResult, len(jobs))
	allFailed := true
	for index, jobItem := range jobs {
		results[index] = <-jobItem.completion
		if results[index].Error == nil {
			allFailed = false
		}
	}

	if allFailed && len(results) > 0 {
		return results, results[0].Error
	}

	return results, nil
}

func (client *BaseRetailer) enqueue(jobs []*Job) {
	queued := 0

	select {
	case <-client.done:
		for _, jobItem := range jobs {
			if !jobItem.isSettled() {
				if jobItem.markSettled() {
					select {
					case jobItem.completion <- RetailClientResult{
						Request: jobItem.Request,
						Error:   errors.New("throttled client shut down before the request ran"),
					}:
					default:
					}
				}
			}
		}
		return
	default:
	}

	client.mutex.Lock()
	for _, jobItem := range jobs {
		if jobItem.isSettled() {
			continue
		}
		client.queue.PushBack(jobItem)
		queued++
	}
	client.mutex.Unlock()

	if queued == 0 {
		return
	}

	client.ensurePump()
	select {
	case client.work <- struct{}{}:
	default:
	}
}

func (client *BaseRetailer) ensurePump() {
	client.pumpOnce.Do(func() {
		go client.pumpLoop()
	})
}

func (client *BaseRetailer) pumpLoop() {
	for {
		func() {
			defer func() {
				if recovery := recover(); recovery != nil {
					if client.Logger != nil {
						client.Logger.Errorf("throttle %s: pump recovered from panic: %v", client.name, recovery)
					}
				}
			}()
			client.pump()
		}()

		select {
		case <-client.done:
			return
		default:
		}
	}
}

func (client *BaseRetailer) pump() {
	for {
		select {
		case <-client.work:
		case <-client.done:
			return
		}

		for {
			select {
			case <-client.done:
				return
			default:
			}

			client.pace()

			batch := client.takeBatch()
			if len(batch) == 0 {
				break
			}

			sessionHeaders := client.sessionHeaders(batch)
			if sessionHeaders == nil && client.session != nil {
				continue
			}

			client.runBatch(batch, sessionHeaders)
			client.report()

			wait := longest(client.nextDelay(), client.backoff)
			client.backoff = 0
			client.nextBatchAt = time.Now().Add(wait)
		}
	}
}

func (client *BaseRetailer) sessionHeaders(batch []*Job) map[string][]string {
	if client.session == nil {
		return nil
	}

	if primeError := client.session.Prime(); primeError != nil && client.Logger != nil {
		client.Logger.Warnf("session prime failed: %v", primeError)
	}

	headers := client.session.Headers(time.Now())
	if headers != nil {
		return headers
	}

	mintRequest := client.session.MintRequest()
	mintContext, mintCancel := context.WithTimeout(context.Background(), client.timeout)
	defer mintCancel()
	result := client.attemptWithTimeout(mintRequest, mintContext)

	if result.response != nil && succeeded(result.response, result.error) && client.session.TryAccept(*result.response) {
		return client.session.Headers(time.Now())
	}

	var mintFailure error
	if result.error != nil || result.response == nil {
		mintFailure = &RetailClientSessionError{
			Message: fmt.Sprintf("session: mint request failed: %v", result.error),
			Inner:   result.error,
		}
	} else if succeeded(result.response, nil) {
		mintFailure = &RetailClientSessionError{
			Message: "session: mint response carried no usable access_token",
		}
	} else {
		mintFailure = &RetailClientSessionError{
			Message: fmt.Sprintf("session: mint request failed with status %d", result.response.StatusCode),
		}
	}

	for _, jobItem := range batch {
		if !jobItem.isSettled() {
			if jobItem.markSettled() {
				jobItem.completion <- RetailClientResult{
					Request:  jobItem.Request,
					Error:    mintFailure,
					Attempts: int(atomic.LoadInt32(&jobItem.attempts)),
				}
			}
		}
	}

	return nil
}

func (client *BaseRetailer) pace() {
	if client.nextBatchAt.IsZero() {
		return
	}
	now := time.Now()
	if client.nextBatchAt.After(now) {
		time.Sleep(client.nextBatchAt.Sub(now))
	}
}

func (client *BaseRetailer) takeBatch() []*Job {
	batch := make([]*Job, 0, client.degree)

	client.mutex.Lock()
	for client.queue.Len() > 0 && len(batch) < client.degree {
		frontElement := client.queue.Front()
		jobItem := frontElement.Value.(*Job)
		client.queue.Remove(frontElement)
		if jobItem.isSettled() {
			continue
		}
		batch = append(batch, jobItem)
	}
	client.mutex.Unlock()

	return batch
}

func (client *BaseRetailer) runBatch(batch []*Job, sessionHeaders map[string][]string) {
	if len(batch) == 0 {
		return
	}

	outcomes := make([]attempt, len(batch))

	var waitGroup sync.WaitGroup
	for index, jobItem := range batch {
		waitGroup.Add(1)
		go func(idx int, jobItem *Job) {
			defer waitGroup.Done()
			outcomes[idx] = client.attemptWithTimeout(WithSession(jobItem.Request, sessionHeaders), jobItem.token)
		}(index, jobItem)
	}
	waitGroup.Wait()

	maxAttempts := int32(client.numRetries + 1)
	var requeue []*Job
	var backoff time.Duration

	for index, jobItem := range batch {
		response, error := outcomes[index].response, outcomes[index].error

		atomic.AddInt32(&jobItem.attempts, 1)

		if jobItem.isSettled() {
			continue
		}

		select {
		case <-jobItem.token.Done():
			if jobItem.markSettled() {
				jobItem.completion <- RetailClientResult{
					Request:  jobItem.Request,
					Error:    jobItem.token.Err(),
					Attempts: int(atomic.LoadInt32(&jobItem.attempts)),
				}
			}
			continue
		default:
		}

		if succeeded(response, error) {
			atomic.AddInt64(&client.succeeded, 1)
			if jobItem.markSettled() {
				jobItem.completion <- RetailClientResult{
					Request:  jobItem.Request,
					Response: response,
					Attempts: int(atomic.LoadInt32(&jobItem.attempts)),
				}
			}
			continue
		}

		if client.session != nil && response != nil && client.session.IsExpired(*response) && !jobItem.reauthorised {
			jobItem.reauthorised = true
			if atomic.LoadInt32(&jobItem.attempts) > 0 {
				atomic.AddInt32(&jobItem.attempts, -1)
			}
			client.session.Invalidate()
			requeue = append(requeue, jobItem)
			continue
		}

		attempts := int(atomic.LoadInt32(&jobItem.attempts))
		exhausted := int32(attempts) >= maxAttempts
		retryable := client.retryable(response, error)

		if retryable && !exhausted {
			atomic.AddInt64(&client.retries, 1)
			backoffDuration := client.backoffFor(response, attempts)
			if backoffDuration > backoff {
				backoff = backoffDuration
			}
			requeue = append(requeue, jobItem)
			continue
		}

		if retryable {
			atomic.AddInt64(&client.failed, 1)
		} else {
			atomic.AddInt64(&client.rejected, 1)
		}

		if jobItem.markSettled() {
			jobItem.completion <- RetailClientResult{
				Request:  jobItem.Request,
				Response: response,
				Attempts: attempts,
				Error:    TerminalError(jobItem, response, error, exhausted && retryable),
			}
		}
	}

	client.backoff = backoff

	if len(requeue) > 0 {
		client.mutex.Lock()
		for index := len(requeue) - 1; index >= 0; index-- {
			client.queue.PushFront(requeue[index])
		}
		client.mutex.Unlock()
		select {
		case client.work <- struct{}{}:
		default:
		}
	}
}

func (client *BaseRetailer) failRemaining(error error) {
	client.mutex.Lock()
	remaining := client.queue
	client.queue = list.New()
	client.mutex.Unlock()

	for element := remaining.Front(); element != nil; element = element.Next() {
		jobItem := element.Value.(*Job)
		if !jobItem.isSettled() {
			if jobItem.markSettled() {
				jobItem.completion <- RetailClientResult{
					Request:  jobItem.Request,
					Error:    error,
					Attempts: int(atomic.LoadInt32(&jobItem.attempts)),
				}
			}
		}
	}
}

// Dispose shuts down the client.
func (client *BaseRetailer) Dispose() {
	client.disposeOnce.Do(func() {
		close(client.done)
	})
	client.failRemaining(errors.New("throttled client shut down"))
}

// stats returns current progress counters.
func (client *BaseRetailer) stats() (succeeded, rejected, failed, retries, queued int64) {
	client.mutex.Lock()
	queued = int64(client.queue.Len())
	client.mutex.Unlock()
	return atomic.LoadInt64(&client.succeeded), atomic.LoadInt64(&client.rejected),
		atomic.LoadInt64(&client.failed), atomic.LoadInt64(&client.retries), queued
}

func (client *BaseRetailer) attemptWithTimeout(request RetailClientRequest, callerContext context.Context) attempt {
	resolvedUrl, error := client.resolveUrl(request.Path)
	if error != nil {
		return attempt{error: &RetailClientError{Message: fmt.Sprintf("build request %s: %v", label(request), error), Inner: error}}
	}

	method := request.Method
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if len(request.Body) > 0 {
		bodyReader = bytes.NewReader(request.Body)
	}

	requestContext := callerContext
	cancel := func() {}
	if callerDeadline, hasDeadline := callerContext.Deadline(); !hasDeadline || time.Until(callerDeadline) > client.timeout {
		requestContext, cancel = context.WithTimeout(callerContext, client.timeout)
	}
	defer cancel()

	httpRequest, error := http.NewRequestWithContext(requestContext, method, resolvedUrl, bodyReader)
	if error != nil {
		return attempt{error: &RetailClientError{Message: fmt.Sprintf("build request %s: %v", label(request), error), Inner: error}}
	}

	for key, values := range client.defaultHeaders {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}
	for key, values := range request.Headers {
		for _, value := range values {
			if value == "" {
				httpRequest.Header.Del(key)
			} else {
				httpRequest.Header.Set(key, value)
			}
		}
	}

	response, error := client.httpClient.Do(httpRequest)
	if error != nil {
		if requestContext.Err() == context.DeadlineExceeded && callerContext.Err() == nil {
			return attempt{error: &RetailClientTimeoutError{
				Message: fmt.Sprintf("%s: timed out after %.3fs", label(request), client.timeout.Seconds()),
				Timeout: client.timeout,
				Inner:   error,
			}}
		}
		return attempt{error: error}
	}
	defer response.Body.Close()

	body, error := io.ReadAll(io.LimitReader(response.Body, MaxBodySize))
	if error != nil {
		return attempt{error: &RetailClientError{Message: fmt.Sprintf("read body %s: %v", label(request), error), Inner: error}}
	}

	headers := make(map[string][]string, len(response.Header))
	for key, values := range response.Header {
		headers[key] = slices.Clone(values)
	}

	return attempt{response: &RetailClientResponse{
		StatusCode: response.StatusCode,
		Body:       body,
		Headers:    headers,
	}}
}

// resolveUrl resolves a path against the base URL.
func (client *BaseRetailer) resolveUrl(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	if client.baseUrl == "" {
		return "", fmt.Errorf("relative path given but BaseUrl is empty")
	}
	return strings.TrimRight(client.baseUrl, "/") + "/" + strings.TrimLeft(path, "/"), nil
}

func (client *BaseRetailer) backoffFor(response *RetailClientResponse, attempts int) time.Duration {
	if stated := RetryAfter(response); stated != nil {
		if *stated > MaxBackoff {
			return MaxBackoff
		}
		return *stated
	}

	baseline := client.delayMin
	if baseline <= 0 {
		baseline = 100 * time.Millisecond
	}
	scaled := time.Duration(float64(baseline) * math.Pow(2, math.Min(float64(attempts-1), 6)))
	if scaled > MaxBackoff {
		scaled = MaxBackoff
	}
	return time.Duration(float64(scaled) * (1 + (randomFloat() * JitterFraction)))
}

func (client *BaseRetailer) nextDelay() time.Duration {
	if client.delayMax > client.delayMin {
		return client.delayMin + time.Duration(float64(client.delayMax-client.delayMin)*randomFloat())
	}
	return client.delayMin
}

func (client *BaseRetailer) report() {
	if client.Logger == nil {
		return
	}
	if !client.heartbeat.ShouldReport() {
		return
	}

	client.mutex.Lock()
	queued := int64(client.queue.Len())
	client.mutex.Unlock()

	rate := client.heartbeat.RatePerSecond(atomic.LoadInt64(&client.succeeded))

	client.Logger.Infof("throttle %s: done %d, queued %d, retries %d, rejected %d, failed %d, %.1f/s",
		client.name, atomic.LoadInt64(&client.succeeded), queued,
		atomic.LoadInt64(&client.retries), atomic.LoadInt64(&client.rejected),
		atomic.LoadInt64(&client.failed), rate)
}

// RetryAfter parses the Retry-After header from a response.
func RetryAfter(response *RetailClientResponse) *time.Duration {
	if response == nil {
		return nil
	}
	values, exists := response.Headers["Retry-After"]
	if !exists || len(values) == 0 {
		return nil
	}
	raw := values[0]
	if raw == "" {
		return nil
	}

	if seconds, error := strconv.Atoi(raw); error == nil {
		if seconds >= 0 {
			duration := time.Duration(seconds) * time.Second
			return &duration
		}
		return nil
	}

	parsedTime, error := time.Parse(time.RFC1123, raw)
	if error == nil {
		until := time.Until(parsedTime)
		if until > 0 {
			return &until
		}
		duration := time.Duration(0)
		return &duration
	}

	return nil
}

// WithSession merges session headers into a request.
func WithSession(request RetailClientRequest, sessionHeaders map[string][]string) RetailClientRequest {
	if len(sessionHeaders) == 0 {
		return request
	}
	merged := make(map[string][]string)
	for key, values := range request.Headers {
		merged[key] = slices.Clone(values)
	}
	for key, values := range sessionHeaders {
		merged[key] = slices.Clone(values)
	}
	request.Headers = merged
	return request
}

func succeeded(response *RetailClientResponse, error error) bool {
	return error == nil && response != nil && response.StatusCode >= 200 && response.StatusCode < 300
}

// TerminalError creates the final error for a failed request.
func TerminalError(jobItem *Job, response *RetailClientResponse, error error, exhausted bool) error {
	requestLabel := jobItem.RequestLabel

	if error != nil {
		message := requestLabel + ": " + error.Error()
		if exhausted {
			message = fmt.Sprintf("%s: gave up after %d attempt(s): %s", requestLabel, atomic.LoadInt32(&jobItem.attempts), error.Error())
		}
		var timeout *RetailClientTimeoutError
		if errors.As(error, &timeout) {
			return &RetailClientTimeoutError{Message: message, Timeout: timeout.Timeout, Inner: error}
		}
		return &RetailClientError{Message: message, Inner: error}
	}

	statusCode := response.StatusCode
	if exhausted {
		return &RetailClientHTTPStatusError{
			Message:    fmt.Sprintf("%s: gave up after %d attempt(s), last status %d: unsuccessful http status", requestLabel, atomic.LoadInt32(&jobItem.attempts), statusCode),
			StatusCode: statusCode,
		}
	}
	return &RetailClientHTTPStatusError{
		Message:    fmt.Sprintf("%s: status %d: unsuccessful http status", requestLabel, statusCode),
		StatusCode: statusCode,
	}
}

func label(request RetailClientRequest) string {
	if request.Label != "" {
		return request.Label
	}
	method := request.Method
	if method == "" {
		method = "GET"
	}
	return method + " " + request.Path
}

func hostOf(baseUrl string) string {
	parsedUrl, error := url.Parse(baseUrl)
	if error != nil {
		return ""
	}
	return parsedUrl.Hostname()
}

func maxInt(valueOne, valueTwo int) int {
	if valueOne > valueTwo {
		return valueOne
	}
	return valueTwo
}

func longest(durationOne, durationTwo time.Duration) time.Duration {
	if durationOne > durationTwo {
		return durationOne
	}
	return durationTwo
}

func randomFloat() float64 {
	return rand.Float64()
}

// progressTicker paces progress reporting.
type progressTicker struct {
	interval  time.Duration
	startedAt time.Time
	nextAt    time.Time
}

// NewProgressTicker creates a new progress ticker.
func NewProgressTicker(interval time.Duration) *progressTicker {
	now := time.Now()
	return &progressTicker{
		interval:  interval,
		startedAt: now,
		nextAt:    now.Add(interval),
	}
}

// ShouldReport returns true if it's time to report progress.
func (ticker *progressTicker) ShouldReport() bool {
	now := time.Now()
	if now.Before(ticker.nextAt) {
		return false
	}
	ticker.nextAt = now.Add(ticker.interval)
	return true
}

// RatePerSecond returns the current rate of completed requests per second.
func (ticker *progressTicker) RatePerSecond(done int64) float64 {
	elapsed := time.Since(ticker.startedAt).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return math.Round(float64(done)/elapsed*10) / 10
}
