package utils

import "net/http"

type ErrorResponse struct {
	Title  string
	Status int
	Detail string
}

func CreateErrorResponse(status int, message string) *ErrorResponse {
	var response ErrorResponse

	response.Status = status
	response.Detail = message

	// Map status codes to human-readable titles
	switch status {
	case http.StatusBadRequest:
		response.Title = "Bad Request"
	case http.StatusUnauthorized:
		response.Title = "Unauthorized"
	case http.StatusForbidden:
		response.Title = "Forbidden"
	case http.StatusNotFound:
		response.Title = "Not Found"
	case http.StatusMethodNotAllowed:
		response.Title = "Method Not Allowed"
	case http.StatusConflict:
		response.Title = "Conflict"
	case http.StatusInternalServerError:
		response.Title = "Internal Server Error"
	case http.StatusBadGateway:
		response.Title = "Bad Gateway"
	case http.StatusServiceUnavailable:
		response.Title = "Service Unavailable"
	case http.StatusGatewayTimeout:
		response.Title = "Gateway Timeout"
	default:
		response.Title = "An Unexpected Error Occurred"
	}

	return &response
}
