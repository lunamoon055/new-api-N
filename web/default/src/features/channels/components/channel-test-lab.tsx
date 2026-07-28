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
import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import {
  CheckCircle2,
  Clipboard,
  Loader2,
  Play,
  RefreshCw,
  XCircle,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { TitledCard } from '@/components/ui/titled-card'
import { getChannels, testChannelLab } from '../api'
import {
  channelsQueryKeys,
  formatResponseTime,
  getChannelTypeLabel,
  supportsChannelConnectionTest,
} from '../lib'
import {
  CHANNEL_TEST_TEMPLATES,
  getChannelTestPreviewPayload,
  getChannelTestTemplate,
  resolveChannelTestEndpointType,
  type ChannelTestEndpointType,
} from '../lib/channel-test-lab'
import type { Channel, ChannelTestResponse } from '../types'

type LabResult = {
  channelId: number
  model: string
  endpointType: ChannelTestEndpointType
  stream: boolean
  response: ChannelTestResponse
  requestParams: Record<string, unknown>
  previewPayload: Record<string, unknown>
  testedAt: string
}

function getInitialModel(channels: Channel[], channelId: string) {
  const channel = channels.find((item) => String(item.id) === channelId)
  const testModel = channel?.test_model?.trim()
  if (testModel) return testModel

  const firstModel = channel?.models
    ?.split(',')
    .map((item) => item.trim())
    .find(Boolean)

  return firstModel ?? ''
}

function getElapsedSeconds(response: ChannelTestResponse) {
  if (typeof response.time === 'number') return response.time
  return response.data?.response_time
}

function formatJson(value: unknown) {
  return JSON.stringify(value, null, 2)
}

function parseTemplateJson(value: string) {
  return JSON.parse(value) as Record<string, unknown>
}

function getTemplateParseError(value: string) {
  if (!value.trim()) {
    return 'Request template cannot be empty'
  }
  try {
    parseTemplateJson(value)
    return null
  } catch (error) {
    return error instanceof Error ? error.message : 'Invalid JSON'
  }
}

function ResultStatus({ result }: { result: LabResult | null }) {
  const { t } = useTranslation()

  if (!result) {
    return (
      <Badge variant='secondary' className='gap-1.5'>
        {t('Waiting for test')}
      </Badge>
    )
  }

  if (result.response.success) {
    return (
      <Badge variant='default' className='gap-1.5'>
        <CheckCircle2 className='size-3.5' />
        {t('Test passed')}
      </Badge>
    )
  }

  return (
    <Badge variant='destructive' className='gap-1.5'>
      <XCircle className='size-3.5' />
      {t('Test failed')}
    </Badge>
  )
}

export function ChannelTestLab() {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()
  const [channelId, setChannelId] = useState('')
  const [model, setModel] = useState('')
  const [endpointType, setEndpointType] =
    useState<ChannelTestEndpointType>('auto')
  const [stream, setStream] = useState(false)
  const [requestTemplateText, setRequestTemplateText] = useState('')
  const [templateDirty, setTemplateDirty] = useState(false)
  const [result, setResult] = useState<LabResult | null>(null)

  const channelsQuery = useQuery({
    queryKey: channelsQueryKeys.list({
      p: 1,
      page_size: 200,
      id_sort: true,
    }),
    queryFn: () => getChannels({ p: 1, page_size: 200, id_sort: true }),
  })

  const channels = useMemo(
    () => channelsQuery.data?.data?.items ?? [],
    [channelsQuery.data?.data?.items]
  )

  const selectedChannel = useMemo(
    () => channels.find((item) => String(item.id) === channelId),
    [channelId, channels]
  )
  const channelModels = useMemo(
    () =>
      selectedChannel?.models
        ?.split(',')
        .map((item) => item.trim())
        .filter(Boolean) ?? [],
    [selectedChannel?.models]
  )

  const fallbackTemplate = getChannelTestTemplate(endpointType)
  const effectiveModel = model.trim() || fallbackTemplate.defaultModel
  const resolvedEndpointType = resolveChannelTestEndpointType(
    endpointType,
    effectiveModel,
    selectedChannel
  )
  const selectedTemplate = getChannelTestTemplate(resolvedEndpointType)
  const defaultPayload = getChannelTestPreviewPayload(
    resolvedEndpointType,
    effectiveModel
  )
  const defaultTemplateText = formatJson(defaultPayload)
  const activeRequestTemplateText = templateDirty
    ? requestTemplateText
    : defaultTemplateText
  const templateParseError = getTemplateParseError(activeRequestTemplateText)
  const requestParams = {
    channel_id: Number(channelId) || null,
    model: effectiveModel,
    endpoint_type:
      resolvedEndpointType === 'auto' ? undefined : resolvedEndpointType,
    stream: selectedTemplate.supportsStream ? stream : false,
  }

  const runTestMutation = useMutation({
    mutationFn: async () => {
      const numericChannelId = Number(channelId)
      if (!numericChannelId) {
        throw new Error(t('Please select a channel first'))
      }
      if (
        selectedChannel &&
        !supportsChannelConnectionTest(selectedChannel.type)
      ) {
        throw new Error(
          t('This channel type does not support connection tests')
        )
      }
      const finalModel = effectiveModel.trim()
      if (!finalModel) {
        throw new Error(t('Please enter a test model'))
      }
      if (templateParseError) {
        throw new Error(t('Invalid request template JSON'))
      }

      const editedPayload = parseTemplateJson(activeRequestTemplateText)
      const payloadModel =
        typeof editedPayload.model === 'string'
          ? editedPayload.model.trim()
          : ''
      const requestModel = payloadModel || finalModel

      const params = {
        model: requestModel,
        ...(resolvedEndpointType === 'auto'
          ? {}
          : { endpoint_type: resolvedEndpointType }),
        ...(selectedTemplate.supportsStream && stream ? { stream: true } : {}),
        payload: editedPayload,
      }
      const response = await testChannelLab(numericChannelId, params)
      return {
        channelId: numericChannelId,
        model: requestModel,
        endpointType: resolvedEndpointType,
        stream: selectedTemplate.supportsStream && stream,
        response,
        requestParams: {
          channel_id: numericChannelId,
          ...params,
        },
        previewPayload: editedPayload,
        testedAt: new Date().toLocaleString(),
      } satisfies LabResult
    },
    onSuccess: (nextResult) => {
      setResult(nextResult)
      if (nextResult.response.success) {
        toast.success(t('Test passed'))
      } else {
        toast.error(nextResult.response.message || t('Test failed'))
      }
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Test failed'))
    },
  })

  const handleChannelChange = (value: string | null) => {
    if (!value) return
    setChannelId(value)
    const nextModel = getInitialModel(channels, value)
    if (nextModel) {
      setModel(nextModel)
      setTemplateDirty(false)
    }
  }

  const handleEndpointChange = (value: string | null) => {
    if (!value) return
    const nextEndpointType = value as ChannelTestEndpointType
    const nextTemplate = getChannelTestTemplate(nextEndpointType)
    setEndpointType(nextEndpointType)
    setTemplateDirty(false)
    if (!nextTemplate.supportsStream) {
      setStream(false)
    }
    if (!model.trim() && nextTemplate.defaultModel) {
      setModel(nextTemplate.defaultModel)
    }
  }

  const selectedChannelUnsupported =
    selectedChannel && !supportsChannelConnectionTest(selectedChannel.type)
  const isRunDisabled =
    runTestMutation.isPending ||
    !Number(channelId) ||
    !effectiveModel.trim() ||
    Boolean(selectedChannelUnsupported) ||
    Boolean(templateParseError)
  const elapsedSeconds = result ? getElapsedSeconds(result.response) : undefined

  return (
    <div className='grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(360px,0.85fr)]'>
      <TitledCard
        title={t('API Test Lab')}
        description={t(
          'Quickly validate a channel, model, and endpoint before wiring it into production flows'
        )}
        icon={<Play className='size-4' />}
      >
        <div className='grid gap-4 lg:grid-cols-2'>
          <div className='grid gap-2'>
            <Label htmlFor='channel-test-lab-channel'>{t('Channel')}</Label>
            <Select
              value={channelId}
              onValueChange={handleChannelChange}
              disabled={channelsQuery.isLoading}
            >
              <SelectTrigger id='channel-test-lab-channel' className='w-full'>
                <SelectValue
                  placeholder={
                    channelsQuery.isLoading
                      ? t('Loading channels...')
                      : t('Select a channel')
                  }
                />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {channels.map((channel) => (
                    <SelectItem key={channel.id} value={String(channel.id)}>
                      #{channel.id} · {channel.name} ·{' '}
                      {t(getChannelTypeLabel(channel.type))}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            {selectedChannelUnsupported && (
              <p className='text-destructive text-xs'>
                {t('This channel type does not support connection tests')}
              </p>
            )}
          </div>

          <div className='grid gap-2'>
            <Label htmlFor='channel-test-lab-model'>{t('Test Model')}</Label>
            <Select
              value={channelModels.includes(model.trim()) ? model.trim() : ''}
              onValueChange={(value) => {
                if (!value) return
                setModel(value)
                setTemplateDirty(false)
              }}
              disabled={channelModels.length === 0}
            >
              <SelectTrigger className='w-full'>
                <SelectValue
                  placeholder={
                    channelModels.length > 0
                      ? t('Choose from channel models')
                      : t('No channel models available')
                  }
                />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {channelModels.map((item) => (
                    <SelectItem key={item} value={item}>
                      {item}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <Input
              id='channel-test-lab-model'
              value={model}
              onChange={(event) => setModel(event.target.value)}
              placeholder={selectedTemplate.defaultModel || 'gpt-4o-mini'}
            />
            <p className='text-muted-foreground text-xs'>
              {t('You can also type or edit the model name manually')}
            </p>
          </div>

          <div className='grid gap-2'>
            <Label htmlFor='channel-test-lab-endpoint'>
              {t('Endpoint Type')}
            </Label>
            <Select
              value={resolvedEndpointType}
              onValueChange={handleEndpointChange}
            >
              <SelectTrigger id='channel-test-lab-endpoint' className='w-full'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {CHANNEL_TEST_TEMPLATES.map((template) => (
                    <SelectItem
                      key={template.endpointType}
                      value={template.endpointType}
                    >
                      {t(template.labelKey)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <p className='text-muted-foreground text-xs'>
              {t(selectedTemplate.descriptionKey)}
            </p>
          </div>

          <div className='grid gap-2'>
            <Label>{t('Stream')}</Label>
            <div
              className={cn(
                'border-input flex h-9 items-center justify-between rounded-md border px-3',
                !selectedTemplate.supportsStream && 'opacity-60'
              )}
            >
              <span className='text-sm'>
                {selectedTemplate.supportsStream
                  ? t('Enable stream test')
                  : t('This endpoint does not support stream tests')}
              </span>
              <Switch
                checked={selectedTemplate.supportsStream && stream}
                onCheckedChange={setStream}
                disabled={!selectedTemplate.supportsStream}
              />
            </div>
          </div>
        </div>

        <div className='mt-4 grid gap-2'>
          <div className='flex items-center justify-between gap-2'>
            <Label htmlFor='channel-test-lab-preview'>
              {t('Request Template Preview')}
            </Label>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={() => void copyToClipboard(activeRequestTemplateText)}
            >
              <Clipboard className='mr-2 size-4' />
              {t('Copy')}
            </Button>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={() => {
                setTemplateDirty(false)
                setRequestTemplateText('')
              }}
            >
              <RefreshCw className='mr-2 size-4' />
              {t('Reset template')}
            </Button>
          </div>
          <Textarea
            id='channel-test-lab-preview'
            value={activeRequestTemplateText}
            onChange={(event) => {
              setRequestTemplateText(event.target.value)
              setTemplateDirty(true)
            }}
            className='min-h-44 resize-y font-mono text-xs'
          />
          {templateParseError ? (
            <p className='text-destructive text-xs'>
              {t('Invalid JSON: {{message}}', {
                message: templateParseError,
              })}
            </p>
          ) : (
            <p className='text-muted-foreground text-xs'>
              {t(
                'This editable template will be submitted to the backend test endpoint.'
              )}
            </p>
          )}
        </div>

        <div className='mt-4 flex flex-wrap items-center justify-between gap-2'>
          <Button
            type='button'
            variant='outline'
            onClick={() => channelsQuery.refetch()}
            disabled={channelsQuery.isFetching}
          >
            <RefreshCw
              className={cn(
                'mr-2 size-4',
                channelsQuery.isFetching && 'animate-spin'
              )}
            />
            {t('Refresh channels')}
          </Button>
          <Button
            type='button'
            onClick={() => runTestMutation.mutate()}
            disabled={isRunDisabled}
          >
            {runTestMutation.isPending ? (
              <Loader2 className='mr-2 size-4 animate-spin' />
            ) : (
              <Play className='mr-2 size-4' />
            )}
            {t('Run Test')}
          </Button>
        </div>
      </TitledCard>

      <TitledCard
        title={t('Test Result')}
        description={t(
          'Inspect the final test parameters and backend response'
        )}
        action={<ResultStatus result={result} />}
      >
        <div className='space-y-4'>
          <div className='grid gap-2 rounded-md border p-3 text-sm'>
            <div className='flex items-center justify-between gap-2'>
              <span className='text-muted-foreground'>{t('Channel')}</span>
              <span className='font-medium'>
                {result?.channelId
                  ? `#${result.channelId}`
                  : selectedChannel
                    ? `#${selectedChannel.id}`
                    : '-'}
              </span>
            </div>
            <div className='flex items-center justify-between gap-2'>
              <span className='text-muted-foreground'>{t('Model')}</span>
              <span className='max-w-56 truncate font-medium'>
                {result?.model || effectiveModel || '-'}
              </span>
            </div>
            <div className='flex items-center justify-between gap-2'>
              <span className='text-muted-foreground'>{t('Elapsed Time')}</span>
              <span className='font-medium'>
                {typeof elapsedSeconds === 'number'
                  ? formatResponseTime(elapsedSeconds * 1000, t)
                  : '-'}
              </span>
            </div>
            <div className='flex items-center justify-between gap-2'>
              <span className='text-muted-foreground'>{t('Tested At')}</span>
              <span className='font-medium'>{result?.testedAt || '-'}</span>
            </div>
          </div>

          <div className='grid gap-2'>
            <div className='flex items-center justify-between gap-2'>
              <Label>{t('Request Parameters')}</Label>
              <Button
                type='button'
                variant='ghost'
                size='sm'
                onClick={() =>
                  void copyToClipboard(
                    formatJson(result?.requestParams ?? requestParams)
                  )
                }
              >
                <Clipboard className='mr-2 size-4' />
                {t('Copy')}
              </Button>
            </div>
            <pre className='bg-muted/60 max-h-52 overflow-auto rounded-md border p-3 font-mono text-xs whitespace-pre-wrap'>
              {formatJson(result?.requestParams ?? requestParams)}
            </pre>
          </div>

          <div className='grid gap-2'>
            <div className='flex items-center justify-between gap-2'>
              <Label>{t('Backend Response')}</Label>
              <Button
                type='button'
                variant='ghost'
                size='sm'
                onClick={() =>
                  void copyToClipboard(formatJson(result?.response ?? null))
                }
                disabled={!result}
              >
                <Clipboard className='mr-2 size-4' />
                {t('Copy')}
              </Button>
            </div>
            <pre className='bg-muted/60 min-h-44 overflow-auto rounded-md border p-3 font-mono text-xs whitespace-pre-wrap'>
              {result
                ? formatJson(result.response)
                : t('Run a test to see the backend response here.')}
            </pre>
          </div>
        </div>
      </TitledCard>
    </div>
  )
}
