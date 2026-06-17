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
import { describe, expect, it } from 'bun:test'
import { getTaskLogVideoPreviewUrl } from '../src/features/usage-logs/lib/task-preview'
import type { TaskLog } from '../src/features/usage-logs/types'

function createTaskLog(overrides: Partial<TaskLog>): TaskLog {
  return {
    id: 1,
    user_id: 1,
    platform: 'sora',
    task_id: 'task_video_123',
    action: 'generate',
    channel_id: 1,
    submit_time: 1,
    status: 'SUCCESS',
    ...overrides,
  }
}

describe('task log video preview helpers', () => {
  it('uses a direct result url for successful video tasks when one is available', () => {
    expect(
      getTaskLogVideoPreviewUrl(
        createTaskLog({
          result_url: 'https://example.com/upstream-video.mp4',
        })
      )
    ).toBe('https://example.com/upstream-video.mp4')
  })

  it('uses the authenticated video proxy when successful video tasks only have a task id', () => {
    expect(
      getTaskLogVideoPreviewUrl(
        createTaskLog({
          result_url: '',
        })
      )
    ).toBe('/v1/videos/task_video_123/content')
  })

  it('uses the authenticated video proxy for upstream api content urls', () => {
    expect(
      getTaskLogVideoPreviewUrl(
        createTaskLog({
          result_url:
            'https://api.example.com/v1/video/async-generations/upstream_123/content',
        })
      )
    ).toBe('/v1/videos/task_video_123/content')
  })

  it('does not expose preview links for unfinished tasks', () => {
    expect(
      getTaskLogVideoPreviewUrl(
        createTaskLog({
          status: 'IN_PROGRESS',
          result_url: 'https://example.com/upstream-video.mp4',
        })
      )
    ).toBeNull()
  })
})
