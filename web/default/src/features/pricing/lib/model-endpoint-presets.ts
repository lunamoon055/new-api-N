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
import type { PricingEndpoint } from '../types'

export type ModelEndpointPreset = PricingEndpoint & {
  type: string
  labelKey: string
  kind: 'video' | 'image'
}

const VIDEO_ASYNC: ModelEndpointPreset = {
  type: 'video-async',
  labelKey: 'Async video',
  kind: 'video',
  path: '/v1/video/async-generations',
  method: 'POST',
  query_path: '/v1/video/async-generations/{task_id}',
  query_method: 'GET',
}

const VIDEO_GENERATIONS: ModelEndpointPreset = {
  type: 'video-generations',
  labelKey: 'Video generations',
  kind: 'video',
  path: '/v1/video/generations',
  method: 'POST',
  query_path: '/v1/video/generations/{task_id}',
  query_method: 'GET',
}

const VIDEO_VIDEOS: ModelEndpointPreset = {
  type: 'video-videos',
  labelKey: 'Video tasks',
  kind: 'video',
  path: '/v1/videos',
  method: 'POST',
  query_path: '/v1/videos/{task_id}',
  query_method: 'GET',
}

function startsWithAny(value: string, prefixes: string[]): boolean {
  return prefixes.some((prefix) => value.startsWith(prefix))
}

/**
 * Return endpoint presets documented for model families that may not yet have
 * an explicit endpoint entry in the model metadata. These are display-only
 * fallbacks; a model's configured endpoint metadata remains authoritative for
 * all models not covered here.
 */
export function getModelEndpointPreset(
  modelName?: string
): ModelEndpointPreset | null {
  const model = modelName?.trim().toLowerCase() ?? ''
  if (!model) return null

  // 官转文档明确使用 /v1/video/generations。
  if (model.startsWith('官转')) return VIDEO_GENERATIONS

  // 004 系列文档、全能系列文档和统一视频文档使用 /v1/videos。
  if (
    model.startsWith('004系列/') ||
    model.startsWith('omni-video-') ||
    model.startsWith('videos-') ||
    model === 'sd-2-c1' ||
    model === 'sd-mini' ||
    model === 'sd-2.5' ||
    model === 'grok-imagine-video-1.5' ||
    model.startsWith('wan3.0(') ||
    model.startsWith('wan3.0（') ||
    startsWithAny(model, ['wan3.0-video', 'wan3.0-image', 'sd2', 'sd-2-'])
  ) {
    return VIDEO_VIDEOS
  }

  // Seedance 2.x 使用统一视频任务路径；Linksky 文档中的这些模型使用异步路径。
  if (
    startsWithAny(model, [
      'seedance2.0',
      'seedance2.5',
      'seedance-2.0',
      'seedance-2.5',
    ])
  ) {
    return VIDEO_VIDEOS
  }

  // 异步视频模型 - 使用 /v1/video/async-generations
  if (
    startsWithAny(model, [
      'sora2',
      'sora-2',
      'sora',
      'veo31',
      'veo',
      'kling-v3',
      'kling',
      'grok-imagine-video',
      'video-2.0',
      'video-2.5',
      'minimax-h3',
    ]) ||
    model === 'ko3' ||
    model === 'wan3.0'
  ) {
    return VIDEO_ASYNC
  }

  return null
}
