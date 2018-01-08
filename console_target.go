package glog

import (
	"log"
)

//输出信息到console 使用log库来实现
type consoleTarget struct {
	name     string   //只读
	minLevel LogLevel //只读
	maxLevel LogLevel //只读
}

func (ct *consoleTarget) Name() string {
	return ct.name
}

func (ct *consoleTarget) MinLevel() LogLevel {
	return ct.minLevel
}

func (ct *consoleTarget) MaxLevel() LogLevel {
	return ct.maxLevel
}

func (ct *consoleTarget) Write(event *LogEvent, sr Serializer) {
	bs := sr.Encode(event)
	if bs != nil {
		log.Println(string(bs))
	} else {
		log.Printf("%+v\n", event)
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
		ct.name = "*"
	} else {
		ct.name = name.(string)
	}
	maxLevel := config["MaxLevel"]
	if maxLevel == nil {
		ct.maxLevel = EveryLevel
	} else {
		ct.maxLevel = toLevel(maxLevel.(string))
	}

	minLevel := config["MinLevel"]
	if minLevel == nil {
		ct.minLevel = EveryLevel
	} else {
		ct.minLevel = toLevel(minLevel.(string))
	}
	return ct
}
