package zapvml

import (
	"os"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc/codes"
)

var (
	// Log is global logger
	Log   *zap.Logger
	Level zap.AtomicLevel
)

type Config struct {
	Level zapcore.Level `required:"true" default:"warn"`
	Debug bool          `required:"true" default:"false"`
}

func Init(globalLevel zapcore.Level) {
	Level.SetLevel(globalLevel)
}

// Use package init to avoid race conditions for GRPC options
// sync.Once still suffers from races, init functions are less complex than sync.once + waitgroup
func init() {
	var cfg Config
	if err := envconfig.Process("log", &cfg); err != nil {
		panic(err)
	}

	Level = zap.NewAtomicLevelAt(cfg.Level)

	// High-priority output should also go to standard error, and low-priority
	// output should also go to standard out.
	// It is useful for Kubernetes deployment.
	// Kubernetes interprets os.Stdout log items as INFO and os.Stderr log items
	// as ERROR by default.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= Level.Level() && lvl < zapcore.ErrorLevel
	})

	// Output channels
	consoleInfos := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	// Setup Config and Encoder
	var ecfg zapcore.EncoderConfig
	var enc zapcore.Encoder
	if cfg.Debug {
		ecfg = zapdriver.NewDevelopmentEncoderConfig()
		enc = zapcore.NewConsoleEncoder(ecfg)
	} else {
		ecfg = zapdriver.NewProductionEncoderConfig()
		enc = zapcore.NewJSONEncoder(ecfg)
	}

	// Join the outputs, encoders, and level-handling functions into
	// zapcore.
	core := zapcore.NewTee(
		zapcore.NewCore(enc, consoleErrors, highPriority),
		zapcore.NewCore(enc, consoleInfos, lowPriority),
	)
	// From a zapcore.Core, it's easy to construct a Logger.
	Log = zap.New(core)
	zap.RedirectStdLog(Log)
}

func CodeToLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.OK:
		return zap.InfoLevel
	case codes.Canceled:
		return zap.WarnLevel
	case codes.Unknown:
		return zap.ErrorLevel
	case codes.InvalidArgument:
		return zap.WarnLevel
	case codes.DeadlineExceeded:
		return zap.WarnLevel
	case codes.NotFound:
		return zap.WarnLevel
	case codes.AlreadyExists:
		return zap.WarnLevel
	case codes.PermissionDenied:
		return zap.WarnLevel
	case codes.Unauthenticated:
		return zap.WarnLevel
	case codes.ResourceExhausted:
		return zap.WarnLevel
	case codes.FailedPrecondition:
		return zap.WarnLevel
	case codes.Aborted:
		return zap.WarnLevel
	case codes.OutOfRange:
		return zap.WarnLevel
	case codes.Unimplemented:
		return zap.ErrorLevel
	case codes.Internal:
		return zap.ErrorLevel
	case codes.Unavailable:
		return zap.WarnLevel
	case codes.DataLoss:
		return zap.ErrorLevel
	default:
		return zap.ErrorLevel
	}
}
