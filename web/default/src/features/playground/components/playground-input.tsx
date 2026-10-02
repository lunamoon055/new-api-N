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
import { useState } from 'react'
import {
  PaperclipIcon,
  FileIcon,
  ImageIcon,
  ScreenShareIcon,
  CameraIcon,
  GlobeIcon,
  SendIcon,
  SquareIcon,
  BarChartIcon,
  BoxIcon,
  NotepadTextIcon,
  CodeSquareIcon,
  GraduationCapIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  PromptInput,
  PromptInputButton,
  PromptInputFooter,
  PromptInputTextarea,
  PromptInputTools,
  type PromptInputMessage,
} from '@/components/ai-elements/prompt-input'
import { Suggestion, Suggestions } from '@/components/ai-elements/suggestion'
import { ModelGroupSelector } from '@/components/model-group-selector'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { resolvePlaygroundMode } from '../lib/media-routing'
import type { ModelOption, GroupOption, PlaygroundConfig } from '../types'

interface PlaygroundInputProps {
  onSubmit: (text: string) => void
  onStop?: () => void
  disabled?: boolean
  isGenerating?: boolean
  models: ModelOption[]
  modelValue: string
  mode: PlaygroundConfig['mode']
  imageEndpoint: PlaygroundConfig['imageEndpoint']
  videoEndpoint: PlaygroundConfig['videoEndpoint']
  chatEndpoint: PlaygroundConfig['chatEndpoint']
  extraBody: string
  onModeChange: (value: PlaygroundConfig['mode']) => void
  onImageEndpointChange: (value: PlaygroundConfig['imageEndpoint']) => void
  onVideoEndpointChange: (value: PlaygroundConfig['videoEndpoint']) => void
  onChatEndpointChange: (value: PlaygroundConfig['chatEndpoint']) => void
  onExtraBodyChange: (value: string) => void
  onModelChange: (value: string) => void
  isModelLoading?: boolean
  groups: GroupOption[]
  groupValue: string
  onGroupChange: (value: string) => void
}

const suggestions = [
  { icon: BarChartIcon, text: 'Analyze data', color: '#76d0eb' },
  { icon: BoxIcon, text: 'Surprise me', color: '#76d0eb' },
  { icon: NotepadTextIcon, text: 'Summarize text', color: '#ea8444' },
  { icon: CodeSquareIcon, text: 'Code', color: '#6c71ff' },
  { icon: GraduationCapIcon, text: 'Get advice', color: '#76d0eb' },
  { icon: null, text: 'More' },
]

