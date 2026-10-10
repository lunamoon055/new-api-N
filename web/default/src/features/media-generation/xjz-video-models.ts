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
import type {
  CreationResolution,
  CreationVideoReferenceMode,
} from './video-options'

export type XJZVideoModelSpec = {
  durations: string[]
  resolutions: CreationResolution[]
  ratios: string[]
  referenceModes: CreationVideoReferenceMode[]
  maxImages: number
  minImages?: number
  maxVideos: number
  maxAudios: number
  imageField?: 'images' | 'reference_images' | 'image_urls'
  videoField?: 'reference_videos' | 'video_urls'
  audioField?: 'reference_audios' | 'audio_urls' | 'audio_reference'
  requestResolution?: string
  selectableResolution?: boolean
  audioNeedsVisualReference?: boolean
  allowFrames?: boolean
}

const RATIOS_WIDE = ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9']
const RATIOS_STANDARD = ['16:9', '9:16', '1:1']
const DURATIONS_4_TO_15 = seconds(4, 15)
const DURATIONS_5_TO_15 = seconds(5, 15)
const DURATIONS_4_TO_30 = seconds(4, 30)

function seconds(min: number, max: number): string[] {
  return Array.from({ length: max - min + 1 }, (_, index) =>
    String(min + index)
  )
}

// Exact API model IDs and capabilities from XJZ's public /api/pricing catalog.
// The model cards and the catalog disagree in a few places. Use the catalog's
// structured parameter limits; where a field is not documented, omit it.
export const XJZ_VIDEO_MODELS: Record<string, XJZVideoModelSpec> = {
  '特价dola2.5-900-720P-固定30s': {
    durations: ['30'],
    resolutions: ['720p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['image'],
    minImages: 1,
    maxImages: 9,
    maxVideos: 0,
    maxAudios: 0,
    imageField: 'images',
    requestResolution: '720p',
  },
  '特价dola-sd2.0-900-720p': {
    durations: ['5', '10', '15'],
    resolutions: ['720p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['image'],
    minImages: 1,
    maxImages: 1,
    maxVideos: 0,
    maxAudios: 0,
    imageField: 'images',
    requestResolution: '720p',
  },
  'TTP-grok': {
    durations: seconds(6, 15),
    resolutions: ['720p', '480p'],
    ratios: RATIOS_STANDARD,
    referenceModes: ['text', 'image'],
    maxImages: 7,
    maxVideos: 0,
    maxAudios: 0,
    imageField: 'images',
    selectableResolution: true,
  },
  'gz稳定sd2.0-903-720P': {
    durations: DURATIONS_5_TO_15,
    resolutions: ['720p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['text', 'image', 'image-audio'],
    maxImages: 9,
    maxVideos: 0,
    maxAudios: 3,
    imageField: 'images',
    audioField: 'reference_audios',
    requestResolution: '720p',
  },
  'Xminimex-h3-2K-933': {
    durations: DURATIONS_5_TO_15,
    resolutions: ['2k'],
    ratios: RATIOS_WIDE,
    referenceModes: ['text', 'image', 'video', 'multimodal'],
    maxImages: 9,
    maxVideos: 3,
    maxAudios: 3,
    imageField: 'images',
    videoField: 'reference_videos',
    audioField: 'reference_audios',
    requestResolution: '2k',
  },
  '特价sd2.0-720p-933-过脸': {
    durations: DURATIONS_5_TO_15,
    resolutions: ['720p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['text', 'image', 'video', 'multimodal'],
    maxImages: 9,
    maxVideos: 3,
    maxAudios: 3,
    imageField: 'images',
    videoField: 'reference_videos',
    audioField: 'reference_audios',
    requestResolution: '720p',
  },
  '特价稳sd-2.5-30s-720P可10图': {
    durations: ['30'],
    resolutions: ['720p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['text', 'image'],
    maxImages: 10,
    maxVideos: 0,
    maxAudios: 0,
    imageField: 'reference_images',
    requestResolution: '720p',
  },
  '特价稳sd2.0-933-720p': {
    // The model description says 5-15 while one catalog parameter says 4-15.
    durations: DURATIONS_5_TO_15,
    resolutions: ['720p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['text', 'image', 'video', 'multimodal'],
    maxImages: 9,
    maxVideos: 3,
    maxAudios: 3,
    imageField: 'reference_images',
    videoField: 'reference_videos',
    audioField: 'reference_audios',
    requestResolution: '720P',
  },
  'gz稳定sd2.5-720全参过脸': {
    durations: DURATIONS_4_TO_30,
    resolutions: ['720p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['text', 'image', 'video', 'multimodal'],
    maxImages: 30,
    maxVideos: 10,
    maxAudios: 10,
    imageField: 'reference_images',
    videoField: 'reference_videos',
    audioField: 'reference_audios',
    requestResolution: '720P',
  },
  'gz稳定sd2.5-480全参过脸': {
    durations: DURATIONS_4_TO_30,
    resolutions: ['480p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['text', 'image', 'video', 'multimodal'],
    maxImages: 30,
    maxVideos: 10,
    maxAudios: 10,
    imageField: 'reference_images',
    videoField: 'reference_videos',
    audioField: 'reference_audios',
    requestResolution: '480P',
  },
  'gz稳定sd2.5-1080全参过脸': {
    durations: DURATIONS_4_TO_30,
    resolutions: ['1080p'],
    ratios: RATIOS_WIDE,
    referenceModes: ['text', 'image', 'video', 'multimodal'],
    maxImages: 30,
    maxVideos: 10,
    maxAudios: 10,
    imageField: 'reference_images',
    videoField: 'reference_videos',
    audioField: 'reference_audios',
    requestResolution: '1080p',
  },
  '特价sd2.0-933-过脸': {
    durations: DURATIONS_4_TO_15,
    resolutions: ['720p'],
    ratios: ['16:9', '9:16', '1:1', '4:3'],
    referenceModes: ['text', 'image', 'video', 'multimodal'],
    maxImages: 9,
    maxVideos: 3,
    maxAudios: 3,
    imageField: 'image_urls',
    videoField: 'video_urls',
    audioField: 'audio_urls',
    audioNeedsVisualReference: true,
    requestResolution: '720p',
  },
  MiniMaxH3: {
    durations: DURATIONS_4_TO_15,
    resolutions: ['2k'],
    ratios: RATIOS_STANDARD,
    referenceModes: ['text', 'image', 'frames', 'image-audio'],
    maxImages: 9,
    maxVideos: 0,
    maxAudios: 3,
    imageField: 'image_urls',
    audioField: 'audio_reference',
    audioNeedsVisualReference: true,
    allowFrames: true,
    requestResolution: '2K',
  },
}

export function getXJZVideoModelSpec(
  modelId: string
): XJZVideoModelSpec | undefined {
  return Object.hasOwn(XJZ_VIDEO_MODELS, modelId)
    ? XJZ_VIDEO_MODELS[modelId]
    : undefined
}
