package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newGatewayRoutesTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
			SoraGateway:   &handler.SoraGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
		nil,
	)

	return router
}

func TestGatewayRoutesOpenAIResponsesCompactPathIsRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()

	for _, path := range []string{"/v1/responses/compact", "/responses/compact"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-5"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI responses handler", path)
	}
}

func TestDispatchOpenAICompatibleByGroupPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newAPIKey := func(platform string) *service.APIKey {
		groupID := int64(7)
		return &service.APIKey{
			GroupID: &groupID,
			Group: &service.Group{
				ID:       groupID,
				Platform: platform,
				Status:   service.StatusActive,
				Hydrated: true,
			},
		}
	}

	tests := []struct {
		name       string
		apiKey     *service.APIKey
		wantStatus int
	}{
		{name: "anthropic group uses gateway handler", apiKey: newAPIKey(service.PlatformAnthropic), wantStatus: http.StatusAccepted},
		{name: "openai group uses openai handler", apiKey: newAPIKey(service.PlatformOpenAI), wantStatus: http.StatusNoContent},
		{name: "missing group falls back to gateway handler", apiKey: nil, wantStatus: http.StatusAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			if tt.apiKey != nil {
				c.Set(string(servermiddleware.ContextKeyAPIKey), tt.apiKey)
			}

			handler := dispatchOpenAICompatibleByGroupPlatform(
				func(c *gin.Context) { c.Status(http.StatusNoContent) },
				func(c *gin.Context) { c.Status(http.StatusAccepted) },
			)

			handler(c)
			require.Equal(t, tt.wantStatus, c.Writer.Status())
		})
	}
}
