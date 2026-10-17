import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { ExchangeRate, UpdateExchangeRateRequest } from '@/types/api'

// 管理员：获取 USD → CNY 结算汇率
export function fetchUSDCNY() {
  return apiClient
    .get('/v1/exchange-rates/usd-cny')
    .then(res => handleApiResponse<ExchangeRate>(res))
}

// 管理员：更新 USD → CNY 结算汇率
export function updateUSDCNY(data: UpdateExchangeRateRequest) {
  return apiClient
    .put('/v1/exchange-rates/usd-cny', data)
    .then(res => handleApiResponse<null>(res))
}
