/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, TestTube, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { SettingsSection } from '../components/settings-section'
import {
  getMediaStorageSettings,
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
    response_url_path: provider.response_url_path.trim() || 'url',
  }
}

export function MediaStorageSettingsSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const settingsQuery = useQuery({
    queryKey: ['media-storage-settings'],
    queryFn: getMediaStorageSettings,
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
      const responseURLPath = provider.response_url_path.trim() || 'url'
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
    </SettingsSection>
  )
}
