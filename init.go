package glog

import (
	"log"

	"github.com/dalixu/glogger"
)

//GLoggerFactory 实现gloggerFactory
type GLoggerFactory struct {
	manager Manager
}

//GetLogger implement GLogger
func (gf *GLoggerFactory) GetLogger(name string) glogger.GLogger {
	return gf.manager.GetLogger(name)
}

//NewGLoggerFactory 返回1个glogger.Factory
func NewGLoggerFactory(path string) glogger.Factory {
	manager := New(path)
	return &GLoggerFactory{
		manager: manager,
	}
}

//全局配置
func init() {
	globalSerializer = make(map[string]Serializer)
	globalSerializer["plain"] = &DefaultSerializer{}
	globalSerializer["json"] = &JSONSerializer{}

	globalTarget = make(map[string]TargetCtor)
	globalTarget["file"] = createFileTarget
	globalTarget["console"] = createConsoleTarget
}

var globalSerializer map[string]Serializer

//TargetCtor 实现自定义Target
type TargetCtor func(config map[string]interface{}) Target

var globalTarget map[string]TargetCtor

//RegisterSerializer 添加一个序列化 在配置文件里指定相同的name 则可以调用这个序列化
func RegisterSerializer(name string, serial Serializer) {
	globalSerializer[name] = serial
}

//RegisterTarget 添加一个Target
func RegisterTarget(name string, ctor TargetCtor) {
	globalTarget[name] = ctor
}

func findSerializer(name string) Serializer {
	var seria = globalSerializer[name]
	if seria == nil {
		seria = globalSerializer["plain"]
	}
	return seria
}

func findTarget(name string, config map[string]interface{}) Target {
	var target = globalTarget[name]
	if target == nil {
		target = globalTarget["file"]
	}
	return target(config)
}

//New 返回1个Manager对象 通常1个程序1个manager就可以了
func New(path string) Manager {
	file := newConfigFile()
	config, err := file.Load(path)
	if err != nil {
		log.Println(err)
		return nil
	}
	return newManager(config, file)
}
