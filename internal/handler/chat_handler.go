package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"Xin-api/internal/adapter"
	"Xin-api/internal/middleware"
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
	logRepo      postgresql.LogRepo  // 请求日志 / 计费落库
	userRepo     postgresql.UserRepo // 读取用户余额
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
	logRepo postgresql.LogRepo,
	userRepo postgresql.UserRepo,
) *ChatHandler {
	return &ChatHandler{
		channelRepo:  channelRepo,
		redis:        rdb,
		cacheExpire:  cacheExpire,
		balancer:     balancer,
		breaker:      breaker,
		proxy:        streamProxy,
		adapterGroup: adapterGroup,
		logRepo:      logRepo,
		userRepo:     userRepo,
	}
}

// --- 指针辅助函数（避免大量取址样板） ---

func strPtr(s string) *string { return &s }
func i64Ptr(v int64) *int64   { return &v }
func i32Ptr(v int) *int       { return &v }

// saveLog 持久化一条请求日志；写库失败仅记录日志，不阻断主流程
func (h *ChatHandler) saveLog(ctx context.Context, log *model.RequestLog) {
	if h.logRepo == nil {
		return
	}
	if err := h.logRepo.Create(ctx, log); err != nil {
		logger.Error(ctx, "write request log failed: %v", err)
	}
}

