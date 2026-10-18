package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"Xin-api/internal/adapter"
	"Xin-api/internal/inbound"
	"Xin-api/internal/middleware"
	"Xin-api/internal/model"
	"Xin-api/internal/requestctx"
	"Xin-api/internal/routing"
	"Xin-api/internal/service"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// ChatHandler 数据面聊天补全处理器
type ChatHandler struct {
	channelRepo      postgresql.ChannelRepo
	redis            *redis.Client
	cacheExpire      time.Duration
	router           routing.Router
	breaker          service.ChannelBreaker
	proxy            *service.StreamProxy
	adapterGroup     *adapter.Registry
	inbound          inbound.Registry            // 入站协议编解码注册表
	logRepo          postgresql.LogRepo          // 请求日志 / 计费落库
	userRepo         postgresql.UserRepo         // 读取用户余额
	exchangeRateRepo postgresql.ExchangeRateRepo // 结算汇率查询（USD→CNY）
	gatewayNode      string                      // 网关实例标识（落库 gateway_node）

	logRequestBody  bool // 是否落库请求原文
	logResponseBody bool // 是否落库响应原文
}

// NewChatHandler 创建聊天补全处理器
func NewChatHandler(
	channelRepo postgresql.ChannelRepo,
	rdb *redis.Client,
	cacheExpire time.Duration,
	router routing.Router,
	breaker service.ChannelBreaker,
	streamProxy *service.StreamProxy,
	adapterGroup *adapter.Registry,
	inboundRegistry inbound.Registry,
	logRepo postgresql.LogRepo,
	userRepo postgresql.UserRepo,
	exchangeRateRepo postgresql.ExchangeRateRepo,
	gatewayNode string,
	logRequestBody, logResponseBody bool,
) *ChatHandler {
	return &ChatHandler{
		channelRepo:      channelRepo,
		redis:            rdb,
		cacheExpire:      cacheExpire,
		router:           router,
		breaker:          breaker,
		proxy:            streamProxy,
		adapterGroup:     adapterGroup,
		inbound:          inboundRegistry,
		logRepo:          logRepo,
		userRepo:         userRepo,
		exchangeRateRepo: exchangeRateRepo,
		gatewayNode:      gatewayNode,
		logRequestBody:   logRequestBody,
		logResponseBody:  logResponseBody,
	}
}

// --- 指针辅助函数（避免大量取址样板） ---

func strPtr(s string) *string { return &s }
func i64Ptr(v int64) *int64   { return &v }
func i32Ptr(v int) *int       { return &v }

// routeStrategy 返回选路策略快照，route 为空时回退为加权轮询
func routeStrategy(route *routing.RouteResult) string {
	if route == nil || route.Strategy == "" {
		return routing.StrategyWeighted
	}
	return route.Strategy
}

// routeWeight 返回命中渠道权重快照
func routeWeight(route *routing.RouteResult) int {
	if route == nil {
		return 0
	}
	return route.Weight
}

// shouldRetry 判断一次转发失败是否值得跨渠道重试
// 4xx（除 429）通常是请求本身问题，换渠道也无意义；网络/5xx 可重试
func shouldRetry(res *service.ForwardResult, err error) bool {
	if err == nil {
		return false
	}
	var ue *service.UpstreamError
	if errors.As(err, &ue) {
		return ue.StatusCode == http.StatusTooManyRequests || ue.StatusCode >= 500
	}
	return true
}

// gatewayNodePtr 网关实例标识；未配置时返回 nil，避免写入空串
func (h *ChatHandler) gatewayNodePtr() *string {
	if h.gatewayNode == "" {
		return nil
	}
	return &h.gatewayNode
}

// localErrorStatus 本地错误类型到 HTTP 状态码的映射，保证错误日志 status_code 可对账
func localErrorStatus(errType string) int {
	switch errType {
	case "invalid_request":
		return 400
	case "insufficient_balance":
		return 403
	case "model_not_found":
		return 404
	case "no_channel", "no_adapter":
		return 503
	case "forward_failed":
		return 502
	default:
		return 500
	}
}

// upstreamErrorCode 从上游错误响应体中提取 error.code / error.type，统一加 upstream: 前缀
func upstreamErrorCode(body []byte, statusCode int) string {
	var payload struct {
		Error struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		if payload.Error.Code != "" {
			return "upstream:" + payload.Error.Code
		}
		if payload.Error.Type != "" {
			return "upstream:" + payload.Error.Type
		}
	}
	return fmt.Sprintf("upstream:http_%d", statusCode)
}

