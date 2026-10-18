package postgresql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Xin-api/internal/model"
)

type ChannelRepo interface {
	// Create 创建上游渠道
	Create(ctx context.Context, channel *model.Channel) error
	// DeleteSoft 软删除上游渠道
	DeleteSoft(ctx context.Context, id int64) error
	// Update 完整更新渠道信息（常用于管理后台编辑页面整体提交）
	Update(ctx context.Context, channel *model.Channel) error
	// UpdateStatus 修改渠道状态 (active-正常 disabled-手动关闭 circuit_open-自动熔断)——常用于健康检查熔断器
	UpdateStatus(ctx context.Context, id int64, status string) error
	// GetByID 根据 ID 查询渠道详情
	GetByID(ctx context.Context, id int64) (*model.Channel, error)
	// GetByName 按名称查询未软删的渠道（大小写不敏感），用于判重；不存在返回 (nil, nil)
	GetByName(ctx context.Context, name string) (*model.Channel, error)
	// ListAll 获取所有渠道（包含关闭和熔断的，供管理后台列表展示）
	ListAll(ctx context.Context) ([]model.Channel, error)
	// ListActiveByModel ⚡️核心流量方法：根据请求的模型，捞出所有当前可用的物理渠道（状态为 active）
	ListActiveByModel(ctx context.Context, modelName string) ([]model.Channel, error)
	// ListModelsByChannel 获取指定渠道下可用的模型（渠道 × 模型，来自 models 表）
	ListModelsByChannel(ctx context.Context, channelID int64) ([]model.Model, error)
	// ListModelsByChannelAll 获取指定渠道下全部模型（含 disabled，不含软删），并带出当前价格方案绑定
	ListModelsByChannelAll(ctx context.Context, channelID int64) ([]model.ChannelModelView, error)
	// GetModelByID 根据模型 ID 查询渠道模型记录（软删记录不返回）
	GetModelByID(ctx context.Context, id int64) (*model.Model, error)
	// GetModelByName 按 (channel_id, name) 查询渠道模型记录（软删记录不返回），用于判重
	GetModelByName(ctx context.Context, channelID int64, name string) (*model.Model, error)
	// GetUpstreamModel 按 (channel_id, name) 取该渠道登记的实际上游模型名；未登记或非 active 返回空串
	GetUpstreamModel(ctx context.Context, channelID int64, name string) (string, error)
	// CreateModel 为渠道新增模型（渠道 × 模型）
	CreateModel(ctx context.Context, m *model.Model) error
	// CreateModelWithBinding 为渠道新增模型，并在同一事务内维护价格方案绑定
	// modelPriceID：nil 或 <=0 时不写入绑定；>0 时绑定到该价格方案
	CreateModelWithBinding(ctx context.Context, m *model.Model, modelPriceID *int64) error
	// UpdateModel 更新模型（整体保存）
	UpdateModel(ctx context.Context, m *model.Model) error
	// UpdateModelWithBinding 更新模型，并在同一事务内维护价格方案绑定
	// modelPriceID：nil=不改绑定；0=解绑；>0=切换到该价格方案
	UpdateModelWithBinding(ctx context.Context, m *model.Model, modelPriceID *int64) error
	// UpdateModelStatus 修改模型状态（active/disabled）
	UpdateModelStatus(ctx context.Context, id int64, status string) error
	// DeleteModelSoft 软删除模型
	DeleteModelSoft(ctx context.Context, id int64) error
	// DeleteModelSoftWithBinding 软删除指定渠道下的模型，并在同一事务内软删其价格方案绑定
	DeleteModelSoftWithBinding(ctx context.Context, channelID, modelID int64) error
	// ListModelChannelRows 查询全部「渠道 × 模型」明细行（JOIN 渠道信息），供模型管理页按逻辑模型名聚合
	ListModelChannelRows(ctx context.Context) ([]model.ModelChannelEntry, error)
	// ListEnabledModels 查询对外可用的逻辑模型名（模型与渠道均 active 且未软删），按名称去重
	ListEnabledModels(ctx context.Context) ([]model.EnabledModelInfo, error)
}

type channelRepo struct {
	db *gorm.DB
}

