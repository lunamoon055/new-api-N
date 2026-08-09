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

const IMAGE_MIME_BY_EXTENSION: Record<string, string> = {
  avif: 'image/avif',
  gif: 'image/gif',
  jpeg: 'image/jpeg',
  jpg: 'image/jpeg',
  png: 'image/png',
  webp: 'image/webp',
}

const SEEDANCE_25_IMAGE_MIME_BY_EXTENSION: Record<string, string> = {
  jpeg: 'image/jpeg',
  jpg: 'image/jpeg',
  png: 'image/png',
  webp: 'image/webp',
}

const VIDEO_MIME_BY_EXTENSION: Record<string, string> = {
  mp4: 'video/mp4',
}

const AUDIO_MIME_BY_EXTENSION: Record<string, string> = {
  mp3: 'audio/mpeg',
  wav: 'audio/wav',
}

const MINIMAX_H3_AUDIO_MIME_BY_EXTENSION: Record<string, string> = {
  ...AUDIO_MIME_BY_EXTENSION,
  m4a: 'audio/mp4',
  aac: 'audio/aac',
  ogg: 'audio/ogg',
  webm: 'audio/webm',
}

const AUDIO_MIME_TYPES = [
  'audio/mpeg',
  'audio/mp3',
  'audio/wav',
  'audio/wave',
  'audio/x-wav',
]

const MINIMAX_H3_AUDIO_MIME_TYPES = [
  ...AUDIO_MIME_TYPES,
  'audio/mp4',
  'audio/x-m4a',
  'audio/aac',
  'audio/x-aac',
  'audio/ogg',
  'application/ogg',
  'audio/webm',
]

export type ReferenceAudioProfile = 'default' | 'minimax-h3'
export type ReferenceImageProfile = 'default' | 'seedance-2.5'

type ReferenceFileLike = Pick<File, 'name' | 'type'>
type ReferenceVideoBlobLike = Pick<File, 'size' | 'slice'>

export type ReferenceMp4VideoMetadata = {
  codec: string
  frameRate: number
}

type Mp4Box = {
  type: string
  start: number
  contentStart: number
  end: number
}

function getReferenceMime(
  file: ReferenceFileLike,
  mimeByExtension: Record<string, string>,
  mimeTypes = Object.values(mimeByExtension)
) {
  const extension = file.name.split('.').pop()?.toLowerCase()
  const mime = extension ? mimeByExtension[extension] : undefined
  if (mime) return mime
  return mimeTypes.includes(file.type) ? file.type : undefined
}

export function getReferenceImageMime(
  file: ReferenceFileLike,
  profile: ReferenceImageProfile = 'default'
) {
  return getReferenceMime(
    file,
    profile === 'seedance-2.5'
      ? SEEDANCE_25_IMAGE_MIME_BY_EXTENSION
      : IMAGE_MIME_BY_EXTENSION
  )
}

export function getReferenceVideoMime(file: ReferenceFileLike) {
  return getReferenceMime(file, VIDEO_MIME_BY_EXTENSION)
}

export function getReferenceAudioMime(
  file: ReferenceFileLike,
  profile: ReferenceAudioProfile = 'default'
) {
  return profile === 'minimax-h3'
    ? getReferenceMime(
        file,
        MINIMAX_H3_AUDIO_MIME_BY_EXTENSION,
        MINIMAX_H3_AUDIO_MIME_TYPES
      )
    : getReferenceMime(file, AUDIO_MIME_BY_EXTENSION, AUDIO_MIME_TYPES)
}

export function isReferenceImageFile(
  file: ReferenceFileLike,
  profile: ReferenceImageProfile = 'default'
) {
  return !!getReferenceImageMime(file, profile)
}

export function isReferenceVideoFile(file: ReferenceFileLike) {
  return !!getReferenceVideoMime(file)
}

export function isReferenceAudioFile(
  file: ReferenceFileLike,
  profile: ReferenceAudioProfile = 'default'
) {
  return !!getReferenceAudioMime(file, profile)
}

