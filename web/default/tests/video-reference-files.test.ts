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
import {
  getReferenceAudioMime,
  getReferenceImageMime,
  getReferenceMp4VideoMetadata,
  getReferenceVideoMime,
  isReferenceAudioFile,
  isReferenceImageFile,
  isSeedance25ReferenceVideoMetadata,
  isReferenceVideoFile,
} from '../src/features/creation-center/video-reference-files'

function bytes(...values: number[]) {
  return new Uint8Array(values)
}

function uint32(value: number) {
  const result = new Uint8Array(4)
  new DataView(result.buffer).setUint32(0, value)
  return result
}

function joinBytes(...parts: Uint8Array[]) {
  const result = new Uint8Array(
    parts.reduce((total, part) => total + part.byteLength, 0)
  )
  let offset = 0
  for (const part of parts) {
    result.set(part, offset)
    offset += part.byteLength
  }
  return result
}

function mp4Box(type: string, ...payload: Uint8Array[]) {
  const body = joinBytes(...payload)
  return joinBytes(
    uint32(body.byteLength + 8),
    new TextEncoder().encode(type),
    body
  )
}

function referenceMp4(codec: string, frameRate: number) {
  const timescale = 30_000
  const sampleDelta = Math.round(timescale / frameRate)
  const mdhd = mp4Box(
    'mdhd',
    bytes(0, 0, 0, 0),
    uint32(0),
    uint32(0),
    uint32(timescale),
    uint32(timescale * 10)
  )
  const hdlr = mp4Box(
    'hdlr',
    bytes(0, 0, 0, 0),
    uint32(0),
    new TextEncoder().encode('vide')
  )
  const stsd = mp4Box(
    'stsd',
    bytes(0, 0, 0, 0),
    uint32(1),
    mp4Box(codec)
  )
  const stts = mp4Box(
    'stts',
    bytes(0, 0, 0, 0),
    uint32(1),
    uint32(Math.round(frameRate * 10)),
    uint32(sampleDelta)
  )
  const moov = mp4Box(
    'moov',
    mp4Box('trak', mp4Box('mdia', mdhd, hdlr, mp4Box('minf', mp4Box('stbl', stsd, stts))))
  )
  return new Blob([mp4Box('ftyp'), moov], { type: 'video/mp4' })
}

describe('video reference files', () => {
  it('accepts images by browser mime type or file extension fallback', () => {
    expect(isReferenceImageFile({ name: 'photo.webp', type: '' })).toBe(true)
    expect(
      isReferenceImageFile({ name: 'photo.bin', type: 'image/png' })
    ).toBe(true)
    expect(
      isReferenceImageFile({ name: 'photo.bmp', type: 'image/bmp' })
    ).toBe(false)
    expect(isReferenceImageFile({ name: 'notes.txt', type: '' })).toBe(false)
  })

  it('limits Seedance 2.5 images to JPG, PNG, and WebP', () => {
    for (const filename of ['photo.jpg', 'photo.jpeg', 'photo.png', 'photo.webp']) {
      expect(
        isReferenceImageFile({ name: filename, type: '' }, 'seedance-2.5')
      ).toBe(true)
    }
    expect(
      isReferenceImageFile(
        { name: 'animation.gif', type: 'image/gif' },
        'seedance-2.5'
      )
    ).toBe(false)
    expect(
      isReferenceImageFile(
        { name: 'photo.avif', type: 'image/avif' },
        'seedance-2.5'
      )
    ).toBe(false)
  })

  it('detects the upload mime from file names when needed', () => {
    expect(getReferenceImageMime({ name: 'reference.jpg', type: '' })).toBe(
      'image/jpeg'
    )
    expect(
      getReferenceImageMime({
        name: 'reference.png',
        type: 'application/octet-stream',
      })
    ).toBe('image/png')
    expect(getReferenceVideoMime({ name: 'clip.mp4', type: '' })).toBe(
      'video/mp4'
    )
    expect(getReferenceAudioMime({ name: 'voice.MP3', type: '' })).toBe(
      'audio/mpeg'
    )
    expect(getReferenceAudioMime({ name: 'voice.wav', type: '' })).toBe(
      'audio/wav'
    )
  })

  it('accepts video and audio reference files by supported type or extension', () => {
    expect(isReferenceVideoFile({ name: 'clip.mp4', type: '' })).toBe(true)
    expect(
      isReferenceVideoFile({ name: 'clip.bin', type: 'video/mp4' })
    ).toBe(true)
    expect(isReferenceVideoFile({ name: 'clip.mov', type: '' })).toBe(false)
    expect(isReferenceAudioFile({ name: 'voice.mp3', type: '' })).toBe(true)
    expect(
      isReferenceAudioFile({ name: 'voice.bin', type: 'audio/wav' })
    ).toBe(true)
    expect(
      isReferenceAudioFile({ name: 'voice.bin', type: 'audio/mp3' })
    ).toBe(true)
    expect(isReferenceAudioFile({ name: 'voice.flac', type: '' })).toBe(false)
  })

  it('checks Seedance 2.5 MP4 codec and frame rate metadata', async () => {
    const valid = await getReferenceMp4VideoMetadata(
      referenceMp4('avc1', 30)
    )
    expect(valid?.codec).toBe('avc1')
    expect(valid?.frameRate).toBeCloseTo(30, 1)
    expect(isSeedance25ReferenceVideoMetadata(valid)).toBe(true)

    expect(
      isSeedance25ReferenceVideoMetadata(
        await getReferenceMp4VideoMetadata(referenceMp4('hvc1', 30))
      )
    ).toBe(false)
    expect(
      isSeedance25ReferenceVideoMetadata(
        await getReferenceMp4VideoMetadata(referenceMp4('avc1', 60))
      )
    ).toBe(false)
  })
})
