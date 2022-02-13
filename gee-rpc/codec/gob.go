package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// 具体实现一个编码对象，在Codec.go中相当于建立一个编码组件的抽象，然后可以具体实现为JSON编解码，或者是这里的Gob编解码
// 所以这里的GobCodec就需要实现Codec.go中定义的编解码组件的所有接口

type GobCodec struct {
	conn io.ReadWriteCloser // conn 支持io的Read、Write、Close三个操作
	buf  *bufio.Writer      // 防止阻塞创建一个带缓冲的 buf, 一般这么做可以提升性能
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn) // 初始化的时候传入 conn
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(conn),
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush() // 将缓冲区写入io中
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc: gob error encoding header:", err)
		return err
	}
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc: gob error encoding body:", err)
		return err
	}
	return
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}
