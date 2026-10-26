/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'
import type { PerformanceMetricsData, PerfSummaryAllData } from './types'

export type SuccessRateVisibilityResponse = {
  success: boolean
  message?: string
  data: { show_success_rate: boolean }
}

export async function getSuccessRateVisibility(): Promise<SuccessRateVisibilityResponse> {
  const response = await api.get<SuccessRateVisibilityResponse>(
    '/api/perf-metrics/success-rate-visibility'
  )
  if (!response.data.success) throw new Error(response.data.message)
  return response.data
}

export async function updateSuccessRateVisibility(
  showSuccessRate: boolean
): Promise<SuccessRateVisibilityResponse> {
  const response = await api.put<SuccessRateVisibilityResponse>(
    '/api/perf-metrics/success-rate-visibility',
    { show_success_rate: showSuccessRate }
  )
  if (!response.data.success) throw new Error(response.data.message)
  return response.data
}

export async function getPerfMetricsSummary(
  hours = 24
): Promise<PerfSummaryAllData> {
  const res = await api.get<PerfSummaryAllData>('/api/perf-metrics/summary', {
    params: { hours },
  })
  return res.data
}

export async function getPerfMetrics(
  modelName: string,
  hours = 24
): Promise<PerformanceMetricsData> {
  const res = await api.get<PerformanceMetricsData>('/api/perf-metrics', {
    params: {
      model: modelName,
      hours,
    },
  })
  return res.data
}
