package dto

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Code    string      `json:"code,omitempty"`
	Field   *FieldError `json:"field,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}
