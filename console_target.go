package logger

import (
	"log"
)

//输出信息到console 使用log库来实现
type consoleTarget struct {
	Name     string   //只读
	MinLevel LogLevel //只读
	MaxLevel LogLevel //只读
}

func (ct *consoleTarget) Match(event *LogEvent) bool {
	return event.Level >= ct.MinLevel && event.Level <= ct.MaxLevel && (ct.Name == "" || ct.Name == event.Name)
}

func (ct *consoleTarget) Write(event *LogEvent, sr Serializer) {
	if !ct.Match(event) {
		return
	}
	bs := sr.Encode(event)
	if bs != nil {
		log.Print(string(bs))
	} else {
		log.Printf("%+v", event)
	}
}

func (ct *consoleTarget) Overflow() bool {
	return false
}

func (ct *consoleTarget) Flush() {
	return
}

func createConsoleTarget(config map[string]interface{}) Target {
	ct := &consoleTarget{}

	name := config["Name"]
	if name == nil {
		ct.Name = ""
	} else {
		ct.Name = name.(string)
	}
	maxLevel := config["MaxLevel"]
	if maxLevel == nil {
		ct.MaxLevel = FatalLevel
	} else {
		ct.MaxLevel = toLevel(maxLevel.(string), FatalLevel)
	}

	minLevel := config["MinLevel"]
	if minLevel == nil {
		ct.MinLevel = TraceLevel
	} else {
		ct.MinLevel = toLevel(minLevel.(string), TraceLevel)
	}
	return ct
}
