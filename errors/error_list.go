package errors

import "net/http"

var (
	SuccessResponse = CommonResponse{StatusCode: http.StatusOK, Code: "0000", Message: "ok"}

	ErrorBadRequest   = CommonResponse{StatusCode: http.StatusBadRequest, Code: "E100", Message: "bad request"}
	ErrorNotFound     = CommonResponse{StatusCode: http.StatusNotFound, Code: "E101", Message: "resource not found"}
	ErrorInternal     = CommonResponse{StatusCode: http.StatusInternalServerError, Code: "E102", Message: "internal server error"}
	ErrorUnauthorized = CommonResponse{StatusCode: http.StatusUnauthorized, Code: "E103", Message: "unauthorized"}
	ErrorForbidden    = CommonResponse{StatusCode: http.StatusForbidden, Code: "E104", Message: "forbidden"}
)
