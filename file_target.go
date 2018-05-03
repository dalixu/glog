package glog

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

//FileTarget 文件项
type fileTarget struct {
	name       string        //只读
	minLevel   LogLevel      //只读
	maxLevel   LogLevel      //只读
	suffix     string        //只读文件名后缀 默认的文件名是 {shortDate}-suffix
	interval   time.Duration //只读 写入的时间间隔
	volumeSize int64         //单个日志文件大小
	cacheSize  int           // 日志缓存大小
	root       string        // 日志存放的根目录

	slice           int //当前写入的文件序号 默认为0
	fullLogFileName string
	currLogSize     int64

	locker        *sync.Mutex
	currLogBuff   int             //protected by locker
	logBuf        [2]bytes.Buffer //protected by locker
	currCacheSize int             //protected by locker 当前buffer中的大小

	nextWriteTime time.Time
	lastPCDate    string
}

func (ft *fileTarget) Name() string {
	return ft.name
}

func (ft *fileTarget) MinLevel() LogLevel {
	return ft.minLevel
}

func (ft *fileTarget) MaxLevel() LogLevel {
	return ft.maxLevel
}

func (ft *fileTarget) Write(event *LogEvent, sr Serializer) {
	bs := sr.Encode(event)
	if bs == nil {
		bs = []byte(fmt.Sprintf("%+v", event))
	}

	ft.locker.Lock()
	defer ft.locker.Unlock()
	index := ft.currLogBuff % len(ft.logBuf)
	ft.logBuf[index].Write(bs)
	ft.logBuf[index].WriteByte('\r')
	ft.logBuf[index].WriteByte('\n')
	ft.currCacheSize += len(bs) + 2

}

func (ft *fileTarget) Overflow() bool {
	//这里ft.CurrCacheSize 没有加锁 但是考虑到CurrCacheSize 不需要太精确
	//只要没有panic就不加锁 避免降低效率
	return time.Now().After(ft.nextWriteTime) || ft.currCacheSize >= ft.cacheSize
}

func (ft *fileTarget) Flush() {
	//写入日志文件
	var cache *bytes.Buffer
	ft.locker.Lock()
	cache = &ft.logBuf[ft.currLogBuff%len(ft.logBuf)]
	ft.currLogBuff = (ft.currLogBuff + 1) % len(ft.logBuf)
	ft.currCacheSize = 0

	ft.locker.Unlock()
	if cache.Len() > 0 {
		//写入日志文件
		ft.createLogFile()
		ft.currLogSize += int64(ft.writeFromCache(cache))
	}
	// nextWritetime 是一个结构 Overflow里读 Flush里写 如果 两个函数不在一个线程会出问题
	//目前为止manager保证了 overflow和flush会在一个线程调用
	ft.nextWriteTime = time.Now().Add(ft.interval)
}

func (ft *fileTarget) createLogFile() {
	currPCDate := getShortDate()
	if ft.fullLogFileName != "" && ft.currLogSize >= ft.volumeSize {
		//文件超过允许的大小 写入到新文件中去
		if ft.slice < 100 {
			ft.slice++
			ft.fullLogFileName = ""
			ft.currLogSize = 0
		}
	}
	//日期切换了 slice也要变成0
	if ft.lastPCDate != currPCDate {
		ft.slice = 0
		ft.currLogSize = 0
		ft.fullLogFileName = "" //文件名置空
		ft.lastPCDate = currPCDate
	}
	if ft.fullLogFileName == "" {
		for {
			//如果文件名不存在 或者 日期切换 要根据slice来生成新的文件名
			sliceDesc := strconv.Itoa(ft.slice)
			ft.fullLogFileName = path.Join(ft.root, ft.lastPCDate+"-"+sliceDesc+"-"+ft.suffix)
			stat, err := os.Stat(ft.fullLogFileName)
			if err == nil {
				ft.currLogSize = stat.Size()
			}
			if ft.currLogSize < ft.volumeSize || ft.slice >= 100 {
				break
			}
			ft.slice++
		}
	}
}

func getShortDate() string {
	return time.Now().Format("2006-01-02")
}

func (ft *fileTarget) writeFromCache(logs *bytes.Buffer) (size int) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("writeFromCache 0:", ft.fullLogFileName, ":", err)
			size = 0
		}
	}()
	if logs.Len() <= 0 {
		return 0
	}
	defer logs.Reset()

	f, err := os.OpenFile(ft.fullLogFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Println("writeFromCache 1:", ft.fullLogFileName, ":", err)
		return 0
	}
	defer f.Close()
	n, err := f.Write(logs.Bytes())
	if err == nil {
		err = f.Sync()
	}
	if err != nil {
		log.Println("writeFromCache 2:", ft.fullLogFileName, ":", err)
		return 0
	}
	return n
}

func createFileTarget(config map[string]interface{}) Target {
	ft := &fileTarget{}
	volumeSize := config["VolumeSize"]
	if volumeSize != nil {
		ft.volumeSize = volumeSize.(int64)
	} else {
		ft.volumeSize = 1024 * 1024 * 10
	}

	root := config["Root"]
	if root != nil {
		ft.root = root.(string)
	} else {
		ft.root = "./logs"
	}
	err := os.MkdirAll(ft.root, os.ModePerm)
	if err != nil {
		log.Println("createFileTarget:path ", ft.root, " ", err)
		return nil
	}
	maxLevel := config["MaxLevel"]
	if maxLevel == nil {
		ft.maxLevel = EveryLevel
	} else {
		ft.maxLevel = toLevel(maxLevel.(string))
	}

	minLevel := config["MinLevel"]
	if minLevel == nil {
		ft.minLevel = EveryLevel
	} else {
		ft.minLevel = toLevel(minLevel.(string))
	}
	name := config["Name"]
	if name == nil {
		ft.name = "*"
	} else {
		ft.name = name.(string)
	}
	suffix := config["Suffix"]
	if suffix == nil {
		ft.suffix = ".log"
	} else {
		ft.suffix = suffix.(string)
	}
	interval := config["Interval"]
	if interval == nil {
		ft.interval = time.Duration(time.Second)
	} else {
		ft.interval = time.Duration(interval.(int)) * time.Second
	}

	cacheSize := config["CacheSize"]
	if cacheSize == nil {
		ft.cacheSize = 1024 * 8
	} else {
		ft.cacheSize = cacheSize.(int)
	}

	ft.locker = &sync.Mutex{}
	ft.currLogBuff = 0
	ft.nextWriteTime = time.Now().Add(ft.interval)
	return ft
}
