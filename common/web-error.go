package common

// Web Error
type WebError struct {
	Code        string
	Msg         string
	InternalMsg string
	hasCode     bool
}

func (e *WebError) Error() string {
	return e.Msg
}

// Create new WebError
func NewWebErr(msg string, internalMsg ...string) *WebError {
	var im string
	if len(internalMsg) > 0 {
		im = internalMsg[0]
	}
	return &WebError{Msg: msg, hasCode: false, InternalMsg: im}
}

// Create new WebError with code
func NewWebErrCode(code string, msg string, internalMsg ...string) *WebError {
	var im string
	if len(internalMsg) > 0 {
		im = internalMsg[0]
	}
	return &WebError{Msg: msg, Code: code, hasCode: true, InternalMsg: im}
}

// Check if WebError has a specified code
func HasCode(e *WebError) bool {
	return e.hasCode
}
