package logger

//全局配置
func init() {
	globalSerializer = make(map[string]Serializer)
	globalSerializer["default"] = &DefaultSerializer{}
	globalSerializer["json"] = &JSONSerializer{}

	globalTarget = make(map[string]TargetCtor)
	globalTarget["file"] = createFileTarget
}

var globalSerializer map[string]Serializer

//TargetCtor 实现自定义Target
type TargetCtor func(config map[string]interface{}) Target

var globalTarget map[string]TargetCtor

//RegisterSerializer 添加一个序列化 在配置文件里指定相同的name 则可以调用这个序列化
//必须在Initialize 之前调用(不保证也不需要线程安全)
func RegisterSerializer(name string, serial Serializer) {
	globalSerializer[name] = serial
}

//RegisterTarget 添加一个Target
//必须在Initialize 之前调用(不保证也不需要线程安全)
func RegisterTarget(name string, ctor TargetCtor) {
	globalTarget[name] = ctor
}

func findSerializer(name string) Serializer {
	var seria = globalSerializer[name]
	if seria == nil {
		seria = globalSerializer["default"]
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

var mr Manager
var fc *ConfigFile

//GetManager 获得 manager
func GetManager() Manager {
	return mr
}

//Initialize log库初始化
func Initialize(path string) error {
	file := newConfigFile()
	config, err := file.Load(path)
	if err != nil {
		return err
	}
	mr = newManager(config)
	fc = file
	fc.StartMonitor(mr.Reload)
	return nil
}

//Finalize 安全退出
func Finalize() {
	fc.StopMonitor()
	mr.Stop()
	fc = nil
	mr = nil
}
