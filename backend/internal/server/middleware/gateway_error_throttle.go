package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// GatewayErrorThrottleConfig 网关错误惩罚限流配置
type GatewayErrorThrottleConfig struct {
	Enabled      bool          // 是否启用
	ErrorLimit   int           // 窗口内最大错误数（超过后限流）
	Window       time.Duration // 时间窗口
	CooldownSecs int           // 触发后冷却时间（秒），0 则等于 Window
}

// errorThrottleScript:
//
//	KEYS[1] = 计数器 key
//	ARGV[1] = 窗口毫秒数
//	返回当前计数（不增加）
var errorThrottleCheckScript = redis.NewScript(`
local cnt = redis.call('GET', KEYS[1])
if cnt == false then return 0 end
return tonumber(cnt)
`)

// errorThrottleIncrScript:
//
//	KEYS[1] = 计数器 key
//	ARGV[1] = 窗口毫秒数
//	增加计数并设置/保持 TTL
var errorThrottleIncrScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
else
  local ttl = redis.call('PTTL', KEYS[1])
  if ttl == -1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
  end
end
return current
`)

// GatewayErrorThrottle 基于 API Key 的错误惩罚限流中间件。
//
// 工作原理：
//   - 请求进入时，检查该 API Key 的最近错误计数。
//     如果超过阈值，直接返回 429，不走认证和路由流程。
//   - 请求完成后，如果响应是 4xx（401/403/429），对该 Key 的错误计数 +1。
//   - 成功请求不增加计数；窗口到期后计数自动清零。
func GatewayErrorThrottle(redisClient *redis.Client, cfg GatewayErrorThrottleConfig) gin.HandlerFunc {
	if !cfg.Enabled || redisClient == nil || cfg.ErrorLimit <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	cooldownMs := cfg.Window.Milliseconds()
	if cfg.CooldownSecs > 0 {
		cooldownMs = int64(cfg.CooldownSecs) * 1000
	}
	if cooldownMs < 1000 {
		cooldownMs = 60000
	}

	return func(c *gin.Context) {
		key := throttleKey(c)
		redisKey := "gw_err_throttle:" + key

		ctx := c.Request.Context()

		// 检查当前错误计数（只读，不增加）
		count := checkErrorCount(ctx, redisClient, redisKey)
		if count > int64(cfg.ErrorLimit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "rate_limit_error",
					"message": "Too many failed requests, please slow down and retry later",
				},
			})
			return
		}

		c.Next()

		// 请求完成后，根据响应状态码决定是否计入错误
		status := c.Writer.Status()
		if status == 401 || status == 403 || status == 429 {
			incrErrorCount(ctx, redisClient, redisKey, cooldownMs)
		}
	}
}

func checkErrorCount(ctx context.Context, rdb *redis.Client, key string) int64 {
	result, err := errorThrottleCheckScript.Run(ctx, rdb, []string{key}).Int64()
	if err != nil {
		return 0 // Redis 故障放行
	}
	return result
}

func incrErrorCount(ctx context.Context, rdb *redis.Client, key string, windowMs int64) {
	_ = errorThrottleIncrScript.Run(ctx, rdb, []string{key}, windowMs).Err()
}

// throttleKey 从请求中提取限流维度 key。
// 优先使用 API Key 后缀作为标识，无 Key 时回退到 IP。
func throttleKey(c *gin.Context) string {
	if token := extractBearerSuffix(c.GetHeader("Authorization")); token != "" {
		return "key:" + token
	}
	if xKey := c.GetHeader("x-api-key"); xKey != "" {
		return "key:" + keySuffix(xKey)
	}
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

func extractBearerSuffix(header string) string {
	if len(header) <= 7 {
		return ""
	}
	token := header[7:] // strip "Bearer "
	return keySuffix(token)
}

func keySuffix(key string) string {
	if len(key) > 16 {
		return key[len(key)-16:]
	}
	return key
}
