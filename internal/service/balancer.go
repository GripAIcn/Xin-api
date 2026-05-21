package service

import (
	"sync"

	"Xin-api/internal/model"
)

// ChannelBalancer 负载均衡接口
type ChannelBalancer interface {
	Select(key string, channels []model.Channel) *model.Channel
}

type roundRobinState struct {
	index int
	mu    sync.Mutex
}

// WeightedRRBalancer 加权轮询负载均衡器
type WeightedRRBalancer struct {
	states map[string]*roundRobinState
	mu     sync.Mutex
}

// NewWeightedRRBalancer 创建加权轮询负载均衡器
func NewWeightedRRBalancer() *WeightedRRBalancer {
	return &WeightedRRBalancer{
		states: make(map[string]*roundRobinState),
	}
}

// Select 按 "groupID:model" 维度加权轮询选择渠道
// 返回 nil 表示无可选渠道
func (b *WeightedRRBalancer) Select(key string, channels []model.Channel) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	// 构建加权虚拟节点列表，跳过权重 <= 0 的渠道
	type weighted struct {
		channel *model.Channel
		weight  int
	}
	var pool []weighted
	totalWeight := 0
	for i := range channels {
		if channels[i].Weight <= 0 {
			continue
		}
		pool = append(pool, weighted{channel: &channels[i], weight: channels[i].Weight})
		totalWeight += channels[i].Weight
	}
	if len(pool) == 0 {
		return nil
	}

	// 获取/创建该 key 对应的轮询状态
	b.mu.Lock()
	state, ok := b.states[key]
	if !ok {
		state = &roundRobinState{}
		b.states[key] = state
	}
	b.mu.Unlock()

	// 原子步进游标
	state.mu.Lock()
	state.index = (state.index + 1) % totalWeight
	idx := state.index
	state.mu.Unlock()

	// 按权重区间选择
	cumulative := 0
	for _, w := range pool {
		cumulative += w.weight
		if idx < cumulative {
			return w.channel
		}
	}

	// fallback
	return pool[len(pool)-1].channel
}
