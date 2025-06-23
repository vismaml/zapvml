package zapvml

import (
	"strings"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc/codes"
)

var (
	// Log is global logger
	Log *zap.Logger
)

// Config represents the configuration options for the logger.
type Config struct {
	Level               zapcore.Level `required:"true" default:"warn"`
	ServiceName         string        `required:"true" default:"default_service"`
	EnableCtxtraceWarns bool          `default:"false"`
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
	switch cfg.Level {
	case zap.DebugLevel:
		config = zapdriver.NewDevelopmentConfig()
		config.Encoding = "console"
	case zap.InfoLevel:
		config = zapdriver.NewProductionConfig()
		config.Encoding = "stackdriver-json"
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case zap.WarnLevel:
		config = zapdriver.NewProductionConfig()
		config.Encoding = "stackdriver-json"
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	default:
		config = zapdriver.NewProductionConfig()
		config.Encoding = "stackdriver-json"
	}

	Log, err = config.Build(zapdriver.WrapCore(
		zapdriver.ReportAllErrors(true),
		zapdriver.ServiceName(cfg.ServiceName),
	), zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return newFilteringCore(core, cfg.EnableCtxtraceWarns)
	}))
	if err != nil {
		panic(err)
	}

	zap.RedirectStdLog(Log)
}

// CodeToLevel maps gRPC status codes to appropriate log levels.
//
//nolint:gocyclo // This function intentionally handles all gRPC status codes explicitly
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

// filteringCore wraps a zapcore.Core to filter out specific log messages
type filteringCore struct {
	zapcore.Core
	enableCtxtraceWarns bool
}

func newFilteringCore(core zapcore.Core, enableCtxtraceWarns bool) zapcore.Core {
	return &filteringCore{
		Core:                core,
		enableCtxtraceWarns: enableCtxtraceWarns,
	}
}

func (f *filteringCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// Filter out ctxtrace warnings if disabled
	if !f.enableCtxtraceWarns &&
		ent.Level == zapcore.WarnLevel &&
		strings.Contains(ent.Caller.File, "ctxtrace") &&
		strings.Contains(ent.Message, "b3 injection failed") {
		return ce
	}
	return f.Core.Check(ent, ce)
}

func (f *filteringCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// Filter out ctxtrace warnings if disabled
	if !f.enableCtxtraceWarns &&
		ent.Level == zapcore.WarnLevel &&
		strings.Contains(ent.Caller.File, "ctxtrace") &&
		strings.Contains(ent.Message, "b3 injection failed") {
		return nil
	}
	return f.Core.Write(ent, fields)
}

func (f *filteringCore) With(fields []zapcore.Field) zapcore.Core {
	return &filteringCore{
		Core:                f.Core.With(fields),
		enableCtxtraceWarns: f.enableCtxtraceWarns,
	}
}
