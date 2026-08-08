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
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))
const zhLocale = JSON.parse(
  readFileSync(join(__dirname, '../src/i18n/locales/zh.json'), 'utf8')
)

describe('Chinese navigation labels', () => {
  it('shows the playground navigation as model testing', () => {
    assert.equal(zhLocale.translation.Playground, '模型测试')
  })

  it('keeps Seedance upload prompts and validation messages in Chinese', () => {
    assert.equal(
      zhLocale.translation[
        'Tip: Reference videos support MP4. Up to {{count}} videos, {{size}} MB each.'
      ],
      '提示：参考视频支持 MP4，最多 {{count}} 个，每个不超过 {{size}} MB。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance accepts at most 9 image references.'
      ],
      'Seedance 最多支持 9 张参考图片。'
    )
    assert.equal(
      zhLocale.translation['Seedance accepts too many reference assets.'],
      'Seedance 参考素材总数超过模型限制。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 tip: Upload up to {{imageCount}} reference images. Each image must not exceed {{imageSize}} MB.'
      ],
      '提示：最多上传 {{imageCount}} 张参考图片，每张不能超过 {{imageSize}} MB。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 tip: Upload up to {{videoCount}} MP4 videos, no more than {{videoSize}} MB each and {{videoTotalSize}} MB in total. Total reference video duration must not exceed 29 seconds.'
      ],
      '提示：最多上传 {{videoCount}} 个 MP4 参考视频，每个不能超过 {{videoSize}} MB，总大小不能超过 {{videoTotalSize}} MB，总时长不能超过 29 秒。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 reference audios must not exceed 29 seconds in total.'
      ],
      'Seedance 2.5 参考音频总时长不能超过 29 秒。'
    )
    assert.equal(
      zhLocale.translation[
        'Reference audios must not exceed {{size}} MB total.'
      ],
      '参考音频总大小不能超过 {{size}} MB。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 tip: Upload up to {{imageCount}} images, {{videoCount}} videos, and {{audioCount}} audio files. Images must not exceed {{imageSize}} MB each. Videos must not exceed {{videoSize}} MB each or {{videoTotalSize}} MB in total. Audio must not exceed {{audioSize}} MB each or {{audioTotalSize}} MB in total. Total reference video and audio duration must not exceed 29 seconds respectively.'
      ],
      '提示：最多上传 {{imageCount}} 张图片、{{videoCount}} 个视频和 {{audioCount}} 个音频。图片每张不能超过 {{imageSize}} MB；视频每个不能超过 {{videoSize}} MB、总大小不能超过 {{videoTotalSize}} MB；音频每个不能超过 {{audioSize}} MB、总大小不能超过 {{audioTotalSize}} MB。参考视频和参考音频的总时长分别不能超过 29 秒。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 duration must be between 4 and 29 seconds.'
      ],
      'Seedance 2.5 时长必须在 4 到 29 秒之间。'
    )
  })
})
