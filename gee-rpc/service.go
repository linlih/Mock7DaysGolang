package geerpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// RPC框架的基础能力是：能像调用本地程序一样调用远程服务
// 在 Go 的实现上就是要把结构体方法映射为服务
// 对于 net/rpc 而言，一个函数需要能够被远程调用的5个条件(参考go源码net/rpc/server.go)：
// 1. 方法所属类型是导出的（也就是方法类型是首字母大写的）
// 2. 方法是导出的
// 3. 两个入参，均为导出或者内置类型
// 4. 第二个参数必须是一个指针
// 5. 返回值为 error 类型

// 客户端发送过来的 serviceMethod 比如为：Foo.Sum ,表示的是调用类型Foo的Sum方法
// 如果用 switch serviceMethod case ... 的方式，代码量较大，并且繁琐不灵活

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type // 第一个参数的类型
	ReplyType reflect.Type // 第二个参数的类型
	numCalls  uint64
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

// 根据参数类型创建 Value，其中指针和普通变量的创建不同
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	if m.ArgType.Kind() == reflect.Ptr {
		// 如果是指针类型，需要从指针指向的内存中取出具体的内容
		// 注意这里reflect.New返回的Ptr指针，刚好ArgType的类型是匹配的，所以这里不需要再次调用Elem()函数了
		argv = reflect.New(m.ArgType.Elem())
	} else {
		// 使用reflect.New(m.ArgType)创建的Value是一个Ptr
		// 因为ArgType是非指针类型，所以argv的值也是非指针类型的
		// 所以为了类型匹配，这里需要返回非指针类型，也就是reflect.New创建的指针指向的元素
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	// replyv 必须是一个指针，所以需要使用 Elem 函数取出
	replyv := reflect.New(m.ReplyType.Elem())
	// 对 Map 和 Slice 类型进行特殊处理
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type service struct {
	name   string                 // 映射的结构体名称，比如 WaitGroup
	typ    reflect.Type           // 结构体的类型
	rcvr   reflect.Value          // 结构体实例本身
	method map[string]*methodType // 存储映射结构体的所有符合上诉条件的方法
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	// Indirect 的目的就是做一个 Ptr 的转换，如果是 Ptr，则需要使用Elem()，如果不是 Ptr，那直接返回它本身就可以
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) { // 利于语法树的函数判断结构体是否可导出
		log.Fatalf("rpc server: %s is not valid service name", s.name)
	}
	s.registerMethods()
	return s
}

func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	// 遍历结构体的所有方法，注册method
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		// 这里注册的方法，限定的输入参数为3个，返回参数为1个
		// 输入参数三个，第一个是自身，第二个是输入参数，第三个是输出参数
		// 返回参数为 error
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		// 输入参数和输出参数都必须是可导出的
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: reigster %s.%s\n", s.name, method.Name)
	}
}

func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv}) // 调用执行注册的函数
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	// 内置类型的 PkgPath=""，如果是main函数同一个pkg下，PkgPath也为空
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}
