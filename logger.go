package logger

import (
	"log"
	"os"
	"sync"
)

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
}

//Log 实现简单的log功能
type Log struct {
	trace    *log.Logger
	debug    *log.Logger
	info     *log.Logger
	warn     *log.Logger
	err      *log.Logger
	critical *log.Logger
}

func newLog() *Log {
	return &Log{
		trace:    log.New(os.Stderr, "[TRACE]", log.LstdFlags),
		debug:    log.New(os.Stderr, "[DEBUG]", log.LstdFlags),
		info:     log.New(os.Stderr, "[INFO]", log.LstdFlags),
		warn:     log.New(os.Stderr, "[WARN]", log.LstdFlags),
		err:      log.New(os.Stderr, "[ERROR]", log.LstdFlags),
		critical: log.New(os.Stderr, "[CRITICAL]", log.LstdFlags),
	}
}

//Trace 打印log
func (l *Log) Trace(v ...interface{}) {
	l.trace.Println(v...)
}

//Tracef 打印log
func (l *Log) Tracef(format string, v ...interface{}) {
	l.trace.Printf(format, v...)
}

//Debug 打印log
func (l *Log) Debug(v ...interface{}) {
	l.debug.Println(v...)
}

//Debugf 打印log
func (l *Log) Debugf(format string, v ...interface{}) {
	l.debug.Printf(format, v...)
}

//Info 打印log
func (l *Log) Info(v ...interface{}) {
	l.info.Println(v...)
}

//Infof 打印log
func (l *Log) Infof(format string, v ...interface{}) {
	l.info.Printf(format, v...)
}

//Warn 打印log
func (l *Log) Warn(v ...interface{}) {
	l.warn.Println(v...)
}

//Warnf 打印log
func (l *Log) Warnf(format string, v ...interface{}) {
	l.warn.Printf(format, v...)
}

//Error 打印log
func (l *Log) Error(v ...interface{}) {
	l.err.Println(v...)
}

//Errorf 打印log
func (l *Log) Errorf(format string, v ...interface{}) {
	l.err.Printf(format, v...)
}

//Critical 打印log
func (l *Log) Critical(v ...interface{}) {
	l.critical.Println(v...)
}

//Criticalf 打印log
func (l *Log) Criticalf(format string, v ...interface{}) {
	l.critical.Printf(format, v...)
}

var mylog Logger = newLog()
var mylock sync.RWMutex

//SetLogger 替换掉默认的log接口 赋值没有加锁 最好是在init里调用
func SetLogger(l Logger) {
	mylock.Lock()
	defer mylock.Unlock()
	mylog = l
}

//Trace 使用Logger接口打印
func Trace(v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Trace(v...)
}

//Tracef 使用Logger接口打印
func Tracef(format string, v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Tracef(format, v...)
}

//Debug 使用Logger接口打印
func Debug(v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Debug(v...)
}

//Debugf 使用Logger接口打印
func Debugf(format string, v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Debugf(format, v...)
}

//Info 使用Logger接口打印
func Info(v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Info(v...)
}

//Infof 使用Logger接口打印
func Infof(format string, v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Infof(format, v...)
}

//Warn 使用Logger接口打印
func Warn(v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Warn(v...)
}

//Warnf 使用Logger接口打印
func Warnf(format string, v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Warnf(format, v...)
}

//Error 使用Logger接口打印
func Error(v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Error(v...)
}

//Errorf 使用Logger接口打印
func Errorf(format string, v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Errorf(format, v...)
}

//Critical 使用Logger接口打印
func Critical(v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Critical(v...)
}

//Criticalf 使用Logger接口打印
func Criticalf(format string, v ...interface{}) {
	mylock.RLock()
	defer mylock.RUnlock()
	mylog.Criticalf(format, v...)
}
