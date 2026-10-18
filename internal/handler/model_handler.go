package handler

import (
	"net/http"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"

	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// openAIModel OpenAI GET /v1/models 列表元素
type openAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// openAIModelList OpenAI GET /v1/models 响应体
type openAIModelList struct {
	Object string        `json:"object"`
	Data   []openAIModel `json:"data"`
}

// ModelHandler 数据面模型列表处理器
type ModelHandler struct {
	channelRepo postgresql.ChannelRepo
}

// NewModelHandler 构造函数注入依赖
func NewModelHandler(channelRepo postgresql.ChannelRepo) *ModelHandler {
	return &ModelHandler{channelRepo: channelRepo}
}

// ListModels 返回当前 API Key 可见的全部启用模型（OpenAI 兼容格式）。
// 权限约定：普通用户与管理员 Key 一致，仅返回启用模型，不区分角色。
func (h *ModelHandler) ListModels(c *gin.Context) {
	rows, err := h.channelRepo.ListEnabledModels(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "list enabled models failed: %v", err)
		response.DataPlaneError(c, response.InternalError)
		return
	}

	data := make([]openAIModel, 0, len(rows))
	for _, row := range rows {
		data = append(data, openAIModel{
			ID:      row.Name,
			Object:  "model",
			Created: row.CreatedAt.Unix(),
			OwnedBy: "xin-api",
		})
	}

	c.JSON(http.StatusOK, openAIModelList{Object: "list", Data: data})
}
