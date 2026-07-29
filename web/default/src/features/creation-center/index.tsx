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
import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { PublicLayout } from '@/components/layout'
import {
  getCreationCatalog,
  getCreationErrorMessage,
  getCreationImageTask,
  getCreationVideoTask,
  saveCreationModelCategories,
  saveCreationModelDescriptions,
  submitCreationTask,
  uploadCreationReferenceFile,
} from './api'
import { getCreationCategoryRows } from './category-rows'
import { Composer } from './components/composer'
import {
  CreationInspector,
  type CreationInspectorView,
} from './components/creation-inspector'
import { CreationPreview } from './components/creation-preview'
import { CreationSidebar } from './components/creation-sidebar'
import { ModelHero } from './components/model-hero'
import {
  ModelCategoryDialog,
  ModelDescriptionDialog,
} from './components/model-management-dialogs'
import { CREATION_MODES } from './constants'
import {
  DEFAULT_CREATION_IMAGE_OPTIONS,
  EMPTY_CREATION_IMAGE_REFERENCES,
  DEFAULT_CREATION_VIDEO_OPTIONS,
  EMPTY_CREATION_VIDEO_REFERENCES,
  filterCreationVideoReferencesByPromptMentions,
  getCreationImageAspectRatioOptions,
  getCreationImageReferenceError,
  getCreationImageReferenceLimits,
  getCreationDurationOptions,
  getCreationHistoryStorageKey,
  getCreationResolutionOptions,
  supportsCreationImageReferences,
  getCreationVideoCapabilities,
  getCreationVideoOptionsError,
  getCreationVideoReferenceError,
  getCreationVideoReferenceLimits,
  getCreationVideoRequestOptions,
  loadCreationHistory,
  normalizeCreationImageOptions,
  normalizeCreationImageReferences,
  normalizeCreationVideoOptions,
  normalizeCreationVideoReferences,
  saveCreationHistory,
  upsertCreationHistoryItem,
  type CreationImageOptions,
  type CreationImageReferences,
  type CreationHistoryItem,
  type CreationVideoOptions,
  type CreationVideoReferences,
} from './session'
import type {
  CreationAsset,
  CreationMode,
  CreationModelCategories,
  CreationModelDescriptions,
  CreationResult,
} from './types'
import {
  getReferenceAudioMime,
  getReferenceImageMime,
  getReferenceVideoMime,
  isReferenceAudioFile,
  isReferenceImageFile,
  isReferenceVideoFile,
} from './video-reference-files'