// saveLog 持久化一条请求日志，并按开关写入请求/响应原文；写库失败仅记录日志，不阻断主流程
func (h *ChatHandler) saveLog(ctx context.Context, log *model.RequestLog, requestBody, responseBody []byte) {
	if h.logRepo == nil {
		return
	}
	if err := h.logRepo.Create(ctx, log); err != nil {
		logger.Error(ctx, "write request log failed: %v", err)
		return
	}
	h.savePayload(ctx, log, requestBody, responseBody)
}

// savePayload 按开关写入 request_payloads；created_at 与父行 request_logs 保持一致
func (h *ChatHandler) savePayload(ctx context.Context, log *model.RequestLog, requestBody, responseBody []byte) {
	if h.logRepo == nil || (!h.logRequestBody && !h.logResponseBody) {
		return
	}
	payload := &model.RequestPayload{
		RequestID: log.RequestID,
		CreatedAt: log.CreatedAt,
	}
	if h.logRequestBody && len(requestBody) > 0 {
		body := string(requestBody)
		payload.RequestBody = &body
	}
	if h.logResponseBody && len(responseBody) > 0 {
		body := string(responseBody)
		payload.ResponseBody = &body
	}
	if payload.RequestBody == nil && payload.ResponseBody == nil {
		return
	}
	if err := h.logRepo.CreatePayload(ctx, payload); err != nil {
		logger.Error(ctx, "write request payload failed: %v", err)
	}
}

