package weberr

// Web Error
type WebError struct {
	Code    string
	Msg     string
	hasCode bool
}

func (e *WebError) Error() string {
	return e.Msg
}

// Create new WebError
func NewWebErr(msg string) *WebError {
	return &WebError{Msg: msg, hasCode: false}
}

// Create new WebError with code
func NewWebErrCode(code string, msg string) *WebError {
	return &WebError{Msg: msg, Code: code, hasCode: true}
}

// Check if WebError has a specified code
func HasCode(e *WebError) bool {
	return e.hasCode
}
