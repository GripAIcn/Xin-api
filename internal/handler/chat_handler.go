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
func (h *ChatHandler) getChannelsWithCache(ctx context.Context, modelName string) ([]model.Channel, error) {
	cacheKey := fmt.Sprintf("xin:channels:active:%s", modelName)

	// 1. 尝试从缓存获取
	if cached, err := h.redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var channels []model.Channel
		if jsonErr := json.Unmarshal(cached, &channels); jsonErr == nil {
			return channels, nil
		}
	}

	// 2. 缓存未命中，查询数据库
	channels, err := h.channelRepo.ListActiveByModel(ctx, modelName)
	if err != nil {
		return nil, err
	}

	// 3. 写入缓存（不阻塞主流程）
	if data, marshalErr := json.Marshal(channels); marshalErr == nil {
		h.redis.Set(ctx, cacheKey, data, h.cacheExpire)
	}

	return channels, nil
}

// getUpstreamModelWithCache 解析渠道登记的「对外名 → 上游名」映射（带缓存）
// 未登记映射时返回空串（表示透传对外名）
func (h *ChatHandler) getUpstreamModelWithCache(ctx context.Context, channelID int64, modelName string) string {
	cacheKey := fmt.Sprintf("xin:modelmap:%d:%s", channelID, modelName)

	if cached, err := h.redis.Get(ctx, cacheKey).Result(); err == nil {
		return cached
	}

	upstream, err := h.channelRepo.GetUpstreamModel(ctx, channelID, modelName)
	if err != nil {
		logger.Error(ctx, "get upstream model mapping failed: %v", err)
		return ""
	}

	h.redis.Set(ctx, cacheKey, upstream, h.cacheExpire)
	return upstream
}

// HandleDataPlane 通用数据面请求处理
// 入站协议固定为 OpenAI Chat；选中渠道后按渠道 provider 选择适配器完成协议转换与转发
func (h *ChatHandler) HandleDataPlane(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		response.DataPlaneError(c, response.InvalidParams)
		return
	}

	// 1. 入站协议固定为 OpenAI Chat，用 openai 适配器提取对外模型名
	inbound, err := h.adapterGroup.GetAdapter("openai")
	if err != nil {
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}
	modelName, err := inbound.ExtractModel(body)
	if err != nil {
		response.DataPlaneError(c, response.InvalidParams)
		return
	}

	// 2. 查询可用渠道（使用缓存）
	channels, err := h.getChannelsWithCache(c.Request.Context(), modelName)
	if err != nil {
		logger.Error(c.Request.Context(), "list channels failed: %v", err)
		response.DataPlaneError(c, response.InternalError)
		return
	}
	if len(channels) == 0 {
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}

	// 3. 负载均衡 + 熔断检查
	balancerKey := modelName
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

	// 4. 按渠道 provider 选择出站适配器
	adpt, err := h.adapterGroup.GetAdapter(selected.Provider)
	if err != nil {
		logger.Error(c.Request.Context(), "no adapter for provider %s: %v", selected.Provider, err)
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}

	// 5. 解析上游模型映射（命中则改写为上游名，未命中透传对外名）
	upstreamModel := h.getUpstreamModelWithCache(c.Request.Context(), selected.ID, modelName)

	// 6. 由适配器构造上游请求（含协议转换与鉴权头）
	upReq, err := adpt.BuildUpstreamRequest(body, upstreamModel, selected.APIKey)
	if err != nil {
		logger.Error(c.Request.Context(), "build upstream request failed: %v", err)
		response.DataPlaneError(c, response.InvalidParams)
		return
	}

	// 7. 转发并按需把响应转换回 OpenAI 格式（model 回写为对外名）
	logger.Info(c.Request.Context(), "routing to channel %s (id=%d) via %s adapter",
		selected.Name, selected.ID, adpt.Name())
	proxyErr := h.proxy.Forward(c, selected, upReq, adpt, modelName)

	// 8. 记录熔断结果
	h.breaker.Record(c.Request.Context(), selected.ID, proxyErr)
	if proxyErr != nil {
		logger.Error(c.Request.Context(), "service to channel %s failed: %v", selected.Name, proxyErr)
	}
}
