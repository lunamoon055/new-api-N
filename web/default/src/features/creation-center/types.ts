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
export type CreationMode = 'chat' | 'image' | 'video'

export type CreationModel = {
  id: string
  description?: string
  manual_description?: string
  icon?: string
  tags?: string[]
  vendor_id?: number
  cost?: CreationModelCost
  metadata?: CreationModelMetadata
  supported_endpoint_types: string[]
}

export type CreationModelMetadata = {
  provider?: string
  id?: string
  upstream_model_id?: string
  name?: string
  description?: string
  type?: string
  category?: string
  model_type?: string
  endpoint?: string
  legacy_endpoint?: string
  billing?: string
  resolutions?: string[]
  ratios?: string[]
  aspect_ratios?: string[]
  sizes?: string[]
  durations?: Array<number | string>
  max_prompt_length?: number
  max_images?: number
  max_videos?: number
  max_audios?: number
  max_media_files?: number
  max_image_size_mb?: number
  max_video_size_mb?: number
  max_audio_size_mb?: number
  min_reference_video_duration_seconds?: number
  max_reference_video_duration_seconds?: number
  concurrency_options?: number[]
  media_inputs?: string[]
}

export type CreationModelCost = {
  billing_mode:
    | 'per_token'
    | 'per_request'
    | 'dynamic'
    | 'tiered_seconds'
    | 'tiered_request'
  input_price_per_million?: number
  output_price_per_million?: number
  request_price?: number
  request_quota?: number
  video_billing_mode?: 'dynamic' | 'fixed' | 'tiered_seconds' | 'tiered_request'
  video_resolution_prices?: Record<string, number>
  video_resolution_quotas?: Record<string, number>
  group_ratio?: number
}

export type CreationModelGroup = {
  mode: CreationMode
  models: CreationModel[]
}

export type CreationVendor = {
  id: number
  name: string
  description?: string
  icon?: string
}

export type CreationModelCatalog = {
  modes: CreationModelGroup[]
  vendors: CreationVendor[]
}

export type CreationModelCategories = Partial<Record<string, CreationMode>>

export type CreationModelDescriptions = Partial<Record<string, string>>

export type CreationAsset = {
  id: string
  name: string
  type: string
  size: number
  text?: string
  dataUrl?: string
}

export type CreationCatalogResponse = {
  success: boolean
  message?: string
  data?: CreationModelCatalog
}

export type CreationView = 'preview' | 'assets' | 'history'

export type CreationResultStatus =
  'queued' | 'processing' | 'completed' | 'failed' | 'unknown'

export type CreationResult = {
  mode: CreationMode
  model: string
  id?: string
  taskId?: string
  createdAt?: number
  estimateSeconds?: number
  duration?: string
  resolution?: string
  status: CreationResultStatus
  outputText?: string
  imageUrl?: string
  videoUrl?: string
  error?: string
  raw?: unknown
}