async function getReferenceMediaDurationSeconds(
  file: File,
  kind: 'audio' | 'video'
) {
  if (
    typeof document === 'undefined' ||
    typeof URL === 'undefined' ||
    typeof URL.createObjectURL !== 'function'
  ) {
    return undefined
  }

  const objectURL = URL.createObjectURL(file)
  return new Promise<number | undefined>((resolve) => {
    const media = document.createElement(kind)
    let settled = false
    const finish = (duration?: number) => {
      if (settled) return
      settled = true
      clearTimeout(timeout)
      media.removeAttribute('src')
      URL.revokeObjectURL(objectURL)
      resolve(duration)
    }
    const timeout = setTimeout(() => finish(), 10_000)
    media.preload = 'metadata'
    media.onloadedmetadata = () =>
      finish(Number.isFinite(media.duration) ? media.duration : undefined)
    media.onerror = () => finish()
    media.src = objectURL
  })
}

export function getReferenceAudioDurationSeconds(file: File) {
  return getReferenceMediaDurationSeconds(file, 'audio')
}

export function getReferenceVideoDurationSeconds(file: File) {
  return getReferenceMediaDurationSeconds(file, 'video')
}

function readFourCC(view: DataView, offset: number) {
  if (offset < 0 || offset + 4 > view.byteLength) return ''
  return String.fromCharCode(
    view.getUint8(offset),
    view.getUint8(offset + 1),
    view.getUint8(offset + 2),
    view.getUint8(offset + 3)
  )
}

function readMp4Box(view: DataView, offset: number, limit: number) {
  if (offset < 0 || offset + 8 > limit || limit > view.byteLength) {
    return undefined
  }

  const size32 = view.getUint32(offset)
  const type = readFourCC(view, offset + 4)
  let headerSize = 8
  let size = size32

  if (size32 === 1) {
    if (offset + 16 > limit) return undefined
    const high = view.getUint32(offset + 8)
    const low = view.getUint32(offset + 12)
    size = high * 2 ** 32 + low
    headerSize = 16
  } else if (size32 === 0) {
    size = limit - offset
  }

  if (
    !type ||
    !Number.isSafeInteger(size) ||
    size < headerSize ||
    offset + size > limit
  ) {
    return undefined
  }

  return {
    type,
    start: offset,
    contentStart: offset + headerSize,
    end: offset + size,
  } satisfies Mp4Box
}

function getMp4Children(view: DataView, parent: Mp4Box) {
  const children: Mp4Box[] = []
  let offset = parent.contentStart
  while (offset + 8 <= parent.end) {
    const child = readMp4Box(view, offset, parent.end)
    if (!child) break
    children.push(child)
    if (child.end <= offset) break
    offset = child.end
  }
  return children
}

function getMp4Child(view: DataView, parent: Mp4Box, type: string) {
  return getMp4Children(view, parent).find((box) => box.type === type)
}

function getMp4TrackTimescale(view: DataView, mdhd: Mp4Box) {
  if (mdhd.contentStart + 4 > mdhd.end) return undefined
  const version = view.getUint8(mdhd.contentStart)
  const timescaleOffset = mdhd.contentStart + (version === 1 ? 20 : 12)
  if (timescaleOffset + 4 > mdhd.end) return undefined
  const timescale = view.getUint32(timescaleOffset)
  return timescale > 0 ? timescale : undefined
}

function getMp4SampleEntryCodec(view: DataView, stsd: Mp4Box) {
  const entryCountOffset = stsd.contentStart + 4
  const firstEntryOffset = stsd.contentStart + 8
  if (entryCountOffset + 4 > stsd.end || firstEntryOffset + 8 > stsd.end) {
    return undefined
  }
  if (view.getUint32(entryCountOffset) < 1) return undefined
  const firstEntry = readMp4Box(view, firstEntryOffset, stsd.end)
  return firstEntry?.type
}

