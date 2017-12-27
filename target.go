package logger

//Target 日志文件写入
type Target interface {
	Write(event *LogEvent, sr Serializer) //manager 可能在多个routine调用
	Overflow() bool                       //manager保证同一时刻只有1个routine调用
	Flush()                               //manager保证同一时刻只有1个routine调用
}

func toLevel(l string, dt LogLevel) LogLevel {
	if l == "Trace" {
		return TraceLevel
	} else if l == "Debug" {
		return DebugLevel
	} else if l == "Info" {
		return InfoLevel
	} else if l == "Warn" {
		return WarnLevel
	} else if l == "Error" {
		return ErrorLevel
	} else if l == "Fatal" {
		return FatalLevel
	}
	return dt
}
