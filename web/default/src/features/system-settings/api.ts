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
import type {
  ConfirmPaymentComplianceResponse,
  DeleteLogsResponse,
  FetchUpstreamRatiosRequest,
  SystemOptionsResponse,
  UpdateOptionRequest,
  UpdateOptionResponse,
  MediaStorageProvider,
  MediaStorageSettingsResponse,
  MediaStorageTestResponse,
  MediaTransferDashboardResponse,
  UpstreamChannelsResponse,
  UpstreamRatiosResponse,
} from './types'

export async function getSystemOptions() {
  const res = await api.get<SystemOptionsResponse>('/api/option/')
  return res.data
}

export async function updateSystemOption(request: UpdateOptionRequest) {
  const res = await api.put<UpdateOptionResponse>('/api/option/', request)
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to update setting')
  }
  return res.data
}

export async function getMediaStorageSettings() {
  const res = await api.get<MediaStorageSettingsResponse>(
    '/api/option/media_storage'
  )
  return res.data
}

export async function updateMediaStorageSettings(
  providers: MediaStorageProvider[]
) {
  const res = await api.put<UpdateOptionResponse>('/api/option/media_storage', {
    providers,
  })
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to update media storage')
  }
  return res.data
}

export async function testMediaStorageProvider(providerId: string) {
  try {
    const res = await api.post<MediaStorageTestResponse>(
      '/api/option/media_storage/test',
      { provider_id: providerId },
      {
        skipErrorHandler: true,
        skipBusinessError: true,
      } as Record<string, unknown>
    )
    if (!res.data.success) {
      throw new Error(res.data.message || 'Media storage test failed')
    }
    return res.data
  } catch (error) {
    const requestError = error as {
      message?: string
      response?: {
        status?: number
        data?: { message?: string } | string
      }
    }
    const responseMessage =
      typeof requestError.response?.data === 'object'
        ? requestError.response.data?.message
        : undefined
    const status = requestError.response?.status
    throw new Error(
      responseMessage ||
        (status
          ? `Media storage test failed (HTTP ${status})`
          : requestError.message || 'Media storage test failed'),
      { cause: error }
    )
  }
}

export async function getMediaTransferDashboard(status?: string) {
  const res = await api.get<MediaTransferDashboardResponse>(
    '/api/option/media_storage/transfers',
    { params: status ? { status } : undefined }
  )
  return res.data
}

export async function retryMediaTransferJob(id: number) {
  const res = await api.post(
    '/api/option/media_storage/transfers/' + id + '/retry'
  )
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to retry media transfer')
  }
  return res.data
}

export async function confirmPaymentCompliance() {
  const res = await api.post<ConfirmPaymentComplianceResponse>(
    '/api/option/payment_compliance',
    { confirmed: true }
  )
  return res.data
}

export async function deleteLogsBefore(targetTimestamp: number) {
  const res = await api.delete<DeleteLogsResponse>('/api/log/', {
    params: { target_timestamp: targetTimestamp },
  })
  return res.data
}

export async function resetModelRatios() {
  const res = await api.post<UpdateOptionResponse>(
    '/api/option/rest_model_ratio'
  )
  return res.data
}

export async function getUpstreamChannels() {
  const res = await api.get<UpstreamChannelsResponse>(
    '/api/ratio_sync/channels'
  )
  return res.data
}

export async function fetchUpstreamRatios(request: FetchUpstreamRatiosRequest) {
  const res = await api.post<UpstreamRatiosResponse>(
    '/api/ratio_sync/fetch',
    request
  )
  return res.data
}
