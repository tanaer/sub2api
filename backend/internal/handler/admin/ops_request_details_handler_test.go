package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type captureRequestDetailsRepo struct {
	service.OpsRepository
	filter *service.OpsRequestDetailFilter
}

func (r *captureRequestDetailsRepo) ListRequestDetails(ctx context.Context, filter *service.OpsRequestDetailFilter) ([]*service.OpsRequestDetail, int64, error) {
	r.filter = filter
	return []*service.OpsRequestDetail{}, 0, nil
}

func TestOpsHandler_ListRequestDetails_ParsesExcludePhases(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	repo := &captureRequestDetailsRepo{}
	opsService := service.NewOpsService(
		repo,
		newTestSettingRepo(),
		&config.Config{Ops: config.OpsConfig{Enabled: true}},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	handler := NewOpsHandler(opsService)

	router := gin.New()
	router.GET("/requests", handler.ListRequestDetails)

	req := httptest.NewRequest(http.MethodGet, "/requests?exclude_phases=auth,%20internal", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.filter == nil {
		t.Fatal("expected request details filter to be passed to repository")
	}

	filterValue := reflect.ValueOf(repo.filter).Elem()
	excludePhasesField := filterValue.FieldByName("ExcludePhases")
	if !excludePhasesField.IsValid() {
		t.Fatal("expected OpsRequestDetailFilter to expose ExcludePhases")
	}

	phases, ok := excludePhasesField.Interface().([]string)
	if !ok {
		t.Fatalf("expected ExcludePhases to be []string, got %T", excludePhasesField.Interface())
	}

	expected := []string{"auth", "internal"}
	if !reflect.DeepEqual(phases, expected) {
		t.Fatalf("exclude phases mismatch: got %v want %v", phases, expected)
	}
}
