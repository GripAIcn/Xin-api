package routing

import (
	"context"
	"errors"

	"Xin-api/internal/model"
	"Xin-api/internal/service"
)

// 选路策略枚举，对应 request_logs.route_strategy
const (
	StrategyWeighted         = "weighted"          // 仅加权轮询
	StrategyPriorityWeighted = "priority_weighted" // 优先级 + 同级加权轮询
)

// 路由错误
var (
	ErrNoChannel = errors.New("no healthy channel available")
)

// RouteRequest 一次选路的输入快照
type RouteRequest struct {
	Model             string          // 对外模型名（同时作为加权轮询的维度 key）
	Protocol          string          // 请求协议 chat/responses/gemini/messages
	Channels          []model.Channel // 候选渠道（通常来自缓存）
	ExcludeChannelIDs []int64         // 需要排除的渠道（重试时已失败的渠道）
}

// RouteResult 一次选路的结果快照
type RouteResult struct {
	Channel    *model.Channel // 命中渠道
	Strategy   string         // 选路策略
	Priority   int            // 命中渠道优先级快照
	Weight     int            // 命中渠道权重快照
	Candidates int            // 参与本次选路的健康候选数
}

// Router 选路接口；后续可替换为按标签/成本/延迟等策略的实现
type Router interface {
	Route(ctx context.Context, req RouteRequest) (*RouteResult, error)
}

// WeightedRouter 优先级 + 同级加权轮询路由器
// 1. 过滤排除渠道、权重非正与不健康渠道
// 2. 取优先级最大的一组渠道
// 3. 在同优先级内按权重加权轮询
type WeightedRouter struct {
	balancer service.ChannelBalancer
	breaker  service.ChannelBreaker
}

// NewWeightedRouter 创建加权路由器
func NewWeightedRouter(balancer service.ChannelBalancer, breaker service.ChannelBreaker) *WeightedRouter {
	return &WeightedRouter{balancer: balancer, breaker: breaker}
}

// Route 执行选路；无可用渠道时返回 ErrNoChannel
func (r *WeightedRouter) Route(ctx context.Context, req RouteRequest) (*RouteResult, error) {
	if len(req.Channels) == 0 {
		return nil, ErrNoChannel
	}

	excluded := make(map[int64]struct{}, len(req.ExcludeChannelIDs))
	for _, id := range req.ExcludeChannelIDs {
		excluded[id] = struct{}{}
	}

	// 1. 过滤：排除集合 / 权重非正 / 熔断不健康
	healthy := make([]model.Channel, 0, len(req.Channels))
	maxPriority := 0
	hasPriority := false
	for i := range req.Channels {
		ch := req.Channels[i]
		if _, skip := excluded[ch.ID]; skip {
			continue
		}
		if ch.Weight <= 0 {
			continue
		}
		if !supportsProtocol(ch, req.Protocol) {
			continue
		}
		if r.breaker != nil {
			ok, _ := r.breaker.IsHealthy(ctx, ch.ID)
			if !ok {
				continue
			}
		}
		healthy = append(healthy, ch)
		if !hasPriority || ch.Priority > maxPriority {
			maxPriority = ch.Priority
			hasPriority = true
		}
	}
	if len(healthy) == 0 {
		return nil, ErrNoChannel
	}

	// 2. 取最高优先级子集（priority 数值越大越优先）
	top := make([]model.Channel, 0, len(healthy))
	for i := range healthy {
		if healthy[i].Priority == maxPriority {
			top = append(top, healthy[i])
		}
	}

	// 3. 同优先级内加权轮询
	var selected *model.Channel
	if r.balancer != nil {
		selected = r.balancer.Select(req.Model, top)
	}
	if selected == nil {
		// 兜底：未注入 balancer 或权重全部异常时取首个候选
		selected = &top[0]
	}

	strategy := StrategyWeighted
	if maxPriority > 0 || len(healthy) != len(top) {
		strategy = StrategyPriorityWeighted
	}

	return &RouteResult{
		Channel:    selected,
		Strategy:   strategy,
		Priority:   selected.Priority,
		Weight:     selected.Weight,
		Candidates: len(healthy),
	}, nil
}

// supportsProtocol 判断渠道是否支持指定协议；protocol 为空或渠道未登记协议视为「全协议支持」
// responses / gemini / messages 入站都会归一化为 OpenAI Chat 上游请求，
// 因此声明 chat 的渠道同样支持这些协议
func supportsProtocol(ch model.Channel, protocol string) bool {
	if protocol == "" || len(ch.Protocol) == 0 {
		return true
	}
	for _, p := range ch.Protocol {
		if p == protocol {
			return true
		}
	}
	if protocol == "responses" || protocol == "messages" || protocol == "gemini" {
		for _, p := range ch.Protocol {
			if p == "chat" {
				return true
			}
		}
	}
	return false
}
