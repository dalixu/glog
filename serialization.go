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
	buf.WriteString("@Name:")
	buf.WriteString(e.Name)
	buf.WriteString(" @Message:")
	if e.Format != "" {
		buf.WriteString(fmt.Sprintf(e.Format, e.Args...))
	} else {
		buf.WriteString(fmt.Sprint(e.Args...))
	}
	if e.StackTrace != "" {
		buf.WriteString(" @StackTrace:")
		buf.WriteString(e.StackTrace)
	}
	for k, v := range e.Properties {
		buf.WriteString(fmt.Sprintf(" @%s:", k))
		buf.WriteString(fmt.Sprint(v))
	}
	buf.WriteString(" @Time:")
	buf.WriteString(e.Time)
	return buf.Bytes()
}

//JSONSerializer json序列化接口
type JSONSerializer struct {
}

//Encode 实现Serialization
func (js *JSONSerializer) Encode(e *LogEvent) []byte {
	bs, err := json.Marshal(e)
	if err != nil {
		fmt.Println("JSONSerialization:", err)
		return nil
	}
	return bs
}
