package llog

import (
	"context"
	"github.com/google/uuid"
	"github.com/mangohow/gowlb/errors"
	"github.com/mangohow/gowlb/transport/http"
	"go.uber.org/zap"
	"time"
)

type loggerKey struct{}

const (
	requestIdKeyName = "X-Request-ID"
)

// WithLogger 将 logger 注入 context
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext 从 context 获取 logger（不存在则返回默认 logger）
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerKey{}).(*zap.SugaredLogger); ok {
		return logger
	}
	return log
}

// LoggerInjectMiddleware 中间件：注入带 RequestID 的 logger
func LoggerInjectMiddleware(requestIdKey string) http.Middleware {
	return func(ctx context.Context, req interface{}, handler http.Handler) (interface{}, error) {
		c := http.FromContext(ctx)
		// 1. 获取/生成 RequestID
		if requestIdKey == "" {
			requestIdKey = requestIdKeyName
		}
		rid := c.Request().Header.Get(requestIdKey)
		if rid == "" {
			rid = uuid.New().String()
		}
		c.ResponseWriter().Header().Set("X-Request-ID", rid) // 响应头返回

		// 2. 创建带 RequestID 的 logger
		requestLogger := log.With("requestId", rid)

		// 3. 注入 logger 到 context
		cc := WithLogger(ctx, requestLogger)

		// 4. 调用下一个 handler
		return handler(cc, req)
	}
}

func RequestLoggingMiddleware() http.Middleware {
	return func(ctx context.Context, req interface{}, handler http.Handler) (interface{}, error) {
		var (
			logger  = FromContext(ctx)
			c       = http.FromContext(ctx)
			request = c.Request()
		)

		// 1. 开始计时
		start := time.Now()
		path := request.URL.Path
		raw := request.URL.RawQuery

		// 2. 处理请求
		resp, err := handler(ctx, req)

		// 3. 计算延迟
		latency := time.Since(start)
		clientIP := request.Header.Get("X-Real-IP")
		if clientIP == "" {
			clientIP = request.Header.Get("X-Forwarded-For")
		}
		if clientIP == "" {
			clientIP = request.RemoteAddr
		}
		method := request.Method

		// 4. 构建日志字段
		fields := []interface{}{
			"method", method,
			"path", path,
			"query", raw,
			"ip", clientIP,
			"latency", latency,
		}

		if err != nil {
			if e, ok := err.(errors.Error); ok {
				fields = append(fields, "status", e.HttpStatus())
				fields = append(fields, "errCode", e.Code())
				fields = append(fields, "errMsg", e.Message())
			} else {
				fields = append(fields, "error", err.Error())
			}
		}

		// 5. 按状态码决定日志级别
		if err != nil {
			logger.Errorw("Server error", fields...)
		} else {
			logger.Infow("Request", fields...)
		}

		return resp, err
	}
}
