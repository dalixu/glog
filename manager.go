package glog

import (
	"container/list"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

//Manager 接口
type Manager interface {
	GetLogger(name string) Logger
	WriteEvent(event LogEvent)
	Close()
}

//manager 日志写入
type manager struct {
	file     *ConfigFile
	loggers  sync.Map
	stop     chan bool
	rwLocker *sync.RWMutex
	config   *LogConfig // protected by rwLocker

	queue        *list.List
	atomicLocker int64
}

//newManager 返回Manager
func newManager(config *LogConfig, file *ConfigFile) Manager {
	mr := &manager{
		file:         file,
		stop:         make(chan bool), //无缓冲 自动会等
		rwLocker:     &sync.RWMutex{},
		config:       config,
		queue:        list.New(),
		atomicLocker: 0,
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
	//stopLoop后 lock前 可能已经有WriteEvent进去 要write以及flush
	if m.config.Async {
		m.asyncWrite()
	}
	for _, v := range m.config.Layouts {
		v.Target.Flush()
	}
	m.config = config
	m.startLoop()
}

func (m *manager) WriteEvent(e LogEvent) {
	m.rwLocker.RLock()
	defer m.rwLocker.RUnlock()
	if m.config.Async {
		m.asyncCache(e)
	} else {
		for _, v := range m.config.Layouts {
			if match(&e, v.Target) {
				v.Target.Write(&e, v.Serializer)
			}
		}
	}
}

func (m *manager) flush(force bool) {
	defer func() {
		//保证外围循环不会挂掉
		if err := recover(); err != nil {
			log.Println("flush 0:", err)
		}
	}()
	m.rwLocker.RLock()
	defer m.rwLocker.RUnlock()

	if m.config.Async {
		m.asyncWrite()
	}
	for _, v := range m.config.Layouts {
		if force || v.Target.Overflow() {
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
			case <-time.After(50 * time.Millisecond):
				m.flush(false)
			}
		}
	}()
}

func (m *manager) stopLoop() {
	m.stop <- true //等待loop退出
	m.flush(true)
}
func (m *manager) asyncCache(e LogEvent) {
	m.atomicLock()
	m.queue.PushBack(&e)
	m.atomicUnLock()
}

func (m *manager) asyncWrite() {
	//如果是异步模式
	var queue *list.List
	m.atomicLock()
	if m.queue.Len() > 0 {
		queue = list.New()
		for {
			if m.queue.Len() <= 0 {
				break
			}
			e := m.queue.Front()
			m.queue.Remove(e)
			queue.PushBack(e.Value)
		}
	}
	m.atomicUnLock()
	for {
		if queue == nil || queue.Len() <= 0 {
			break
		}
		node := queue.Front()
		queue.Remove(node)
		e := node.Value.(*LogEvent)
		for _, v := range m.config.Layouts {
			if match(e, v.Target) {
				v.Target.Write(e, v.Serializer)
			}
		}
	}
}

func (m *manager) atomicLock() {
	for {
		if atomic.CompareAndSwapInt64(&m.atomicLocker, 0, 1) {
			break
		}
	}
}

func (m *manager) atomicUnLock() {
	atomic.AddInt64(&m.atomicLocker, -1)
}

func match(event *LogEvent, t Target) bool {
	return (t.Name() == "*" || event.Name == t.Name()) &&
		(t.MaxLevel() == EveryLevel || event.Level <= t.MaxLevel()) &&
		(t.MinLevel() == EveryLevel || event.Level >= t.MinLevel())
}
