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
import { useTranslation } from 'react-i18next'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SectionPageLayout } from '@/components/layout'
import { ChannelTestLab } from './components/channel-test-lab'
import { ChannelsDialogs } from './components/channels-dialogs'
import { ChannelsPrimaryButtons } from './components/channels-primary-buttons'
import { ChannelsProvider } from './components/channels-provider'
import { ChannelsTable } from './components/channels-table'

type ChannelsView = 'list' | 'test-lab'

export function Channels() {
  const { t } = useTranslation()
  const [view, setView] = useState<ChannelsView>('list')

  return (
    <ChannelsProvider>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Channels')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('Manage API channels and provider configurations')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <ChannelsPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            <Tabs
              value={view}
              onValueChange={(value) => setView(value as ChannelsView)}
            >
              <TabsList>
                <TabsTrigger value='list'>{t('Channel List')}</TabsTrigger>
                <TabsTrigger value='test-lab'>{t('API Test Lab')}</TabsTrigger>
              </TabsList>
            </Tabs>
            {view === 'list' ? <ChannelsTable /> : <ChannelTestLab />}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <ChannelsDialogs />
    </ChannelsProvider>
  )
}