function getMp4AverageFrameRate(
  view: DataView,
  stts: Mp4Box,
  timescale: number
) {
  const entryCountOffset = stts.contentStart + 4
  if (entryCountOffset + 4 > stts.end) return undefined
  const entryCount = view.getUint32(entryCountOffset)
  let offset = stts.contentStart + 8
  let sampleCount = 0
  let totalTicks = 0

  for (let index = 0; index < entryCount; index += 1) {
    if (offset + 8 > stts.end) return undefined
    const count = view.getUint32(offset)
    const delta = view.getUint32(offset + 4)
    sampleCount += count
    totalTicks += count * delta
    offset += 8
  }

  if (sampleCount <= 0 || totalTicks <= 0) return undefined
  return (sampleCount * timescale) / totalTicks
}

function parseReferenceMp4VideoMetadata(buffer: ArrayBuffer) {
  const view = new DataView(buffer)
  const moov = readMp4Box(view, 0, view.byteLength)
  if (!moov || moov.type !== 'moov') return undefined

  for (const track of getMp4Children(view, moov).filter(
    (box) => box.type === 'trak'
  )) {
    const mdia = getMp4Child(view, track, 'mdia')
    if (!mdia) continue
    const hdlr = getMp4Child(view, mdia, 'hdlr')
    if (!hdlr || readFourCC(view, hdlr.contentStart + 8) !== 'vide') continue

    const mdhd = getMp4Child(view, mdia, 'mdhd')
    const minf = getMp4Child(view, mdia, 'minf')
    const stbl = minf ? getMp4Child(view, minf, 'stbl') : undefined
    const stsd = stbl ? getMp4Child(view, stbl, 'stsd') : undefined
    const stts = stbl ? getMp4Child(view, stbl, 'stts') : undefined
    if (!mdhd || !stsd || !stts) continue

    const timescale = getMp4TrackTimescale(view, mdhd)
    const codec = getMp4SampleEntryCodec(view, stsd)
    const frameRate = timescale
      ? getMp4AverageFrameRate(view, stts, timescale)
      : undefined
    if (!codec || !frameRate || !Number.isFinite(frameRate)) continue
    return { codec, frameRate } satisfies ReferenceMp4VideoMetadata
  }

  return undefined
}

async function findMp4MoovBox(file: ReferenceVideoBlobLike) {
  let offset = 0
  while (offset + 8 <= file.size) {
    const headerBuffer = await file
      .slice(offset, Math.min(offset + 16, file.size))
      .arrayBuffer()
    const headerView = new DataView(headerBuffer)
    const size32 = headerView.getUint32(0)
    const type = readFourCC(headerView, 4)
    let boxSize = size32
    let headerSize = 8
    if (size32 === 1) {
      if (headerView.byteLength < 16) return undefined
      boxSize = headerView.getUint32(8) * 2 ** 32 + headerView.getUint32(12)
      headerSize = 16
    } else if (size32 === 0) {
      boxSize = file.size - offset
    }
    if (
      !type ||
      !Number.isSafeInteger(boxSize) ||
      boxSize < headerSize ||
      offset + boxSize > file.size
    ) {
      return undefined
    }

    if (type === 'moov') {
      if (boxSize > 32 * 1024 * 1024) return undefined
      return file.slice(offset, offset + boxSize).arrayBuffer()
    }
    if (boxSize <= 0) return undefined
    offset += boxSize
  }
  return undefined
}

export async function getReferenceMp4VideoMetadata(
  file: ReferenceVideoBlobLike
) {
  const moovBuffer = await findMp4MoovBox(file)
  return moovBuffer ? parseReferenceMp4VideoMetadata(moovBuffer) : undefined
}

export function isSeedance25ReferenceVideoMetadata(
  metadata?: ReferenceMp4VideoMetadata
) {
  if (!metadata || !['avc1', 'avc3'].includes(metadata.codec)) return false
  return [24, 25, 30].some(
    (allowedFrameRate) => Math.abs(metadata.frameRate - allowedFrameRate) <= 0.1
  )
}
