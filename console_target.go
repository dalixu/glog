package logger

import (
	"container/list"
	"log"
	"sync"
	"time"
)

//输出信息到console 使用log库来实现
type consoleTarget struct {
	Name     string        //只读
	MinLevel LogLevel      //只读
	MaxLevel LogLevel      //只读
	Interval time.Duration //只读 写入的时间间隔
	Async    bool          //只读 false直接写入
	Locker   *sync.Mutex
	Queue    *list.List

	NextWriteTime time.Time
}

func (ct *consoleTarget) Match(event *LogEvent) bool {
	return event.Level >= ct.MinLevel && event.Level <= ct.MaxLevel && (ct.Name == "" || ct.Name == event.Name)
}

func (ct *consoleTarget) Write(event *LogEvent, sr Serializer) {
	if !ct.Match(event) {
		return
	}

	if ct.Async {
		ct.Locker.Lock()
		defer ct.Locker.Unlock()
		ct.Queue.PushBack(asyncLogNode{event: event, serializer: sr})
	} else {
		bs := sr.Encode(event)
		if bs != nil {
			log.Print(string(bs))
		} else {
			log.Printf("%+v", event)
		}
	}
}

func (ct *consoleTarget) Filled() bool {
	now := time.Now()
	return now.After(ct.NextWriteTime)
}

func (ct *consoleTarget) Flush() {
	var queue *list.List
	ct.Locker.Lock()
	if ct.Queue.Len() > 0 {
		queue = list.New()
		for {
			if ct.Queue.Len() <= 0 {
				break
			}
			e := ct.Queue.Front()
			ct.Queue.Remove(e)
			queue.PushBack(e.Value)
		}
	}
	ct.Locker.Unlock()
	for {
		if queue == nil || queue.Len() <= 0 {
			break
		}
		node := queue.Front()
		queue.Remove(node)
		e := node.Value.(asyncLogNode)
		bs := e.serializer.Encode(e.event)
		if bs != nil {
			log.Print(string(bs))
		} else {
			log.Printf("%+v", e.event)
		}
	}
	ct.NextWriteTime = time.Now().Add(ct.Interval)
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
	interval := config["Interval"]
	if interval == nil {
		ct.Interval = time.Duration(time.Second)
	} else {
		ct.Interval = time.Duration(interval.(int)) * time.Second
	}
	async := config["Async"]
	if async == nil {
		ct.Async = true
	} else {
		ct.Async = async.(bool)
	}
	ct.Locker = &sync.Mutex{}
	ct.Queue = list.New()
	ct.NextWriteTime = time.Now().Add(ct.Interval)
	return ct
}
