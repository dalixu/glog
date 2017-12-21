package logger

//全局配置
func init() {
	globalSerializer = make(map[string]Serializer)
	globalSerializer["default"] = &DefaultSerializer{}
	globalSerializer["json"] = &JSONSerializer{}
}

var globalSerializer map[string]Serializer

//RegisterSerializer 添加一个序列化 在配置文件里指定相同的name 则可以调用这个序列化
//必须在Initialize 之前调用(不保证也不需要线程安全)
func RegisterSerializer(name string, serial Serializer) {
	globalSerializer[name] = serial
}

//FindSerializer 寻找一个Serializer
func FindSerializer(name string) Serializer {
	var seria = globalSerializer[name]
	if seria == nil {
		seria = globalSerializer["default"]
	}
	return seria
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