// writeErrorLog 组装并写入一条错误请求日志（本地错误分支复用）
func (h *ChatHandler) writeErrorLog(
	ctx context.Context,
	requestID, modelName string,
	userID, apiKeyID int64,
	clientIP string,
	isStream bool,
	startedAt time.Time,
	logType, stage, errType, errMsg string,
	statusCode *int,
	channelID *int64,
	provider *string,
) {
	now := time.Now()
	log := &model.RequestLog{
		RequestID:    requestID,
		UserID:       i64Ptr(userID),
		APIKeyID:     i64Ptr(apiKeyID),
		Model:        modelName,
		LogType:      logType,
		Status:       model.LogStatusError,
		StatusCode:   statusCode,
		IsStream:     isStream,
		UsageSource:  model.UsageSourceUpstream,
		ErrorStage:   strPtr(stage),
		ErrorType:    strPtr(errType),
		ErrorMessage: strPtr(errMsg),
		ChannelID:    channelID,
		Provider:     provider,
		StartedAt:    &startedAt,
		CompletedAt:  &now,
	}
	// client_ip 为 inet 类型，空串会报错，故非空才写入
	if clientIP != "" {
		log.ClientIP = &clientIP
	}
	h.saveLog(ctx, log)
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
	ctx := c.Request.Context()
	requestID := uuid.NewString()
	startedAt := time.Now()

	// 0. 读取鉴权上下文（由 ApiKeyAuth 注入，类型均为 int64）
	uidVal, uidOK := c.Get(middleware.CtxUserIDKey)
	kidVal, kidOK := c.Get(middleware.CtxAPIKeyIDKey)
	userID, uidCast := uidVal.(int64)
	apiKeyID, kidCast := kidVal.(int64)
	if !uidOK || !kidOK || !uidCast || !kidCast {
		response.DataPlaneError(c, response.Unauthorized)
		return
	}

	// 客户端 IP（为空则不写入，避免 inet 类型写入空串报错）
	clientIP := c.ClientIP()

	body, err := c.GetRawData()
	if err != nil {
		h.writeErrorLog(ctx, requestID, "", userID, apiKeyID, clientIP, false, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "invalid_request", "read request body failed", nil, nil, nil)
		response.DataPlaneError(c, response.InvalidParams)
		return
	}

	// 解析入站 body 的 stream 标记（用于日志快照）
	var streamMeta struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(body, &streamMeta)
	isStream := streamMeta.Stream

	// 1. 入站协议固定为 OpenAI Chat，用 openai 适配器提取对外模型名
	inbound, err := h.adapterGroup.GetAdapter("openai")
	if err != nil {
		h.writeErrorLog(ctx, requestID, "", userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "no_adapter", err.Error(), nil, nil, nil)
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}
	modelName, err := inbound.ExtractModel(body)
	if err != nil {
		h.writeErrorLog(ctx, requestID, "", userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "invalid_request", err.Error(), nil, nil, nil)
		response.DataPlaneError(c, response.InvalidParams)
		return
	}

	// 2. 路由前余额校验
	user, err := h.userRepo.GetByID(ctx, uint(userID))
	if err != nil {
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "internal_error", err.Error(), nil, nil, nil)
		response.DataPlaneError(c, response.InternalError)
		return
	}
	if user == nil {
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "internal_error", "user not found", nil, nil, nil)
		response.DataPlaneError(c, response.Unauthorized)
		return
	}
	if user.Balance <= 0 {
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "insufficient_balance", "insufficient balance", nil, nil, nil)
		response.DataPlaneError(c, response.Forbidden)
		return
	}

	// 3. 查询可用渠道（使用缓存）
	channels, err := h.getChannelsWithCache(ctx, modelName)
	if err != nil {
		logger.Error(ctx, "list channels failed: %v", err)
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "no_channel", err.Error(), nil, nil, nil)
		response.DataPlaneError(c, response.InternalError)
		return
	}
	if len(channels) == 0 {
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "no_channel", "no active channel for model", nil, nil, nil)
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}

	// 4. 负载均衡 + 熔断检查
	balancerKey := modelName
	var selected *model.Channel
	for attempts := 0; attempts < len(channels); attempts++ {
		candidate := h.balancer.Select(balancerKey, channels)
		if candidate == nil {
			break
		}
		healthy, _ := h.breaker.IsHealthy(ctx, candidate.ID)
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
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "no_channel", "no healthy channel available", nil, nil, nil)
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}

	// 5. 按渠道 provider 选择出站适配器
	adpt, err := h.adapterGroup.GetAdapter(selected.Provider)
	if err != nil {
		logger.Error(ctx, "no adapter for provider %s: %v", selected.Provider, err)
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "no_adapter", err.Error(), nil, i64Ptr(selected.ID), strPtr(selected.Provider))
		response.DataPlaneError(c, response.ServiceUnavailable)
		return
	}

	// 6. 解析上游模型映射（命中则改写为上游名，未命中透传对外名）
	upstreamModel := h.getUpstreamModelWithCache(ctx, selected.ID, modelName)

	// 7. 由适配器构造上游请求（含协议转换与鉴权头）
	upReq, err := adpt.BuildUpstreamRequest(body, upstreamModel, selected.APIKey)
	if err != nil {
		logger.Error(ctx, "build upstream request failed: %v", err)
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "invalid_request", err.Error(), nil, i64Ptr(selected.ID), strPtr(selected.Provider))
		response.DataPlaneError(c, response.InvalidParams)
		return
	}

	// 8. 转发并按需把响应转换回 OpenAI 格式（model 回写为对外名）
	logger.Info(ctx, "routing to channel %s (id=%d) via %s adapter",
		selected.Name, selected.ID, adpt.Name())
	res, proxyErr := h.proxy.Forward(c, selected, upReq, adpt, modelName)

	// 9. 记录熔断结果
	h.breaker.Record(ctx, selected.ID, proxyErr)

	if proxyErr != nil {
		logger.Error(ctx, "service to channel %s failed: %v", selected.Name, proxyErr)
		h.writeForwardErrorLog(ctx, requestID, modelName, upstreamModel, userID, apiKeyID, clientIP, isStream,
			startedAt, selected, res, proxyErr)
		// Forward 失败时通常已把响应写给了客户端，切勿再调用 DataPlaneError
		return
	}

	// 理论成功时不应出现 res == nil，按本地错误兜底，避免 panic
	if res == nil {
		h.writeErrorLog(ctx, requestID, modelName, userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "forward_failed", "nil forward result", nil, i64Ptr(selected.ID), strPtr(selected.Provider))
		return
	}

	// 10. 成功：组装请求日志与计费流水并落库
	h.settleSuccess(ctx, requestID, modelName, upstreamModel, userID, apiKeyID, clientIP, startedAt, body, selected, user, res)
}

