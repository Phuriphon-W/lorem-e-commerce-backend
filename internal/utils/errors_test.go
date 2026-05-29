package utils

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateErrorResponse(t *testing.T) {
	tests := []struct {
		status        int
		message       string
		expectedTitle string
	}{
		{http.StatusBadRequest, "bad request message", "Bad Request"},
		{http.StatusUnauthorized, "unauthorized message", "Unauthorized"},
		{http.StatusForbidden, "forbidden message", "Forbidden"},
		{http.StatusNotFound, "not found message", "Not Found"},
		{http.StatusMethodNotAllowed, "method not allowed message", "Method Not Allowed"},
		{http.StatusConflict, "conflict message", "Conflict"},
		{http.StatusInternalServerError, "internal error message", "Internal Server Error"},
		{http.StatusBadGateway, "bad gateway message", "Bad Gateway"},
		{http.StatusServiceUnavailable, "service unavailable message", "Service Unavailable"},
		{http.StatusGatewayTimeout, "gateway timeout message", "Gateway Timeout"},
		{http.StatusTeapot, "teapot message", "An Unexpected Error Occurred"}, // default case
	}

	for _, tt := range tests {
		t.Run(tt.expectedTitle, func(t *testing.T) {
			resp := CreateErrorResponse(tt.status, tt.message)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.status, resp.Status)
			assert.Equal(t, tt.message, resp.Detail)
			assert.Equal(t, tt.expectedTitle, resp.Title)
		})
	}
}
