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

const VIDEO_MIME_BY_EXTENSION: Record<string, string> = {
  mp4: 'video/mp4',
}

const AUDIO_MIME_BY_EXTENSION: Record<string, string> = {
  mp3: 'audio/mpeg',
  wav: 'audio/wav',
}

const AUDIO_MIME_TYPES = [
  'audio/mpeg',
  'audio/mp3',
  'audio/wav',
  'audio/wave',
  'audio/x-wav',
]

type ReferenceFileLike = Pick<File, 'name' | 'type'>

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

export function getReferenceImageMime(file: ReferenceFileLike) {
  return getReferenceMime(file, IMAGE_MIME_BY_EXTENSION)
}

export function getReferenceVideoMime(file: ReferenceFileLike) {
  return getReferenceMime(file, VIDEO_MIME_BY_EXTENSION)
}

export function getReferenceAudioMime(file: ReferenceFileLike) {
  return getReferenceMime(file, AUDIO_MIME_BY_EXTENSION, AUDIO_MIME_TYPES)
}

export function isReferenceImageFile(file: ReferenceFileLike) {
  return !!getReferenceImageMime(file)
}

export function isReferenceVideoFile(file: ReferenceFileLike) {
  return !!getReferenceVideoMime(file)
}

export function isReferenceAudioFile(file: ReferenceFileLike) {
  return !!getReferenceAudioMime(file)
}
