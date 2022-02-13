package geerpc

import (
	"encoding/json"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodecType   codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

// ServerConn 通信协议的格式：
// | Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
// | <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   -------> |
// 在一条连接中可能有多个请求，数据流的格式如下：
// | Option | Header1 | Body1 | Header2 | Body2 | ...
func (server *Server) ServerConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	// 在连接开始的时候协商通信协议信息
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error:", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x\n", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: not supporting codec type %s\n", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

var invalidRequest = struct {
}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // 针对的是一条连接
	wg := new(sync.WaitGroup)
	// 处理多个请求
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending) // 处理错误场景
		}
		wg.Add(1)
		go server.handleResponse(cc, req, sending, wg) // 并行处理多个请求
	}
	wg.Wait()
	_ = cc.Close()
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value // 数据可以支持多种类型，所以这里使用反射
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		// 如果不是文件末尾，表示读取错误
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	req.argv = reflect.New(reflect.TypeOf("")) // 这里暂时创建一个字符串的Value
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleResponse(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}

func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServerConn(conn)
	}
}

func Accept(lis net.Listener) { DefaultServer.Accept(lis) }
