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
import { useQuery } from '@tanstack/react-query'
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
  Send,
  Sparkles,
  Upload,
  Video,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { PublicLayout } from '@/components/layout'
import { getCreationCatalog } from './api'
import type { CreationMode, CreationModel, CreationView } from './types'

const MODES: CreationMode[] = ['chat', 'image', 'video']

export function CreationCenter() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { auth } = useAuthStore()
  const [mode, setMode] = useState<CreationMode>('chat')
  const [selectedByMode, setSelectedByMode] = useState<
    Partial<Record<CreationMode, string>>
  >({})
  const [view, setView] = useState<CreationView>('preview')
  const [prompt, setPrompt] = useState('')
  const [assets, setAssets] = useState<string[]>([])
  const [uploadOpen, setUploadOpen] = useState(false)
  const [sessionNumber, setSessionNumber] = useState(1)

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
  const selectedModel = useMemo(
    () =>
      models.find((model) => model.id === selectedByMode[mode]) ?? models[0],
    [mode, models, selectedByMode]
  )

  useEffect(() => {
    if (!models.length || selectedByMode[mode]) return
    setSelectedByMode((current) => ({ ...current, [mode]: models[0].id }))
  }, [mode, models, selectedByMode])

  const selectMode = (nextMode: CreationMode) => {
    setMode(nextMode)
    setView('preview')
  }

  const startNewSession = () => {
    setPrompt('')
    setView('preview')
    setSessionNumber((current) => current + 1)
    toast.success(t('A new creation session is ready.'))
  }

  const addFiles = (files: FileList | null) => {
    if (!files?.length) return
    setAssets((current) => [
      ...current,
      ...Array.from(files, (file) => file.name),
    ])
    setUploadOpen(false)
    setView('assets')
    toast.success(t('Assets added to the current session.'))
  }

  const submit = () => {
    if (!auth.user) {
      navigate({ to: '/sign-in', search: { redirect: '/creation' } })
      return
    }
    toast.info(t('Generation submission will be connected in the next step.'))
  }

  return (
    <PublicLayout showMainContainer={false}>
      <main className='dark:bg-background dark:text-foreground min-h-svh bg-[#f3f7fd] pt-16 text-slate-900'>
        <div className='grid min-h-[calc(100svh-4rem)] lg:grid-cols-[19rem_minmax(0,1fr)]'>
          <CreationSidebar
            mode={mode}
            models={models}
            selectedModel={selectedModel}
            loading={catalogQuery.isLoading}
            error={catalogQuery.isError}
            onModeChange={selectMode}
            onModelChange={(model) =>
              setSelectedByMode((current) => ({
                ...current,
                [mode]: model.id,
              }))
            }
            onHistory={() => setView('history')}
            onNewSession={startNewSession}
            onUpload={() => setUploadOpen(true)}
          />

          <section className='flex min-w-0 flex-col gap-4 p-3 md:p-5'>
            <div className='grid min-h-[36rem] flex-1 gap-4 xl:grid-cols-[minmax(20rem,0.86fr)_minmax(24rem,1.14fr)]'>
              <ModelHero mode={mode} model={selectedModel} />
              <CreationPreview
                mode={mode}
                model={selectedModel}
                view={view}
                assets={assets}
                onViewChange={setView}
              />
            </div>

            <Composer
              prompt={prompt}
              assets={assets}
              authenticated={!!auth.user}
              sessionNumber={sessionNumber}
              onPromptChange={setPrompt}
              onUpload={() => setUploadOpen(true)}
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
    </PublicLayout>
  )
}

type SidebarProps = {
  mode: CreationMode
  models: CreationModel[]
  selectedModel?: CreationModel
  loading: boolean
  error: boolean
  onModeChange: (mode: CreationMode) => void
  onModelChange: (model: CreationModel) => void
  onHistory: () => void
  onNewSession: () => void
  onUpload: () => void
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
          <div className='text-muted-foreground mb-2 text-xs font-medium'>
            {t('Available models')}
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
      <span>{label}</span>
    </button>
  )
}

function ModelButton(props: {
  model: CreationModel
  active: boolean
  onClick: () => void
}) {
  const { t } = useTranslation()

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
      {!!props.model.tags?.length && (
        <span className='mt-2 flex flex-wrap gap-1'>
          {props.model.tags.slice(0, 2).map((tag) => (
            <Badge
              key={tag}
              variant='secondary'
              className='h-4 px-1.5 text-[10px]'
            >
              {tag}
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
      </div>
    </section>
  )
}

type PreviewProps = {
  mode: CreationMode
  model?: CreationModel
  view: CreationView
  assets: string[]
  onViewChange: (view: CreationView) => void
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
          <AssetPreview assets={props.assets} />
        ) : props.view === 'history' ? (
          <HistoryPreview />
        ) : (
          <EmptyPreview mode={props.mode} model={props.model} />
        )}
      </div>
    </section>
  )
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

function AssetPreview(props: { assets: string[] }) {
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
          key={`${asset}-${index}`}
          className='bg-muted/40 flex min-w-0 items-center gap-3 rounded-lg border p-3'
        >
          <FileImage className='text-primary size-4 shrink-0' />
          <span className='truncate text-xs'>{asset}</span>
        </div>
      ))}
    </div>
  )
}

function HistoryPreview() {
  const { t } = useTranslation()

  return (
    <div className='max-w-md text-center'>
      <Clock3 className='text-muted-foreground mx-auto size-8' />
      <h3 className='mt-3 text-sm font-semibold'>
        {t('No session history yet')}
      </h3>
      <p className='text-muted-foreground mt-2 text-xs leading-5'>
        {t(
          'Completed creation tasks will appear here after submission is enabled.'
        )}
      </p>
    </div>
  )
}

type ComposerProps = {
  prompt: string
  assets: string[]
  authenticated: boolean
  sessionNumber: number
  onPromptChange: (value: string) => void
  onUpload: () => void
  onSubmit: () => void
}

function Composer(props: ComposerProps) {
  const { t } = useTranslation()

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
            placeholder={t(
              'Describe the task you want the selected model to complete...'
            )}
            className='min-h-20 resize-none border-0 px-0 py-1 shadow-none focus-visible:ring-0'
          />
          <div className='text-muted-foreground text-right text-[11px]'>
            {props.prompt.length}/5000
          </div>
        </div>
        <Button
          size='icon-lg'
          aria-label={t('Submit')}
          onClick={props.onSubmit}
          disabled={!props.prompt.trim()}
        >
          <Send className='size-4' />
        </Button>
      </div>
      <div className='bg-primary/5 text-primary border-primary/15 mt-3 flex flex-wrap items-center justify-between gap-2 rounded-lg border px-3 py-2 text-xs'>
        <span>
          {props.authenticated
            ? t('Submission is reserved for the next integration step.')
            : t('Sign in before submitting a real creation task.')}
        </span>
        <span className='text-muted-foreground'>
          {t('Session')} #{props.sessionNumber} · {props.assets.length}{' '}
          {t('assets')}
        </span>
      </div>
    </section>
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
