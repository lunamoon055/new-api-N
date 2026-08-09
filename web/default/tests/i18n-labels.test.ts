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

  it('shows the requested Creation Center model prompt', () => {
    assert.equal(
      zhLocale.translation['Select a generation model'],
      '请选择生成模型'
    )
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
        'Seedance 2.5 tip: Upload up to {{imageCount}} JPG, PNG, or WebP reference images. Each image must not exceed {{imageSize}} MB. Recommended resolution: 1080p to 4K.'
      ],
      '提示：最多上传 {{imageCount}} 张 JPG、PNG 或 WebP 参考图片，每张不能超过 {{imageSize}} MB，建议分辨率为 1080p～4K。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 tip: Upload up to {{videoCount}} MP4 (H.264) videos at 24, 25, or 30 fps. Each video must be 2-30 seconds and no more than {{videoSize}} MB; all reference videos together must not exceed 30 seconds.'
      ],
      '提示：最多上传 {{videoCount}} 个 MP4（H.264）参考视频，帧率支持 24/25/30fps；单个 2～30 秒且不超过 {{videoSize}} MB；所有参考视频总时长不能超过 30 秒。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 reference audios must not exceed 30 seconds in total.'
      ],
      'Seedance 2.5 参考音频总时长不能超过 30 秒。'
    )
    assert.equal(
      zhLocale.translation[
        'Reference audios must not exceed {{size}} MB total.'
      ],
      '参考音频总大小不能超过 {{size}} MB。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 tip: Images support JPG, PNG, or WebP, up to {{imageCount}}, {{imageSize}} MB each, recommended 1080p to 4K. Videos support MP4 (H.264) at 24, 25, or 30 fps, up to {{videoCount}}, 2-30 seconds and {{videoSize}} MB each, 30 seconds total. Audio supports MP3 or WAV, up to {{audioCount}}, 2-30 seconds and {{audioSize}} MB each, 30 seconds total.'
      ],
      '提示：图片支持 JPG、PNG 或 WebP，最多 {{imageCount}} 张，每张不超过 {{imageSize}} MB，建议 1080p～4K；视频支持 MP4（H.264）、24/25/30fps，最多 {{videoCount}} 个，单个 2～30 秒且不超过 {{videoSize}} MB，总时长不超过 30 秒；音频支持 MP3 或 WAV，最多 {{audioCount}} 个，单个 2～30 秒且不超过 {{audioSize}} MB，总时长不超过 30 秒。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 reference videos must be MP4 (H.264) at 24, 25, or 30 fps.'
      ],
      'Seedance 2.5 参考视频必须为 MP4（H.264），帧率仅支持 24、25 或 30fps。'
    )
    assert.equal(
      zhLocale.translation[
        'Seedance 2.5 duration must be between 4 and 29 seconds.'
      ],
      'Seedance 2.5 时长必须在 4 到 29 秒之间。'
    )
  })
})