// writeErrorLog 组装并写入一条错误请求日志（本地错误分支复用）
func (h *ChatHandler) writeErrorLog(
	ctx context.Context,
	requestID, modelName, protocol string,
	userID, apiKeyID int64,
	clientIP string,
	isStream bool,
	startedAt time.Time,
	logType, stage, errType, errMsg string,
	statusCode *int,
	channelID *int64,
	provider *string,
	requestBody []byte,
) {
	now := time.Now()
	// 本地错误未显式传状态码时，按错误类型补齐，便于日志与响应对账
	if statusCode == nil {
		statusCode = i32Ptr(localErrorStatus(errType))
	}
	log := &model.RequestLog{
		RequestID:    requestID,
		UserID:       i64Ptr(userID),
		APIKeyID:     i64Ptr(apiKeyID),
		Model:        modelName,
		Protocol:     strPtr(protocol),
		LogType:      logType,
		Status:       model.LogStatusError,
		StatusCode:   statusCode,
		IsStream:     isStream,
		UsageSource:  model.UsageSourceUpstream,
		ErrorStage:   strPtr(stage),
		ErrorType:    strPtr(errType),
		ErrorCode:    strPtr(errType),
		ErrorMessage: strPtr(errMsg),
		ChannelID:    channelID,
		Provider:     provider,
		GatewayNode:  h.gatewayNodePtr(),
		StartedAt:    &startedAt,
		CompletedAt:  &now,
	}
	// client_ip 为 inet 类型，空串会报错，故非空才写入
	if clientIP != "" {
		log.ClientIP = &clientIP
	}
	h.saveLog(ctx, log, requestBody, nil)
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

	// 3. 写入缓存（仅缓存命中结果：未登记模型不缓存负结果，
	// 避免新增模型或渠道恢复后仍被空结果挡住直到 TTL 到期）
	if len(channels) > 0 {
		if data, marshalErr := json.Marshal(channels); marshalErr == nil {
			h.redis.Set(ctx, cacheKey, data, h.cacheExpire)
		}
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

// HandleDataPlane 数据面 OpenAI Chat 兼容入口
func (h *ChatHandler) HandleDataPlane(c *gin.Context) {
	h.handle(c, inbound.ProtocolChat, inbound.NormalizeOptions{})
}

// HandleResponses 数据面 OpenAI Responses API 入口
func (h *ChatHandler) HandleResponses(c *gin.Context) {
	h.handle(c, inbound.ProtocolResponses, inbound.NormalizeOptions{})
}

// HandleMessages 数据面 Anthropic Messages API 入口
func (h *ChatHandler) HandleMessages(c *gin.Context) {
	h.handle(c, inbound.ProtocolMessages, inbound.NormalizeOptions{})
}

// HandleGemini 数据面 Google Gemini generateContent 入口
// 路由 :model 捕获整段（如 gemini-2.0-flash:generateContent），此处切分动作后缀
func (h *ChatHandler) HandleGemini(c *gin.Context) {
	model, stream := parseGeminiAction(c.Param("model"))
	if c.Query("alt") == "sse" {
		stream = true
	}
	h.handle(c, inbound.ProtocolGemini, inbound.NormalizeOptions{PathModel: model, Stream: stream})
}

// parseGeminiAction 从路径段解析 Gemini 模型名与流式标记
func parseGeminiAction(raw string) (model string, stream bool) {
	if idx := strings.LastIndex(raw, ":"); idx >= 0 {
		return raw[:idx], raw[idx+1:] == "streamGenerateContent"
	}
	return raw, false
}

// abortLocalError 按入站协议编码并写出网关本地错误，同时阻断后续中间件
func (h *ChatHandler) abortLocalError(c *gin.Context, codec inbound.Codec, bizCode int) {
	status, body := codec.LocalError(bizCode)
	c.Data(status, "application/json", body)
	c.Abort()
}

// handle 数据面通用处理：入站归一化 → 现有链路（选路 / 适配器 / 转发）→ 出站再编码
func (h *ChatHandler) handle(c *gin.Context, protocol string, opts inbound.NormalizeOptions) {
	codec := h.inbound.New(protocol)

	// 请求上下文由中间件创建；此处兜底 Ensure，保证 request_id 与起始时间唯一
	rc := requestctx.Ensure(c)
	ctx := c.Request.Context()
	requestID := rc.RequestID
	startedAt := rc.StartTime
	rc.Protocol = codec.Name()

	// 0. 读取鉴权上下文（由 ApiKeyAuth 注入，类型均为 int64）
	uidVal, uidOK := c.Get(middleware.CtxUserIDKey)
	kidVal, kidOK := c.Get(middleware.CtxAPIKeyIDKey)
	userID, uidCast := uidVal.(int64)
	apiKeyID, kidCast := kidVal.(int64)
	if !uidOK || !kidOK || !uidCast || !kidCast {
		h.abortLocalError(c, codec, response.Unauthorized)
		return
	}

	// 客户端 IP（为空则不写入，避免 inet 类型写入空串报错）
	clientIP := c.ClientIP()

	rawBody, err := c.GetRawData()
	if err != nil {
		h.writeErrorLog(ctx, requestID, "", codec.Name(), userID, apiKeyID, clientIP, false, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "invalid_request", "read request body failed", nil, nil, nil, nil)
		h.abortLocalError(c, codec, response.InvalidParams)
		return
	}

	// 入站归一化：responses / gemini → 规范 OpenAI Chat 请求体（chat 为恒等）
	body, err := codec.NormalizeRequest(rawBody, opts)
	if err != nil {
		h.writeErrorLog(ctx, requestID, "", codec.Name(), userID, apiKeyID, clientIP, opts.Stream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "invalid_request", err.Error(), nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.InvalidParams)
		return
	}

	// 解析规范 body 的 stream 标记（用于日志快照）
	var streamMeta struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(body, &streamMeta)
	isStream := streamMeta.Stream || opts.Stream
	rc.IsStream = isStream

	// 1. 统一用 openai 适配器从规范 body 提取对外模型名
	openaiAdapter, err := h.adapterGroup.GetAdapter("openai")
	if err != nil {
		h.writeErrorLog(ctx, requestID, "", codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "no_adapter", err.Error(), nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.ServiceUnavailable)
		return
	}
	modelName, err := openaiAdapter.ExtractModel(body)
	if err != nil {
		h.writeErrorLog(ctx, requestID, "", codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "invalid_request", err.Error(), nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.InvalidParams)
		return
	}
	rc.Model = modelName

	// 2. 路由前余额校验
	user, err := h.userRepo.GetByID(ctx, uint(userID))
	if err != nil {
		h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "internal_error", err.Error(), nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.InternalError)
		return
	}
	if user == nil {
		h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "internal_error", "user not found", nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.Unauthorized)
		return
	}
	// 用户被禁用后，其名下 API Key 立即失效（不能仅依赖 Key 自身状态）
	if user.Status != model.UserStatusActive {
		h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "user_disabled", "user disabled", nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.Unauthorized)
		return
	}
	// 余额允许因历史消费扣成负数，因此这里按非正数拦截：负余额账户直接拒绝，不请求上游
	if !user.Balance.IsPositive() {
		h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "insufficient_balance", "insufficient balance", nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.Forbidden)
		return
	}

	// 3. 查询可用渠道（使用缓存）
	channels, err := h.getChannelsWithCache(ctx, modelName)
	if err != nil {
		logger.Error(ctx, "list channels failed: %v", err)
		h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "no_channel", err.Error(), nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.InternalError)
		return
	}
	if len(channels) == 0 {
		h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "model_not_found", "model not registered or no active channel", nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.ModelNotFound)
		return
	}

	// 4. 选路（priority 优先，同级加权轮询）+ 跨渠道重试
	excluded := make([]int64, 0, 2)
	retryChannels := make([]int64, 0, 2)
	var (
		route         *routing.RouteResult
		selected      *model.Channel
		adpt          adapter.ProviderAdapter
		upstreamModel string
		upReq         *adapter.UpstreamRequest
		res           *service.ForwardResult
		proxyErr      error
		maxAttempts   int
	)

	for attempt := 1; ; attempt++ {
		route, err = h.router.Route(ctx, routing.RouteRequest{
			Model:             modelName,
			Protocol:          codec.Name(),
			Channels:          channels,
			ExcludeChannelIDs: excluded,
		})
		if err != nil {
			proxyErr = err
			break
		}
		selected = route.Channel

		// 重试预算取自首个命中渠道的 max_retries（含首次请求）
		if maxAttempts == 0 {
			maxAttempts = selected.MaxRetries + 1
			if maxAttempts < 1 {
				maxAttempts = 1
			}
		}

		// 5. 按渠道 provider 选择出站适配器
		adpt, err = h.adapterGroup.GetAdapter(selected.Provider)
		if err != nil {
			logger.Error(ctx, "no adapter for provider %s: %v", selected.Provider, err)
			h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
				model.LogTypeError, model.ErrorStageLocal, "no_adapter", err.Error(), nil, i64Ptr(selected.ID), strPtr(selected.Provider), rawBody)
			h.abortLocalError(c, codec, response.ServiceUnavailable)
			return
		}

		// 6. 解析上游模型映射（命中则改写为上游名，未命中透传对外名）
		upstreamModel = h.getUpstreamModelWithCache(ctx, selected.ID, modelName)

		// 7. 由适配器构造上游请求（含协议转换与鉴权头）
		upReq, err = adpt.BuildUpstreamRequest(body, upstreamModel, selected.APIKey)
		if err != nil {
			logger.Error(ctx, "build upstream request failed: %v", err)
			h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
				model.LogTypeError, model.ErrorStageLocal, "invalid_request", err.Error(), nil, i64Ptr(selected.ID), strPtr(selected.Provider), rawBody)
			h.abortLocalError(c, codec, response.InvalidParams)
			return
		}

		// 8. 转发并按需把响应转换回 OpenAI 格式，再由 codec 编码为入站协议格式（model 回写为对外名）
		logger.Info(ctx, "routing to channel %s (id=%d) via %s adapter",
			selected.Name, selected.ID, adpt.Name())
		res, proxyErr = h.proxy.Forward(c, selected, upReq, adpt, modelName, codec, h.logResponseBody)
		h.breaker.Record(ctx, selected.ID, proxyErr)

		if proxyErr == nil {
			break
		}

		// 流式响应已开始输出或客户端已中断，无法换渠道重试
		if (res != nil && (res.IsStream || res.IsCancelled)) || c.Writer.Written() {
			break
		}
		if !shouldRetry(res, proxyErr) || attempt >= maxAttempts {
			break
		}

		// 记录失败渠道并排除后重新选路
		retryChannels = append(retryChannels, selected.ID)
		excluded = append(excluded, selected.ID)
	}

	// 9. 选路失败（无可用渠道）：可能发生在首次或重试时全部渠道被排除
	if selected == nil {
		h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "no_channel", proxyErr.Error(), nil, nil, nil, rawBody)
		h.abortLocalError(c, codec, response.ServiceUnavailable)
		return
	}
	rc.ChannelID = selected.ID
	rc.Provider = selected.Provider

	if proxyErr != nil {
		logger.Error(ctx, "service to channel %s failed: %v", selected.Name, proxyErr)
		h.writeForwardErrorLog(ctx, requestID, modelName, upstreamModel, codec.Name(), userID, apiKeyID, clientIP, isStream,
			startedAt, selected, route, retryChannels, res, proxyErr, rawBody)
		// Forward 不再向客户端写出错误响应；此处按入站协议编码后统一回写（已写出则跳过）
		if !c.Writer.Written() {
			var ue *service.UpstreamError
			if errors.As(proxyErr, &ue) {
				status, errBody := codec.UpstreamError(ue.StatusCode, ue.Body)
				c.Data(status, "application/json", errBody)
			} else {
				h.abortLocalError(c, codec, response.ServiceUnavailable)
			}
		}
		return
	}

	// 理论成功时不应出现 res == nil，按本地错误兜底，避免 panic
	if res == nil {
		h.writeErrorLog(ctx, requestID, modelName, codec.Name(), userID, apiKeyID, clientIP, isStream, startedAt,
			model.LogTypeError, model.ErrorStageLocal, "forward_failed", "nil forward result", nil, i64Ptr(selected.ID), strPtr(selected.Provider), rawBody)
		return
	}

	// 10. 成功：组装请求日志与计费流水并落库
	h.settleSuccess(ctx, requestID, modelName, upstreamModel, codec.Name(), userID, apiKeyID, clientIP, startedAt, rawBody, selected, route, retryChannels, user, res)
}

