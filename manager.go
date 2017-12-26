package logger

import (
	"fmt"
	"runtime"
	"sync"
)

//Manager 接口
type Manager interface {
	GetLogger(name string) Logger
	WriteEvent(event LogEvent)
	Reload(config *LogConfig)
	Close()
}

//manager 日志写入
type manager struct {
	file     *ConfigFile
	loggers  sync.Map
	stop     chan bool
	rwLocker *sync.RWMutex
	config   *LogConfig // protected by rwLocker
}

//newManager 返回Manager
func newManager(config *LogConfig, file *ConfigFile) Manager {
	mr := &manager{
		stop:     make(chan bool), //无缓冲 自动会等
		rwLocker: &sync.RWMutex{},
		config:   config,
		file:     file,
	}
	mr.startLoop()
	mr.file.StartMonitor(mr.Reload)
	return mr
}

func (m *manager) Close() {
	m.file.StopMonitor()
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

	for _, v := range m.config.Layouts {
		v.Target.Flush()
	}
	m.config = config
	m.startLoop()
}

func (m *manager) WriteEvent(e LogEvent) {
	m.rwLocker.RLock()
	defer m.rwLocker.RUnlock()
	for _, v := range m.config.Layouts {
		v.Target.Write(&e, v.Serializer)
	}
}

func (m *manager) flush(force bool) {
	defer func() {
		//保证外围循环不会挂掉
		if err := recover(); err != nil {
			fmt.Println("flush 0:", err)
		}
	}()
	m.rwLocker.RLock()
	defer m.rwLocker.RUnlock()
	for _, v := range m.config.Layouts {
		if force || v.Target.Filled() {
			v.Target.Flush()
		}
	}
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
