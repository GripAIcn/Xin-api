import apiClient, { handleApiResponse } from '@/lib/api-client'
import type {
  CreateModelPriceRequest,
  ModelPrice,
  ModelPriceWithCount,
  PriceBoundModelEntry,
  UpdateModelPriceRequest,
  UpdateModelPriceStatusRequest,
} from '@/types/api'

// 获取全部价格方案（含绑定模型数）
export function fetchPrices() {
  return apiClient.get('/v1/prices').then(res => handleApiResponse<ModelPriceWithCount[]>(res))
}

// 获取价格方案详情
export function fetchPrice(id: number) {
  return apiClient.get(`/v1/prices/${id}`).then(res => handleApiResponse<ModelPrice>(res))
}

// 新增价格方案
export function createPrice(data: CreateModelPriceRequest) {
  return apiClient.post('/v1/prices', data).then(res => handleApiResponse<ModelPrice>(res))
}

// 编辑价格方案
export function updatePrice(id: number, data: UpdateModelPriceRequest) {
  return apiClient.put(`/v1/prices/${id}`, data).then(res => handleApiResponse<ModelPrice>(res))
}

// 启用/禁用价格方案
export function updatePriceStatus(id: number, data: UpdateModelPriceStatusRequest) {
  return apiClient.put(`/v1/prices/${id}/status`, data).then(res => handleApiResponse<ModelPrice>(res))
}

// 删除价格方案
export function deletePrice(id: number) {
  return apiClient.delete(`/v1/prices/${id}`).then(res => handleApiResponse<null>(res))
}

// 查询绑定到指定价格方案的「渠道 × 模型」明细
export function fetchPriceModels(id: number) {
  return apiClient.get(`/v1/prices/${id}/models`).then(res => handleApiResponse<PriceBoundModelEntry[]>(res))
}