// writeForwardErrorLog 转发失败后落一条错误日志（区分上游错误 / 本地失败）
func (h *ChatHandler) writeForwardErrorLog(
	ctx context.Context,
	requestID, modelName, upstreamModel, protocol string,
	userID, apiKeyID int64,
	clientIP string,
	isStream bool,
	startedAt time.Time,
	selected *model.Channel,
	route *routing.RouteResult,
	retryChannels []int64,
	res *service.ForwardResult,
	proxyErr error,
	requestBody []byte,
) {
	now := time.Now()
	log := &model.RequestLog{
		RequestID:     requestID,
		UserID:        i64Ptr(userID),
		APIKeyID:      i64Ptr(apiKeyID),
		Model:         modelName,
		ChannelID:     i64Ptr(selected.ID),
		Provider:      strPtr(selected.Provider),
		Protocol:      strPtr(protocol),
		LogType:       model.LogTypeError,
		Status:        model.LogStatusError,
		IsStream:      isStream,
		UsageSource:   model.UsageSourceUpstream,
		ErrorStage:    strPtr(model.ErrorStageLocal),
		ErrorType:     strPtr("forward_failed"),
		ErrorCode:     strPtr("forward_failed"),
		ErrorMessage:  strPtr(proxyErr.Error()),
		RetryCount:    len(retryChannels),
		RetryChannels: model.Int64Array(retryChannels),
		RouteStrategy: strPtr(routeStrategy(route)),
		RouteWeight:   i32Ptr(routeWeight(route)),
		GatewayNode:   h.gatewayNodePtr(),
		StartedAt:     &startedAt,
		CompletedAt:   &now,
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
		log.ErrorCode = strPtr(upstreamErrorCode(ue.Body, ue.StatusCode))
		log.ErrorMessage = strPtr(msg)
		log.StatusCode = i32Ptr(ue.StatusCode)
	}

	var responseBody []byte
	if res != nil {
		responseBody = res.ResponseBody
	}
	h.saveLog(ctx, log, requestBody, responseBody)
}

