package logger

//Target 日志文件写入
type Target interface {
	Write(event *LogEvent, sr Serializer)
	Filled() bool
	Flush()
}

type asyncLogNode struct {
	event      *LogEvent
	serializer Serializer
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
