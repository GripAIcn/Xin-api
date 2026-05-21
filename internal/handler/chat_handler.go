package handler

import (
	"fmt"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"

	"Xin-api/internal/adapter"
	"Xin-api/internal/model"
	"Xin-api/internal/service"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// ChatHandler 数据面聊天补全处理器
type ChatHandler struct {
	channelRepo  postgresql.ChannelRepo
	balancer     service.ChannelBalancer
	breaker      service.ChannelBreaker
	proxy        *service.StreamProxy
	adapterGroup *adapter.Registry
}

// NewChatHandler 创建聊天补全处理器
func NewChatHandler(
	channelRepo postgresql.ChannelRepo,
	balancer service.ChannelBalancer,
	breaker service.ChannelBreaker,
	streamProxy *service.StreamProxy,
	adapterGroup *adapter.Registry,
) *ChatHandler {
	return &ChatHandler{
		channelRepo:  channelRepo,
		balancer:     balancer,
		breaker:      breaker,
		proxy:        streamProxy,
		adapterGroup: adapterGroup,
	}
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

	// 4. 查询可用渠道
	channels, err := h.channelRepo.ListActiveByGroupAndModel(c.Request.Context(), gid, modelName)
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
