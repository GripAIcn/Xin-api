package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"Xin-api/internal/adapter"
	"Xin-api/internal/model"
	"Xin-api/internal/service"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// ChatHandler 数据面聊天补全处理器
type ChatHandler struct {
	channelRepo  postgresql.ChannelRepo
	redis        *redis.Client
	cacheExpire  time.Duration
	balancer     service.ChannelBalancer
	breaker      service.ChannelBreaker
	proxy        *service.StreamProxy
	adapterGroup *adapter.Registry
}

// NewChatHandler 创建聊天补全处理器
func NewChatHandler(
	channelRepo postgresql.ChannelRepo,
	rdb *redis.Client,
	cacheExpire time.Duration,
	balancer service.ChannelBalancer,
	breaker service.ChannelBreaker,
	streamProxy *service.StreamProxy,
	adapterGroup *adapter.Registry,
) *ChatHandler {
	return &ChatHandler{
		channelRepo:  channelRepo,
		redis:        rdb,
		cacheExpire:  cacheExpire,
		balancer:     balancer,
		breaker:      breaker,
		proxy:        streamProxy,
		adapterGroup: adapterGroup,
	}
}

// getChannelsWithCache 优先从缓存获取渠道，未命中则查数据库
func (h *ChatHandler) getChannelsWithCache(ctx context.Context, groupID int64, modelName string) ([]model.Channel, error) {
	cacheKey := fmt.Sprintf("xin:channels:active:%d:%s", groupID, modelName)

	// 1. 尝试从缓存获取
	if cached, err := h.redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var channels []model.Channel
		if jsonErr := json.Unmarshal(cached, &channels); jsonErr == nil {
			return channels, nil
		}
	}

	// 2. 缓存未命中，查询数据库
	channels, err := h.channelRepo.ListActiveByGroupAndModel(ctx, groupID, modelName)
	if err != nil {
		return nil, err
	}

	// 3. 写入缓存（不阻塞主流程）
	if data, marshalErr := json.Marshal(channels); marshalErr == nil {
		h.redis.Set(ctx, cacheKey, data, h.cacheExpire)
	}

	return channels, nil
}

// HandleDataPlane 通用数据面请求处理
// 根据请求路径查找对应 adapter，委托 adapter 完成协议解析和转发
func (h *ChatHandler) HandleDataPlane(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		response.DataPlaneError(c, response.InvalidParams)
		return
	}

	// 1. 根据请求路径查找适配器
	adpt, err := h.adapterGroup.GetAdapter(c.FullPath())
	if err != nil {
		response.DataPlaneError(c, response.URLNotFound)
		return
	}

	// 2. 提取模型名
	modelName, err := adpt.ExtractModel(body)
	if err != nil {
		response.DataPlaneError(c, response.InvalidParams)
		return
	}

	// 3. 获取 group_id
	groupID, exists := c.Get("group_id")
	if !exists {
		response.DataPlaneError(c, response.Unauthorized)
		return
	}
	gid := groupID.(int64)

	// 4. 查询可用渠道（使用缓存）
	channels, err := h.getChannelsWithCache(c.Request.Context(), gid, modelName)
	if err != nil {
		logger.Error(c.Request.Context(), "list channels failed: %v", err)
		response.DataPlaneError(c, response.InternalError)
		return
	}
	if len(channels) == 0 {
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}

	// 5. 负载均衡 + 熔断检查
	balancerKey := fmt.Sprintf("%d:%s", gid, modelName)
	var selected *model.Channel
	for attempts := 0; attempts < len(channels); attempts++ {
		candidate := h.balancer.Select(balancerKey, channels)
		if candidate == nil {
			break
		}
		healthy, _ := h.breaker.IsHealthy(c.Request.Context(), candidate.ID)
		if healthy {
			selected = candidate
			break
		}
		for i, ch := range channels {
			if ch.ID == candidate.ID {
				channels = append(channels[:i], channels[i+1:]...)
				break
			}
		}
	}
	if selected == nil {
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}

	// 6. 代理转发（使用 adapter 提供的端点）
	logger.Info(c.Request.Context(), "routing to channel %s (id=%d) via %s adapter",
		selected.Name, selected.ID, adpt.Name())
	proxyErr := h.proxy.Forward(c, selected, body, adpt.UpstreamEndpoint())

	// 7. 记录熔断结果
	h.breaker.Record(c.Request.Context(), selected.ID, proxyErr)
	if proxyErr != nil {
		logger.Error(c.Request.Context(), "service to channel %s failed: %v", selected.Name, proxyErr)
	}
}
