package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusHandler(t *testing.T) {
	server := NewCategoryServer(&StubCategoryStore{})
	req := NewGetRequest(t, "/status")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)
	result := res.Result()

	assertStatusCode(t, result.StatusCode, http.StatusOK)

	body := readBodyBytes(t, result.Body)
	assertBodyString(t, string(body), statusBodyString)

}
