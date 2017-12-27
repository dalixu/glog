package logger

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
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
	FatalLevel
)

//Layout 用于内部描述
type Layout struct {
	Target     Target
	Serializer Serializer
}

//LogConfig 文件配置
type LogConfig struct {
	Layouts []*Layout //只读
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
func (file *ConfigFile) Load(path string) (lc *LogConfig, e error) {
	defer func() {
		if err := recover(); err != nil {
			lc = nil
			e = log.Errorf("%+v", err)
		}
	}()
	if path == "" {
		return nil, errors.New("empty path")
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, log.Errorf("path is dir:%s", path)
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ct map[string]interface{}
	err = json.Unmarshal(content, &ct)
	if err != nil {
		return nil, err
	}
	cf, err := convert(ct)
	if err != nil {
		return nil, err
	}
	file.path = path
	file.modTime = stat.ModTime()
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
					log.Println("StartMonitor 0:", file.path, ":", err)
					continue loop
				}
				//文件修改时间不等则准备更新config
				if stat.ModTime().Before(file.modTime) || stat.ModTime().After(file.modTime) {
					file.modTime = stat.ModTime()
					config, err := file.Load(file.path)
					if err != nil {
						log.Println("StartMonitor 1 load fail:", file.path, ":", err)
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
			log.Println("invoke 0:", err)
		}
	}()
	delegate(config)
}

func convert(content map[string]interface{}) (*LogConfig, error) {

	if _, ok := content["Layouts"]; !ok {
		return nil, errors.New("Layouts missed")
	}
	layouts := content["Layouts"].([]interface{})
	//设置默认值
	config := &LogConfig{
		Layouts: nil,
	}
	for _, v := range layouts {
		tmp := v.(map[string]interface{})
		layout := &Layout{}
		se := tmp["Serializer"].(map[string]interface{})
		seType := se["Type"].(string)
		layout.Serializer = findSerializer(seType)
		tt := tmp["Target"].(map[string]interface{})
		ttType := tt["Type"].(string)
		layout.Target = findTarget(ttType, tt)
		if layout.Serializer != nil && layout.Target != nil {
			config.Layouts = append(config.Layouts, layout)
		}
	}

	return config, nil
}
