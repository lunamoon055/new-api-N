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
import {
  ArrowRight,
  Bot,
  Clock3,
  FileImage,
  FolderUp,
  History,
  Image,
  Library,
  MessageSquare,
  Plus,
  RefreshCw,
  RotateCcw,
  Send,
  Settings2,
  Sparkles,
  Timer,
  Trash2,
  Upload,
  Video,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { PublicLayout } from '@/components/layout'
import {
  getCreationCatalog,
  getCreationErrorMessage,
  getCreationVideoTask,
  saveCreationModelCategories,
  submitCreationTask,
} from './api'
import {
  CREATION_RESOLUTION_OPTIONS,
  DEFAULT_CREATION_VIDEO_OPTIONS,
  formatCreationCountdown,
  getCreationCountdownSeconds,
  getCreationDurationOptions,
  getCreationHistoryStorageKey,
  getCreationTimedOut,
  getCreationVideoRequestOptions,
  loadCreationHistory,
  normalizeCreationVideoOptions,
  saveCreationHistory,
  upsertCreationHistoryItem,
  type CreationDuration,
  type CreationHistoryItem,
  type CreationResolution,
  type CreationVideoOptions,
} from './session'
import type {
  CreationAsset,
  CreationMode,
  CreationModel,
  CreationModelCost,
  CreationModelCategories,
  CreationModelGroup,
  CreationResult,
  CreationView,
} from './types'

const MODES: CreationMode[] = ['chat', 'image', 'video']
const MAX_TEXT_ASSET_CHARS = 8000
const MAX_INLINE_IMAGE_BYTES = 4 * 1024 * 1024

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
  const [uploadOpen, setUploadOpen] = useState(false)
  const [categoryOpen, setCategoryOpen] = useState(false)
  const [sessionNumber, setSessionNumber] = useState(1)
  const [result, setResult] = useState<CreationResult>()
  const [historyItems, setHistoryItems] = useState<CreationHistoryItem[]>([])
  const [videoOptions, setVideoOptions] = useState<CreationVideoOptions>(
    DEFAULT_CREATION_VIDEO_OPTIONS
  )
  const [previewNow, setPreviewNow] = useState(Date.now())
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
  const modeCounts = useMemo(
    () =>
      MODES.reduce(
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

  useEffect(() => {
    if (!models.length || selectedByMode[mode]) return
    setSelectedByMode((current) => ({ ...current, [mode]: models[0].id }))
  }, [mode, models, selectedByMode])

  useEffect(() => {
    if (typeof window === 'undefined') return
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

    setPreviewNow(Date.now())
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
    setVideoOptions((current) => {
      const normalized = normalizeCreationVideoOptions(
        current,
        selectedModel?.id
      )
      return normalized.duration === current.duration &&
        normalized.resolution === current.resolution
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
    setView('preview')
    setResult(undefined)
    setSessionNumber((current) => current + 1)
    toast.success(t('A new creation session is ready.'))
  }

  const addFiles = async (files: FileList | null) => {
    if (!files?.length) return
    const nextAssets = await Promise.all(Array.from(files, readCreationAsset))
    setAssets((current) => [...current, ...nextAssets])
    setUploadOpen(false)
    setView('assets')
    toast.success(t('Assets added to the current session.'))
  }

  const removeAsset = (index: number) => {
    setAssets((current) =>
      current.filter((_, itemIndex) => itemIndex !== index)
    )
    toast.success(t('Asset removed.'))
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

    setSubmitting(true)
    setView('preview')
    const createdAt = Date.now()
    const videoRequestOptions =
      mode === 'video'
        ? getCreationVideoRequestOptions(videoOptions, selectedModel.id)
        : undefined
    const normalizedVideoOptions =
      mode === 'video'
        ? normalizeCreationVideoOptions(videoOptions, selectedModel.id)
        : undefined
    try {
      const nextResult = await submitCreationTask({
        mode,
        model: selectedModel,
        prompt: trimmedPrompt,
        assets,
        videoOptions: normalizedVideoOptions,
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
        videoOptions: normalizedVideoOptions,
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
        videoOptions: normalizedVideoOptions,
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
            onUpload={() => setUploadOpen(true)}
            onManageCategories={() => setCategoryOpen(true)}
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
              videoOptions={videoOptions}
              durationOptions={durationOptions}
              submitting={submitting}
              sessionNumber={sessionNumber}
              onPromptChange={setPrompt}
              onVideoOptionsChange={setVideoOptions}
              onUpload={() => setUploadOpen(true)}
              onRemoveAsset={removeAsset}
              onSubmit={submit}
            />
          </section>
        </div>
      </main>

      <UploadDialog
        open={uploadOpen}
        onOpenChange={setUploadOpen}
        onFilesSelected={addFiles}
      />
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
    </PublicLayout>
  )
}

type SidebarProps = {
  mode: CreationMode
  models: CreationModel[]
  selectedModel?: CreationModel
  modeCounts: Record<CreationMode, number>
  loading: boolean
  error: boolean
  canManageCategories: boolean
  onModeChange: (mode: CreationMode) => void
  onModelChange: (model: CreationModel) => void
  onHistory: () => void
  onNewSession: () => void
  onUpload: () => void
  onManageCategories: () => void
}

function CreationSidebar(props: SidebarProps) {
  const { t } = useTranslation()

  return (
    <aside className='border-border/70 bg-background/90 flex flex-col border-r'>
      <div className='space-y-4 border-b p-4'>
        <div>
          <h1 className='text-lg font-semibold'>{t('Creation Center')}</h1>
          <p className='text-muted-foreground mt-1 text-xs leading-5'>
            {t(
              'Create chat, image, and video tasks with the models configured in your workspace.'
            )}
          </p>
        </div>
        <div className='grid grid-cols-3 gap-2' role='tablist'>
          {MODES.map((item) => (
            <ModeButton
              key={item}
              mode={item}
              count={props.modeCounts[item]}
              active={props.mode === item}
              onClick={() => props.onModeChange(item)}
            />
          ))}
        </div>
        <div className='grid grid-cols-2 gap-2'>
          <Button variant='outline' size='sm' onClick={props.onHistory}>
            <History />
            {t('History')}
          </Button>
          <Button variant='outline' size='sm' onClick={props.onNewSession}>
            <Plus />
            {t('New session')}
          </Button>
        </div>
      </div>

      <div className='space-y-4 p-4'>
        <button
          type='button'
          onClick={props.onUpload}
          className='border-border bg-card hover:bg-muted flex w-full items-center gap-3 rounded-lg border p-3 text-left transition-colors'
        >
          <span className='bg-primary/10 text-primary flex size-9 shrink-0 items-center justify-center rounded-lg'>
            <FolderUp className='size-4' />
          </span>
          <span className='min-w-0 flex-1'>
            <span className='block text-sm font-medium'>
              {t('Asset library')}
            </span>
            <span className='text-muted-foreground mt-0.5 block text-xs leading-4'>
              {t('Add reference files to the current session.')}
            </span>
          </span>
          <ArrowRight className='text-muted-foreground size-4 shrink-0' />
        </button>

        <div>
          <div className='mb-2 flex items-center justify-between gap-2'>
            <div className='text-muted-foreground text-xs font-medium'>
              {t('Available models')}
            </div>
            {props.canManageCategories && (
              <Button
                variant='ghost'
                size='xs'
                className='h-6 px-1.5 text-[11px]'
                onClick={props.onManageCategories}
              >
                <Settings2 className='size-3.5' />
                {t('Manage categories')}
              </Button>
            )}
          </div>
          <div className='space-y-2'>
            {props.loading ? (
              <ModelSkeletons />
            ) : props.error ? (
              <SidebarNotice>
                {t('Unable to load model catalog.')}
              </SidebarNotice>
            ) : props.models.length === 0 ? (
              <SidebarNotice>
                {t('No models are configured for this creation type.')}
              </SidebarNotice>
            ) : (
              props.models.map((model) => (
                <ModelButton
                  key={model.id}
                  model={model}
                  active={props.selectedModel?.id === model.id}
                  onClick={() => props.onModelChange(model)}
                />
              ))
            )}
          </div>
        </div>
      </div>

      <div className='mt-auto border-t p-4'>
        <div className='bg-muted/50 rounded-lg border p-3'>
          <div className='text-xs font-medium'>{t('Browsing mode')}</div>
          <p className='text-muted-foreground mt-1 text-xs leading-5'>
            {t(
              'The model catalog is synced with live configuration. Sign in before submitting a real task.'
            )}
          </p>
        </div>
      </div>
    </aside>
  )
}

function ModeButton(props: {
  mode: CreationMode
  count: number
  active: boolean
  onClick: () => void
}) {
  const { t } = useTranslation()
  const Icon =
    props.mode === 'chat'
      ? MessageSquare
      : props.mode === 'image'
        ? Image
        : Video
  const label =
    props.mode === 'chat'
      ? t('Chat')
      : props.mode === 'image'
        ? t('Image')
        : t('Video')

  return (
    <button
      type='button'
      role='tab'
      aria-selected={props.active}
      onClick={props.onClick}
      className={cn(
        'flex h-16 flex-col items-center justify-center gap-1 rounded-lg border text-xs transition-colors',
        props.active
          ? 'border-primary/40 bg-primary/10 text-primary'
          : 'border-border bg-card text-muted-foreground hover:bg-muted'
      )}
    >
      <Icon className='size-4' />
      <span>
        {label}
        {props.count > 0 ? ` ${props.count}` : ''}
      </span>
    </button>
  )
}

function formatCreationModelCost(
  cost: CreationModelCost | undefined,
  t: (key: string) => string
) {
  if (!cost) return t('Pricing pending')
  const groupSuffix =
    cost.group_ratio && cost.group_ratio !== 1
      ? ` · ${t('Group')} x${formatCostNumber(cost.group_ratio)}`
      : ''

  switch (cost.billing_mode) {
    case 'dynamic':
      return `${t('Dynamic pricing')}${groupSuffix}`
    case 'per_request': {
      const requestPrice = formatCurrencyFromUSD(cost.request_price, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })
      const requestQuota = cost.request_quota
        ? ` · ${formatQuota(cost.request_quota)}`
        : ''
      return `${requestPrice} ${t('per request')}${requestQuota}${groupSuffix}`
    }
    case 'per_token': {
      const inputPrice = formatCurrencyFromUSD(cost.input_price_per_million, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })
      const outputPrice = formatCurrencyFromUSD(cost.output_price_per_million, {
        digitsLarge: 4,
        digitsSmall: 6,
        abbreviate: false,
      })
      return `${t('Input')} ${inputPrice}/1M · ${t('Output')} ${outputPrice}/1M${groupSuffix}`
    }
  }
}

function formatCostNumber(value: number) {
  return Number.parseFloat(value.toFixed(6)).toString()
}

async function readCreationAsset(file: File): Promise<CreationAsset> {
  const asset: CreationAsset = {
    id: createCreationAssetId(file),
    name: file.name,
    type: file.type,
    size: file.size,
  }

  if (file.type.startsWith('image/') && file.size <= MAX_INLINE_IMAGE_BYTES) {
    asset.dataUrl = await readFileAsDataUrl(file)
    return asset
  }

  if (isReadableTextAsset(file)) {
    asset.text = await readFileAsText(file.slice(0, MAX_TEXT_ASSET_CHARS))
  }

  return asset
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

function createCreationAssetId(file: File) {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `${file.name}-${file.size}-${file.lastModified}-${Math.random()
    .toString(36)
    .slice(2, 8)}`
}

function isReadableTextAsset(file: File) {
  return (
    file.type.startsWith('text/') ||
    /\.(csv|json|md|txt|tsv|yaml|yml)$/i.test(file.name)
  )
}

function readFileAsDataUrl(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.addEventListener('load', () => resolve(String(reader.result || '')))
    reader.addEventListener('error', () => reject(reader.error))
    reader.readAsDataURL(file)
  })
}

function readFileAsText(file: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.addEventListener('load', () => resolve(String(reader.result || '')))
    reader.addEventListener('error', () => reject(reader.error))
    reader.readAsText(file)
  })
}

function ModelButton(props: {
  model: CreationModel
  active: boolean
  onClick: () => void
}) {
  const { t } = useTranslation()
  const costLabel = formatCreationModelCost(props.model.cost, t)
  const tagLabels: Record<string, string> = {
    advanced: t('Advanced'),
    chat: t('Chat'),
    code: t('Code'),
    fast: t('Fast'),
    generation: t('Generation'),
    image: t('Image'),
    reasoning: t('Reasoning'),
    video: t('Video'),
  }
  const visibleTags = props.model.tags?.filter((tag) => tag !== 'async') ?? []

  return (
    <button
      type='button'
      onClick={props.onClick}
      className={cn(
        'w-full rounded-lg border p-3 text-left transition-colors',
        props.active
          ? 'border-primary/45 bg-primary/8'
          : 'border-border bg-card hover:bg-muted'
      )}
    >
      <div className='flex items-start gap-2.5'>
        <span
          className={cn(
            'flex size-8 shrink-0 items-center justify-center rounded-lg',
            props.active
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground'
          )}
        >
          <Bot className='size-4' />
        </span>
        <span className='min-w-0'>
          <span className='block truncate text-xs font-semibold'>
            {props.model.id}
          </span>
          <span className='text-muted-foreground mt-1 line-clamp-2 block text-[11px] leading-4'>
            {props.model.description || t('Ready for creation tasks.')}
          </span>
        </span>
      </div>
      <span className='text-primary mt-2 block truncate text-[11px] font-medium'>
        {t('Consumption')}: {costLabel}
      </span>
      {!!visibleTags.length && (
        <span className='mt-2 flex flex-wrap gap-1'>
          {visibleTags.slice(0, 2).map((tag) => (
            <Badge
              key={tag}
              variant='secondary'
              className='h-4 px-1.5 text-[10px]'
            >
              {tagLabels[tag] ?? tag}
            </Badge>
          ))}
        </span>
      )}
    </button>
  )
}

function ModelSkeletons() {
  return (
    <>
      <Skeleton className='h-[5.25rem] w-full' />
      <Skeleton className='h-[5.25rem] w-full' />
      <Skeleton className='h-[5.25rem] w-full' />
    </>
  )
}

function SidebarNotice(props: { children: React.ReactNode }) {
  return (
    <div className='text-muted-foreground rounded-lg border border-dashed px-3 py-5 text-center text-xs leading-5'>
      {props.children}
    </div>
  )
}

function ModelHero(props: { mode: CreationMode; model?: CreationModel }) {
  const { t } = useTranslation()
  const title = props.model?.id || t('Select a model')
  const costLabel = props.model
    ? formatCreationModelCost(props.model.cost, t)
    : undefined
  const fallback =
    props.mode === 'chat'
      ? t('Choose a configured chat model for writing, coding, and analysis.')
      : props.mode === 'image'
        ? t(
            'Choose an image model and add references before composing a prompt.'
          )
        : t('Choose a video model and prepare a prompt for the next step.')

  return (
    <section className='bg-card flex min-h-[22rem] items-center justify-center rounded-lg border p-6 text-center'>
      <div className='max-w-lg'>
        <div className='bg-primary text-primary-foreground mx-auto flex size-16 items-center justify-center rounded-lg shadow-sm'>
          <Sparkles className='size-7' />
        </div>
        <div className='text-primary mt-5 flex items-center justify-center gap-2 text-[11px] font-medium'>
          <span className='bg-primary/30 h-px w-10' />
          {t('Current model')}
          <span className='bg-primary/30 h-px w-10' />
        </div>
        <h2 className='mt-3 text-2xl font-semibold break-words'>{title}</h2>
        <p className='text-muted-foreground mt-3 text-sm leading-6'>
          {props.model?.description || fallback}
        </p>
        {costLabel && (
          <div className='text-primary bg-primary/10 mx-auto mt-4 inline-flex max-w-full items-center rounded-full px-3 py-1 text-xs font-medium'>
            <span className='truncate'>
              {t('Consumption')}: {costLabel}
            </span>
          </div>
        )}
      </div>
    </section>
  )
}

type PreviewProps = {
  mode: CreationMode
  model?: CreationModel
  view: CreationView
  assets: CreationAsset[]
  historyItems: CreationHistoryItem[]
  result?: CreationResult
  now: number
  submitting: boolean
  refreshingTask: boolean
  onViewChange: (view: CreationView) => void
  onSelectHistory: (item: CreationHistoryItem) => void
  onClearHistory: () => void
  onRefreshTask: () => void
  onRemoveAsset: (index: number) => void
}

function CreationPreview(props: PreviewProps) {
  const { t } = useTranslation()

  return (
    <section className='bg-card flex min-h-[22rem] min-w-0 flex-col rounded-lg border'>
      <div className='flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('Creation workspace')}</h2>
        <div className='flex gap-1'>
          {(['preview', 'assets', 'history'] as CreationView[]).map((item) => (
            <Button
              key={item}
              variant={props.view === item ? 'secondary' : 'ghost'}
              size='xs'
              onClick={() => props.onViewChange(item)}
            >
              {item === 'preview'
                ? t('Preview')
                : item === 'assets'
                  ? t('Assets')
                  : t('History')}
            </Button>
          ))}
        </div>
      </div>

      <div className='flex min-h-0 flex-1 items-center justify-center p-4'>
        {props.view === 'assets' ? (
          <AssetPreview
            assets={props.assets}
            onRemoveAsset={props.onRemoveAsset}
          />
        ) : props.view === 'history' ? (
          <HistoryPreview
            items={props.historyItems}
            onClearHistory={props.onClearHistory}
            onSelectHistory={props.onSelectHistory}
          />
        ) : props.submitting ? (
          <SubmittingPreview mode={props.mode} />
        ) : props.result ? (
          <ResultPreview
            result={props.result}
            now={props.now}
            refreshingTask={props.refreshingTask}
            onRefreshTask={props.onRefreshTask}
          />
        ) : (
          <EmptyPreview mode={props.mode} model={props.model} />
        )}
      </div>
    </section>
  )
}

function SubmittingPreview(props: { mode: CreationMode }) {
  const { t } = useTranslation()
  const title =
    props.mode === 'video'
      ? t('Submitting async media task')
      : t('Submitting creation task')

  return (
    <div className='max-w-md text-center'>
      <div className='bg-primary/10 text-primary mx-auto flex size-14 items-center justify-center rounded-lg'>
        <RefreshCw className='size-6 animate-spin' />
      </div>
      <h3 className='mt-4 text-sm font-semibold'>{title}</h3>
      <p className='text-muted-foreground mt-2 text-xs leading-5'>
        {t('Please keep this page open while the request is being submitted.')}
      </p>
    </div>
  )
}

function ResultPreview(props: {
  result: CreationResult
  now: number
  refreshingTask: boolean
  onRefreshTask: () => void
}) {
  const { t } = useTranslation()
  const statusLabel = getStatusLabel(props.result.status, t)
  const countdownSeconds = getCreationCountdownSeconds(
    props.result.createdAt,
    props.result.estimateSeconds,
    props.now
  )
  const timedOut = getCreationTimedOut(
    props.result.createdAt,
    props.result.estimateSeconds,
    props.now
  )

  if (props.result.status === 'failed') {
    return (
      <div className='max-w-lg text-center'>
        <div className='bg-destructive/10 text-destructive mx-auto flex size-14 items-center justify-center rounded-lg'>
          <MessageSquare className='size-6' />
        </div>
        <h3 className='mt-4 text-sm font-semibold'>{t('Task failed')}</h3>
        <p className='text-muted-foreground mt-2 text-xs leading-5'>
          {props.result.error || t('The upstream provider returned an error.')}
        </p>
      </div>
    )
  }

  if (props.result.mode === 'chat') {
    return (
      <div className='w-full max-w-2xl'>
        <div className='text-muted-foreground mb-2 text-xs'>
          {props.result.model} · {statusLabel}
        </div>
        <div className='bg-muted/40 max-h-[30rem] overflow-auto rounded-lg p-4 text-sm leading-6 whitespace-pre-wrap'>
          {props.result.outputText || t('No text content was returned.')}
        </div>
      </div>
    )
  }

  if (props.result.mode === 'image') {
    return (
      <div className='w-full max-w-2xl text-center'>
        {props.result.imageUrl ? (
          <img
            src={props.result.imageUrl}
            alt={t('Generated image')}
            className='mx-auto max-h-[31rem] w-auto max-w-full rounded-lg object-contain'
          />
        ) : (
          <EmptyMediaResult title={t('Image task returned no image URL')} />
        )}
        {props.result.outputText && (
          <p className='text-muted-foreground mx-auto mt-3 max-w-xl text-xs leading-5'>
            {props.result.outputText}
          </p>
        )}
      </div>
    )
  }

  return (
    <div className='w-full max-w-2xl text-center'>
      {props.result.videoUrl ? (
        <video
          controls
          src={props.result.videoUrl}
          className='mx-auto max-h-[31rem] w-auto max-w-full rounded-lg'
        />
      ) : (
        <EmptyMediaResult title={t('Video task is waiting for a result')} />
      )}
      <div className='text-muted-foreground mt-4 space-y-1 text-xs leading-5'>
        <div>
          {props.result.model} · {statusLabel}
        </div>
        {!props.result.videoUrl && props.result.status !== 'completed' && (
          <div
            className={cn(
              'inline-flex items-center justify-center gap-1.5',
              timedOut ? 'text-amber-600 dark:text-amber-400' : 'text-primary'
            )}
          >
            <Timer className='size-3.5' />
            {countdownSeconds > 0
              ? `${t('Estimated remaining')} ${formatCreationCountdown(
                  countdownSeconds
                )}`
              : t(
                  'Generation is taking longer than expected. You can refresh later or check the task log.'
                )}
          </div>
        )}
        {(props.result.resolution || props.result.duration) && (
          <div>
            {[
              formatCreationResolution(props.result.resolution),
              props.result.duration && `${props.result.duration}s`,
            ]
              .filter(Boolean)
              .join(' · ')}
          </div>
        )}
        {props.result.taskId && <div>{props.result.taskId}</div>}
      </div>
      {props.result.taskId && (
        <Button
          variant='outline'
          size='sm'
          className='mt-4'
          onClick={props.onRefreshTask}
          disabled={props.refreshingTask}
        >
          <RefreshCw
            className={cn('size-4', props.refreshingTask && 'animate-spin')}
          />
          {t('Refresh status')}
        </Button>
      )}
    </div>
  )
}

function EmptyMediaResult(props: { title: string }) {
  return (
    <div className='bg-muted/40 text-muted-foreground flex min-h-56 items-center justify-center rounded-lg border border-dashed px-6 text-sm'>
      {props.title}
    </div>
  )
}

function getStatusLabel(
  status: CreationResult['status'],
  t: (key: string) => string
) {
  switch (status) {
    case 'queued':
      return t('Queued')
    case 'processing':
      return t('Processing')
    case 'completed':
      return t('Completed')
    case 'failed':
      return t('Failed')
    default:
      return t('Unknown status')
  }
}

function getCreationHistoryItemId(result: CreationResult, mode: CreationMode) {
  return (
    result.taskId ||
    result.id ||
    `${mode}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  )
}

function formatCreationResolution(resolution?: string) {
  switch (resolution) {
    case '1080p':
      return '1080'
    case '2k':
      return '2K'
    case '4k':
      return '4K'
    default:
      return resolution
  }
}

function formatCreationTime(value: number) {
  return new Date(value).toLocaleString()
}

function EmptyPreview(props: { mode: CreationMode; model?: CreationModel }) {
  const { t } = useTranslation()
  const Icon =
    props.mode === 'chat'
      ? MessageSquare
      : props.mode === 'image'
        ? Image
        : Video
  const title =
    props.mode === 'chat'
      ? t('No conversation yet')
      : props.mode === 'image'
        ? t('No image task yet')
        : t('No video task yet')

  return (
    <div className='max-w-md text-center'>
      <div className='bg-muted text-muted-foreground mx-auto flex size-14 items-center justify-center rounded-lg'>
        <Icon className='size-6' />
      </div>
      <h3 className='mt-4 text-sm font-semibold'>{title}</h3>
      <p className='text-muted-foreground mt-2 text-xs leading-5'>
        {props.model
          ? t(
              'Add assets if needed, write a prompt below, and sign in before submitting a real task.'
            )
          : t('Select a configured model from the sidebar to begin.')}
      </p>
    </div>
  )
}

function AssetPreview(props: {
  assets: CreationAsset[]
  onRemoveAsset: (index: number) => void
}) {
  const { t } = useTranslation()

  if (!props.assets.length) {
    return (
      <div className='text-muted-foreground text-center text-xs'>
        {t('No assets have been added to this session.')}
      </div>
    )
  }

  return (
    <div className='grid w-full gap-2 sm:grid-cols-2'>
      {props.assets.map((asset, index) => (
        <div
          key={asset.id}
          className='bg-muted/40 flex min-w-0 items-center gap-3 rounded-lg border p-3'
        >
          <FileImage className='text-primary size-4 shrink-0' />
          <span className='min-w-0 flex-1 truncate text-xs'>{asset.name}</span>
          <Button
            type='button'
            variant='ghost'
            size='icon-xs'
            className='ml-auto shrink-0'
            aria-label={`${t('Remove asset')}: ${asset.name}`}
            onClick={() => props.onRemoveAsset(index)}
          >
            <Trash2 className='size-3.5' />
          </Button>
        </div>
      ))}
    </div>
  )
}

function HistoryPreview(props: {
  items: CreationHistoryItem[]
  onClearHistory: () => void
  onSelectHistory: (item: CreationHistoryItem) => void
}) {
  const { t } = useTranslation()

  if (props.items.length) {
    return (
      <div className='flex h-full w-full flex-col gap-3'>
        <div className='flex items-center justify-between gap-3'>
          <div>
            <h3 className='text-sm font-semibold'>{t('History')}</h3>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t('History is saved in this browser.')}
            </p>
          </div>
          <Button variant='ghost' size='sm' onClick={props.onClearHistory}>
            <Trash2 className='size-4' />
            {t('Clear history')}
          </Button>
        </div>
        <div className='min-h-0 flex-1 space-y-2 overflow-auto pr-1'>
          {props.items.map((item) => (
            <button
              key={item.id}
              type='button'
              onClick={() => props.onSelectHistory(item)}
              className='border-border bg-muted/30 hover:bg-muted flex w-full min-w-0 flex-col gap-2 rounded-lg border p-3 text-left transition-colors'
            >
              <span className='flex min-w-0 items-center justify-between gap-2'>
                <span className='min-w-0 truncate text-xs font-semibold'>
                  {item.model}
                </span>
                <Badge variant='secondary' className='shrink-0 text-[10px]'>
                  {getStatusLabel(item.result.status, t)}
                </Badge>
              </span>
              <span className='text-muted-foreground line-clamp-2 text-xs leading-5'>
                {item.prompt}
              </span>
              <span className='text-muted-foreground flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px]'>
                <span>{formatCreationTime(item.createdAt)}</span>
                {item.result.taskId && <span>{item.result.taskId}</span>}
              </span>
            </button>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className='max-w-md text-center'>
      <Clock3 className='text-muted-foreground mx-auto size-8' />
      <h3 className='mt-3 text-sm font-semibold'>
        {t('No session history yet')}
      </h3>
      <p className='text-muted-foreground mt-2 text-xs leading-5'>
        {t('Completed creation tasks will appear here after you submit tasks.')}
      </p>
    </div>
  )
}

type ComposerProps = {
  prompt: string
  assets: CreationAsset[]
  authenticated: boolean
  mode: CreationMode
  model?: CreationModel
  videoOptions: CreationVideoOptions
  durationOptions: ReturnType<typeof getCreationDurationOptions>
  submitting: boolean
  sessionNumber: number
  onPromptChange: (value: string) => void
  onVideoOptionsChange: (options: CreationVideoOptions) => void
  onUpload: () => void
  onRemoveAsset: (index: number) => void
  onSubmit: () => void
}

function Composer(props: ComposerProps) {
  const { t } = useTranslation()
  const canSubmit = !!props.prompt.trim() && !!props.model && !props.submitting

  return (
    <section className='bg-card rounded-lg border p-3'>
      <div className='flex min-w-0 items-start gap-3'>
        <Button
          variant='outline'
          className='h-20 w-20 shrink-0 flex-col gap-1 text-xs'
          onClick={props.onUpload}
        >
          <Upload className='size-4' />
          {t('Add assets')}
        </Button>
        <div className='min-w-0 flex-1'>
          <Textarea
            aria-label={t('Prompt')}
            value={props.prompt}
            maxLength={5000}
            onChange={(event) => props.onPromptChange(event.target.value)}
            onKeyDown={(event) => {
              if (
                event.key !== 'Enter' ||
                event.shiftKey ||
                event.nativeEvent.isComposing
              ) {
                return
              }
              event.preventDefault()
              if (canSubmit) props.onSubmit()
            }}
            placeholder={t(
              'Describe the task you want the selected model to complete...'
            )}
            className='min-h-20 resize-none border-0 px-0 py-1 shadow-none focus-visible:ring-0'
          />
          {!!props.assets.length && (
            <div className='mt-2 flex flex-wrap gap-1.5'>
              {props.assets.map((asset, index) => (
                <button
                  key={asset.id}
                  type='button'
                  onClick={() => props.onRemoveAsset(index)}
                  className='bg-muted text-muted-foreground hover:bg-muted/80 inline-flex max-w-full items-center gap-1.5 rounded-md border px-2 py-1 text-[11px] transition-colors'
                  aria-label={`${t('Remove asset')}: ${asset.name}`}
                >
                  <FileImage className='size-3 shrink-0' />
                  <span className='truncate'>{asset.name}</span>
                  <Trash2 className='size-3 shrink-0' />
                </button>
              ))}
            </div>
          )}
          <div className='text-muted-foreground text-right text-[11px]'>
            {props.prompt.length}/5000
          </div>
        </div>
        <Button
          size='icon-lg'
          aria-label={t('Submit')}
          onClick={props.onSubmit}
          disabled={!canSubmit}
        >
          {props.submitting ? (
            <RefreshCw className='size-4 animate-spin' />
          ) : (
            <Send className='size-4' />
          )}
        </Button>
      </div>
      {props.mode === 'video' && (
        <div className='border-border/70 mt-3 grid gap-3 border-t pt-3 sm:grid-cols-2'>
          <ComposerOptionGroup
            label={t('Resolution')}
            value={props.videoOptions.resolution}
            options={CREATION_RESOLUTION_OPTIONS}
            onChange={(value) =>
              props.onVideoOptionsChange({
                ...props.videoOptions,
                resolution: value as CreationResolution,
              })
            }
          />
          <ComposerOptionGroup
            label={t('Video duration')}
            value={props.videoOptions.duration}
            options={props.durationOptions}
            onChange={(value) =>
              props.onVideoOptionsChange({
                ...props.videoOptions,
                duration: value as CreationDuration,
              })
            }
          />
        </div>
      )}
      <div className='text-muted-foreground mt-3 flex flex-wrap items-center justify-end gap-2 rounded-lg border px-3 py-2 text-xs'>
        {!props.authenticated && (
          <span className='mr-auto'>
            {t('Sign in before submitting a real creation task.')}
          </span>
        )}
        <span className='text-muted-foreground'>
          {t('Session')} #{props.sessionNumber} · {props.assets.length}{' '}
          {t('assets')}
        </span>
        <span>{t('Press Enter to send, Shift+Enter for newline.')}</span>
      </div>
    </section>
  )
}

function ComposerOptionGroup(props: {
  label: string
  value: string
  options: Array<{ value: string; label: string }>
  onChange: (value: string) => void
}) {
  return (
    <div className='flex min-w-0 items-center gap-2'>
      <span className='text-muted-foreground w-14 shrink-0 text-xs font-medium'>
        {props.label}
      </span>
      <div className='bg-muted/40 grid min-w-0 flex-1 grid-cols-3 gap-1 rounded-lg border p-1'>
        {props.options.map((option) => (
          <button
            key={option.value}
            type='button'
            onClick={() => props.onChange(option.value)}
            className={cn(
              'h-8 rounded-md px-2 text-xs font-medium transition-colors',
              props.value === option.value
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-background'
            )}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  )
}

type CreationCategoryRow = CreationModel & {
  mode: CreationMode
}

function getCreationCategoryRows(groups: CreationModelGroup[]) {
  const rows = new Map<string, CreationCategoryRow>()
  for (const group of groups) {
    for (const model of group.models) {
      const key = model.id.toLowerCase()
      if (rows.has(key)) continue
      rows.set(key, { ...model, mode: group.mode })
    }
  }
  return [...rows.values()].sort((a, b) => a.id.localeCompare(b.id))
}

function getCreationModeLabel(mode: CreationMode, t: (key: string) => string) {
  switch (mode) {
    case 'chat':
      return t('Chat')
    case 'image':
      return t('Image')
    case 'video':
      return t('Video')
  }
}

type ModelCategoryDialogProps = {
  open: boolean
  models: CreationCategoryRow[]
  saving: boolean
  onOpenChange: (open: boolean) => void
  onSave: (categories: CreationModelCategories) => void
  onReset: () => void
}

function ModelCategoryDialog(props: ModelCategoryDialogProps) {
  const { t } = useTranslation()
  const save = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const formData = new FormData(event.currentTarget)
    const categories = props.models.reduce<CreationModelCategories>(
      (next, model) => {
        const value = formData.get(model.id)
        next[model.id] =
          typeof value === 'string' && MODES.includes(value as CreationMode)
            ? (value as CreationMode)
            : model.mode
        return next
      },
      {}
    )
    props.onSave(categories)
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-w-3xl'>
        <form onSubmit={save}>
          <DialogHeader>
            <DialogTitle>{t('Creation model category management')}</DialogTitle>
            <DialogDescription>
              {t(
                'Manually assign visible creation models to chat, image, or video filters.'
              )}
            </DialogDescription>
          </DialogHeader>

          <div className='mt-4 rounded-lg border'>
            <div className='bg-muted/40 grid grid-cols-[minmax(0,1fr)_8.5rem] gap-3 border-b px-3 py-2 text-xs font-medium'>
              <span>{t('Model')}</span>
              <span>{t('Category')}</span>
            </div>
            <div className='max-h-[min(28rem,60svh)] overflow-auto'>
              {props.models.length === 0 ? (
                <div className='text-muted-foreground px-3 py-8 text-center text-sm'>
                  {t('No creation models available.')}
                </div>
              ) : (
                <div className='divide-y'>
                  {props.models.map((model) => (
                    <div
                      key={model.id}
                      className='grid grid-cols-[minmax(0,1fr)_8.5rem] items-center gap-3 px-3 py-3'
                    >
                      <div className='min-w-0'>
                        <div className='truncate text-sm font-medium'>
                          {model.id}
                        </div>
                        <div className='text-muted-foreground mt-1 line-clamp-1 text-xs'>
                          {model.description || t('Ready for creation tasks.')}
                        </div>
                      </div>
                      <NativeSelect
                        size='sm'
                        className='w-full'
                        aria-label={t('Category')}
                        name={model.id}
                        defaultValue={model.mode}
                        disabled={props.saving}
                      >
                        {MODES.map((item) => (
                          <NativeSelectOption key={item} value={item}>
                            {getCreationModeLabel(item, t)}
                          </NativeSelectOption>
                        ))}
                      </NativeSelect>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <DialogFooter className='mt-4 sm:justify-between'>
            <Button
              type='button'
              variant='outline'
              onClick={props.onReset}
              disabled={props.saving}
            >
              <RotateCcw className='size-4' />
              {t('Reset to auto')}
            </Button>
            <Button
              type='submit'
              disabled={props.saving || props.models.length === 0}
            >
              {props.saving ? (
                <RefreshCw className='size-4 animate-spin' />
              ) : (
                <Settings2 className='size-4' />
              )}
              {t('Save categories')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

type UploadDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onFilesSelected: (files: FileList | null) => void
}

function UploadDialog(props: UploadDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('Add assets')}</DialogTitle>
          <DialogDescription>
            {t(
              'Choose local reference files for the current creation session.'
            )}
          </DialogDescription>
        </DialogHeader>
        <label className='border-border bg-muted/30 hover:bg-muted flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-5 py-10 text-center transition-colors'>
          <Library className='text-primary size-7' />
          <span className='text-sm font-medium'>{t('Choose files')}</span>
          <span className='text-muted-foreground text-xs'>
            {t('Files stay in this browser preview until upload is connected.')}
          </span>
          <input
            type='file'
            className='sr-only'
            multiple
            onChange={(event) => props.onFilesSelected(event.target.files)}
          />
        </label>
        <DialogFooter>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Cancel')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
