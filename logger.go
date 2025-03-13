package zapvml

import (
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
	Level       zapcore.Level `required:"true" default:"warn"`
	Debug       bool          `required:"true" default:"false"`
	ServiceName string        `required:"true" default:"default_service"`
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

	err := zap.RegisterEncoder("stackdriver-json", newEncoder)
	if err != nil {
		panic(err)
	}

	var config zap.Config
	if cfg.Debug {
		config = zapdriver.NewDevelopmentConfig()
		config.Encoding = "console"
	} else {
		config = zapdriver.NewProductionConfig()
		config.Encoding = "stackdriver-json"
	}

	Log, err = config.Build(zapdriver.WrapCore(
		zapdriver.ReportAllErrors(true),
		zapdriver.ServiceName(cfg.ServiceName),
	))
	if err != nil {

		panic(err)
	}

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
