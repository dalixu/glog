# logger
<p>日志库 实现了 文件按天按大小分割 配置文件热更新 可同时写入多个文件方便接入其他日志系统</p>
配置文件使用json
{
    "Layouts":
    [
        {
            "Serializer":{"Type":"json"},
            "Target":{"Type":"file"}
        }
    ]
}
使用Layouts数组来支持多个文件的输出 
配合file Target字段的MinLevel 和MaxLevel可以把 不同级别的日志输出到不同的文件
file Target支持的字段参照target.go createFileTarget

初始化日志库 Initialize
然后通过GetManager->GetLogger来获取logger 就可以打印日志

Serializer 目前支持plain 和json
自定义Serializer
1.实现Serializer
2.RegisterSerializer(key, Serializer)
3.在配置文件中Serializer的Type字段中指定同样的key
4.初始化日志库 Initialize

Target 目前支持file
fileTarget 使用异步写入日志 Async字段为true时 异步序列化 否则同步序列化
自定义Target
1.实现TargetCtor
2.RegisterTarget(key, TargetCtor)
3.在配置文件中Target的Type字段指定同样的key
4.初始化日志库 Initialize


