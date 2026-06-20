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
  bmp: 'image/bmp',
  gif: 'image/gif',
  heic: 'image/heic',
  heif: 'image/heif',
  jpeg: 'image/jpeg',
  jpg: 'image/jpeg',
  png: 'image/png',
  webp: 'image/webp',
}

type ImageFileLike = Pick<File, 'name' | 'type'>

export function getReferenceImageMime(file: ImageFileLike) {
  if (file.type.startsWith('image/')) return file.type
  const extension = file.name.split('.').pop()?.toLowerCase()
  return extension ? IMAGE_MIME_BY_EXTENSION[extension] : undefined
}

export function isReferenceImageFile(file: ImageFileLike) {
  return !!getReferenceImageMime(file)
}

export function normalizeReferenceImageDataURL(
  file: ImageFileLike,
  dataURL: string
) {
  const mime = getReferenceImageMime(file)
  if (!mime) return dataURL
  return dataURL.replace(
    /^data:(?:application\/octet-stream)?;base64,/i,
    `data:${mime};base64,`
  )
}

export function readReferenceImageAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.addEventListener('load', () => {
      resolve(normalizeReferenceImageDataURL(file, String(reader.result || '')))
    })
    reader.addEventListener('error', () => reject(reader.error))
    reader.readAsDataURL(file)
  })
}
