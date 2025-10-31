package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthEndpoints(t *testing.T) {
    h := NewHealthHandler()
    req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
    rr := httptest.NewRecorder()
    h.Liveness(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }

    req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
    rr = httptest.NewRecorder()
    h.Readiness(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
}


