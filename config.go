package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

//LogLevel log的等级
type LogLevel int

//log的等级
const (
	TraceLevel = 1 + iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	CriticalLevel
)

//LogBufferID 当前使用的buffer
type LogBufferID byte

//log的bufferID
const (
	LogBufferIDA = 'A'
	LogBufferIDB = 'B'
)

//FileTarget 文件项
type FileTarget struct {
	Name       string        //只读
	MinLevel   LogLevel      //只读
	MaxLevel   LogLevel      //只读
	FileSuffix string        //只读文件名后缀 默认的文件名是 {shortDate}-suffix
	Serializer Serializer    //只读序列化
	Interval   time.Duration //只读 写入的时间间隔

	Slice           int //当前写入的文件序号 默认为0
	LogFileName     string
	FullLogFileName string
	CurrLogSize     int64

	Locker        *sync.Mutex
	CurrLogBuff   LogBufferID  //protected by locker
	LogBufA       bytes.Buffer //protected by locker
	LogBufB       bytes.Buffer //protected by locker
	CurrCacheSize int          //protected by locker 当前buffer中的大小

	NextWriteTime time.Time
	LastPCDate    string
}

//Match 是否可以写入Target
func (ft FileTarget) Match(l LogLevel, name string) bool {
	return l >= ft.MinLevel && l <= ft.MaxLevel && (ft.Name == "" || ft.Name == name)
}

//LogConfig 文件配置
type LogConfig struct {
	SingleFileSize int64         //只读
	LogCacheSize   int           //只读
	Root           string        //只读
	Targets        []*FileTarget //只读
}

//Descriptor 指定的文件Target
type Descriptor struct {
	Name       string
	MinLevel   string
	MaxLevel   string
	FileSuffix string
	Serializer string
	Interval   int
}

//FileContent 配置文件对应的结构
type FileContent struct {
	SingleFileSize int64
	LogCacheSize   int
	Root           string
	Targets        []Descriptor
}

//ConfigFile 文件配置管理器
type ConfigFile struct {
	stop    chan bool
	path    string
	modTime time.Time
}

func newConfigFile() *ConfigFile {
	return &ConfigFile{
		stop: make(chan bool), //无缓冲信号
	}
}

//Load 载入配置
func (file *ConfigFile) Load(path string) (*LogConfig, error) {
	if path == "" {
		return nil, errors.New("empty path")
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("path is dir:%s", path)
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ct FileContent
	err = json.Unmarshal(content, ct)
	if err != nil {
		return nil, err
	}
	cf, err := convert(ct)
	if err != nil {
		return nil, err
	}
	fc.path = path
	fc.modTime = stat.ModTime()
	return cf, nil
}

//StartMonitor 监控文件变化
func (file *ConfigFile) StartMonitor(delegate func(config *LogConfig)) {
	if file.path == "" {
		return
	}
	go func() {
	loop:
		for {
			select {
			case <-file.stop:
				break loop
			case <-time.After(10 * time.Second):
				stat, err := os.Stat(file.path)
				//必须是文件
				if err != nil || stat.IsDir() {
					fmt.Println("StartMonitor 0:", file.path, ":", err)
					continue loop
				}
				//文件修改时间不等则准备更新config
				if stat.ModTime().Before(file.modTime) || stat.ModTime().After(file.modTime) {
					file.modTime = stat.ModTime()
					config, err := file.Load(file.path)
					if err != nil {
						fmt.Println("StartMonitor 1 load fail:", file.path, ":", err)
						continue loop
					}
					file.invoke(delegate, config)
				}

			}

		}
	}()
}

//StopMonitor 停止监控文件变化
func (file *ConfigFile) StopMonitor() {
	file.stop <- true
}

func (file *ConfigFile) invoke(delegate func(config *LogConfig), config *LogConfig) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("invoke 0:", err)
		}
	}()
	delegate(config)
}

func convert(fc FileContent) (*LogConfig, error) {
	//设置默认值
	config := &LogConfig{
		SingleFileSize: 1024 * 1024 * 10,
		LogCacheSize:   1024 * 10,
		Root:           "./logs",
		Targets:        nil,
	}
	if fc.SingleFileSize != 0 {
		config.SingleFileSize = fc.SingleFileSize
	}
	if fc.LogCacheSize != 0 {
		config.LogCacheSize = fc.LogCacheSize
	}
	if fc.Root != "" {
		config.Root = fc.Root
	}
	//检测路径是否可用
	err := os.MkdirAll(config.Root, os.ModePerm)
	if err != nil {
		return nil, err
	}
	//根据配置文件生成target
	for _, v := range fc.Targets {
		tmp := &FileTarget{}
		if v.MaxLevel == "*" || v.MaxLevel == "" {
			tmp.MaxLevel = CriticalLevel
		} else {
			tmp.MaxLevel = toLevel(v.MaxLevel)
		}
		if v.MinLevel == "*" || v.MinLevel == "" {
			tmp.MinLevel = TraceLevel
		} else {
			tmp.MinLevel = toLevel(v.MinLevel)
		}
		if v.Name == "*" || v.Name == "" {
			tmp.Name = ""
		} else {
			tmp.Name = v.Name
		}
		if v.FileSuffix == "" {
			tmp.FileSuffix = ".log"
		} else {
			tmp.FileSuffix = v.FileSuffix
		}
		if v.Interval <= 0 {
			tmp.Interval = time.Duration(time.Second)
		} else {
			tmp.Interval = time.Duration(v.Interval) * time.Second
		}
		tmp.Serializer = FindSerializer(v.Serializer)

		tmp.Locker = &sync.Mutex{}
		tmp.CurrLogBuff = LogBufferIDA
		config.Targets = append(config.Targets, tmp)
	}
	return config, nil
}

func toLevel(l string) LogLevel {
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
	return CriticalLevel
}