export function CreationCenter() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { auth } = useAuthStore()
  const isSuperAdmin = auth.user?.role === ROLE.SUPER_ADMIN
  const [mode, setMode] = useState<CreationMode>('chat')
  const [selectedByMode, setSelectedByMode] = useState<
    Partial<Record<CreationMode, string>>
  >({})
  const [inspectorView, setInspectorView] =
    useState<CreationInspectorView>('assets')
  const [prompt, setPrompt] = useState('')
  const [assets, setAssets] = useState<CreationAsset[]>([])
  const [categoryOpen, setCategoryOpen] = useState(false)
  const [descriptionOpen, setDescriptionOpen] = useState(false)
  const [sessionNumber, setSessionNumber] = useState(1)
  const [result, setResult] = useState<CreationResult>()
  const [historyItems, setHistoryItems] = useState<CreationHistoryItem[]>([])
  const [imageOptions, setImageOptions] = useState<CreationImageOptions>(
    DEFAULT_CREATION_IMAGE_OPTIONS
  )
  const [imageReferences, setImageReferences] =
    useState<CreationImageReferences>({
      ...EMPTY_CREATION_IMAGE_REFERENCES,
      imageUrls: [],
    })
  const [videoOptions, setVideoOptions] = useState<CreationVideoOptions>(
    DEFAULT_CREATION_VIDEO_OPTIONS
  )
  const [videoReferences, setVideoReferences] =
    useState<CreationVideoReferences>({
      ...EMPTY_CREATION_VIDEO_REFERENCES,
      imageUrls: [],
      videoUrls: [],
      audioUrls: [],
    })
  const [previewNow, setPreviewNow] = useState(() => Date.now())
  const [submitting, setSubmitting] = useState(false)
  const [refreshingTask, setRefreshingTask] = useState(false)
  const historyStorageKey = useMemo(
    () => getCreationHistoryStorageKey(auth.user?.id),
    [auth.user?.id]
  )

  const catalogQuery = useQuery({
    queryKey: ['creation-models'],
    queryFn: getCreationCatalog,
    staleTime: 5 * 60 * 1000,
  })

  const models = useMemo(
    () =>
      catalogQuery.data?.data?.modes.find((group) => group.mode === mode)
        ?.models ?? [],
    [catalogQuery.data?.data?.modes, mode]
  )
  const categoryModels = useMemo(
    () => getCreationCategoryRows(catalogQuery.data?.data?.modes ?? []),
    [catalogQuery.data?.data?.modes]
  )
  const selectedModel = useMemo(
    () =>
      models.find((model) => model.id === selectedByMode[mode]) ?? models[0],
    [mode, models, selectedByMode]
  )
  const durationOptions = useMemo(
    () => getCreationDurationOptions(selectedModel),
    [selectedModel]
  )
  const resolutionOptions = useMemo(
    () => getCreationResolutionOptions(selectedModel),
    [selectedModel]
  )
  const videoCapabilities = useMemo(
    () => getCreationVideoCapabilities(selectedModel),
    [selectedModel]
  )
  const imageReferencesSupported = useMemo(
    () => supportsCreationImageReferences(selectedModel),
    [selectedModel]
  )
  const imageAspectRatioOptions = useMemo(
    () => getCreationImageAspectRatioOptions(selectedModel),
    [selectedModel]
  )
  const imageReferenceLimits = useMemo(
    () => getCreationImageReferenceLimits(selectedModel),
    [selectedModel]
  )
  const modeCounts = useMemo(
    () =>
      CREATION_MODES.reduce(
        (counts, item) => ({
          ...counts,
          [item]:
            catalogQuery.data?.data?.modes.find((group) => group.mode === item)
              ?.models.length ?? 0,
        }),
        {} as Record<CreationMode, number>
      ),
    [catalogQuery.data?.data?.modes]
  )
  const saveCategoryMutation = useMutation({
    mutationFn: (variables: {
      categories: CreationModelCategories
      reset?: boolean
    }) => saveCreationModelCategories(variables.categories),
    onSuccess: async (response, variables) => {
      if (!response.success) {
        toast.error(response.message || t('Unable to save categories.'))
        return
      }
      toast.success(
        variables.reset
          ? t('Automatic categories restored.')
          : t('Categories saved.')
      )
      setCategoryOpen(false)
      setSelectedByMode({})
      await queryClient.invalidateQueries({ queryKey: ['creation-models'] })
    },
    onError: () => {
      toast.error(t('Unable to save categories.'))
    },
  })
  const saveDescriptionMutation = useMutation({
    mutationFn: (variables: {
      descriptions: CreationModelDescriptions
      reset?: boolean
    }) => saveCreationModelDescriptions(variables.descriptions),
    onSuccess: async (response, variables) => {
      if (!response.success) {
        toast.error(response.message || t('Unable to save descriptions.'))
        return
      }
      toast.success(
        variables.reset
          ? t('Automatic descriptions restored.')
          : t('Descriptions saved.')
      )
      setDescriptionOpen(false)
      await queryClient.invalidateQueries({ queryKey: ['creation-models'] })
    },
    onError: () => {
      toast.error(t('Unable to save descriptions.'))
    },
  })

  useEffect(() => {
    if (typeof window === 'undefined') return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setHistoryItems(loadCreationHistory(window.localStorage, historyStorageKey))
  }, [historyStorageKey])

  useEffect(() => {
    if (
      result?.mode !== 'video' ||
      result.videoUrl ||
      result.status === 'completed' ||
      result.status === 'failed'
    ) {
      return
    }

    const timer = window.setInterval(() => setPreviewNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [
    result?.createdAt,
    result?.mode,
    result?.status,
    result?.taskId,
    result?.videoUrl,
  ])

  useEffect(() => {
    if (mode !== 'image') return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setImageOptions((current) => {
      const normalized = normalizeCreationImageOptions(current, selectedModel)
      return normalized.aspectRatio === current.aspectRatio
        ? current
        : normalized
    })
  }, [mode, selectedModel])

  useEffect(() => {
    if (mode !== 'video') return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setVideoOptions((current) => {
      const normalized = normalizeCreationVideoOptions(current, selectedModel)
      return normalized.duration === current.duration &&
        normalized.resolution === current.resolution &&
        normalized.aspectRatio === current.aspectRatio
        ? current
        : normalized
    })
    setVideoReferences((current) => {
      const normalized = normalizeCreationVideoReferences(
        current,
        selectedModel
      )
      return JSON.stringify(normalized) === JSON.stringify(current)
        ? current
        : normalized
    })
  }, [mode, selectedModel])

  const persistHistoryItem = (item: CreationHistoryItem) => {
    if (typeof window === 'undefined') return
    setHistoryItems((current) => {
      const next = upsertCreationHistoryItem(current, item)
      saveCreationHistory(window.localStorage, historyStorageKey, next)
      return next
    })
  }

  const updateHistoryResult = (nextResult: CreationResult) => {
    if (typeof window === 'undefined') return
    const identity = nextResult.taskId || nextResult.id
    if (!identity) return

    setHistoryItems((current) => {
      const next = current.map((item) => {
        const itemIdentity = item.result.taskId || item.result.id || item.id
        if (itemIdentity !== identity) return item
        return {
          ...item,
          model: nextResult.model,
          result: {
            ...item.result,
            ...nextResult,
            createdAt: nextResult.createdAt ?? item.result.createdAt,
            duration: nextResult.duration ?? item.result.duration,
            estimateSeconds:
              nextResult.estimateSeconds ?? item.result.estimateSeconds,
            resolution: nextResult.resolution ?? item.result.resolution,
          },
        }
      })
      saveCreationHistory(window.localStorage, historyStorageKey, next)
      return next
    })
  }

  const selectMode = (nextMode: CreationMode) => {
    setMode(nextMode)
    setInspectorView('assets')
    setResult(undefined)
  }

  const startNewSession = () => {
    setPrompt('')
    setAssets([])
    setImageOptions(DEFAULT_CREATION_IMAGE_OPTIONS)
    setImageReferences({
      ...EMPTY_CREATION_IMAGE_REFERENCES,
      imageUrls: [],
    })
    setVideoReferences({
      ...EMPTY_CREATION_VIDEO_REFERENCES,
      imageUrls: [],
      videoUrls: [],
      audioUrls: [],
    })
    setInspectorView('assets')
    setResult(undefined)
    setSessionNumber((current) => current + 1)
    toast.success(t('A new creation session is ready.'))
  }

  const addImageReferenceFiles = async (files: File[]) => {
    if (!files.length) return

    const imageFiles = files.filter(isReferenceImageFile)
    if (!imageFiles.length) {
      toast.error(t('Choose supported reference files.'))
      return
    }
    if (imageFiles.length < files.length) {
      toast.error(t('Choose supported reference files.'))
    }

    const imageReferenceLimits = getCreationImageReferenceLimits(selectedModel)
    const supportedImageFiles = imageFiles.filter(
      (file) => file.size <= imageReferenceLimits.maxImageSizeBytes
    )
    const imageOversizedCount = imageFiles.length - supportedImageFiles.length
    const imageRemainingSlots =
      imageReferenceLimits.maxImages - imageReferences.imageUrls.length
    const referenceImageFiles = supportedImageFiles.slice(
      0,
      Math.max(imageRemainingSlots, 0)
    )

    if (
      imageFiles.length > 0 &&
      (imageRemainingSlots <= 0 ||
        supportedImageFiles.length > referenceImageFiles.length)
    ) {
      toast.error(
        t('This model accepts at most {{count}} reference images.', {
          count: imageReferenceLimits.maxImages,
        })
      )
    }
    if (imageOversizedCount > 0) {
      toast.error(
        t('Reference images must not exceed {{size}} MB each.', {
          size: imageReferenceLimits.maxImageSizeMB,
        })
      )
    }
    if (!referenceImageFiles.length) return

    let imageUrls: CreationImageReferences['imageUrls'] = []
    try {
      imageUrls = await Promise.all(
        referenceImageFiles.map((file) =>
          createUploadedReferenceValue(
            file,
            'image',
            getReferenceImageMime(file)
          )
        )
      )
    } catch {
      toast.error(t('Unable to upload reference file.'))
      return
    }

    setImageReferences((current) =>
      normalizeCreationImageReferences(
        {
          ...current,
          imageUrls: [...current.imageUrls, ...imageUrls],
        },
        selectedModel
      )
    )
    toast.success(t('Reference images added.'))
  }

  const addVideoReferenceFiles = async (files: File[]) => {
    if (!files.length) return

    const referenceMode = videoReferences.referenceMode
    const videoReferenceLimits = getCreationVideoReferenceLimits(selectedModel)
    const acceptsImages =
      (referenceMode === 'image' || referenceMode === 'multimodal') &&
      videoReferenceLimits.maxImages > 0
    const acceptsVideos =
      (referenceMode === 'video' || referenceMode === 'multimodal') &&
      videoReferenceLimits.maxVideos > 0
    const acceptsAudio =
      referenceMode === 'multimodal' && videoReferenceLimits.maxAudios > 0
    const acceptedFiles = files.filter((file) => {
      if (acceptsImages && isReferenceImageFile(file)) return true
      if (acceptsVideos && isReferenceVideoFile(file)) return true
      if (acceptsAudio && isReferenceAudioFile(file)) return true
      return false
    })

    if (!acceptedFiles.length) {
      toast.error(t('Choose supported reference files.'))
      return
    }
    if (acceptedFiles.length < files.length) {
      toast.error(t('Choose supported reference files.'))
    }

    const currentImageCount = videoReferences.imageUrls.filter(Boolean).length
    const currentVideoCount = videoReferences.videoUrls.filter(Boolean).length
    const currentAudioCount = videoReferences.audioUrls.filter(Boolean).length
    const maxMediaFiles = videoReferenceLimits.maxMediaFiles
    let remainingTotalSlots =
      typeof maxMediaFiles === 'number'
        ? Math.max(
            maxMediaFiles -
              currentImageCount -
              currentVideoCount -
              currentAudioCount,
            0
          )
        : Number.POSITIVE_INFINITY
    const referenceImageFiles: File[] = []
    const referenceVideoFiles: File[] = []
    const referenceAudioFiles: File[] = []
    let imageOversizedCount = 0
    let videoOversizedCount = 0
    let audioOversizedCount = 0
    let imageLimitHit = false
    let videoLimitHit = false
    let audioLimitHit = false
    let totalLimitHit = false

    for (const file of acceptedFiles) {
      if (acceptsImages && isReferenceImageFile(file)) {
        if (file.size > videoReferenceLimits.maxImageSizeBytes) {
          imageOversizedCount += 1
          continue
        }
        if (currentImageCount + referenceImageFiles.length >= videoReferenceLimits.maxImages) {
          imageLimitHit = true
          continue
        }
        if (Number.isFinite(remainingTotalSlots) && remainingTotalSlots <= 0) {
          totalLimitHit = true
          continue
        }
        referenceImageFiles.push(file)
        if (Number.isFinite(remainingTotalSlots)) {
          remainingTotalSlots -= 1
        }
        continue
      }

      if (acceptsVideos && isReferenceVideoFile(file)) {
        if (file.size > videoReferenceLimits.maxVideoSizeBytes) {
          videoOversizedCount += 1
          continue
        }
        if (currentVideoCount + referenceVideoFiles.length >= videoReferenceLimits.maxVideos) {
          videoLimitHit = true
          continue
        }
        if (Number.isFinite(remainingTotalSlots) && remainingTotalSlots <= 0) {
          totalLimitHit = true
          continue
        }
        referenceVideoFiles.push(file)
        if (Number.isFinite(remainingTotalSlots)) {
          remainingTotalSlots -= 1
        }
        continue
      }

      if (acceptsAudio && isReferenceAudioFile(file)) {
        if (file.size > videoReferenceLimits.maxAudioSizeBytes) {
          audioOversizedCount += 1
          continue
        }
        if (currentAudioCount + referenceAudioFiles.length >= videoReferenceLimits.maxAudios) {
          audioLimitHit = true
          continue
        }
        if (Number.isFinite(remainingTotalSlots) && remainingTotalSlots <= 0) {
          totalLimitHit = true
          continue
        }
        referenceAudioFiles.push(file)
        if (Number.isFinite(remainingTotalSlots)) {
          remainingTotalSlots -= 1
        }
      }
    }

    if (
      !referenceImageFiles.length &&
      !referenceVideoFiles.length &&
      !referenceAudioFiles.length
    ) {
      return
    }

    if (imageLimitHit) {
      toast.error(
        t('This model accepts at most {{count}} image references.', {
          count: videoReferenceLimits.maxImages,
        })
      )
    }
    if (videoLimitHit) {
      toast.error(
        t('This model accepts at most {{count}} video references.', {
          count: videoReferenceLimits.maxVideos,
        })
      )
    }
    if (audioLimitHit) {
      toast.error(
        t('This model accepts at most {{count}} audio references.', {
          count: videoReferenceLimits.maxAudios,
        })
      )
    }
    if (imageOversizedCount > 0) {
      toast.error(
        t('Reference images must not exceed {{size}} MB each.', {
          size: videoReferenceLimits.maxImageSizeMB,
        })
      )
    }
    if (videoOversizedCount > 0) {
      toast.error(
        t('Reference videos must not exceed {{size}} MB each.', {
          size: videoReferenceLimits.maxVideoSizeMB,
        })
      )
    }
    if (audioOversizedCount > 0) {
      toast.error(
        t('Reference audio must not exceed {{size}} MB each.', {
          size: videoReferenceLimits.maxAudioSizeMB,
        })
      )
    }
    if (totalLimitHit && maxMediaFiles) {
      toast.error(
        t('This model accepts at most {{count}} reference assets.', {
          count: maxMediaFiles,
        })
      )
    }

    let imageUrls: CreationVideoReferences['imageUrls'] = []
    let videoUrls: CreationVideoReferences['videoUrls'] = []
    let audioUrls: CreationVideoReferences['audioUrls'] = []
    try {
      imageUrls = await Promise.all(
        referenceImageFiles.map((file) =>
          createUploadedReferenceValue(
            file,
            'image',
            getReferenceImageMime(file)
          )
        )
      )
      videoUrls = await Promise.all(
        referenceVideoFiles.map((file) =>
          createUploadedReferenceValue(
            file,
            'video',
            getReferenceVideoMime(file)
          )
        )
      )
      audioUrls = await Promise.all(
        referenceAudioFiles.map((file) =>
          createUploadedReferenceValue(
            file,
            'audio',
            getReferenceAudioMime(file)
          )
        )
      )
    } catch {
      toast.error(t('Unable to upload reference file.'))
      return
    }
    setVideoReferences((current) =>
      normalizeCreationVideoReferences(
        {
          ...current,
          imageUrls: [...current.imageUrls, ...imageUrls],
          videoUrls: [...current.videoUrls, ...videoUrls],
          audioUrls: [...current.audioUrls, ...audioUrls],
          audioUrl: [...current.audioUrls, ...audioUrls][0] ?? current.audioUrl,
        },
        selectedModel
      )
    )
    toast.success(t('Reference assets added.'))
  }

  const removeAsset = (index: number) => {
    setAssets((current) =>
      current.filter((_, itemIndex) => itemIndex !== index)
    )
    toast.success(t('Asset removed.'))
  }

  const removeImageReferenceImage = (index: number) => {
    setImageReferences((current) =>
      normalizeCreationImageReferences(
        {
          ...current,
          imageUrls: current.imageUrls.filter(
            (_, itemIndex) => itemIndex !== index
          ),
        },
        selectedModel
      )
    )
  }

  const removeVideoReferenceImage = (index: number) => {
    setVideoReferences((current) =>
      normalizeCreationVideoReferences(
        {
          ...current,
          imageUrls: current.imageUrls.filter(
            (_, itemIndex) => itemIndex !== index
          ),
        },
        selectedModel
      )
    )
  }

  const removeVideoReferenceVideo = (index: number) => {
    setVideoReferences((current) =>
      normalizeCreationVideoReferences(
        {
          ...current,
          videoUrls: current.videoUrls.filter(
            (_, itemIndex) => itemIndex !== index
          ),
        },
        selectedModel
      )
    )
  }

  const removeVideoReferenceAudio = (index: number) => {
    setVideoReferences((current) => {
      const audioUrls = current.audioUrls.filter(
        (_, itemIndex) => itemIndex !== index
      )
      return normalizeCreationVideoReferences(
        {
          ...current,
          audioUrls,
          audioUrl: audioUrls[0] ?? '',
        },
        selectedModel
      )
    })
  }

  const submit = async () => {
    if (!auth.user) {
      navigate({ to: '/sign-in', search: { redirect: '/creation' } })
      return
    }
    if (!selectedModel) {
      toast.error(t('Select a model before submitting.'))
      return
    }
    const trimmedPrompt = prompt.trim()
    if (!trimmedPrompt) {
      toast.error(t('Write a prompt before submitting.'))
      return
    }
    const mentionedVideoReferences =
      mode === 'video' && videoCapabilities
        ? filterCreationVideoReferencesByPromptMentions(
            trimmedPrompt,
            videoReferences,
            selectedModel
          )
        : undefined
    if (mode === 'video') {
      const optionError = getCreationVideoOptionsError(
        videoOptions,
        selectedModel
      )
      if (optionError) {
        toast.error(t(optionError))
        return
      }
      const referenceError = getCreationVideoReferenceError(
        selectedModel,
        mentionedVideoReferences ?? videoReferences
      )
      if (referenceError) {
        toast.error(t(referenceError))
        return
      }
    }
    if (mode === 'image') {
      const referenceError = getCreationImageReferenceError(
        selectedModel,
        imageReferences
      )
      if (referenceError) {
        toast.error(t(referenceError))
        return
      }
    }

    setSubmitting(true)
    setInspectorView('task')
    const createdAt = Date.now()
    const videoRequestOptions =
      mode === 'video' && videoCapabilities
        ? getCreationVideoRequestOptions(
            videoOptions,
            selectedModel,
            mentionedVideoReferences ?? videoReferences
          )
        : undefined
    const normalizedVideoOptions =
      mode === 'video' && videoCapabilities
        ? normalizeCreationVideoOptions(videoOptions, selectedModel)
        : undefined
    const normalizedVideoReferences =
      mode === 'video' && videoCapabilities
        ? normalizeCreationVideoReferences(
            mentionedVideoReferences ?? videoReferences,
            selectedModel
          )
        : undefined
    const normalizedImageReferences =
      mode === 'image'
        ? normalizeCreationImageReferences(imageReferences, selectedModel)
        : undefined
    const normalizedImageOptions =
      mode === 'image'
        ? normalizeCreationImageOptions(imageOptions, selectedModel)
        : undefined
    try {
      const nextResult = await submitCreationTask({
        mode,
        model: selectedModel,
        prompt: trimmedPrompt,
        assets,
        imageOptions: normalizedImageOptions,
        imageReferences: normalizedImageReferences,
        videoOptions: normalizedVideoOptions,
        videoReferences: normalizedVideoReferences,
      })
      const enrichedResult: CreationResult = {
        ...nextResult,
        createdAt,
        duration: normalizedVideoOptions?.duration,
        estimateSeconds: videoRequestOptions?.estimateSeconds,
        resolution: normalizedVideoOptions?.resolution,
      }
      setResult(enrichedResult)
      persistHistoryItem({
        createdAt,
        id: getCreationHistoryItemId(enrichedResult, mode),
        mode,
        model: selectedModel.id,
        prompt: trimmedPrompt,
        assets: getCreationAssetSnapshots(assets),
        result: enrichedResult,
        imageOptions: normalizedImageOptions,
        imageReferences: normalizedImageReferences,
        videoOptions: normalizedVideoOptions,
        videoReferences: normalizedVideoReferences,
      })
      if (nextResult.status === 'failed') {
        toast.error(nextResult.error || t('Creation task failed.'))
      } else if (nextResult.taskId && nextResult.status !== 'completed') {
        toast.success(t('Task submitted. Refresh its status later.'))
      } else {
        toast.success(t('Creation task completed.'))
      }
    } catch (error) {
      const message = getCreationErrorMessage(error)
      const failedResult: CreationResult = {
        mode,
        model: selectedModel.id,
        createdAt,
        duration: normalizedVideoOptions?.duration,
        estimateSeconds: videoRequestOptions?.estimateSeconds,
        resolution: normalizedVideoOptions?.resolution,
        status: 'failed',
        error: message,
      }
      setResult(failedResult)
      persistHistoryItem({
        createdAt,
        id: getCreationHistoryItemId(failedResult, mode),
        mode,
        model: selectedModel.id,
        prompt: trimmedPrompt,
        assets: getCreationAssetSnapshots(assets),
        result: failedResult,
        imageOptions: normalizedImageOptions,
        imageReferences: normalizedImageReferences,
        videoOptions: normalizedVideoOptions,
        videoReferences: normalizedVideoReferences,
      })
      toast.error(message)
    } finally {
      setSubmitting(false)
    }
  }

  const refreshMediaTask = async () => {
    if (
      !result?.taskId ||
      (result.mode !== 'image' && result.mode !== 'video')
    ) {
      return
    }
    setRefreshingTask(true)
    try {
      const nextResult =
        result.mode === 'image'
          ? await getCreationImageTask({
              taskId: result.taskId,
              model: result.model,
            })
          : await getCreationVideoTask({
              taskId: result.taskId,
              model: result.model,
            })
      const enrichedResult = {
        ...nextResult,
        createdAt: result.createdAt,
        duration: result.duration,
        estimateSeconds: result.estimateSeconds,
        resolution: result.resolution,
      }
      setResult(enrichedResult)
      updateHistoryResult(enrichedResult)
      toast.success(t('Task status refreshed.'))
    } catch (error) {
      const message = getCreationErrorMessage(error)
      const failedResult: CreationResult = {
        ...result,
        status: 'failed',
        error: message,
      }
      setResult(failedResult)
      updateHistoryResult(failedResult)
      toast.error(message)
    } finally {
      setRefreshingTask(false)
    }
  }

  const selectHistoryItem = (item: CreationHistoryItem) => {
    setMode(item.mode)
    setSelectedByMode((current) => ({
      ...current,
      [item.mode]: item.model,
    }))
    setPrompt(item.prompt)
    setAssets(normalizeStoredCreationAssets(item.assets))
    setImageReferences(
      item.imageReferences
        ? normalizeCreationImageReferences(item.imageReferences, item.model)
        : {
            ...EMPTY_CREATION_IMAGE_REFERENCES,
            imageUrls: [],
          }
    )
    setImageOptions(
      item.imageOptions
        ? normalizeCreationImageOptions(item.imageOptions, item.model)
        : DEFAULT_CREATION_IMAGE_OPTIONS
    )
    setVideoReferences(
      item.videoReferences
        ? normalizeCreationVideoReferences(item.videoReferences, item.model)
        : {
            ...EMPTY_CREATION_VIDEO_REFERENCES,
            imageUrls: [],
            videoUrls: [],
            audioUrls: [],
          }
    )
    setResult(item.result)
    if (item.videoOptions) {
      setVideoOptions(
        normalizeCreationVideoOptions(item.videoOptions, item.model)
      )
    }
    setInspectorView('task')
  }

  const clearHistory = () => {
    if (typeof window !== 'undefined') {
      window.localStorage.removeItem(historyStorageKey)
    }
    setHistoryItems([])
    toast.success(t('Creation history cleared.'))
  }

  const requestReferenceUpload = () => {
    document.getElementById('creation-reference-upload')?.click()
  }

  const canUploadReferences =
    mode === 'image'
      ? imageReferencesSupported
      : mode === 'video'
        ? !!videoCapabilities &&
          (videoCapabilities.referenceLimits.maxImages > 0 ||
            videoCapabilities.referenceLimits.maxVideos > 0 ||
            videoCapabilities.referenceLimits.maxAudios > 0)
        : false

  return (
    <PublicLayout showMainContainer={false}>
      <main className='text-foreground min-h-svh bg-slate-100/80 pt-16 dark:bg-[#080d12]'>
        <div className='grid min-h-[calc(100svh-4rem)] lg:grid-cols-[18rem_minmax(0,1fr)]'>
          <CreationSidebar
            mode={mode}
            models={models}
            selectedModel={selectedModel}
            selectedResolution={
              mode === 'video' && videoCapabilities
                ? videoOptions.resolution
                : undefined
            }
            modeCounts={modeCounts}
            loading={catalogQuery.isLoading}
            error={catalogQuery.isError}
            canManageCategories={isSuperAdmin}
            canManageDescriptions={isSuperAdmin}
            onModeChange={selectMode}
            onModelChange={(model) => {
              setSelectedByMode((current) => ({
                ...current,
                [mode]: model.id,
              }))
              setResult(undefined)
            }}
            onHistory={() => setInspectorView('history')}
            onNewSession={startNewSession}
            onManageCategories={() => setCategoryOpen(true)}
            onManageDescriptions={() => setDescriptionOpen(true)}
          />

          <section className='flex min-w-0 flex-col p-3 md:p-4 xl:p-5'>
            <div className='mx-auto grid w-full max-w-[120rem] flex-1 items-start gap-3 xl:grid-cols-[minmax(0,1fr)_20rem]'>
              <div className='xl:col-span-2'>
                <ModelHero
                  mode={mode}
                  model={selectedModel}
                  selectedResolution={
                    mode === 'video' && videoCapabilities
                      ? videoOptions.resolution
                      : undefined
                  }
                />
              </div>
              <CreationPreview
                className='xl:col-start-1 xl:row-start-2'
                mode={mode}
                model={selectedModel}
                result={result}
                now={previewNow}
                submitting={submitting}
                refreshingTask={refreshingTask}
                onRefreshTask={refreshMediaTask}
              />

              <div className='xl:col-start-1 xl:row-start-3'>
                <Composer
                  prompt={prompt}
                  assets={assets}
                  authenticated={!!auth.user}
                  mode={mode}
                  model={selectedModel}
                  imageOptions={imageOptions}
                  imageReferences={imageReferences}
                  imageReferencesSupported={imageReferencesSupported}
                  imageAspectRatioOptions={imageAspectRatioOptions}
                  imageReferenceLimits={imageReferenceLimits}
                  videoOptions={videoOptions}
                  videoReferences={videoReferences}
                  videoCapabilities={videoCapabilities}
                  resolutionOptions={resolutionOptions}
                  durationOptions={durationOptions}
                  submitting={submitting}
                  sessionNumber={sessionNumber}
                  onPromptChange={setPrompt}
                  onImageOptionsChange={setImageOptions}
                  onImageReferenceFilesSelected={addImageReferenceFiles}
                  onRemoveImageReferenceImage={removeImageReferenceImage}
                  onVideoOptionsChange={setVideoOptions}
                  onVideoReferencesChange={setVideoReferences}
                  onVideoReferenceFilesSelected={addVideoReferenceFiles}
                  onRemoveVideoReferenceImage={removeVideoReferenceImage}
                  onRemoveVideoReferenceVideo={removeVideoReferenceVideo}
                  onRemoveVideoReferenceAudio={removeVideoReferenceAudio}
                  onRemoveAsset={removeAsset}
                  onSubmit={submit}
                />
              </div>

              <CreationInspector
                className='xl:sticky xl:top-20 xl:col-start-2 xl:row-span-2 xl:row-start-2 xl:h-[calc(100svh-6rem)]'
                mode={mode}
                value={inspectorView}
                assets={assets}
                imageReferences={imageReferences}
                videoReferences={videoReferences}
                historyItems={historyItems}
                result={result}
                now={previewNow}
                refreshingTask={refreshingTask}
                canUploadReferences={canUploadReferences}
                onValueChange={setInspectorView}
                onRequestUpload={requestReferenceUpload}
                onSelectHistory={selectHistoryItem}
                onClearHistory={clearHistory}
                onRefreshTask={refreshMediaTask}
                onRemoveAsset={removeAsset}
                onRemoveImageReference={removeImageReferenceImage}
                onRemoveVideoReferenceImage={removeVideoReferenceImage}
                onRemoveVideoReferenceVideo={removeVideoReferenceVideo}
                onRemoveVideoReferenceAudio={removeVideoReferenceAudio}
              />
            </div>
          </section>
        </div>
      </main>

      <ModelCategoryDialog
        open={categoryOpen}
        models={categoryModels}
        saving={saveCategoryMutation.isPending}
        onOpenChange={setCategoryOpen}
        onSave={(categories) => saveCategoryMutation.mutate({ categories })}
        onReset={() =>
          saveCategoryMutation.mutate({ categories: {}, reset: true })
        }
      />
      <ModelDescriptionDialog
        open={descriptionOpen}
        models={categoryModels}
        saving={saveDescriptionMutation.isPending}
        onOpenChange={setDescriptionOpen}
        onSave={(descriptions) =>
          saveDescriptionMutation.mutate({ descriptions })
        }
        onReset={() =>
          saveDescriptionMutation.mutate({ descriptions: {}, reset: true })
        }
      />
    </PublicLayout>
  )
}

function getCreationAssetSnapshots(assets: CreationAsset[]): CreationAsset[] {
  return assets.map((asset) => ({
    id: asset.id,
    name: asset.name,
    type: asset.type,
    size: asset.size,
    text: asset.text?.slice(0, 2000),
  }))
}

function normalizeStoredCreationAssets(assets: unknown): CreationAsset[] {
  if (!Array.isArray(assets)) return []
  return assets.flatMap((asset, index) => {
    if (typeof asset === 'string') {
      return [
        {
          id: `legacy-${index}-${asset}`,
          name: asset,
          type: '',
          size: 0,
        },
      ]
    }
    if (!asset || typeof asset !== 'object') return []
    const item = asset as Partial<CreationAsset>
    if (!item.name) return []
    return [
      {
        id: item.id || `history-${index}-${item.name}`,
        name: item.name,
        type: item.type || '',
        size: item.size || 0,
        text: item.text,
      },
    ]
  })
}

function getCreationHistoryItemId(result: CreationResult, mode: CreationMode) {
  return (
    result.taskId ||
    result.id ||
    `${mode}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  )
}

async function createUploadedReferenceValue(
  file: File,
  kind: 'image' | 'video' | 'audio',
  mimeType: string | undefined
) {
  const url = await uploadCreationReferenceFile(file, kind, mimeType)
  return {
    url,
    previewUrl:
      typeof URL !== 'undefined' && typeof URL.createObjectURL === 'function'
        ? URL.createObjectURL(file)
        : url,
  }
}
