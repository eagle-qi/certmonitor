package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	instance *zap.Logger
	once     sync.Once
	sugar    *zap.SugaredLogger
)

type Config struct {
	Level     string `yaml:"level"`     // debug / info / warn / error
	Format    string `yaml:"format"`    // console / json
	Path      string `yaml:"path"`      // 日志目录路径
	MaxSize   int    `yaml:"max_size"`  // MB
	MaxBackups int   `yaml:"max_backups"`
	MaxAge    int    `yaml:"max_age"`   // days
	Compress  bool   `yaml:"compress"`
}

// Init 初始化全局日志实例
func Init(cfg Config) {
	once.Do(func() {
		var zapLevel zapcore.Level
		switch cfg.Level {
		case "debug":
			zapLevel = zapcore.DebugLevel
		case "info":
			zapLevel = zapcore.InfoLevel
		case "warn":
			zapLevel = zapcore.WarnLevel
		case "error":
			zapLevel = zapcore.ErrorLevel
		default:
			zapLevel = zapcore.InfoLevel
		}

		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		if cfg.Format == "json" {
			encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		}

		var encoder zapcore.Encoder
		if cfg.Format == "json" {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		}

		// 多输出：同时输出到控制台和文件
		var cores []zapcore.Core

		consoleCore := zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			zapLevel,
		)
		cores = append(cores, consoleCore)

		// 如果配置了日志路径，则同时写入文件
		if cfg.Path != "" {
			os.MkdirAll(cfg.Path, 0755)
			fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

			fileWriter, err := os.OpenFile(cfg.Path+"/app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				fileCore := zapcore.NewCore(
					fileEncoder,
					zapcore.AddSync(fileWriter),
					zapLevel,
				)
				cores = append(cores, fileCore)
			}
		}

		core := zapcore.NewTee(cores...)
		instance = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		sugar = instance.Sugar()
	})
}

// GetLogger 获取原生 Logger
func GetLogger() *zap.Logger {
	return instance
}

// GetSugar 获取 SugaredLogger (更方便使用)
func GetSugar() *zap.SugaredLogger {
	return sugar
}

// =========================================== 便捷方法（全局调用）===========================================

func Debug(args ...interface{}) { sugar.Debugw(fmt.Sprint(args...)) }
func Info(args ...interface{})  { sugar.Infow(fmt.Sprint(args...)) }
func Warn(args ...interface{})  { sugar.Warnw(fmt.Sprint(args...)) }
func Error(args ...interface{}) { sugar.Errorw(fmt.Sprint(args...)) }
func Fatal(args ...interface{}) { sugar.Fatalw(fmt.Sprint(args...)) }

func Debugf(format string, args ...interface{}) { sugar.Debugf(format, args...) }
func Infof(format string, args ...interface{})  { sugar.Infof(format, args...) }
func Warnf(format string, args ...interface{})  { sugar.Warnf(format, args...) }
func Errorf(format string, args ...interface{}) { sugar.Errorf(format, args...) }
func Fatalf(format string, args ...interface{}) { sugar.Fatalf(format, args...) }

func Debugw(msg string, keysAndValues ...interface{}) { sugar.Debugw(msg, keysAndValues...) }
func Infow(msg string, keysAndValues ...interface{})  { sugar.Infow(msg, keysAndValues...) }
func Warnw(msg string, keysAndValues ...interface{})  { sugar.Warnw(msg, keysAndValues...) }
func Errorw(msg string, keysAndValues ...interface{}) { sugar.Errorw(msg, keysAndValues...) }
func Fatalw(msg string, keysAndValues ...interface{}) { sugar.Fatalw(msg, keysAndValues...) }

// WithContext 返回带上下文的 logger
func WithContext(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		return sugar
	}
	requestID, _ := ctx.Value("request_id").(string)
	return sugar.With("request_id", requestID)
}