// writeForwardErrorLog 转发失败后落一条错误日志（区分上游错误 / 本地失败）
func (h *ChatHandler) writeForwardErrorLog(
	ctx context.Context,
	requestID, modelName, upstreamModel string,
	userID, apiKeyID int64,
	clientIP string,
	isStream bool,
	startedAt time.Time,
	selected *model.Channel,
	res *service.ForwardResult,
	proxyErr error,
) {
	now := time.Now()
	log := &model.RequestLog{
		RequestID:    requestID,
		UserID:       i64Ptr(userID),
		APIKeyID:     i64Ptr(apiKeyID),
		Model:        modelName,
		ChannelID:    i64Ptr(selected.ID),
		Provider:     strPtr(selected.Provider),
		Protocol:     strPtr("chat"),
		LogType:      model.LogTypeError,
		Status:       model.LogStatusError,
		IsStream:     isStream,
		UsageSource:  model.UsageSourceUpstream,
		ErrorStage:   strPtr(model.ErrorStageLocal),
		ErrorType:    strPtr("forward_failed"),
		ErrorMessage: strPtr(proxyErr.Error()),
		StartedAt:    &startedAt,
		CompletedAt:  &now,
	}
	// 命中上游模型映射时记录实际上游名
	if upstreamModel != "" {
		log.UpstreamModel = &upstreamModel
	}
	// 客户端 IP（非空才写，inet 类型不接受空串）
	if clientIP != "" {
		log.ClientIP = &clientIP
	}
	// res 可能为 nil（构造失败等）；非空时补齐延迟与流式信息
	if res != nil {
		log.IsStream = res.IsStream
		if res.LatencyMs > 0 {
			log.LatencyMs = i64Ptr(res.LatencyMs)
		}
	}

	// 上游错误：状态码与响应体归因到 upstream
	var ue *service.UpstreamError
	if errors.As(proxyErr, &ue) {
		msg := string(ue.Body)
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		log.ErrorStage = strPtr(model.ErrorStageUpstream)
		log.ErrorType = strPtr("upstream_error")
		log.ErrorMessage = strPtr(msg)
		log.StatusCode = i32Ptr(ue.StatusCode)
	}

	h.saveLog(ctx, log)
}

