package logger

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//Manager 接口
type Manager interface {
	GetLogger(name string) Logger
	WriteEvent(event LogEvent)
	Reload(config *LogConfig)
	Stop()
}

//manager 日志写入
type manager struct {
	loggers sync.Map

	stop     chan bool
	rwLocker *sync.RWMutex
	config   *LogConfig // protected by rwLocker
}

//newManager 返回Manager
func newManager(config *LogConfig) Manager {
	mr := &manager{
		stop:     make(chan bool), //无缓冲 自动会等
		rwLocker: &sync.RWMutex{},
		config:   config,
	}
	mr.startLoop()
	return mr
}

func (m *manager) Stop() {
	m.stopLoop()
}

func (m *manager) GetLogger(name string) Logger {
	l, ok := m.loggers.Load(name)
	if !ok {
		l, _ = m.loggers.LoadOrStore(name, newLogger(m, name))
	}
	return l.(Logger)
}

//Reload 重新加载Config
func (m *manager) Reload(config *LogConfig) {
	m.stopLoop()
	m.rwLocker.Lock()
	defer m.rwLocker.Unlock()
	//把旧的config里的缓存全部写入到文件 防止日志丢失
	for _, v := range m.config.Targets {
		//写入日志文件
		var cache *bytes.Buffer
		if v.CurrLogBuff == LogBufferIDA {
			cache = &v.LogBufA
		} else {
			cache = &v.LogBufB
		}
		//写入日志文件
		m.createLogFile(v)
		m.writeToFile(v.FullLogFileName, cache)
	}
	m.config = config
	m.startLoop()
}

func (m *manager) WriteEvent(e LogEvent) {
	m.rwLocker.RLock()
	defer m.rwLocker.RUnlock()
	for _, v := range m.config.Targets {
		if !v.Match(e.Level, e.Name) {
			continue
		}
		bs := v.Serializer.Encode(&e)
		if bs == nil {
			continue
		}
		v.Locker.Lock()
		if v.CurrLogBuff == LogBufferIDA {
			v.LogBufA.Write(bs)
			v.LogBufA.WriteByte('\n')
		} else {
			v.LogBufB.Write(bs)
			v.LogBufB.WriteByte('\n')
		}
		v.CurrCacheSize += len(bs) + 1
		v.Locker.Unlock()
	}
}

func (m *manager) flush(force bool) {
	defer func() {
		//保证外围循环不会挂掉
		if err := recover(); err != nil {
			fmt.Println("flush 0:", err)
		}
	}()
	now := time.Now()
	m.rwLocker.RLock()
	defer m.rwLocker.RUnlock()
	for _, v := range m.config.Targets {
		if now.After(v.NextWriteTime) || v.CurrCacheSize >= m.config.LogCacheSize || force {
			//写入日志文件
			var cache *bytes.Buffer
			v.Locker.Lock()
			if v.CurrLogBuff == LogBufferIDA {
				cache = &v.LogBufA
				v.CurrLogBuff = LogBufferIDB
			} else {
				cache = &v.LogBufB
				v.CurrLogBuff = LogBufferIDA
			}
			v.CurrCacheSize = 0
			v.Locker.Unlock()
			//写入日志文件
			m.createLogFile(v)

			v.CurrLogSize += int64(m.writeToFile(v.FullLogFileName, cache))
			v.NextWriteTime = now.Add(v.Interval)
		}
	}
}

func (m *manager) createLogFile(ft *FileTarget) {
	currPCDate := getShortDate()
	if ft.FullLogFileName != "" && ft.CurrLogSize >= m.config.SingleFileSize {
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
			path := m.config.Root + "/" + ft.LastPCDate + "-" + sliceDesc + "-" + ft.FileSuffix
			stat, err := os.Stat(path)
			if err == nil {
				ft.CurrLogSize = stat.Size()
			}
			if ft.CurrLogSize < m.config.SingleFileSize || ft.Slice >= 100000 {
				break
			}
			ft.Slice++
		}
	}
}

func (m *manager) writeToFile(fn string, logs *bytes.Buffer) int {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("writeToFile 0:", fn, ":", err)
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

func (m *manager) startLoop() {
	go func() {
	loop:
		for {
			select {
			case <-m.stop:
				break loop
			default:
			}
			m.flush(false)
			runtime.Gosched()
		}
	}()
}

func (m *manager) stopLoop() {
	m.stop <- true //等待loop退出
	m.flush(true)
}

func getShortDate() string {
	return time.Now().Format("2006-01-02")
}
