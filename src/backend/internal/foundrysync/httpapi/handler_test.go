package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync"
)

type fakeSyncRunner struct {
	report foundrysync.SyncReport
	err    error
}

func (f fakeSyncRunner) SyncAll(context.Context) (foundrysync.SyncReport, error) {
	return f.report, f.err
}

func TestHandlerRunsWorldSyncAndReturnsReport(t *testing.T) {
	handler, err := NewHandler(fakeSyncRunner{report: foundrysync.SyncReport{WorldID: "world", Discovered: 4, CharactersSynced: 2, SnapshotsOnly: 2}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, SyncPath, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := response.Body.String(); got == "" || !containsAll(got, `"discovered":4`, `"charactersSynced":2`, `"snapshotsOnly":2`) {
		t.Fatalf("unexpected report body: %s", got)
	}
}

func TestHandlerRejectsWrongMethodAndPath(t *testing.T) {
	handler, _ := NewHandler(fakeSyncRunner{})
	for _, test := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, SyncPath, http.StatusMethodNotAllowed},
		{http.MethodPost, "/wrong", http.StatusNotFound},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Errorf("%s %s: status = %d, want %d", test.method, test.path, response.Code, test.want)
		}
	}
}

func TestHandlerReturnsConflictWhenSyncIsAlreadyRunning(t *testing.T) {
	handler, _ := NewHandler(fakeSyncRunner{err: foundrysync.ErrSyncInProgress})
	request := httptest.NewRequest(http.MethodPost, SyncPath, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestNewHandlerRequiresSyncRunner(t *testing.T) {
	if _, err := NewHandler(nil); err == nil {
		t.Fatal("expected nil runner to be rejected")
	}
}

func containsAll(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(value, fragment) {
			return false
		}
	}
	return true
}
