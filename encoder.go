// Package zapvml provides zap logging configuration and customizations for VML services.
package zapvml

import (
	"regexp"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func newEncoder(cfg zapcore.EncoderConfig) (zapcore.Encoder, error) {
	return &Encoder{zapcore.NewJSONEncoder(cfg)}, nil
}

// Encoder wraps zapcore.Encoder to customize stack traces to be picked up by Stackdriver error reporting.
// The following issue might make this unnecessary:
// https://github.com/uber-go/zap/issues/514
type Encoder struct {
	zapcore.Encoder
}

// multiline pattern to match the function name line
var functionNamePattern = regexp.MustCompile(`(?m)^(\S+)$`)

// EncodeEntry customizes log entry encoding to format stack traces for Stackdriver error reporting.
func (s *Encoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	if ent.Stack != "" {
		// Make the message look like a real panic, so Stackdriver error reporting picks it up.
		// This used to need the string "panic: " at the beginning, but no longer seems to need it!
		// ent.Message = "panic: " + ent.Message + "\n\ngoroutine 1 [running]:\n"
		ent.Message = ent.Message + "\n\ngoroutine 1 [running]:\n"

		// Trial-and-error: On App Engine Standard go111 the () are needed after function calls
		// zap does not add them, so hack it with a regexp
		replaced := functionNamePattern.ReplaceAllString(ent.Stack, "$1(...)")
		ent.Message += replaced
		ent.Stack = ""
	}
	return s.Encoder.EncodeEntry(ent, fields)
}

// Clone creates a copy of the encoder.
func (s *Encoder) Clone() zapcore.Encoder {
	return &Encoder{s.Encoder.Clone()}
}
