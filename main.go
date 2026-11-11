package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

type Proxy struct {
	Addr    string
	dialer  proxy.Dialer
	Latency time.Duration
	Alive   bool
	mu      sync.Mutex
}

func NewProxy(addr string) *Proxy {
	d, _ := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
	return &Proxy{Addr: addr, dialer: d}
}

func (p *Proxy) Check(timeout time.Duration) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", p.Addr, timeout)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.Alive = false
		return
	}
	conn.Close()
	p.Alive = true
	p.Latency = time.Since(start)
}

type Pool struct {
	proxies []*Proxy
	idx     int
	mu      sync.Mutex
}

func NewPool(addrs []string) *Pool {
	pp := make([]*Proxy, 0, len(addrs))
	for _, a := range addrs {
		pp = append(pp, NewProxy(a))
	}
	return &Pool{proxies: pp}
}

func (pool *Pool) CheckAll(timeout time.Duration) {
	var wg sync.WaitGroup
	for _, p := range pool.proxies {
		wg.Add(1)
		go func(p *Proxy) {
			defer wg.Done()
			p.Check(timeout)
		}(p)
	}
	wg.Wait()
}

func (pool *Pool) Best() *Proxy {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	var best *Proxy
	for _, p := range pool.proxies {
		if !p.Alive {
			continue
		}
		if best == nil || p.Latency < best.Latency {
			best = p
		}
	}
	return best
}

func main() {
	pool := NewPool([]string{"127.0.0.1:1080"})
	pool.CheckAll(2 * time.Second)
	if b := pool.Best(); b != nil {
		fmt.Printf("лучший прокси: %s (%s)\n", b.Addr, b.Latency)
	} else {
		fmt.Println("нет доступных прокси")
	}
}
