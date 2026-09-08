package responses

type ScrapingExceptionResponse struct {
	Message string
	Inner   error
}

func (exception *ScrapingExceptionResponse) Error() string {
	if exception.Inner != nil {
		return exception.Message + ": " + exception.Inner.Error()
	}
	return exception.Message
}

func (exception *ScrapingExceptionResponse) Unwrap() error {
	return exception.Inner
}

func NewScrapingExceptionResponse(message string, inner error) *ScrapingExceptionResponse {
	return &ScrapingExceptionResponse{Message: message, Inner: inner}
}
