package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCreateAndRedeemHandler creates a RedeemHandler with a non-nil (but minimal)
// RedeemService so that CreateAndRedeem's nil guard passes and we can test the
// parameter-validation layer that runs before any service call.
func newCreateAndRedeemHandler() *RedeemHandler {
	return &RedeemHandler{
		adminService:  newStubAdminService(),
		redeemService: &service.RedeemService{}, // non-nil to pass nil guard
	}
}

// postCreateAndRedeemValidation calls CreateAndRedeem and returns the response
// status code. For cases that pass validation and proceed into the service layer,
// a panic may occur (because RedeemService internals are nil); this is expected
// and treated as "validation passed" (returns 0 to indicate panic).
func postCreateAndRedeemValidation(t *testing.T, handler *RedeemHandler, body any) (code int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBytes, err := json.Marshal(body)
	require.NoError(t, err)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes/create-and-redeem", bytes.NewReader(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	defer func() {
		if r := recover(); r != nil {
			// Panic means we passed validation and entered service layer (expected for minimal stub).
			code = 0
		}
	}()
	handler.CreateAndRedeem(c)
	return w.Code
}

func postGenerateValidation(t *testing.T, handler *RedeemHandler, body any) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBytes, err := json.Marshal(body)
	require.NoError(t, err)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes/generate", bytes.NewReader(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Generate(c)
	return w.Code
}

func TestCreateAndRedeem_TypeDefaultsToBalance(t *testing.T) {
	// 不传 type 字段时应默认 balance，不触发 subscription 校验。
	// 验证通过后进入 service 层会 panic（返回 0），说明默认值生效。
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-balance-default",
		"value":   10.0,
		"user_id": 1,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"omitting type should default to balance and pass validation")
}

func TestCreateAndRedeem_SubscriptionRequiresGroupID(t *testing.T) {
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-no-group",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"validity_days": 30,
		// group_id 缺失
	})

	assert.Equal(t, http.StatusBadRequest, code)
}

func TestCreateAndRedeem_SubscriptionRequiresPositiveValidityDays(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()

	cases := []struct {
		name         string
		validityDays int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := postCreateAndRedeemValidation(t, h, map[string]any{
				"code":          "test-sub-bad-days-" + tc.name,
				"type":          "subscription",
				"value":         29.9,
				"user_id":       1,
				"group_id":      groupID,
				"validity_days": tc.validityDays,
			})

			assert.Equal(t, http.StatusBadRequest, code)
		})
	}
}

func TestCreateAndRedeem_SubscriptionValidParamsPassValidation(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-valid",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"group_id":      groupID,
		"validity_days": 31,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"valid subscription params should pass validation")
}

func TestCreateAndRedeem_BalanceIgnoresSubscriptionFields(t *testing.T) {
	h := newCreateAndRedeemHandler()
	// balance 类型不传 group_id 和 validity_days，不应报 400
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-balance-no-extras",
		"type":    "balance",
		"value":   50.0,
		"user_id": 1,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"balance type should not require group_id or validity_days")
}

func TestGenerate_GroupRequestQuotaRequiresGroupID(t *testing.T) {
	h := &RedeemHandler{
		adminService: newStubAdminService(),
	}
	code := postGenerateValidation(t, h, map[string]any{
		"count": 1,
		"type":  "group_request_quota",
		"value": 5,
	})

	assert.Equal(t, http.StatusBadRequest, code)
}

func TestGenerate_GroupRequestQuotaRequiresWholePositiveValue(t *testing.T) {
	h := &RedeemHandler{
		adminService: newStubAdminService(),
	}

	tests := []struct {
		name  string
		value any
	}{
		{name: "zero", value: 0},
		{name: "fraction", value: 1.5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code := postGenerateValidation(t, h, map[string]any{
				"count":    1,
				"type":     "group_request_quota",
				"group_id": 5,
				"value":    tc.value,
			})
			assert.Equal(t, http.StatusBadRequest, code)
		})
	}
}

func TestGenerate_GroupRequestQuotaValidParamsPassValidation(t *testing.T) {
	h := &RedeemHandler{
		adminService: newStubAdminService(),
	}
	code := postGenerateValidation(t, h, map[string]any{
		"count":    2,
		"type":     "group_request_quota",
		"group_id": 5,
		"value":    8,
	})

	assert.Equal(t, http.StatusOK, code)
}

func TestCreateAndRedeem_GroupRequestQuotaRequiresGroupID(t *testing.T) {
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-group-rq-no-group",
		"type":    "group_request_quota",
		"value":   5,
		"user_id": 1,
	})

	assert.Equal(t, http.StatusBadRequest, code)
}

func TestCreateAndRedeem_GroupRequestQuotaRequiresWholePositiveValue(t *testing.T) {
	h := newCreateAndRedeemHandler()

	tests := []struct {
		name  string
		value any
	}{
		{name: "zero", value: 0},
		{name: "fraction", value: 2.5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code := postCreateAndRedeemValidation(t, h, map[string]any{
				"code":     "test-group-rq-bad-" + tc.name,
				"type":     "group_request_quota",
				"value":    tc.value,
				"user_id":  1,
				"group_id": 5,
			})
			assert.Equal(t, http.StatusBadRequest, code)
		})
	}
}

func TestCreateAndRedeem_GroupRequestQuotaValidParamsPassValidation(t *testing.T) {
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":     "test-group-rq-valid",
		"type":     "group_request_quota",
		"value":    12,
		"user_id":  1,
		"group_id": 5,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"valid group_request_quota params should pass validation")
}
