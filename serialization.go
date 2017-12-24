package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
)

//Serializer 序列化接口
type Serializer interface {
	Encode(e *LogEvent) []byte
}

//DefaultSerializer 默认的序列化接口
type DefaultSerializer struct {
}

//Encode 实现Serialization
func (ds *DefaultSerializer) Encode(e *LogEvent) []byte {
	var buf bytes.Buffer
	buf.WriteByte('[')
	buf.WriteString(e.LevelDesc)
	buf.WriteByte(']')
	buf.WriteString("@Time:")
	buf.WriteString(e.Time)

	buf.WriteString("@Name:")
	buf.WriteString(e.Name)
	buf.WriteString("@Message:")
	if e.Format != "" {
		buf.WriteString(fmt.Sprintf(e.Format, e.Args...))
	} else {
		buf.WriteString(fmt.Sprint(e.Args...))
	}
	if e.StackTrace != "" {
		buf.WriteString("@StackTrace:")
		buf.WriteString(e.StackTrace)
	}
	for k, v := range e.Properties {
		buf.WriteString(fmt.Sprintf("@%s:", k))
		buf.WriteString(fmt.Sprint(v))
	}

	return buf.Bytes()
}

//JSONSerializer json序列化接口
type JSONSerializer struct {
}

//Encode 实现Serialization
func (js *JSONSerializer) Encode(e *LogEvent) []byte {
	if e.Properties == nil {
		e.Properties = make(map[string]interface{})
	}
	e.Properties["Level"] = e.LevelDesc
	e.Properties["Name"] = e.Name
	if e.Format != "" {
		e.Properties["Message"] = fmt.Sprintf(e.Format, e.Args...)
	} else {
		e.Properties["Message"] = fmt.Sprint(e.Args...)
	}
	if e.StackTrace != "" {
		e.Properties["StackTrace"] = e.StackTrace
	}
	e.Properties["Time"] = e.Time
	bs, err := json.Marshal(e.Properties)
	if err != nil {
		fmt.Println("JSONSerialization:", err)
		return nil
	}
	return bs
}
