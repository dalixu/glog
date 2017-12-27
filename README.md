# logger
<p>日志库 实现了 文件按天按大小分割 配置文件热更新 可同时写入多个文件方便接入其他日志系统</p>
<p>配置文件使用json
<pre>
{
    "Layouts":
    [
        {
            "Serializer":{"Type":"json"},
            "Target":{"Type":"file"}
        }
    ],
    "Async":false
}
</pre>
</p>
<p>使用Layouts数组来支持多个文件的输出<br/>
配合file Target字段的MinLevel 和MaxLevel可以把 不同级别的日志输出到不同的文件<br />
file Target支持的字段参照target.go createFileTarget<br />
</p>
<p>
通过NewManager->GetLogger来获取logger 就可以打印日志 通常1个程序只需要1个manager<br/>

Serializer 目前支持plain 和json<br/>
自定义Serializer<br/>
1.实现Serializer<br/>
2.RegisterSerializer(key, Serializer)<br/>
3.在配置文件中Serializer的Type字段中指定同样的key<br/>
4.NewManager<br/>

Target 目前支持file<br/>
fileTarget 使用异步写入日志 Async字段为true时 异步序列化 否则同步序列化<br/>
自定义Target<br/>
1.实现TargetCtor<br/>
2.RegisterTarget(key, TargetCtor)<br/>
3.在配置文件中Target的Type字段指定同样的key<br/>
4.NewManager<br/>
</p>
