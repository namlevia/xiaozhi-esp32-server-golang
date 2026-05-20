package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	nested "github.com/antonfisher/nested-logrus-formatter"
	log "github.com/sirupsen/logrus"
)

const (
	TYPE_HTTP = 1
)

func init() {
	log.SetFormatter(Formatter(false))
}

func SetOutput(out *os.File) {
	log.SetOutput(out)
}

func SetLevel(level log.Level) {
	log.SetLevel(level)
}

func UseStdout() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(Formatter(true))
}

/*
func getUserInfo(ctx *gin.Context) int {
	if data, ok := ctx.Get("uid"); ok {
		if uid, ok := data.(int); ok {
			return uid
		}
	}
	return 0
}
*/

func getCaller() (string, int) {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown", 0
	}
	shortFile := filepath.Base(file)
	return shortFile, line
}

func addCallerField() *log.Entry {
	file, line := getCaller()
	return log.WithField("caller", fmt.Sprintf("%s:%d", file, line))
}

func Info(args ...interface{}) {
	addCallerField().Info(args...)
}

func Error(args ...interface{}) {
	addCallerField().Error(args...)
}

func Debug(args ...interface{}) {
	addCallerField().Debug(args...)
}

func Warn(args ...interface{}) {
	addCallerField().Warn(args...)
}

func Fatal(args ...interface{}) {
	addCallerField().Fatal(args...)
}

func Infof(format string, args ...interface{}) {
	addCallerField().Infof(format, args...)
}

func Errorf(format string, args ...interface{}) {
	addCallerField().Errorf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	addCallerField().Debugf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	addCallerField().Warnf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	addCallerField().Fatalf(format, args...)
}

func Log(args ...interface{}) *log.Entry {
	fields := log.Fields{}
	lenArgs := len(args)
	for i := 0; i < lenArgs; i = i + 2 {
		var key string
		var ok bool
		if key, ok = args[i].(string); !ok {
			continue
		}

		if i <= lenArgs-2 {
			fields[key] = args[i+1]
			continue
		}
		fields[key] = ""
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}
	shortFile := filepath.Base(file)
	fields["caller"] = fmt.Sprintf("%s:%d", shortFile, line)

	log.SetFormatter(Formatter(true))
	return log.WithFields(fields)
}

func Formatter(isConsole bool) *nested.Formatter {
	fmtter := &nested.Formatter{
		FieldsOrder:      []string{"time", "level", "caller", "msg"},
		HideKeys:         true,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		CallerFirst:      true,
		NoUppercaseLevel: true,
		ShowFullLevel:    true,
		//NoFieldsSpace:    true,
		CustomCallerFormatter: func(frame *runtime.Frame) string {
			return ""
		},
	}
	if isConsole {
		fmtter.NoColors = false
	} else {
		fmtter.NoColors = true
	}
	return fmtter
}

func DebugStack() {
	for i := 0; i < 5; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		shortFile := filepath.Base(file)
		log.Infof("Call stack[%d]: %s:%d", i, shortFile, line)
	}
}
