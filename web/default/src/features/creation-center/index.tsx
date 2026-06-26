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
  getCreationVideoTask,
  saveCreationModelCategories,
  saveCreationModelDescriptions,
  submitCreationTask,
  uploadCreationReferenceFile,
} from './api'
import { getCreationCategoryRows } from './category-rows'
import { Composer } from './components/composer'
import { CreationPreview } from './components/creation-preview'
import { CreationSidebar } from './components/creation-sidebar'
import { ModelHero } from './components/model-hero'
import {
  ModelCategoryDialog,
  ModelDescriptionDialog,
} from './components/model-management-dialogs'
import { CREATION_MODES } from './constants'
import {
  CREATION_IMAGE_REFERENCE_MAX_BYTES,
  CREATION_IMAGE_REFERENCE_MAX_COUNT,
  CREATION_VIDEO_IMAGE_REFERENCE_MAX_BYTES,
  CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT,
  CREATION_VIDEO_AUDIO_REFERENCE_MAX_BYTES,
  CREATION_VIDEO_VIDEO_REFERENCE_MAX_BYTES,
  CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT,
  EMPTY_CREATION_IMAGE_REFERENCES,
  DEFAULT_CREATION_VIDEO_OPTIONS,
  EMPTY_CREATION_VIDEO_REFERENCES,
  getCreationImageReferenceError,
  getCreationDurationOptions,
  getCreationHistoryStorageKey,
  getCreationResolutionOptions,
  supportsCreationImageReferences,
  getCreationVideoCapabilities,
  getCreationVideoOptionsError,
  getCreationVideoReferenceError,
  getCreationVideoRequestOptions,
  loadCreationHistory,
  normalizeCreationImageReferences,
  normalizeCreationVideoOptions,
  normalizeCreationVideoReferences,
  saveCreationHistory,
  upsertCreationHistoryItem,
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
  CreationView,
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
  const [view, setView] = useState<CreationView>('preview')
  const [prompt, setPrompt] = useState('')
  const [assets, setAssets] = useState<CreationAsset[]>([])
  const [categoryOpen, setCategoryOpen] = useState(false)
  const [descriptionOpen, setDescriptionOpen] = useState(false)
  const [sessionNumber, setSessionNumber] = useState(1)
  const [result, setResult] = useState<CreationResult>()
  const [historyItems, setHistoryItems] = useState<CreationHistoryItem[]>([])
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
    () => getCreationDurationOptions(selectedModel?.id),
    [selectedModel?.id]
  )
  const resolutionOptions = useMemo(
    () => getCreationResolutionOptions(selectedModel?.id),
    [selectedModel?.id]
  )
  const videoCapabilities = useMemo(
    () => getCreationVideoCapabilities(selectedModel?.id),
    [selectedModel?.id]
  )
  const imageReferencesSupported = useMemo(
    () => supportsCreationImageReferences(selectedModel?.id),
    [selectedModel?.id]
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
    if (mode !== 'video') return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setVideoOptions((current) => {
      const normalized = normalizeCreationVideoOptions(
        current,
        selectedModel?.id
      )
      return normalized.duration === current.duration &&
        normalized.resolution === current.resolution &&
        normalized.aspectRatio === current.aspectRatio
        ? current
        : normalized
    })
    setVideoReferences((current) => {
      const normalized = normalizeCreationVideoReferences(
        current,
        selectedModel?.id
      )
      return JSON.stringify(normalized) === JSON.stringify(current)
        ? current
        : normalized
    })
  }, [mode, selectedModel?.id])

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
    setView('preview')
    setResult(undefined)
  }

  const startNewSession = () => {
    setPrompt('')
    setAssets([])
    setImageReferences({
      ...EMPTY_CREATION_IMAGE_REFERENCES,
      imageUrls: [],
    })
    setVideoReferences({
      ...EMPTY_CREATION_VIDEO_REFERENCES,
      imageUrls: [],
      videoUrls: [],
    })
    setView('preview')
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

    const supportedImageFiles = imageFiles.filter(
      (file) => file.size <= CREATION_IMAGE_REFERENCE_MAX_BYTES
    )
    const imageOversizedCount = imageFiles.length - supportedImageFiles.length
    const imageRemainingSlots =
      CREATION_IMAGE_REFERENCE_MAX_COUNT - imageReferences.imageUrls.length
    const referenceImageFiles = supportedImageFiles.slice(
      0,
      Math.max(imageRemainingSlots, 0)
    )

    if (
      imageFiles.length > 0 &&
      (imageRemainingSlots <= 0 ||
        supportedImageFiles.length > referenceImageFiles.length)
    ) {
      toast.error(t('Gpt-image2 accepts at most 6 reference images.'))
    }
    if (imageOversizedCount > 0) {
      toast.error(t('Reference images must not exceed 20 MB each.'))
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
        selectedModel?.id
      )
    )
    toast.success(t('Reference images added.'))
  }

  const addVideoReferenceFiles = async (files: File[]) => {
    if (!files.length) return

    const referenceMode = videoReferences.referenceMode
    const acceptsImages =
      referenceMode === 'image' || referenceMode === 'multimodal'
    const acceptsVideos =
      referenceMode === 'video' || referenceMode === 'multimodal'
    const acceptsAudio = referenceMode === 'multimodal'
    const imageFiles = acceptsImages ? files.filter(isReferenceImageFile) : []
    const videoFiles = acceptsVideos ? files.filter(isReferenceVideoFile) : []
    const audioFiles = acceptsAudio ? files.filter(isReferenceAudioFile) : []
    const acceptedCount =
      imageFiles.length + videoFiles.length + audioFiles.length

    if (!acceptedCount) {
      toast.error(t('Choose supported reference files.'))
      return
    }
    if (acceptedCount < files.length) {
      toast.error(t('Choose supported reference files.'))
    }

    const supportedImageFiles = imageFiles.filter(
      (file) => file.size <= CREATION_VIDEO_IMAGE_REFERENCE_MAX_BYTES
    )
    const imageOversizedCount = imageFiles.length - supportedImageFiles.length
    const imageRemainingSlots =
      CREATION_VIDEO_IMAGE_REFERENCE_MAX_COUNT -
      videoReferences.imageUrls.length
    const referenceImageFiles = supportedImageFiles.slice(
      0,
      Math.max(imageRemainingSlots, 0)
    )

    if (
      imageFiles.length > 0 &&
      (imageRemainingSlots <= 0 ||
        supportedImageFiles.length > referenceImageFiles.length)
    ) {
      toast.error(t('Video2 accepts at most 4 image references.'))
    }
    if (imageOversizedCount > 0) {
      toast.error(t('Reference images must not exceed 20 MB each.'))
    }

    const selectedVideoBytes = videoFiles.reduce(
      (sum, file) => sum + file.size,
      0
    )
    const supportedVideoFiles =
      selectedVideoBytes <= CREATION_VIDEO_VIDEO_REFERENCE_MAX_BYTES
        ? videoFiles
        : []
    const videoRemainingSlots =
      CREATION_VIDEO_VIDEO_REFERENCE_MAX_COUNT -
      videoReferences.videoUrls.filter(Boolean).length
    const referenceVideoFiles = supportedVideoFiles.slice(
      0,
      Math.max(videoRemainingSlots, 0)
    )

    if (
      videoFiles.length > 0 &&
      (videoRemainingSlots <= 0 ||
        supportedVideoFiles.length > referenceVideoFiles.length)
    ) {
      toast.error(t('Video2 accepts at most 3 video references.'))
    }
    if (selectedVideoBytes > CREATION_VIDEO_VIDEO_REFERENCE_MAX_BYTES) {
      toast.error(t('Reference videos must not exceed 200 MB total.'))
    }

    const audioFile = audioFiles.find(
      (file) => file.size <= CREATION_VIDEO_AUDIO_REFERENCE_MAX_BYTES
    )
    if (audioFiles.length > 0 && !audioFile) {
      toast.error(t('Reference audio must not exceed 15 MB.'))
    }
    if (
      !referenceImageFiles.length &&
      !referenceVideoFiles.length &&
      !audioFile
    ) {
      return
    }

    let imageUrls: CreationVideoReferences['imageUrls'] = []
    let videoUrls: CreationVideoReferences['videoUrls'] = []
    let audioUrl: CreationVideoReferences['audioUrl'] | undefined
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
      if (audioFile) {
        audioUrl = await createUploadedReferenceValue(
          audioFile,
          'audio',
          getReferenceAudioMime(audioFile)
        )
      }
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
          audioUrl: audioUrl ?? current.audioUrl,
        },
        selectedModel?.id
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
        selectedModel?.id
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
        selectedModel?.id
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
        selectedModel?.id
      )
    )
  }

  const removeVideoReferenceAudio = () => {
    setVideoReferences((current) =>
      normalizeCreationVideoReferences(
        {
          ...current,
          audioUrl: '',
        },
        selectedModel?.id
      )
    )
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
    if (mode === 'video') {
      const optionError = getCreationVideoOptionsError(
        videoOptions,
        selectedModel.id
      )
      if (optionError) {
        toast.error(t(optionError))
        return
      }
      const referenceError = getCreationVideoReferenceError(
        selectedModel.id,
        videoReferences
      )
      if (referenceError) {
        toast.error(t(referenceError))
        return
      }
    }
    if (mode === 'image') {
      const referenceError = getCreationImageReferenceError(
        selectedModel.id,
        imageReferences
      )
      if (referenceError) {
        toast.error(t(referenceError))
        return
      }
    }

    setSubmitting(true)
    setView('preview')
    const createdAt = Date.now()
    const videoRequestOptions =
      mode === 'video'
        ? getCreationVideoRequestOptions(
            videoOptions,
            selectedModel.id,
            videoReferences
          )
        : undefined
    const normalizedVideoOptions =
      mode === 'video'
        ? normalizeCreationVideoOptions(videoOptions, selectedModel.id)
        : undefined
    const normalizedVideoReferences =
      mode === 'video'
        ? normalizeCreationVideoReferences(videoReferences, selectedModel.id)
        : undefined
    const normalizedImageReferences =
      mode === 'image'
        ? normalizeCreationImageReferences(imageReferences, selectedModel.id)
        : undefined
    try {
      const nextResult = await submitCreationTask({
        mode,
        model: selectedModel,
        prompt: trimmedPrompt,
        assets,
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
        imageReferences: normalizedImageReferences,
        videoOptions: normalizedVideoOptions,
        videoReferences: normalizedVideoReferences,
      })
      if (nextResult.status === 'failed') {
        toast.error(nextResult.error || t('Creation task failed.'))
      } else if (nextResult.mode === 'video') {
        toast.success(t('Video task submitted. Refresh its status later.'))
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
        imageReferences: normalizedImageReferences,
        videoOptions: normalizedVideoOptions,
        videoReferences: normalizedVideoReferences,
      })
      toast.error(message)
    } finally {
      setSubmitting(false)
    }
  }

  const refreshVideoTask = async () => {
    if (!result?.taskId || result.mode !== 'video') return
    setRefreshingTask(true)
    try {
      const nextResult = await getCreationVideoTask({
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
    setVideoReferences(
      item.videoReferences
        ? normalizeCreationVideoReferences(item.videoReferences, item.model)
        : {
            ...EMPTY_CREATION_VIDEO_REFERENCES,
            imageUrls: [],
            videoUrls: [],
          }
    )
    setResult(item.result)
    if (item.videoOptions) {
      setVideoOptions(
        normalizeCreationVideoOptions(item.videoOptions, item.model)
      )
    }
    setView('preview')
  }

  const clearHistory = () => {
    if (typeof window !== 'undefined') {
      window.localStorage.removeItem(historyStorageKey)
    }
    setHistoryItems([])
    toast.success(t('Creation history cleared.'))
  }

  return (
    <PublicLayout showMainContainer={false}>
      <main className='dark:bg-background dark:text-foreground min-h-svh bg-[#f3f7fd] pt-16 text-slate-900'>
        <div className='grid min-h-[calc(100svh-4rem)] lg:grid-cols-[19rem_minmax(0,1fr)]'>
          <CreationSidebar
            mode={mode}
            models={models}
            selectedModel={selectedModel}
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
            onHistory={() => setView('history')}
            onNewSession={startNewSession}
            onManageCategories={() => setCategoryOpen(true)}
            onManageDescriptions={() => setDescriptionOpen(true)}
          />

          <section className='flex min-w-0 flex-col gap-4 p-3 md:p-5'>
            <div className='grid min-h-[36rem] flex-1 gap-4 xl:grid-cols-[minmax(20rem,0.86fr)_minmax(24rem,1.14fr)]'>
              <ModelHero mode={mode} model={selectedModel} />
              <CreationPreview
                mode={mode}
                model={selectedModel}
                view={view}
                assets={assets}
                historyItems={historyItems}
                result={result}
                now={previewNow}
                submitting={submitting}
                refreshingTask={refreshingTask}
                onViewChange={setView}
                onSelectHistory={selectHistoryItem}
                onClearHistory={clearHistory}
                onRefreshTask={refreshVideoTask}
                onRemoveAsset={removeAsset}
              />
            </div>

            <Composer
              prompt={prompt}
              assets={assets}
              authenticated={!!auth.user}
              mode={mode}
              model={selectedModel}
              imageReferences={imageReferences}
              imageReferencesSupported={imageReferencesSupported}
              videoOptions={videoOptions}
              videoReferences={videoReferences}
              videoCapabilities={videoCapabilities}
              resolutionOptions={resolutionOptions}
              durationOptions={durationOptions}
              submitting={submitting}
              sessionNumber={sessionNumber}
              onPromptChange={setPrompt}
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
