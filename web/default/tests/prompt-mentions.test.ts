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
import { getPromptMentionTrigger } from '../src/features/creation-center/lib/prompt-mentions'

describe('prompt mention helpers', () => {
  it('opens a reference mention after normal Chinese prompt text', () => {
    const prompt = '@参考图片1里的人物参考@'
    const start = prompt.lastIndexOf('@')

    expect(getPromptMentionTrigger(prompt, prompt.length)).toEqual({
      start,
      end: prompt.length,
      query: '',
    })
  })
})
