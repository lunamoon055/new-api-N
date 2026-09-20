/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, RefreshCw, RotateCcw, TestTube, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SettingsSection } from '../components/settings-section'
import {
  getMediaStorageSettings,
  getMediaTransferDashboard,
  retryMediaTransferJob,
  testMediaStorageProvider,
  updateMediaStorageSettings,
} from '../api'
import type { MediaStorageProvider } from '../types'

const createProvider = (): MediaStorageProvider => ({
  id: `media-${Date.now()}`,
  name: 'GHLINK ImgHub',
  enabled: true,
  upload_url: 'https://media.ghlink.top/upload?returnFormat=full',
  auth_header: 'Authorization',
  auth_prefix: 'Bearer ',
  token: '',
  field_name: 'file',
  priority: 0,
  response_url_path: '0.src',
})

const sanyueImgHubPreset = {
  upload_url: 'https://media.ghlink.top/upload?returnFormat=full',
  auth_header: 'Authorization',
  auth_prefix: 'Bearer ',
  field_name: 'file',
  response_url_path: '0.src',
} satisfies Partial<MediaStorageProvider>

function normalizeProvider(
  provider: MediaStorageProvider
): MediaStorageProvider {
  return {
    ...provider,
    id: provider.id.trim(),
    name: provider.name.trim(),
    upload_url: provider.upload_url.trim(),
    auth_header: provider.auth_header.trim() || 'Authorization',
    auth_prefix: provider.auth_prefix,
    field_name: provider.field_name.trim() || 'file',
    priority: Number.isFinite(provider.priority) ? provider.priority : 0,
    response_url_path: provider.response_url_path.trim() || '0.src',
  }
}

