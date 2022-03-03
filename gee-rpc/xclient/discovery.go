package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int

const (
	RandomSelect SelectMode = iota
	RoundRobinSelect
)

type Discovery interface {
	Refresh() error                      // 从注册中心更新服务列表
	Update(servers []string) error       // 手动更新服务列表
	Get(mode SelectMode) (string, error) // 根据负载均衡策略，选择一个服务实例
	GetAll() ([]string, error)           // 返回所有的服务实例
}

var _ Discovery = (*MultiServersDiscovery)(nil)

// MultiServersDiscovery 是一个多服务器的发现服务，它不需要注册中心
// 用户需要显式地提供服务地址列表
type MultiServersDiscovery struct {
	r       *rand.Rand // 是一个产生随机数的实例
	mu      sync.RWMutex
	servers []string
	index   int // 记录 Round Robin 算法已经轮询到的位置，为了避免每次从0开始，初始化的时候会随机设定一个值
}

func (d *MultiServersDiscovery) Refresh() error {
	return nil
}

func (d *MultiServersDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}

// Get 获取可用的服务器地址
// 这里实现的是相应的负载均衡策略
// 举例几种负载均衡策略
// 1. 随机选择策略，从服务列表中随机选择一个
// 2. 轮询算法（Round Robin），依次调度不同的服务器，每次调度执行i = (i+1) mod n
// 3. 加权轮询（Weight Round Robin），在轮询算法的基础上，为每个服务实例设置一个权重，高性能的机器赋予更高的权重，
//    也可以根据服务实例当前的负载情况做动态的调整，例如考虑最近5分钟部署服务器的CPU、内存消耗情况
// 4. 哈希/一致性策略，根据请求的某些特征，计算一个hash值，根据hash值将请求发送到对应的机器。
//    一致性hash还可以解决服务实例动态添加的情况下，调度抖动的问题，一致性哈希的一个典型应用场景是分布式缓存服务。
func (d *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.servers[d.index%n]
		d.index = (d.index + 1) % n
		return s, nil
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}

func (d *MultiServersDiscovery) GetAll() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	servers := make([]string, len(d.servers), len(d.servers))
	copy(servers, d.servers)
	return servers, nil
}

func NewMultiServersDiscovery(servers []string) *MultiServersDiscovery {
	d := &MultiServersDiscovery{
		servers: servers,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}
