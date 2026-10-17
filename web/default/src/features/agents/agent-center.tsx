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
import { useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { CopyButton } from '@/components/copy-button'
import { applyForAgent, getAgentCenter, getMyReferralRewards } from './api'

export function AgentCenter() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [note, setNote] = useState('')
  const center = useQuery({
    queryKey: ['agent', 'center'],
    queryFn: getAgentCenter,
  })
  const rewards = useQuery({
    queryKey: ['agent', 'rewards'],
    queryFn: getMyReferralRewards,
    enabled: !!center.data,
  })
  const apply = useMutation({
    mutationFn: () => applyForAgent(note.trim()),
    onSuccess: () => {
      toast.success(t('Agent application submitted'))
      setNote('')
      void queryClient.invalidateQueries({ queryKey: ['agent', 'center'] })
    },
  })
  const application = center.data?.application
  const approved = application?.status === 'approved'
  const agentLink =
    approved && center.data?.agent_code
      ? `${window.location.origin}/sign-up?aff=${encodeURIComponent(center.data.agent_code)}`
      : ''
  const promotionLink = center.data?.promotion_code
    ? `${window.location.origin}/sign-up?aff=${encodeURIComponent(center.data.promotion_code)}`
    : ''

  const pendingUnits = center.data?.reward_pending_units ?? 0
  const paidUnits = center.data?.reward_paid_units ?? 0
  const rewardStatusLabels = {
    pending: t('Pending payout'),
    paid: t('Paid'),
    reversed: t('Reversed'),
  }
  let agentContent: ReactNode
  if (center.isLoading) {
    agentContent = (
      <p className='text-muted-foreground text-sm'>{t('Loading...')}</p>
    )
  } else if (center.isError) {
    agentContent = (
      <div className='flex flex-col items-start gap-3'>
        <p className='text-destructive text-sm'>
          {t(
            'Agent service is unavailable. Please refresh after the backend is updated.'
          )}
        </p>
        <Button variant='outline' onClick={() => void center.refetch()}>
          {t('Retry')}
        </Button>
      </div>
    )
  } else if (approved) {
    agentContent = (
      <>
        <Badge>{t('Agent approved')}</Badge>
        <div className='flex flex-col gap-2'>
          <label className='text-sm font-medium' htmlFor='agent-link'>
            {t('Your agent link')}
          </label>
          <div className='flex gap-2'>
            <Input id='agent-link' readOnly value={agentLink} />
            <CopyButton value={agentLink} tooltip={t('Copy agent link')} />
          </div>
        </div>
        <div className='rounded-lg border p-4'>
          <p className='text-muted-foreground text-sm'>
            {t('Direct customers')}
          </p>
          <p className='mt-1 text-2xl font-semibold tabular-nums'>
            {center.data?.customer_count ?? 0}
          </p>
        </div>
      </>
    )
  } else if (application?.status === 'pending') {
    agentContent = (
      <p className='text-sm'>
        {t('Your application is awaiting administrator review.')}
      </p>
    )
  } else {
    agentContent = (
      <div className='flex flex-col gap-4'>
        {application?.status === 'rejected' ||
        application?.status === 'revoked' ? (
          <Badge variant='secondary'>
            {application.status === 'rejected' ? t('Rejected') : t('Revoked')}
          </Badge>
        ) : null}
        {application?.review_note ? (
          <p className='text-muted-foreground text-sm'>
            {t('Review note')}: {application.review_note}
          </p>
        ) : null}
        <p className='text-muted-foreground text-sm'>
          {t(
            'Apply to become an agent. Your agent link becomes available after approval.'
          )}
        </p>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='agent-application-note'>
              {t('Application details')}
            </FieldLabel>
            <Textarea
              id='agent-application-note'
              value={note}
              maxLength={500}
              onChange={(event) => setNote(event.target.value)}
              placeholder={t('Describe how you plan to promote the service')}
            />
          </Field>
        </FieldGroup>
        <Button
          disabled={apply.isPending || !note.trim()}
          onClick={() => apply.mutate()}
        >
          {t('Submit application')}
        </Button>
      </div>
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Promotion and Agents')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t(
          'Rewards are based on successful usage by directly referred customers. Customer prices remain unchanged.'
        )}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <div className='grid gap-4 lg:grid-cols-[minmax(0,1.4fr)_minmax(300px,1fr)]'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Agent Center')}</CardTitle>
            </CardHeader>
            <CardContent className='flex flex-col gap-5'>
              {agentContent}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>{t('Reward rules')}</CardTitle>
            </CardHeader>
            <CardContent className='text-muted-foreground flex flex-col gap-3 text-sm'>
              <p>
                {t(
                  'Regular promotion: 20% of verified margin, capped at 5% of the sale, for 180 days.'
                )}
              </p>
              <p>
                {t(
                  'Approved agents: 30% of verified margin, capped at 8% of the sale, for 12 months.'
                )}
              </p>
              <p>
                {t(
                  'The platform retains at least 15% of the sale. Failed or refunded usage earns no reward.'
                )}
              </p>
              <p>
                {t(
                  'Rewards become payable after a 14-day review period and cost verification.'
                )}
              </p>
            </CardContent>
          </Card>
        </div>
        {!approved && promotionLink ? (
          <Card className='mt-4'>
            <CardHeader>
              <CardTitle>{t('Your Referral Link')}</CardTitle>
            </CardHeader>
            <CardContent className='flex gap-2'>
              <Input
                readOnly
                value={promotionLink}
                aria-label={t('Your Referral Link')}
              />
              <CopyButton
                value={promotionLink}
                tooltip={t('Copy referral link')}
              />
            </CardContent>
          </Card>
        ) : null}
        <Card className='mt-4'>
          <CardHeader>
            <CardTitle>{t('Verified rewards')}</CardTitle>
          </CardHeader>
          <CardContent className='flex flex-col gap-4'>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Only direct customers with verified paid orders and costs appear here. Payouts are recorded by an administrator.'
              )}
            </p>
            <div className='grid gap-3 sm:grid-cols-2'>
              <div className='rounded-lg border p-3'>
                <p className='text-muted-foreground text-sm'>
                  {t('Pending payout')}
                </p>
                <p className='font-semibold tabular-nums'>
                  ¥{(pendingUnits / 10000).toFixed(4)}
                </p>
              </div>
              <div className='rounded-lg border p-3'>
                <p className='text-muted-foreground text-sm'>
                  {t('Paid rewards')}
                </p>
                <p className='font-semibold tabular-nums'>
                  ¥{(paidUnits / 10000).toFixed(4)}
                </p>
              </div>
            </div>
            {(rewards.data?.items ?? []).map((item) => (
              <div
                key={item.id}
                className='flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3 text-sm'
              >
                <div>
                  <p className='font-medium'>{item.model_name}</p>
                  <p className='text-muted-foreground text-xs'>
                    {new Date(item.created_at * 1000).toLocaleString()}
                  </p>
                </div>
                <div className='flex items-center gap-2'>
                  <Badge variant='secondary'>
                    {rewardStatusLabels[item.status]}
                  </Badge>
                  <span className='font-semibold tabular-nums'>
                    ¥{(item.reward_units / 10000).toFixed(4)}
                  </span>
                </div>
              </div>
            ))}
            {!rewards.isLoading && !rewards.data?.items.length ? (
              <p className='text-muted-foreground text-sm'>
                {t('No verified rewards yet')}
              </p>
            ) : null}
          </CardContent>
        </Card>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