// settleSuccess 成功转发后组装日志 + 计费流水并在同一事务内落库
func (h *ChatHandler) settleSuccess(
	ctx context.Context,
	requestID, modelName, upstreamModel, protocol string,
	userID, apiKeyID int64,
	clientIP string,
	startedAt time.Time,
	body []byte,
	selected *model.Channel,
	route *routing.RouteResult,
	retryChannels []int64,
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
		inputPrice, outputPrice, cachedInputPrice, cacheCreationPrice        model.Money
		inputCost, outputCost, cachedInputCost, cacheCreationCost, totalCost model.Money
		ratio                                                                decimal.Decimal
	)

	if price == nil {
		// 未命中价格：按零成本记账，但仍落一条计费流水
		ratio = decimal.NewFromInt(1)
	} else {
		inputPrice = price.InputPrice
		outputPrice = price.OutputPrice
		cachedInputPrice = price.CachedInputPrice
		cacheCreationPrice = price.CacheCreationPrice
		ratio = decimal.NewFromFloat(price.Ratio)
		if price.Currency != "" {
			currency = price.Currency
		}

		// 普通输入 = 总输入 - 缓存命中 - 缓存创建（防负）
		normalInput := input - cached - cacheCreation
		if normalInput < 0 {
			normalInput = 0
		}
		// 价格方案单价单位为「币种 / 1M tokens」，按实际 tokens 折算
		priceUnit := decimal.NewFromInt(1_000_000)
		cachedInputCost = model.Money{Decimal: decimal.NewFromInt(cached).
			Mul(cachedInputPrice.Decimal).Div(priceUnit)}
		cacheCreationCost = model.Money{Decimal: decimal.NewFromInt(cacheCreation).
			Mul(cacheCreationPrice.Decimal).Div(priceUnit)}
		// 输入总成本 = 普通输入成本 + 缓存命中成本 + 缓存创建成本
		inputCost = model.Money{Decimal: decimal.NewFromInt(normalInput).
			Mul(inputPrice.Decimal).
			Add(decimal.NewFromInt(cached).Mul(cachedInputPrice.Decimal)).
			Add(decimal.NewFromInt(cacheCreation).Mul(cacheCreationPrice.Decimal)).
			Div(priceUnit)}
		outputCost = model.Money{Decimal: decimal.NewFromInt(output).
			Mul(outputPrice.Decimal).Div(priceUnit)}
		totalCost = model.Money{Decimal: inputCost.Decimal.Add(outputCost.Decimal).Mul(ratio)}
	}

	// 结算币种固定为 CNY：CNY 定价直接结算，其他币种（USD）按配置汇率折算后扣减余额
	settleCurrency := model.CurrencyCNY
	settleCost := model.Money{Decimal: decimal.Zero}
	var exchangeRate *model.Money
	billingStatus := model.BillingStatusSettled

	if strings.EqualFold(currency, model.CurrencyCNY) {
		settleCost = totalCost
	} else {
		rateRecord, rateErr := h.exchangeRateRepo.Get(ctx, currency, model.CurrencyCNY)
		if rateErr != nil || rateRecord == nil || !rateRecord.Rate.IsPositive() {
			// 汇率缺失：仅标记结算失败，日志与计费流水仍照常写入，响应不受影响
			logger.Error(ctx, "resolve exchange rate failed (from=%s to=%s): err=%v", currency, model.CurrencyCNY, rateErr)
			billingStatus = model.BillingStatusFailed
		} else {
			rate := model.Money{Decimal: rateRecord.Rate.Decimal.Round(8)}
			settleCost = model.Money{Decimal: totalCost.Decimal.Mul(rate.Decimal).Round(8)}
			exchangeRate = &rate
		}
	}

	log := &model.RequestLog{
		RequestID:           requestID,
		UserID:              i64Ptr(userID),
		APIKeyID:            i64Ptr(apiKeyID),
		Model:               modelName,
		ChannelID:           i64Ptr(selected.ID),
		Provider:            strPtr(selected.Provider),
		Protocol:            strPtr(protocol),
		LogType:             model.LogTypeConsumption,
		Status:              model.LogStatusSuccess,
		StatusCode:          i32Ptr(res.StatusCode),
		IsStream:            res.IsStream,
		IsCancelled:         res.IsCancelled,
		UsageSource:         usageSource,
		RetryCount:          len(retryChannels),
		RetryChannels:       model.Int64Array(retryChannels),
		RouteStrategy:       strPtr(routeStrategy(route)),
		RouteWeight:         i32Ptr(routeWeight(route)),
		InputTokens:         input,
		OutputTokens:        output,
		CachedInputTokens:   cached,
		CacheCreationTokens: cacheCreation,
		ReasoningTokens:     reasoning,
		TotalCost:           totalCost,
		Currency:            currency,
		GatewayNode:         h.gatewayNodePtr(),
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
		// 生成耗时 = 总延迟 − 首 token 延迟；无 TTFT（非流式）时取总延迟
		generationMs := res.LatencyMs
		if res.TTFTMs > 0 && res.LatencyMs >= res.TTFTMs {
			generationMs = res.LatencyMs - res.TTFTMs
		}
		log.GenerationMs = i64Ptr(generationMs)
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
		Ratio:               ratio.InexactFloat64(),
		InputCost:           inputCost,
		OutputCost:          outputCost,
		CachedInputCost:     cachedInputCost,
		CacheCreationCost:   cacheCreationCost,
		TotalCost:           totalCost,
		SettleCurrency:      settleCurrency,
		SettleCost:          settleCost,
		ExchangeRate:        exchangeRate,
		Status:              billingStatus,
	}

	// 同一事务写日志 + 扣减余额 + 落计费流水；失败仅记录，不影响已发出的响应
	if err := h.logRepo.CreateWithBilling(ctx, log, billing); err != nil {
		logger.Error(ctx, "write log with billing failed: %v", err)
		return
	}
	h.savePayload(ctx, log, body, res.ResponseBody)
}
