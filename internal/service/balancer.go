package service

import (
	"strconv"
	"strings"
	"sync"

	"Xin-api/internal/model"
)

// ChannelBalancer 负载均衡接口
type ChannelBalancer interface {
	Select(key string, channels []model.Channel) *model.Channel
}

type roundRobinState struct {
	index       int
	fingerprint string // 新增：当前渠道列表的特征值
	mu          sync.Mutex
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

// computeFingerprint 根据渠道列表生成唯一指纹
// 当渠道的 ID、权重或顺序发生变化时，指纹也会变化
func computeFingerprint(channels []model.Channel) string {
	var sb strings.Builder
	for _, ch := range channels {
		sb.WriteString(strconv.FormatInt(ch.ID, 10))
		sb.WriteByte(':')
		sb.WriteString(strconv.Itoa(ch.Weight))
		sb.WriteByte(',')
	}
	return sb.String()
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

	// 计算当前渠道列表的指纹（包含权重和顺序）
	fp := computeFingerprint(channels)

	// 获取或重置状态（根据指纹是否变化）
	b.mu.Lock()
	state, ok := b.states[key]
	if !ok || state.fingerprint != fp {
		// 列表发生变化：新建状态，索引归零
		state = &roundRobinState{
			index:       0,
			fingerprint: fp,
		}
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
