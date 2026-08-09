package logging

const (
	HeaderRequestID = "X-Request-ID"

	FieldDurationMS  = "duration_ms"
	FieldEnvironment = "environment"
	FieldError       = "error"
	FieldErrorCode   = "error_code"
	FieldEvent       = "event"
	FieldHost        = "host"
	FieldHTTPMethod  = "http_method"
	FieldHTTPPath    = "http_path"
	FieldHTTPStatus  = "http_status"
	FieldPort        = "port"
	FieldRequestID   = "request_id"

	EventHTTPRequestCompleted = "http.request.completed"
	EventServiceStarting      = "service.starting"
)
