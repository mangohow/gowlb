package llog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"time"
)

type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

var (
	log      *zap.SugaredLogger
	zapCore  zapcore.Core
	logLevel = zap.NewAtomicLevel()
)

const (
	logTimeFormat = "2006-01-02 15:04:05.000"
)

// config 日志配置
type config struct {
	// 日志级别 (debug, info, warn, error, dpanic, panic, fatal)
	level string
	// 日志输出类型 (console, json)
	encoding string
	// 文件输出路径（为空则不写文件）
	filename string
	// 是否启用 caller（记录调用位置）
	enableCaller bool

	serviceName string

	// 日期格式化器
	timeEncoder zapcore.TimeEncoder
}

func (c *config) init() {
	if c.level == "" {
		c.level = "info"
	}

	if c.encoding == "" {
		c.encoding = "json"
	}

	if c.timeEncoder == nil {
		c.timeEncoder = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			type appendTimeEncoder interface {
				AppendTimeLayout(time.Time, string)
			}

			if enc, ok := enc.(appendTimeEncoder); ok {
				enc.AppendTimeLayout(t, logTimeFormat)
				return
			}

			enc.AppendString(t.Format(logTimeFormat))
		}
	}
}

func SetLevel(level string) error {
	return logLevel.UnmarshalText([]byte(level))
}

type LoggerOption func(cfg *config)

func WithLevel(level string) LoggerOption {
	return func(cfg *config) {
		cfg.level = level
	}
}

func WithEncoding(encoding string) LoggerOption {
	return func(cfg *config) {
		cfg.encoding = encoding
	}
}

func WithFilename(filename string) LoggerOption {
	return func(cfg *config) {
		cfg.filename = filename
	}
}

func WithEnableCaller(enableCaller bool) LoggerOption {
	return func(cfg *config) {
		cfg.enableCaller = enableCaller
	}
}

func WithServiceName(serviceName string) LoggerOption {
	return func(cfg *config) {
		cfg.serviceName = serviceName
	}
}

func WithTimeEncoder(enc zapcore.TimeEncoder) LoggerOption {
	return func(cfg *config) {
		cfg.timeEncoder = enc
	}
}

// InitLogger 初始化日志实例
func InitLogger(opts ...LoggerOption) (*zap.SugaredLogger, func()) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	// 1. 解析日志级别
	if err := logLevel.UnmarshalText([]byte(cfg.level)); err != nil {
		panic("invalid log level: " + cfg.level)
	}

	// 2. 配置编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = cfg.timeEncoder
	encoderConfig.StacktraceKey = "" // 禁用 zap 自身栈跟踪（由中间件处理）

	// 3. 构建写入器
	var cores []zapcore.Core
	if cfg.filename != "" {
		// 文件输出（带轮转）
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(&lumberjack.Logger{
				Filename:   cfg.filename,
				MaxSize:    10, // MB
				MaxBackups: 7,
				MaxAge:     30, // days
				Compress:   true,
			}),
			logLevel,
		)
		cores = append(cores, fileCore)
	}

	// 4. 始终输出到控制台（开发环境友好）
	if cfg.encoding == "console" {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			logLevel,
		)
		cores = append(cores, consoleCore)
	} else {
		// 生产环境 JSON 格式
		consoleCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			logLevel,
		)
		cores = append(cores, consoleCore)
	}

	// 5. 多输出合并
	zapCore = zapcore.NewTee(cores...)

	// 6. 构建 logger
	zapLogger := zap.New(zapCore, zap.AddCallerSkip(1))
	if cfg.enableCaller {
		zapLogger = zapLogger.WithOptions(zap.AddCaller())
	}
	if cfg.serviceName != "" {
		zapLogger = zapLogger.With(zap.String("service", cfg.serviceName))
	}

	// 7. 设置全局 SugaredLogger（推荐使用）
	log = zapLogger.Sugar()

	return log, func() {
		_ = log.Sync()
	}
}

func GetLogger() *zap.SugaredLogger {
	return log
}
