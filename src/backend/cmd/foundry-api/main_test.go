package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerTokenMiddleware(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := bearerTokenMiddleware("a-test-token-with-more-than-32-characters", next)

	for _, test := range []struct {
		name   string
		header string
		status int
	}{
		{name: "missing", status: http.StatusUnauthorized},
		{name: "wrong", header: "Bearer wrong-token", status: http.StatusUnauthorized},
		{name: "valid", header: "Bearer a-test-token-with-more-than-32-characters", status: http.StatusNoContent},
	} {
		called = false
		request := httptest.NewRequest(http.MethodPost, "/api/integrations/foundry/sync", nil)
		if test.header != "" {
			request.Header.Set("Authorization", test.header)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Errorf("%s: status = %d, want %d", test.name, response.Code, test.status)
		}
		if called != (test.status == http.StatusNoContent) {
			t.Errorf("%s: downstream handler called = %v", test.name, called)
		}
	}
}
