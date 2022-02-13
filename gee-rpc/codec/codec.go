package codec

import "io"

type Header struct {
	ServiceMethod string // 调用服务方法的格式为："Service.Method"
	Seq           uint64 // 由客户端选择相应的序列号
	Error         string
}

// Codec 定义编码的工厂接口
// 因为要支持多种编解码方式，所以这抽象出一个Codec接口
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// NewCodecFunc 定义工厂方法返回的内容，返回的不是一个示例，而是一个构造函数
type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string // 为了便于代码的阅读，在一些带有特定含义的类型命名别名

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