// settleSuccess 成功转发后组装日志 + 计费流水并在同一事务内落库
func (h *ChatHandler) settleSuccess(
	ctx context.Context,
	requestID, modelName, upstreamModel string,
	userID, apiKeyID int64,
	clientIP string,
	startedAt time.Time,
	body []byte,
	selected *model.Channel,
	user *model.User,
	res *service.ForwardResult,
) {
	completedAt := time.Now()

	// tokens 快照
	input := res.InputTokens
	output := res.OutputTokens
	cached := res.CachedInputTokens
	cacheCreation := res.CacheCreationTokens
	reasoning := res.ReasoningTokens

	// 用量来源：上游未返回 usage 时按 body 粗略估算输入
	usageSource := model.UsageSourceUpstream
	if !res.HasUsage {
		usageSource = model.UsageSourceEstimated
		input = int64(len(body) / 4)
	}

	// 解析计价单价（未命中返回 nil）
	price, priceErr := h.logRepo.ResolvePrice(ctx, selected.ID, modelName)
	if priceErr != nil {
		logger.Error(ctx, "resolve price failed (channel=%d model=%s): %v", selected.ID, modelName, priceErr)
	} else if price == nil {
		logger.Warn(ctx, "no price plan bound (channel=%d model=%s)", selected.ID, modelName)
	}

	currency := user.Currency
	if currency == "" {
		currency = "USD"
	}

	var (
		inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice, ratio float64
		inputCost, outputCost, cachedInputCost, cacheCreationCost, totalCost float64
	)

	if price == nil {
		// 未命中价格：按零成本记账，但仍落一条计费流水
		ratio = 1
	} else {
		inputPrice = price.InputPrice
		outputPrice = price.OutputPrice
		cachedInputPrice = price.CachedInputPrice
		cacheCreationPrice = price.CacheCreationPrice
		ratio = price.Ratio
		if price.Currency != "" {
			currency = price.Currency
		}

		// 普通输入 = 总输入 - 缓存命中 - 缓存创建（防负）
		normalInput := input - cached - cacheCreation
		if normalInput < 0 {
			normalInput = 0
		}
		// 价格方案单价单位为「币种 / 1M tokens」，按实际 tokens 折算
		const priceUnit = 1_000_000
		cachedInputCost = float64(cached) * cachedInputPrice / priceUnit
		cacheCreationCost = float64(cacheCreation) * cacheCreationPrice / priceUnit
		// 输入总成本 = 普通输入成本 + 缓存命中成本 + 缓存创建成本
		inputCost = (float64(normalInput)*inputPrice +
			float64(cached)*cachedInputPrice +
			float64(cacheCreation)*cacheCreationPrice) / priceUnit
		outputCost = float64(output) * outputPrice / priceUnit
		totalCost = (inputCost + outputCost) * ratio
	}

	log := &model.RequestLog{
		RequestID:           requestID,
		UserID:              i64Ptr(userID),
		APIKeyID:            i64Ptr(apiKeyID),
		Model:               modelName,
		ChannelID:           i64Ptr(selected.ID),
		Provider:            strPtr(selected.Provider),
		Protocol:            strPtr("chat"),
		LogType:             model.LogTypeConsumption,
		Status:              model.LogStatusSuccess,
		StatusCode:          i32Ptr(res.StatusCode),
		IsStream:            res.IsStream,
		IsCancelled:         res.IsCancelled,
		UsageSource:         usageSource,
		InputTokens:         input,
		OutputTokens:        output,
		CachedInputTokens:   cached,
		CacheCreationTokens: cacheCreation,
		ReasoningTokens:     reasoning,
		TotalCost:           totalCost,
		Currency:            currency,
		StartedAt:           &startedAt,
		CompletedAt:         &completedAt,
	}
	if upstreamModel != "" {
		log.UpstreamModel = &upstreamModel
	}
	if res.UpstreamReqID != "" {
		log.UpstreamReqID = &res.UpstreamReqID
	}
	if res.FinishReason != "" {
		log.FinishReason = &res.FinishReason
	}
	if res.CancelledBy != "" {
		log.CancelledBy = &res.CancelledBy
	}
	if res.TTFTMs > 0 {
		log.TTFTMs = i64Ptr(res.TTFTMs)
	}
	if res.LatencyMs > 0 {
		log.LatencyMs = i64Ptr(res.LatencyMs)
	}
	if clientIP != "" {
		log.ClientIP = &clientIP
	}

	billing := &model.BillingRecord{
		RequestID:           requestID,
		UserID:              userID,
		APIKeyID:            i64Ptr(apiKeyID),
		Model:               modelName,
		ChannelID:           i64Ptr(selected.ID),
		InputTokens:         input,
		OutputTokens:        output,
		CachedInputTokens:   cached,
		CacheCreationTokens: cacheCreation,
		Currency:            currency,
		InputPrice:          inputPrice,
		OutputPrice:         outputPrice,
		CachedInputPrice:    cachedInputPrice,
		CacheCreationPrice:  cacheCreationPrice,
		Ratio:               ratio,
		InputCost:           inputCost,
		OutputCost:          outputCost,
		CachedInputCost:     cachedInputCost,
		CacheCreationCost:   cacheCreationCost,
		TotalCost:           totalCost,
		Status:              model.BillingStatusSettled,
	}

	// 同一事务写日志 + 扣减余额 + 落计费流水；失败仅记录，不影响已发出的响应
	if err := h.logRepo.CreateWithBilling(ctx, log, billing); err != nil {
		logger.Error(ctx, "write log with billing failed: %v", err)
	}
}
