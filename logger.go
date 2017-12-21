//Package logger  学习自FLogger https://github.com/cyfonly/FLogger.git
package logger

import (
	"runtime/debug"
	"time"
)

// import (
// 	"log"
// 	"os"
// )

//Properties LogEvent属性 方便添加自定义字段
type Properties map[interface{}]interface{}

//LogEvent log的具体内容
type LogEvent struct {
	Properties
	Level      LogLevel
	LevelDesc  string //level的文本描述
	Name       string
	Format     string //format或者message
	Args       []interface{}
	StackTrace string
	Time       string
}

//Logger 日志打印接口 方便替换为第三方log
type Logger interface {
	Trace(v ...interface{})
	Tracef(format string, v ...interface{})
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Critical(v ...interface{})
	Criticalf(format string, v ...interface{})

	WriteEvent(e LogEvent) //也许应该用*LogEvent
}

//newLogger 返回Flogger
func newLogger(mr Manager, name string) Logger {
	return &logger{
		Manager: mr,
		name:    name,
	}
}

type logger struct {
	Manager
	name string
}

func (lr *logger) WriteEvent(e LogEvent) {
	lr.Manager.WriteEvent(e)
}

//Trace 实现接口
func (lr *logger) Trace(v ...interface{}) {
	lr.write(TraceLevel, "TRACE", v...)
}

//Tracef 实现接口
func (lr *logger) Tracef(format string, v ...interface{}) {
	lr.writef(TraceLevel, "TRACE", format, v...)
}

//Debug 实现接口
func (lr *logger) Debug(v ...interface{}) {
	lr.write(DebugLevel, "DEBUG", v...)
}

//Debugf 实现接口
func (lr *logger) Debugf(format string, v ...interface{}) {
	lr.writef(DebugLevel, "DEBUG", format, v...)
}

//Info 实现接口
func (lr *logger) Info(v ...interface{}) {
	lr.write(InfoLevel, "INFO", v...)
}

//Infof 实现接口
func (lr *logger) Infof(format string, v ...interface{}) {
	lr.writef(InfoLevel, "INFO", format, v...)
}

//Warn 实现接口
func (lr *logger) Warn(v ...interface{}) {
	lr.write(WarnLevel, "WARN", v...)
}

//Warnf 实现接口
func (lr *logger) Warnf(format string, v ...interface{}) {
	lr.writef(WarnLevel, "WARN", format, v...)
}

//Error 实现接口
func (lr *logger) Error(v ...interface{}) {
	lr.write(ErrorLevel, "ERROR", v...)
}

//Errorf 实现接口
func (lr *logger) Errorf(format string, v ...interface{}) {
	lr.writef(ErrorLevel, "ERROR", format, v...)
}

//Critical 实现接口
func (lr *logger) Critical(v ...interface{}) {
	lr.write(CriticalLevel, "CRITICAL", v...)
}

//Criticalf 实现接口
func (lr *logger) Criticalf(format string, v ...interface{}) {
	lr.writef(CriticalLevel, "CRITICAL", format, v...)
}

func (lr *logger) write(level LogLevel, desc string, args ...interface{}) {
	stackTrace := ""
	if level >= ErrorLevel {
		stackTrace = string(debug.Stack())
	}
	lr.WriteEvent(LogEvent{
		Level:      level,
		LevelDesc:  desc,
		Name:       lr.name,
		Args:       args,
		StackTrace: stackTrace,
		Time:       time.Now().Format("2006-01-02 15:04:05.0000"),
	})
}

func (lr *logger) writef(level LogLevel, desc string, format string, args ...interface{}) {
	stackTrace := ""
	if level >= ErrorLevel {
		stackTrace = string(debug.Stack())
	}
	lr.WriteEvent(LogEvent{
		Level:      level,
		LevelDesc:  desc,
		Name:       lr.name,
		Format:     format,
		Args:       args,
		StackTrace: stackTrace,
		Time:       time.Now().Format("2006-01-02 15:04:05.0000"),
	})

}