export function PlaygroundInput({
  onSubmit,
  onStop,
  disabled,
  isGenerating,
  models,
  modelValue,
  mode,
  imageEndpoint,
  videoEndpoint,
  chatEndpoint,
  extraBody,
  onModeChange,
  onImageEndpointChange,
  onVideoEndpointChange,
  onChatEndpointChange,
  onExtraBodyChange,
  onModelChange,
  isModelLoading = false,
  groups,
  groupValue,
  onGroupChange,
}: PlaygroundInputProps) {
  const { t } = useTranslation()
  const [text, setText] = useState('')

  const isModelSelectDisabled =
    disabled || isModelLoading || models.length === 0
  const isGroupSelectDisabled = disabled || groups.length === 0
  const resolvedMode = resolvePlaygroundMode(modelValue, mode)

  const handleSubmit = (message: PromptInputMessage) => {
    if (!message.text?.trim() || disabled) return
    onSubmit(message.text)
    setText('')
  }

  const handleFileAction = (action: string) => {
    toast.info(t('Feature in development'), {
      description: action,
    })
  }

  const handleSuggestionClick = (suggestion: string) => {
    onSubmit(suggestion)
  }

  return (
    <div className='grid shrink-0 gap-4 px-1 md:pb-4'>
      <div className='flex flex-wrap items-end gap-3 px-1'>
        <div className='grid gap-1.5'>
          <Label htmlFor='playground-mode'>{t('Test type')}</Label>
          <Select
            value={mode}
            onValueChange={(value) => {
              if (value) onModeChange(value as PlaygroundConfig['mode'])
            }}
            disabled={disabled}
          >
            <SelectTrigger id='playground-mode' className='min-w-36'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value='auto'>{t('Auto detect')}</SelectItem>
                <SelectItem value='chat'>{t('Chat')}</SelectItem>
                <SelectItem value='image'>{t('Image')}</SelectItem>
                <SelectItem value='video'>{t('Video')}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        {resolvedMode === 'image' && (
          <div className='grid gap-1.5'>
            <Label htmlFor='playground-image-endpoint'>
              {t('Image endpoint')}
            </Label>
            <Select
              value={imageEndpoint}
              onValueChange={(value) => {
                if (value)
                  onImageEndpointChange(
                    value as PlaygroundConfig['imageEndpoint']
                  )
              }}
              disabled={disabled}
            >
              <SelectTrigger
                id='playground-image-endpoint'
                className='min-w-40'
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value='auto'>{t('Auto detect')}</SelectItem>
                  <SelectItem value='sync'>{t('Image Generation')}</SelectItem>
                  <SelectItem value='async'>
                    {t('Async Image Generation')}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        )}
        {resolvedMode === 'video' && (
          <div className='grid gap-1.5'>
            <Label htmlFor='playground-video-endpoint'>
              {t('Video endpoint')}
            </Label>
            <Select
              value={videoEndpoint}
              onValueChange={(value) => {
                if (value)
                  onVideoEndpointChange(
                    value as PlaygroundConfig['videoEndpoint']
                  )
              }}
              disabled={disabled}
            >
              <SelectTrigger
                id='playground-video-endpoint'
                className='min-w-40'
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value='auto'>{t('Auto detect')}</SelectItem>
                  <SelectItem value='standard'>
                    {t('Video Generation')}
                  </SelectItem>
                  <SelectItem value='async'>
                    {t('Async Video Generation')}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        )}
        {resolvedMode === 'chat' && (
          <div className='grid gap-1.5'>
            <Label htmlFor='playground-chat-endpoint'>
              {t('Chat endpoint')}
            </Label>
            <Select
              value={chatEndpoint}
              onValueChange={(value) => {
                if (value)
                  onChatEndpointChange(
                    value as PlaygroundConfig['chatEndpoint']
                  )
              }}
              disabled={disabled}
            >
              <SelectTrigger id='playground-chat-endpoint' className='min-w-40'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value='completions'>
                    {t('Chat Completions')}
                  </SelectItem>
                  <SelectItem value='responses'>{t('Responses')}</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        )}
        <p className='text-muted-foreground pb-1 text-xs'>
          {t('Select a test type for newly added models.')}
        </p>
      </div>
      <details className='px-1 text-sm'>
        <summary className='cursor-pointer'>
          {t('Advanced request parameters')}
        </summary>
        <div className='mt-2 grid gap-1.5'>
          <Label htmlFor='playground-extra-body'>
            {t('Extra parameters (JSON object)')}
          </Label>
          <Textarea
            id='playground-extra-body'
            value={extraBody}
            onChange={(event) => onExtraBodyChange(event.target.value)}
            disabled={disabled}
            placeholder='{"size":"1024x1024"}'
            className='min-h-20 font-mono text-xs'
            aria-describedby='playground-extra-body-help'
          />
          <p
            id='playground-extra-body-help'
            className='text-muted-foreground text-xs'
          >
            {t(
              'Extra parameters are added to the request. Model, prompt, group, and chat messages stay controlled above.'
            )}
          </p>
        </div>
      </details>
      <PromptInput groupClassName='rounded-xl' onSubmit={handleSubmit}>
        <PromptInputTextarea
          autoComplete='off'
          autoCorrect='off'
          autoCapitalize='off'
          spellCheck={false}
          className='px-5 md:text-base'
          disabled={disabled}
          onChange={(event) => setText(event.target.value)}
          placeholder={
            resolvedMode === 'chat'
              ? t('Ask anything')
              : t('Enter a generation prompt')
          }
          value={text}
        />

        <PromptInputFooter className='p-2.5'>
          <PromptInputTools>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <PromptInputButton
                    className='border font-medium'
                    disabled={disabled}
                    variant='outline'
                  />
                }
              >
                <PaperclipIcon size={16} />
                <span className='hidden sm:inline'>{t('Attach')}</span>
                <span className='sr-only sm:hidden'>{t('Attach')}</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align='start'>
                <DropdownMenuItem
                  onClick={() => handleFileAction('upload-file')}
                >
                  <FileIcon className='mr-2' size={16} />
                  {t('Upload file')}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => handleFileAction('upload-photo')}
                >
                  <ImageIcon className='mr-2' size={16} />
                  {t('Upload photo')}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => handleFileAction('take-screenshot')}
                >
                  <ScreenShareIcon className='mr-2' size={16} />
                  {t('Take screenshot')}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => handleFileAction('take-photo')}
                >
                  <CameraIcon className='mr-2' size={16} />
                  {t('Take photo')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <PromptInputButton
              className='border font-medium'
              disabled={disabled}
              onClick={() => toast.info(t('Search feature in development'))}
              variant='outline'
            >
              <GlobeIcon size={16} />
              <span className='hidden sm:inline'>{t('Search')}</span>
              <span className='sr-only sm:hidden'>{t('Search')}</span>
            </PromptInputButton>
          </PromptInputTools>

          <div className='flex items-center gap-1.5 md:gap-2'>
            <ModelGroupSelector
              selectedModel={modelValue}
              models={models}
              onModelChange={onModelChange}
              selectedGroup={groupValue}
              groups={groups}
              onGroupChange={onGroupChange}
              disabled={isModelSelectDisabled || isGroupSelectDisabled}
            />

            {isGenerating && onStop ? (
              <PromptInputButton
                className='text-foreground font-medium'
                onClick={onStop}
                variant='secondary'
              >
                <SquareIcon className='fill-current' size={16} />
                <span className='hidden sm:inline'>{t('Stop')}</span>
                <span className='sr-only sm:hidden'>{t('Stop')}</span>
              </PromptInputButton>
            ) : (
              <PromptInputButton
                className='text-foreground font-medium'
                disabled={disabled || !text.trim()}
                type='submit'
                variant='secondary'
              >
                <SendIcon size={16} />
                <span className='hidden sm:inline'>{t('Send')}</span>
                <span className='sr-only sm:hidden'>{t('Send')}</span>
              </PromptInputButton>
            )}
          </div>
        </PromptInputFooter>
      </PromptInput>

      <Suggestions>
        {suggestions.map(({ icon: Icon, text, color }) => (
          <Suggestion
            className={`text-xs font-normal sm:text-sm ${
              text === 'More' ? 'hidden sm:flex' : ''
            }`}
            key={text}
            onClick={() => handleSuggestionClick(text)}
            suggestion={text}
          >
            {Icon && <Icon size={16} style={{ color }} />}
            {text}
          </Suggestion>
        ))}
      </Suggestions>
    </div>
  )
}
