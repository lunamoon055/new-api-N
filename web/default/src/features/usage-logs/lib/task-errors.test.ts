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
import { describe, test } from 'node:test'
import { getTaskFailureDetails } from './task-errors'

const taskFailure = {
  fail_reason: '参考图片链接不存在或已失效，请重新上传图片后再试。',
  raw_fail_reason:
    '{"error":{"message":"invalid image_urls[0]: image url returned 404"}}',
}

describe('task failure details visibility', () => {
  test('returns both downstream and upstream messages to a super administrator', () => {
    assert.deepEqual(getTaskFailureDetails(taskFailure, true), {
      downstreamMessage: '参考图片链接不存在或已失效，请重新上传图片后再试。',
      upstreamRawError:
        '{"error":{"message":"invalid image_urls[0]: image url returned 404"}}',
    })
  })

  test('never returns the upstream raw error to other viewers', () => {
    assert.deepEqual(getTaskFailureDetails(taskFailure, false), {
      downstreamMessage: '参考图片链接不存在或已失效，请重新上传图片后再试。',
      upstreamRawError: undefined,
    })
  })

  test('does not retranslate an already-Chinese upstream message', () => {
    const rawMessage = '内容触发安全审核或版权限制，请调整输入内容或素材后重试'

    assert.deepEqual(
      getTaskFailureDetails(
        {
          fail_reason:
            '视频生成失败，请稍后重试；如持续失败，请提供任务 ID 联系管理员。',
          raw_fail_reason: rawMessage,
        },
        true
      ),
      {
        downstreamMessage: rawMessage,
        upstreamRawError: rawMessage,
      }
    )
  })

  test('extracts a Chinese message from a structured upstream error', () => {
    const rawMessage = JSON.stringify({
      error: {
        message: '内容触发安全审核或版权限制，请调整输入内容或素材后重试',
      },
    })

    assert.equal(
      getTaskFailureDetails(
        {
          fail_reason:
            '视频生成失败，请稍后重试；如持续失败，请提供任务 ID 联系管理员。',
          raw_fail_reason: rawMessage,
        },
        true
      ).downstreamMessage,
      '内容触发安全审核或版权限制，请调整输入内容或素材后重试'
    )
  })

  test('keeps the unified insufficient-credit message', () => {
    assert.equal(
      getTaskFailureDetails(
        {
          fail_reason:
            '视频生成失败，请稍后重试；如持续失败，请提供任务 ID 联系管理员。',
          raw_fail_reason: '上游积分不足，请充值后重试',
        },
        true
      ).downstreamMessage,
      '积分不足，请联系管理员'
    )
  })
})
