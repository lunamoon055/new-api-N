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
import {
  getReferenceAudioMime,
  isReferenceImageFile,
  isReferenceAudioFile,
} from './video-reference-files'

describe('Creation Center reference audio formats', () => {
  test('keeps Seedance 2.5 images limited to JPG, PNG, and WebP', () => {
    assert.equal(
      isReferenceImageFile(
        { name: 'reference.webp', type: '' },
        'seedance-2.5'
      ),
      true
    )
    assert.equal(
      isReferenceImageFile(
        { name: 'reference.gif', type: 'image/gif' },
        'seedance-2.5'
      ),
      false
    )
  })

  test('keeps legacy models limited to MP3 and WAV', () => {
    assert.equal(
      getReferenceAudioMime({ name: 'reference.mp3', type: '' }),
      'audio/mpeg'
    )
    assert.equal(
      isReferenceAudioFile({ name: 'reference.m4a', type: 'audio/mp4' }),
      false
    )
  })

  test('accepts every MiniMax H3 audio format from the integration document', () => {
    for (const [filename, mime] of [
      ['reference.mp3', 'audio/mpeg'],
      ['reference.wav', 'audio/wav'],
      ['reference.m4a', 'audio/mp4'],
      ['reference.aac', 'audio/aac'],
      ['reference.ogg', 'audio/ogg'],
      ['reference.webm', 'audio/webm'],
    ] as const) {
      assert.equal(
        getReferenceAudioMime({ name: filename, type: '' }, 'minimax-h3'),
        mime
      )
      assert.equal(
        isReferenceAudioFile({ name: filename, type: mime }, 'minimax-h3'),
        true
      )
    }
  })
})
