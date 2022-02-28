package biz

import (
	"sync"
	"time"
)

// RouteTable 负责记录设备与其相关联的grpc连接channel以及自动注销其网关路由的计时器的对应关系
// 考虑到单个底层设备客户端通常会传输多台设备的信息，即往往多台同类设备的信息传输与一条grpc连接(一个协程)相关联，
// 利用这样的关系，可以采用并查集实现RouteTable，让在同一个协程传输信息的设备的所有路由信息相关联(利用tag,进行同时自动注销)
// 并使用同一个channel接收设备更新请求。
// todo 利用并查集以及sync包实现协程安全的RouteTable
type RouteTable struct {
	sync.Map
}
type RouteTableNode struct {
	//
	UpdateChannel chan interface{}
	// 自动向网关注销路由协程使用的计时器
	UnregisterTicker *time.Ticker
	// 与计时器相关联的路由tag信息，用于一次性注销若干相关联设备的route
	RouteTag string
	// 父节点
	Parent *RouteTableNode
}

// Find 查询指定key对应的节点的parent，返回查询到的节点和查询结果
// 当key对应的节点不存在时，返回nil,false
func (t *RouteTable) Find(key string) (*RouteTableNode, bool) {
	n, ok := t.Load(key)
	if !ok {
		return nil, false
	}
	node := n.(*RouteTableNode)
	for node.Parent != nil {
		node = node.Parent
	}
	return node, true
}

// Join 对两个给定节点执行并操作(将node2连接到node1所在node群中)
func (t *RouteTable) Join(node1, node2 *RouteTableNode) {
	for node1.Parent != nil {
		node1 = node1.Parent
	}
	for node2.Parent != nil {
		node2 = node2.Parent
	}
	node2.Parent = node1
}
