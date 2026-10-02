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
import { API_ENDPOINTS } from './constants'
import {
  buildPlaygroundResponsesPayload,
  parsePlaygroundResponsesText,
} from './lib/responses'
import {
  buildPlaygroundMediaRequest,
  getPlaygroundMediaEndpoint,
  parsePlaygroundExtraBody,
  parsePlaygroundMediaResult,
  resolvePlaygroundMode,
  type PlaygroundMediaResult,
} from './lib/media-routing'
import type {
  ChatCompletionRequest,
  ChatCompletionResponse,
  Message,
  ModelOption,
  GroupOption,
  PlaygroundConfig,
} from './types'

/**
 * Send chat completion request (non-streaming)
 */
export async function sendChatCompletion(
  payload: ChatCompletionRequest
): Promise<ChatCompletionResponse> {
  const res = await api.post(API_ENDPOINTS.CHAT_COMPLETIONS, payload, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return res.data
}

export async function sendPlaygroundResponses(
  config: PlaygroundConfig,
  messages: Message[]
): Promise<string> {
  const payload = buildPlaygroundResponsesPayload(
    config.model,
    config.group,
    messages,
    parsePlaygroundExtraBody(config.extraBody)
  )
  const response = await api.post(API_ENDPOINTS.RESPONSES, payload, {
    skipErrorHandler: true,
  } as Record<string, unknown>)
  return parsePlaygroundResponsesText(response.data)
}

/**
 * Send image/video generation request for media-only playground models.
 */
export async function sendPlaygroundMediaGeneration(
  config: PlaygroundConfig,
  messages: Message[]
): Promise<PlaygroundMediaResult> {
  const endpoint = getPlaygroundMediaEndpoint(
    config.model,
    config.mode,
    config.imageEndpoint,
    config.videoEndpoint
  )
  const extraBody = parsePlaygroundExtraBody(config.extraBody)
  delete extraBody.group
  const payload = buildPlaygroundMediaRequest(
    config.model,
    messages,
    config.mode,
    config.imageEndpoint,
    extraBody
  )
  if (!endpoint || !payload) {
    throw new Error('Current model does not support media generation')
  }

  const res = await api.post(endpoint, payload, {
    skipErrorHandler: true,
    headers: { 'X-Playground-Group': config.group },
  } as Record<string, unknown>)
  const initialResult = parsePlaygroundMediaResult(
    res.data,
    config.model,
    config.mode
  )

  if (
    !initialResult.taskId ||
    initialResult.mediaUrl ||
    (resolvePlaygroundMode(config.model, config.mode) === 'image' &&
      endpoint !== API_ENDPOINTS.IMAGE_ASYNC_GENERATIONS)
  ) {
    return initialResult
  }

  return pollPlaygroundMediaTask(
    config.model,
    config.mode,
    endpoint,
    initialResult
  )
}

async function pollPlaygroundMediaTask(
  model: string,
  selectedMode: PlaygroundConfig['mode'],
  endpoint: string,
  initialResult: PlaygroundMediaResult
): Promise<PlaygroundMediaResult> {
  const taskId = initialResult.taskId
  if (!taskId) return initialResult

  let latestResult = initialResult
  const path = endpoint
  for (let attempt = 0; attempt < 45; attempt += 1) {
    await delay(4000)
    const response = await api.get(`${path}/${encodeURIComponent(taskId)}`, {
      skipErrorHandler: true,
      disableDuplicate: true,
    } as Record<string, unknown>)
    latestResult = parsePlaygroundMediaResult(
      response.data,
      model,
      selectedMode
    )
    if (latestResult.mediaUrl || isTerminalMediaStatus(latestResult.status)) {
      return latestResult
    }
  }

  return latestResult
}

function isTerminalMediaStatus(status: string | undefined) {
  switch (status?.toLowerCase()) {
    case 'completed':
    case 'failed':
    case 'cancelled':
    case 'canceled':
      return true
    default:
      return false
  }
}

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

/**
 * Get user available models
 */
export async function getUserModels(): Promise<ModelOption[]> {
  const res = await api.get(API_ENDPOINTS.USER_MODELS)
  const { data } = res

  if (!data.success || !Array.isArray(data.data)) {
    return []
  }

  return data.data.map((model: string) => ({
    label: model,
    value: model,
  }))
}

/**
 * Get user groups
 */
export async function getUserGroups(): Promise<GroupOption[]> {
  const res = await api.get(API_ENDPOINTS.USER_GROUPS)
  const { data } = res

  if (!data.success || !data.data) {
    return []
  }

  const groupData = data.data as Record<string, { desc: string; ratio: number }>

  // label is for button display (name only); desc is for dropdown content
  return Object.entries(groupData).map(([group, info]) => ({
    label: group,
    value: group,
    ratio: info.ratio,
    desc: info.desc,
  }))
}