// NewChannelRepo 创建上游渠道仓储
func NewChannelRepo(db *gorm.DB) ChannelRepo {
	return &channelRepo{db: db}
}

func (r *channelRepo) Create(ctx context.Context, channel *model.Channel) error {
	return gorm.G[model.Channel](r.db).Create(ctx, channel)
}

// DeleteSoft 软删除渠道，并在同一事务内链式软删其名下全部 models 及其价格方案绑定，避免孤儿数据
// 顺序：先取模型 id 列表 → 软删绑定 → 软删 models → 软删渠道（绑定清理须在 models 软删之前，否则查不到模型 id）。
func (r *channelRepo) DeleteSoft(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var modelIDs []int64
		if err := tx.Model(&model.Model{}).Where("channel_id = ?", id).Pluck("id", &modelIDs).Error; err != nil {
			return err
		}
		if len(modelIDs) > 0 {
			if err := tx.Where("model_id IN ?", modelIDs).Delete(&model.ModelPriceBinding{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("channel_id = ?", id).Delete(&model.Model{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&model.Channel{}).Error
	})
}

func (r *channelRepo) Update(ctx context.Context, channel *model.Channel) error {
	// Save 会根据主键 ID 是否存在来自动执行 UPDATE，并且会更新所有字段（包括零值如 status=0）
	return r.db.WithContext(ctx).Save(channel).Error
}

func (r *channelRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	rows, err := gorm.G[model.Channel](r.db).Where("id = ?", id).Update(ctx, "status", status)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

func (r *channelRepo) GetByID(ctx context.Context, id int64) (*model.Channel, error) {
	channel, err := gorm.G[model.Channel](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

func (r *channelRepo) GetByName(ctx context.Context, name string) (*model.Channel, error) {
	channel, err := gorm.G[model.Channel](r.db).Where("lower(name) = lower(?)", name).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

func (r *channelRepo) ListAll(ctx context.Context) ([]model.Channel, error) {
	channels, err := gorm.G[model.Channel](r.db).Find(ctx)
	return channels, err
}

// ListActiveByModel 根据模型选出支持该模型的可用渠道
// 模型支持关系由 models 表（渠道 × 模型）行式表达；无精确匹配时回退全部可用渠道。
func (r *channelRepo) ListActiveByModel(ctx context.Context, modelName string) ([]model.Channel, error) {
	var channels []model.Channel
	err := r.db.WithContext(ctx).
		Model(&model.Channel{}).
		Select("channels.*").
		Joins("JOIN models ON models.channel_id = channels.id AND models.deleted_at IS NULL AND models.status = ?", model.ModelStatusActive).
		Where("channels.status = ?", model.ChannelStatusActive).
		Where("models.name = ?", modelName).
		Order("channels.priority DESC, channels.weight DESC").
		Find(&channels).Error
	if err != nil {
		return nil, err
	}

	// 未登记模型不再回退到全部活跃渠道：兜底透传会让未登记的模型名也能
	// 调用成功，但计价侧按模型名匹配不到价格方案，导致零成本调用。
	// 统一返回空列表，由上层按 model_not_found 拒绝。
	return channels, nil
}

func (r *channelRepo) ListModelsByChannel(ctx context.Context, channelID int64) ([]model.Model, error) {
	models, err := gorm.G[model.Model](r.db).
		Where("channel_id = ? AND status = ?", channelID, model.ModelStatusActive).
		Find(ctx)
	return models, err
}

// ListModelsByChannelAll 获取指定渠道下全部模型（含 disabled，不含软删），
// 并补充当前绑定的 model_price_id / price_name，供渠道管理页展示与编辑预选。
// 说明：分两步查询——模型本体走 GORM Find（保证 capabilities 数组正确解码），绑定信息单独聚合后合并。
func (r *channelRepo) ListModelsByChannelAll(ctx context.Context, channelID int64) ([]model.ChannelModelView, error) {
	models, err := gorm.G[model.Model](r.db).
		Where("channel_id = ?", channelID).
		Order("id ASC").
		Find(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]model.ChannelModelView, 0, len(models))
	if len(models) == 0 {
		return views, nil
	}

	ids := make([]int64, 0, len(models))
	for i := range models {
		ids = append(ids, models[i].ID)
	}

	// 该渠道下全部有效绑定的模型 id → 价格方案
	var binds []struct {
		ModelID      int64   `gorm:"column:model_id"`
		ModelPriceID int64   `gorm:"column:model_price_id"`
		PriceName    *string `gorm:"column:price_name"`
	}
	if err := r.db.WithContext(ctx).
		Table("model_price_bindings AS bindings").
		Select("bindings.model_id AS model_id, bindings.model_price_id AS model_price_id, prices.name AS price_name").
		Joins("LEFT JOIN model_prices prices ON prices.id = bindings.model_price_id").
		Where("bindings.model_id IN ? AND bindings.deleted_at IS NULL", ids).
		Scan(&binds).Error; err != nil {
		return nil, err
	}

	bindByModelID := make(map[int64]*int64, len(binds))
	nameByModelID := make(map[int64]*string, len(binds))
	for _, b := range binds {
		priceID := b.ModelPriceID
		bindByModelID[b.ModelID] = &priceID
		nameByModelID[b.ModelID] = b.PriceName
	}

	for i := range models {
		m := models[i]
		views = append(views, model.ChannelModelView{
			ID:              m.ID,
			ChannelID:       m.ChannelID,
			Name:            m.Name,
			UpstreamModel:   m.UpstreamModel,
			ContextLength:   m.ContextLength,
			MaxOutputTokens: m.MaxOutputTokens,
			Capabilities:    m.Capabilities,
			Status:          m.Status,
			Ratio:           m.Ratio,
			ModelPriceID:    bindByModelID[m.ID],
			PriceName:       nameByModelID[m.ID],
			CreatedAt:       m.CreatedAt,
			UpdatedAt:       m.UpdatedAt,
		})
	}

	return views, nil
}

// GetModelByID 根据模型 ID 查询渠道模型记录
func (r *channelRepo) GetModelByID(ctx context.Context, id int64) (*model.Model, error) {
	m, err := gorm.G[model.Model](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// GetModelByName 按 (channel_id, name) 查询渠道模型记录
func (r *channelRepo) GetModelByName(ctx context.Context, channelID int64, name string) (*model.Model, error) {
	m, err := gorm.G[model.Model](r.db).
		Where("channel_id = ? AND name = ?", channelID, name).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// GetUpstreamModel 按 (channel_id, name) 取上游真实模型名（仅未软删且 active），未登记返回空串
func (r *channelRepo) GetUpstreamModel(ctx context.Context, channelID int64, name string) (string, error) {
	m, err := gorm.G[model.Model](r.db).
		Select("upstream_model").
		Where("channel_id = ? AND name = ? AND status = ?", channelID, name, model.ModelStatusActive).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return m.UpstreamModel, nil
}

// CreateModel 为渠道新增模型
func (r *channelRepo) CreateModel(ctx context.Context, m *model.Model) error {
	return gorm.G[model.Model](r.db).Create(ctx, m)
}

// CreateModelWithBinding 为渠道新增模型，并在同一事务内写入价格方案绑定
func (r *channelRepo) CreateModelWithBinding(ctx context.Context, m *model.Model, modelPriceID *int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		if modelPriceID != nil && *modelPriceID > 0 {
			if err := tx.Create(&model.ModelPriceBinding{ModelPriceID: *modelPriceID, ModelID: m.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateModel 更新模型（整体保存，允许清空价格等可空字段）
func (r *channelRepo) UpdateModel(ctx context.Context, m *model.Model) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// UpdateModelWithBinding 更新模型，并在同一事务内维护价格方案绑定
func (r *channelRepo) UpdateModelWithBinding(ctx context.Context, m *model.Model, modelPriceID *int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Save 全字段更新，保留 models 预留价格列存量值不变
		if err := tx.Save(m).Error; err != nil {
			return err
		}
		if modelPriceID == nil {
			return nil // 不改动绑定
		}
		// 先软删该模型的既有有效绑定，保证未软删范围内最多一条绑定
		if err := tx.Where("model_id = ?", m.ID).Delete(&model.ModelPriceBinding{}).Error; err != nil {
			return err
		}
		if *modelPriceID > 0 {
			if err := tx.Create(&model.ModelPriceBinding{ModelPriceID: *modelPriceID, ModelID: m.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateModelStatus 修改模型状态
func (r *channelRepo) UpdateModelStatus(ctx context.Context, id int64, status string) error {
	rows, err := gorm.G[model.Model](r.db).Where("id = ?", id).Update(ctx, "status", status)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

// DeleteModelSoft 软删除模型
func (r *channelRepo) DeleteModelSoft(ctx context.Context, id int64) error {
	rows, err := gorm.G[model.Model](r.db).Where("id = ?", id).Delete(ctx)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

// DeleteModelSoftWithBinding 软删除指定渠道下的模型，并在同一事务内软删其价格方案绑定
func (r *channelRepo) DeleteModelSoftWithBinding(ctx context.Context, channelID, modelID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND channel_id = ?", modelID, channelID).Delete(&model.Model{}).Error; err != nil {
			return err
		}
		return tx.Where("model_id = ?", modelID).Delete(&model.ModelPriceBinding{}).Error
	})
}

// ListModelChannelRows 查询全部「渠道 × 模型」明细行（JOIN 渠道信息），供模型管理页按逻辑模型名聚合
// 说明：以 models 为主表，GORM 自动追加 models.deleted_at IS NULL；从表 channels 的软删条件写在 JOIN 内；
// 价格信息通过 LEFT JOIN 绑定表与价格方案表补充 model_price_id / price_name。
func (r *channelRepo) ListModelChannelRows(ctx context.Context) ([]model.ModelChannelEntry, error) {
	var rows []model.ModelChannelEntry
	err := r.db.WithContext(ctx).
		Model(&model.Model{}).
		Select("models.name AS name, models.id AS model_id, models.channel_id AS channel_id, " +
			"channels.name AS channel_name, channels.provider AS channel_provider, channels.status AS channel_status, " +
			"models.upstream_model AS upstream_model, models.status AS status, " +
			"models.context_length AS context_length, models.max_output_tokens AS max_output_tokens, " +
			"models.capabilities AS capabilities, models.ratio AS ratio, " +
			"bindings.model_price_id AS model_price_id, prices.name AS price_name").
		Joins("JOIN channels ON channels.id = models.channel_id AND channels.deleted_at IS NULL").
		Joins("LEFT JOIN model_price_bindings bindings ON bindings.model_id = models.id AND bindings.deleted_at IS NULL").
		Joins("LEFT JOIN model_prices prices ON prices.id = bindings.model_price_id").
		Order("models.name ASC, channels.priority DESC, channels.weight DESC").
		Scan(&rows).Error
	return rows, err
}

// ListEnabledModels 查询对外可用的逻辑模型名（按名称去重），供数据面 GET /v1/models 使用。
// 判定与 ListActiveByModel 保持一致：模型 active + 渠道 active + 双方未软删。
// 说明：按 createdAt 升序取出全部可用「渠道 × 模型」行后在 Go 内按名称去重，
// 首条即该逻辑模型的最早创建时间；避免 MIN(created_at) 聚合结果在不同数据库驱动下类型不一致。
func (r *channelRepo) ListEnabledModels(ctx context.Context) ([]model.EnabledModelInfo, error) {
	rows := make([]model.EnabledModelInfo, 0)
	err := r.db.WithContext(ctx).
		Table("models AS m").
		Select("m.name AS name, m.created_at AS created_at").
		Joins("JOIN channels c ON c.id = m.channel_id").
		Where("m.deleted_at IS NULL").
		Where("m.status = ?", model.ModelStatusActive).
		Where("c.deleted_at IS NULL").
		Where("c.status = ?", model.ChannelStatusActive).
		Order("m.name ASC, m.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 同名模型跨渠道去重：排序后相邻同名只保留首条（created_at 最早）
	deduped := make([]model.EnabledModelInfo, 0, len(rows))
	for _, row := range rows {
		if len(deduped) > 0 && deduped[len(deduped)-1].Name == row.Name {
			continue
		}
		deduped = append(deduped, row)
	}
	return deduped, nil
}
