package logger

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

//Target 日志文件写入
type Target interface {
	Match(event *LogEvent) bool
	Write(log []byte)

	NeedFlush() bool
	Flush()
}

//FileTarget 文件项
type fileTarget struct {
	Name       string        //只读
	MinLevel   LogLevel      //只读
	MaxLevel   LogLevel      //只读
	Suffix     string        //只读文件名后缀 默认的文件名是 {shortDate}-suffix
	Serializer Serializer    //只读序列化
	Interval   time.Duration //只读 写入的时间间隔
	VolumeSize int64         //单个日志文件大小
	CacheSize  int           // 日志缓存大小
	Root       string        // 日志存放的根目录

	Slice           int //当前写入的文件序号 默认为0
	LogFileName     string
	FullLogFileName string
	CurrLogSize     int64

	Locker        *sync.Mutex
	CurrLogBuff   int             //protected by locker
	LogBuf        [2]bytes.Buffer //protected by locker
	CurrCacheSize int             //protected by locker 当前buffer中的大小

	NextWriteTime time.Time
	LastPCDate    string
}

func (ft *fileTarget) Match(event *LogEvent) bool {
	return event.Level >= ft.MinLevel && event.Level <= ft.MaxLevel && (ft.Name == "" || ft.Name == event.Name)
}

func (ft *fileTarget) Write(log []byte) {
	ft.Locker.Lock()
	index := ft.CurrLogBuff % len(ft.LogBuf)
	ft.LogBuf[index].Write(log)
	ft.LogBuf[index].WriteByte('\n')
	ft.CurrCacheSize += len(log) + 1
	ft.Locker.Unlock()
}

func (ft *fileTarget) NeedFlush() bool {
	now := time.Now()
	return now.After(ft.NextWriteTime) || ft.CurrCacheSize >= ft.CacheSize
}

func (ft *fileTarget) Flush() {
	//写入日志文件
	var cache *bytes.Buffer
	ft.Locker.Lock()
	cache = &ft.LogBuf[ft.CurrLogBuff%len(ft.LogBuf)]
	ft.CurrLogBuff = (ft.CurrLogBuff + 1) % len(ft.LogBuf)
	ft.CurrCacheSize = 0
	ft.Locker.Unlock()
	//写入日志文件
	ft.createLogFile()

	ft.CurrLogSize += int64(ft.writeToFile(ft.FullLogFileName, cache))
	ft.NextWriteTime = time.Now().Add(ft.Interval)
}

func (ft *fileTarget) createLogFile() {
	currPCDate := getShortDate()
	if ft.FullLogFileName != "" && ft.CurrLogSize >= ft.VolumeSize {
		//文件超过允许的大小 写入到新文件中去
		if ft.Slice < 100000 {
			ft.Slice++
			ft.FullLogFileName = ""
			ft.CurrLogSize = 0
		}
	}
	//日期切换了 slice也要变成0
	if ft.LastPCDate != currPCDate {
		ft.Slice = 0
		ft.CurrLogSize = 0
		ft.FullLogFileName = "" //文件名置空
		ft.LastPCDate = currPCDate
	}
	if ft.FullLogFileName == "" {
		for {
			//如果文件名不存在 或者 日期切换 要根据slice来生成新的文件名
			sliceDesc := strconv.Itoa(ft.Slice)
			path := ft.Root + "/" + ft.LastPCDate + "-" + sliceDesc + "-" + ft.Suffix
			stat, err := os.Stat(path)
			if err == nil {
				ft.CurrLogSize = stat.Size()
			}
			if ft.CurrLogSize < ft.VolumeSize || ft.Slice >= 100000 {
				break
			}
			ft.Slice++
		}
	}
}

func getShortDate() string {
	return time.Now().Format("2006-01-02")
}

func (ft *fileTarget) writeToFile(fn string, logs *bytes.Buffer) (size int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("writeToFile 0:", fn, ":", err)
			size = 0
		}
	}()
	if logs.Len() <= 0 {
		return 0
	}
	defer logs.Reset()

	f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		fmt.Println("writeToFile 1:", fn, ":", err)
		return 0
	}
	defer f.Close()
	n, err := f.Write(logs.Bytes())
	if err == nil {
		err = f.Sync()
	}
	if err != nil {
		fmt.Println("writeToFile 2:", fn, ":", err)
		return 0
	}
	return n
}

func createFileTarget(config map[string]interface{}) Target {
	ft := &fileTarget{}
	volumeSize := config["VolumeSize"]
	if volumeSize != nil {
		ft.VolumeSize = volumeSize.(int64)
	} else {
		ft.VolumeSize = 1024 * 1024 * 10
	}
	cacheSize := config["CacheSize"]
	if cacheSize != nil {
		ft.CacheSize = cacheSize.(int)
	} else {
		ft.CacheSize = 1024 * 10
	}
	root := config["Root"]
	if root != nil {
		ft.Root = root.(string)
	} else {
		ft.Root = "./logs"
	}
	err := os.MkdirAll(ft.Root, os.ModePerm)
	if err != nil {
		fmt.Println("createFileTarget:path ", ft.Root, "\n", err)
		return nil
	}
	maxLevel := config["MaxLevel"]
	if maxLevel == nil {
		ft.MaxLevel = CriticalLevel
	} else {
		ft.MaxLevel = toLevel(maxLevel.(string), CriticalLevel)
	}

	minLevel := config["MinLevel"]
	if minLevel == nil {
		ft.MinLevel = TraceLevel
	} else {
		ft.MinLevel = toLevel(minLevel.(string), TraceLevel)
	}
	name := config["Name"]
	if name == nil {
		ft.Name = ""
	} else {
		ft.Name = name.(string)
	}
	suffix := config["Suffix"]
	if suffix == nil {
		ft.Suffix = ".log"
	} else {
		ft.Suffix = suffix.(string)
	}
	interval := config["Interval"]
	if interval == nil {
		ft.Interval = time.Duration(time.Second)
	} else {
		ft.Interval = time.Duration(interval.(int)) * time.Second
	}
	ft.Locker = &sync.Mutex{}
	ft.CurrLogBuff = 0
	return ft
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
	} else if l == "Critical" {
		return CriticalLevel
	}
	return dt
}
