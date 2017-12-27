package logger

import (
	"bytes"
	"container/list"
	"log"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

//FileTarget 文件项
type fileTarget struct {
	Name       string        //只读
	MinLevel   LogLevel      //只读
	MaxLevel   LogLevel      //只读
	Suffix     string        //只读文件名后缀 默认的文件名是 {shortDate}-suffix
	Interval   time.Duration //只读 写入的时间间隔
	VolumeSize int64         //单个日志文件大小
	CacheSize  int           // 日志缓存大小
	Root       string        // 日志存放的根目录
	Async      bool          //异步序列化日志

	Slice           int //当前写入的文件序号 默认为0
	FullLogFileName string
	CurrLogSize     int64

	Locker        *sync.Mutex
	CurrLogBuff   int             //protected by locker
	LogBuf        [2]bytes.Buffer //protected by locker
	CurrCacheSize int             //protected by locker 当前buffer中的大小
	Queue         *list.List

	NextWriteTime time.Time
	LastPCDate    string
}

func (ft *fileTarget) Match(event *LogEvent) bool {
	return event.Level >= ft.MinLevel && event.Level <= ft.MaxLevel && (ft.Name == "" || ft.Name == event.Name)
}

func (ft *fileTarget) Write(event *LogEvent, sr Serializer) {
	if !ft.Match(event) {
		return
	}
	ft.Locker.Lock()
	defer ft.Locker.Unlock()
	if ft.Async {
		ft.Queue.PushBack(asyncLogNode{event: event, serializer: sr})
		ft.CurrCacheSize++
	} else {
		bs := sr.Encode(event)
		if bs == nil {
			return
		}
		index := ft.CurrLogBuff % len(ft.LogBuf)
		ft.LogBuf[index].Write(bs)
		ft.CurrCacheSize += len(bs)
	}

}

func (ft *fileTarget) Filled() bool {
	now := time.Now()
	//这里ft.CurrCacheSize 没有加锁 但是考虑到CurrCacheSize 不需要太精确
	//只要没有panic就不加锁 避免降低效率
	return now.After(ft.NextWriteTime) || ft.CurrCacheSize >= ft.CacheSize
}

func (ft *fileTarget) Flush() {
	//写入日志文件
	var cache *bytes.Buffer
	var queue *list.List
	ft.Locker.Lock()
	cache = &ft.LogBuf[ft.CurrLogBuff%len(ft.LogBuf)]
	ft.CurrLogBuff = (ft.CurrLogBuff + 1) % len(ft.LogBuf)
	ft.CurrCacheSize = 0
	if ft.Async || ft.Queue.Len() > 0 {
		queue = list.New()
		for {
			if ft.Queue.Len() <= 0 {
				break
			}
			e := ft.Queue.Front()
			ft.Queue.Remove(e)
			queue.PushBack(e.Value)
		}
	}
	ft.Locker.Unlock()
	for {
		if queue == nil || queue.Len() <= 0 {
			break
		}
		node := queue.Front()
		queue.Remove(node)
		e := node.Value.(asyncLogNode)
		bs := e.serializer.Encode(e.event)
		if bs != nil {
			cache.Write(bs)
		}
	}
	//写入日志文件
	ft.createLogFile()
	ft.CurrLogSize += int64(ft.writeFromCache(cache))
	ft.NextWriteTime = time.Now().Add(ft.Interval)
}

func (ft *fileTarget) createLogFile() {
	currPCDate := getShortDate()
	if ft.FullLogFileName != "" && ft.CurrLogSize >= ft.VolumeSize {
		//文件超过允许的大小 写入到新文件中去
		if ft.Slice < 100 {
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
			ft.FullLogFileName = path.Join(ft.Root, ft.LastPCDate+"-"+sliceDesc+"-"+ft.Suffix)
			stat, err := os.Stat(ft.FullLogFileName)
			if err == nil {
				ft.CurrLogSize = stat.Size()
			}
			if ft.CurrLogSize < ft.VolumeSize || ft.Slice >= 100 {
				break
			}
			ft.Slice++
		}
	}
}

func getShortDate() string {
	return time.Now().Format("2006-01-02")
}

func (ft *fileTarget) writeFromCache(logs *bytes.Buffer) (size int) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("writeFromCache 0:", ft.FullLogFileName, ":", err)
			size = 0
		}
	}()
	if logs.Len() <= 0 {
		return 0
	}
	defer logs.Reset()

	f, err := os.OpenFile(ft.FullLogFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Println("writeFromCache 1:", ft.FullLogFileName, ":", err)
		return 0
	}
	defer f.Close()
	n, err := f.Write(logs.Bytes())
	if err == nil {
		err = f.Sync()
	}
	if err != nil {
		log.Println("writeFromCache 2:", ft.FullLogFileName, ":", err)
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

	root := config["Root"]
	if root != nil {
		ft.Root = root.(string)
	} else {
		ft.Root = "./logs"
	}
	err := os.MkdirAll(ft.Root, os.ModePerm)
	if err != nil {
		log.Println("createFileTarget:path ", ft.Root, "\n", err)
		return nil
	}
	maxLevel := config["MaxLevel"]
	if maxLevel == nil {
		ft.MaxLevel = FatalLevel
	} else {
		ft.MaxLevel = toLevel(maxLevel.(string), FatalLevel)
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
	async := config["Async"]
	if async == nil {
		ft.Async = true
	} else {
		ft.Async = async.(bool)
	}

	cacheSize := config["CacheSize"]
	if cacheSize == nil {
		if ft.Async {
			ft.CacheSize = 10 //如果async为true 则cachesize 为日志队列的数量
		} else {
			ft.CacheSize = 1024 * 8
		}
	} else {
		ft.CacheSize = cacheSize.(int)
	}
	ft.Locker = &sync.Mutex{}
	ft.Queue = list.New()
	ft.CurrLogBuff = 0
	ft.NextWriteTime = time.Now().Add(ft.Interval)
	return ft
}