export function MediaStorageSettingsSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const settingsQuery = useQuery({
    queryKey: ['media-storage-settings'],
    queryFn: getMediaStorageSettings,
  })
  const transferQuery = useQuery({
    queryKey: ['media-transfer-dashboard'],
    queryFn: () => getMediaTransferDashboard(),
    refetchInterval: 15000,
  })
  const [draftProviders, setDraftProviders] = useState<
    MediaStorageProvider[] | null
  >(null)
  const [testingProviderId, setTestingProviderId] = useState<string | null>(
    null
  )
  const providers =
    draftProviders ??
    settingsQuery.data?.data?.providers ??
    ([] as MediaStorageProvider[])

  const setProviders = (
    update:
      | MediaStorageProvider[]
      | ((current: MediaStorageProvider[]) => MediaStorageProvider[])
  ) => {
    setDraftProviders((current) => {
      const base = current ?? settingsQuery.data?.data?.providers ?? []
      return typeof update === 'function' ? update(base) : update
    })
  }

  const saveMutation = useMutation({
    mutationFn: () =>
      updateMediaStorageSettings(providers.map(normalizeProvider)),
    onSuccess: () => {
      toast.success(t('Media storage settings saved'))
      setDraftProviders(null)
      queryClient.invalidateQueries({ queryKey: ['media-storage-settings'] })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const testMutation = useMutation({
    mutationFn: testMediaStorageProvider,
    onSuccess: (response) => {
      const url = response.data?.url
      toast.success(
        url
          ? `${t('Test upload succeeded')}: ${url}`
          : t('Test upload succeeded')
      )
    },
    onError: (error: Error) => toast.error(error.message),
    onSettled: () => setTestingProviderId(null),
  })
  const retryMutation = useMutation({
    mutationFn: retryMediaTransferJob,
    onSuccess: () => {
      toast.success(t('Media transfer retry scheduled'))
      queryClient.invalidateQueries({ queryKey: ['media-transfer-dashboard'] })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const updateProvider = (
    index: number,
    patch: Partial<MediaStorageProvider>
  ) => {
    setProviders((current) =>
      current.map((provider, providerIndex) =>
        providerIndex === index ? { ...provider, ...patch } : provider
      )
    )
  }

  const save = () => {
    for (const provider of providers) {
      if (!provider.id.trim() || !provider.upload_url.trim()) {
        toast.error(t('Each media storage provider needs an ID and upload URL'))
        return
      }
      try {
        const url = new URL(provider.upload_url)
        if (!['http:', 'https:'].includes(url.protocol)) throw new Error()
      } catch {
        toast.error(t('Provide a valid media storage upload URL'))
        return
      }
      const responseURLPath = provider.response_url_path.trim() || '0.src'
      if (
        !responseURLPath
          .split('.')
          .every((segment) => /^[A-Za-z0-9_-]+$/.test(segment))
      ) {
        toast.error(t('Provide a valid response URL path'))
        return
      }
    }
    saveMutation.mutate()
  }

  return (
    <SettingsSection
      title={t('Media storage')}
      description={t(
        'Copy generated images, videos and audio to one of the configured hosts. If every upload fails, the upstream URL or binary response is kept.'
      )}
    >
      <div className='space-y-4'>
        {settingsQuery.isLoading && (
          <p className='text-muted-foreground text-sm'>{t('Loading...')}</p>
        )}
        {!settingsQuery.isLoading && providers.length === 0 && (
          <p className='text-muted-foreground rounded-lg border border-dashed p-4 text-sm'>
            {t('No media storage providers configured')}
          </p>
        )}
        {providers.map((provider, index) => (
          <div
            key={provider.id || index}
            className='space-y-4 rounded-lg border p-4'
          >
            <div className='flex items-center justify-between gap-3'>
              <div>
                <p className='font-medium'>{provider.name || provider.id}</p>
                <p className='text-muted-foreground text-xs'>
                  {t('Priority')} {provider.priority}
                </p>
              </div>
              <div className='flex items-center gap-3'>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={
                    testMutation.isPending ||
                    saveMutation.isPending ||
                    settingsQuery.isLoading ||
                    draftProviders !== null
                  }
                  title={
                    draftProviders !== null
                      ? t('Save media storage settings before testing')
                      : undefined
                  }
                  onClick={() => {
                    setTestingProviderId(provider.id)
                    testMutation.mutate(provider.id)
                  }}
                  aria-label={t('Test upload for this provider')}
                >
                  <TestTube />
                  {testingProviderId === provider.id && testMutation.isPending
                    ? t('Testing upload...')
                    : t('Test upload')}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={saveMutation.isPending}
                  onClick={() => updateProvider(index, sanyueImgHubPreset)}
                >
                  {t('Apply Sanyue ImgHub preset')}
                </Button>
                <label className='text-muted-foreground flex items-center gap-2 text-sm'>
                  {t('Enabled')}
                  <Switch
                    checked={provider.enabled}
                    onCheckedChange={(enabled) =>
                      updateProvider(index, { enabled })
                    }
                    aria-label={t('Enable media storage provider')}
                  />
                </label>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-sm'
                  className='text-destructive'
                  onClick={() =>
                    setProviders((current) =>
                      current.filter(
                        (_, providerIndex) => providerIndex !== index
                      )
                    )
                  }
                  aria-label={t('Delete media storage provider')}
                >
                  <Trash2 />
                </Button>
              </div>
            </div>

            <div className='grid gap-4 md:grid-cols-2'>
              <label className='space-y-2 text-sm'>
                <span>{t('Provider ID')}</span>
                <Input
                  value={provider.id}
                  onChange={(event) =>
                    updateProvider(index, { id: event.target.value })
                  }
                  autoComplete='off'
                />
              </label>
              <label className='space-y-2 text-sm'>
                <span>{t('Display name')}</span>
                <Input
                  value={provider.name}
                  onChange={(event) =>
                    updateProvider(index, { name: event.target.value })
                  }
                  autoComplete='off'
                />
              </label>
            </div>

            <label className='block space-y-2 text-sm'>
              <span>{t('Response URL path')}</span>
              <Input
                value={provider.response_url_path}
                onChange={(event) =>
                  updateProvider(index, {
                    response_url_path: event.target.value,
                  })
                }
                placeholder='0.src'
                autoComplete='off'
              />
              <span className='text-muted-foreground text-xs'>
                {t('Dot-separated JSON path, for example 0.src or data.url')}
              </span>
            </label>

            <label className='block space-y-2 text-sm'>
              <span>{t('Upload URL')}</span>
              <Input
                type='url'
                inputMode='url'
                value={provider.upload_url}
                onChange={(event) =>
                  updateProvider(index, { upload_url: event.target.value })
                }
                placeholder='https://media.ghlink.top/upload?returnFormat=full'
                autoComplete='off'
              />
            </label>

            <div className='grid gap-4 md:grid-cols-2'>
              <label className='space-y-2 text-sm'>
                <span>{t('Auth header')}</span>
                <Input
                  value={provider.auth_header}
                  onChange={(event) =>
                    updateProvider(index, { auth_header: event.target.value })
                  }
                  placeholder='Authorization'
                  autoComplete='off'
                />
              </label>
              <label className='space-y-2 text-sm'>
                <span>{t('Auth prefix')}</span>
                <Input
                  value={provider.auth_prefix}
                  onChange={(event) =>
                    updateProvider(index, { auth_prefix: event.target.value })
                  }
                  placeholder={t('Leave empty for a raw token')}
                  autoComplete='off'
                />
              </label>
            </div>

            <div className='grid gap-4 md:grid-cols-[1fr_140px_140px]'>
              <label className='space-y-2 text-sm'>
                <span>{t('Token')}</span>
                <Input
                  type='password'
                  value={provider.token}
                  onChange={(event) =>
                    updateProvider(index, { token: event.target.value })
                  }
                  placeholder={t('Leave ******** unchanged to keep the token')}
                  autoComplete='new-password'
                />
              </label>
              <label className='space-y-2 text-sm'>
                <span>{t('File field')}</span>
                <Input
                  value={provider.field_name}
                  onChange={(event) =>
                    updateProvider(index, { field_name: event.target.value })
                  }
                  placeholder='file'
                  autoComplete='off'
                />
              </label>
              <label className='space-y-2 text-sm'>
                <span>{t('Priority')}</span>
                <Input
                  type='number'
                  min={0}
                  value={provider.priority}
                  onChange={(event) =>
                    updateProvider(index, {
                      priority: Number(event.target.value) || 0,
                    })
                  }
                  inputMode='numeric'
                />
              </label>
            </div>
          </div>
        ))}

        <div className='flex flex-wrap gap-2'>
          <Button
            type='button'
            variant='outline'
            onClick={() =>
              setProviders((current) => [...current, createProvider()])
            }
          >
            <Plus />
            {t('Add media storage provider')}
          </Button>
          <Button
            type='button'
            onClick={save}
            disabled={saveMutation.isPending || settingsQuery.isLoading}
          >
            {saveMutation.isPending
              ? t('Saving...')
              : t('Save media storage settings')}
          </Button>
        </div>
      </div>

      <MediaTransferMonitor
        dashboard={transferQuery.data?.data}
        isLoading={transferQuery.isLoading}
        isRefreshing={transferQuery.isFetching}
        onRefresh={() => transferQuery.refetch()}
        onRetry={(id) => retryMutation.mutate(id)}
        retryingId={
          retryMutation.isPending ? (retryMutation.variables ?? null) : null
        }
      />
    </SettingsSection>
  )
}

type MediaTransferMonitorProps = {
  dashboard?: Awaited<ReturnType<typeof getMediaTransferDashboard>>['data']
  isLoading: boolean
  isRefreshing: boolean
  onRefresh: () => void
  onRetry: (id: number) => void
  retryingId: number | null
}

function formatTransferTime(timestamp?: number) {
  if (!timestamp) return '-'
  return new Date(timestamp * 1000).toLocaleString()
}

function formatBytes(bytes: number) {
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function statusVariant(status: string) {
  if (status === 'FAILED') return 'destructive' as const
  if (status === 'READY') return 'secondary' as const
  return 'outline' as const
}

function MediaTransferMonitor({
  dashboard,
  isLoading,
  isRefreshing,
  onRefresh,
  onRetry,
  retryingId,
}: MediaTransferMonitorProps) {
  const { t } = useTranslation()
  const summary = dashboard?.summary
  return (
    <SettingsSection
      title={t('Transfer monitoring')}
      description={t(
        'Transfers are retried without generating upstream content again. Failed jobs stay available for manual review and retry.'
      )}
    >
      <div className='flex items-center justify-between gap-3'>
        <div className='grid flex-1 gap-3 sm:grid-cols-4'>
          <div className='rounded-md border p-3'>
            <p className='text-muted-foreground text-xs'>{t('Queue depth')}</p>
            <p className='text-lg font-semibold'>
              {summary?.queue_depth ?? '-'}
            </p>
          </div>
          <div className='rounded-md border p-3'>
            <p className='text-muted-foreground text-xs'>{t('Failed')}</p>
            <p className='text-lg font-semibold'>{summary?.failed ?? '-'}</p>
          </div>
          <div className='rounded-md border p-3'>
            <p className='text-muted-foreground text-xs'>
              {t('Recent failure rate')}
            </p>
            <p className='text-lg font-semibold'>
              {summary
                ? `${(summary.recent_failure_rate * 100).toFixed(1)}%`
                : '-'}
            </p>
          </div>
          <div className='rounded-md border p-3'>
            <p className='text-muted-foreground text-xs'>
              {t('Average transfer time')}
            </p>
            <p className='text-lg font-semibold'>
              {summary
                ? `${summary.average_transfer_seconds.toFixed(1)}s`
                : '-'}
            </p>
          </div>
        </div>
        <Button
          type='button'
          variant='outline'
          size='icon-sm'
          onClick={onRefresh}
          disabled={isRefreshing}
          aria-label={t('Refresh transfer monitoring')}
        >
          <RefreshCw className={isRefreshing ? 'animate-spin' : undefined} />
        </Button>
      </div>

      {isLoading && (
        <p className='text-muted-foreground text-sm'>{t('Loading...')}</p>
      )}
      {!isLoading && (!dashboard?.items || dashboard.items.length === 0) && (
        <p className='text-muted-foreground rounded-lg border border-dashed p-4 text-sm'>
          {t('No transfer jobs')}
        </p>
      )}
      {!!dashboard?.items?.length && (
        <div className='overflow-x-auto rounded-lg border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Task ID')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
                <TableHead>{t('Attempts')}</TableHead>
                <TableHead>{t('File size')}</TableHead>
                <TableHead>{t('Updated')}</TableHead>
                <TableHead>{t('Last error')}</TableHead>
                <TableHead className='text-right'>{t('Action')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {dashboard.items.map((job) => (
                <TableRow key={job.id}>
                  <TableCell className='font-mono text-xs'>
                    {job.task_id || job.id}
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(job.status)}>
                      {t(job.status)}
                    </Badge>
                  </TableCell>
                  <TableCell>{job.attempts}</TableCell>
                  <TableCell>
                    {job.byte_size ? formatBytes(job.byte_size) : '-'}
                  </TableCell>
                  <TableCell className='whitespace-nowrap text-xs'>
                    {formatTransferTime(job.updated_at)}
                  </TableCell>
                  <TableCell
                    className='max-w-[260px] truncate text-xs'
                    title={job.last_error}
                  >
                    {job.last_error || '-'}
                  </TableCell>
                  <TableCell className='text-right'>
                    {(job.status === 'FAILED' || job.status === 'RETRY') && (
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() => onRetry(job.id)}
                        disabled={retryingId === job.id}
                      >
                        <RotateCcw data-icon='inline-start' />
                        {retryingId === job.id
                          ? t('Retrying...')
                          : t('Retry transfer')}
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {!!dashboard?.providers?.length && (
        <div className='flex flex-wrap gap-2 text-xs'>
          {dashboard.providers.map((provider) => (
            <Badge
              key={provider.provider_id}
              variant={provider.available ? 'secondary' : 'destructive'}
            >
              {provider.provider_id}:{' '}
              {provider.available ? t('Circuit closed') : t('Circuit open')}
            </Badge>
          ))}
        </div>
      )}
    </SettingsSection>
  )
}
