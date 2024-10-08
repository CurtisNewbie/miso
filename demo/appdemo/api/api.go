package api

type ApiReq struct {
	Name   string
	Extras []ApiReqExtra
}
type ApiReqExtra struct {
	Special bool
}
type ApiRes struct{}
