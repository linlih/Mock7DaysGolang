package geerpc

import (
	"encoding/json"
	"errors"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
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
	serviceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined" + s.name)
	}
	return nil
}

// 代码逻辑传入的为"Service.Method"，根据Service找到service实例，在根据Method找到具体的方法
func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed:" + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}

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
		go server.handleRequest(cc, req, sending, wg) // 并行处理多个请求
	}
	wg.Wait()
	_ = cc.Close()
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value // 数据可以支持多种类型，所以这里使用反射
	mtype        *methodType
	svc          *service
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
	/*
		// day 1 和 day2 的代码
		req := &request{h: h}
		req.argv = reflect.New(reflect.TypeOf("")) // 这里暂时创建一个字符串的Value
		if err = cc.ReadBody(req.argv.Interface()); err != nil {
			log.Println("rpc server: read argv err:", err)
		}
		return req, nil
	*/
	// day 3
	req := &request{h: h}
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// 因为ReadBody需要是指针类型的参数，所以需要保证argvi是指针类型
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
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

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	/*
		// day 1 and day 2
		defer wg.Done()
		log.Println(req.h, req.argv.Elem())
		req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
		server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
	*/
	defer wg.Done()
	err := req.svc.call(req.mtype, req.argv, req.replyv)
	if err != nil {
		req.h.Error = err.Error()
		server.sendResponse(cc, req.h, invalidRequest, sending)
	}
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

func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }
