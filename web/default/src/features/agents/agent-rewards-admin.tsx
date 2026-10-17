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
import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  getAdminReferralRewards,
  getAgentTaskPreview,
  createReferralReward,
  markReferralRewardPaid,
  searchAgentTasks,
  type ReferralReward,
} from './api'

function toUnits(value: string): number {
  const amount = Number(value)
  return Number.isFinite(amount) ? Math.round(amount * 10000) : -1
}

export function AgentRewardsAdmin() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [taskID, setTaskID] = useState('')
  const [searchedTaskID, setSearchedTaskID] = useState('')
  const [lookedUpID, setLookedUpID] = useState(0)
  const [orderID, setOrderID] = useState('')
  const [costEvidenceID, setCostEvidenceID] = useState('')
  const [sale, setSale] = useState('')
  const [cost, setCost] = useState('')
  const [extraCost, setExtraCost] = useState('0')
  const [payoutRefs, setPayoutRefs] = useState<Record<number, string>>({})
  const [page, setPage] = useState(1)
  const [nowSeconds, setNowSeconds] = useState(0)
  useEffect(() => {
    const update = () => setNowSeconds(Math.floor(Date.now() / 1000))
    update()
    const timer = window.setInterval(update, 60_000)
    return () => window.clearInterval(timer)
  }, [])
  const preview = useQuery({
    queryKey: ['agent', 'task-preview', lookedUpID],
    queryFn: () => getAgentTaskPreview(lookedUpID),
    enabled: lookedUpID > 0,
  })
  const taskMatches = useQuery({
    queryKey: ['agent', 'task-search', searchedTaskID],
    queryFn: () => searchAgentTasks(searchedTaskID),
    enabled: searchedTaskID.length > 0,
  })
  const rewards = useQuery({
    queryKey: ['agent', 'admin-rewards', page],
    queryFn: () => getAdminReferralRewards('', page),
  })
  const create = useMutation({
    mutationFn: () =>
      createReferralReward({
        task_record_id: lookedUpID,
        source_order_id: orderID.trim(),
        cost_evidence_id: costEvidenceID.trim(),
        sale_units: toUnits(sale),
        cost_units: toUnits(cost),
        extra_cost_units: toUnits(extraCost),
      }),
    onSuccess: () => {
      toast.success(t('Verified reward recorded'))
      setTaskID('')
      setSearchedTaskID('')
      setLookedUpID(0)
      setOrderID('')
      setCostEvidenceID('')
      setSale('')
      setCost('')
      setExtraCost('0')
      void queryClient.invalidateQueries({
        queryKey: ['agent', 'admin-rewards'],
      })
    },
  })
  const markPaid = useMutation({
    mutationFn: (item: ReferralReward) =>
      markReferralRewardPaid(item.id, payoutRefs[item.id] ?? ''),
    onSuccess: () => {
      toast.success(t('Payout recorded'))
      void queryClient.invalidateQueries({
        queryKey: ['agent', 'admin-rewards'],
      })
    },
  })
  const rewardStatusLabels = {
    pending: t('Pending payout'),
    paid: t('Paid'),
    reversed: t('Reversed'),
  }
  const canCreate =
    preview.data?.status === 'SUCCESS' &&
    preview.data?.billing_status === 'SETTLED' &&
    !!preview.data?.inviter_id &&
    !!orderID.trim() &&
    !!costEvidenceID.trim() &&
    !!sale.trim() &&
    !!cost.trim() &&
    !!extraCost.trim() &&
    toUnits(sale) > 0 &&
    toUnits(cost) >= 0 &&
    toUnits(extraCost) >= 0

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Reward settlement')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t(
          'Verify the paid shop order and actual upstream cost before recording a reward. Payouts are made externally and recorded here.'
        )}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <Card>
          <CardHeader>
            <CardTitle>{t('Verify a task')}</CardTitle>
          </CardHeader>
          <CardContent className='flex flex-col gap-5'>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Set the old inviter signup reward to zero in System Settings before recording margin-based rewards.'
              )}
            </p>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor='agent-task-id'>{t('Task ID')}</FieldLabel>
                <div className='flex gap-2'>
                  <Input
                    id='agent-task-id'
                    value={taskID}
                    onChange={(event) => setTaskID(event.target.value)}
                  />
                  <Button
                    variant='outline'
                    disabled={!taskID.trim()}
                    onClick={() => {
                      setSearchedTaskID(taskID.trim())
                      setLookedUpID(0)
                    }}
                  >
                    {t('Look up')}
                  </Button>
                </div>
              </Field>
            </FieldGroup>
            {(taskMatches.data ?? []).map((item) => (
              <div
                key={item.id}
                className='flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3 text-sm'
              >
                <span>
                  #{item.id} · {item.model_name} · {t('Customer ID')}{' '}
                  {item.user_id} · {item.status}
                </span>
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() => setLookedUpID(item.id)}
                >
                  {t('Select')}
                </Button>
              </div>
            ))}
            {searchedTaskID &&
            !taskMatches.isLoading &&
            !taskMatches.data?.length ? (
              <p className='text-muted-foreground text-sm'>
                {t('No matching tasks')}
              </p>
            ) : null}
            {preview.data ? (
              <div className='grid gap-2 rounded-lg border p-4 text-sm sm:grid-cols-2'>
                <p>
                  {t('Task ID')}: {preview.data.task_id}
                </p>
                <p>
                  {t('Model')}: {preview.data.model_name}
                </p>
                <p>
                  {t('Customer ID')}: {preview.data.user_id}
                </p>
                <p>
                  {t('Referrer ID')}: {preview.data.inviter_id || '—'}
                </p>
                <p>
                  {t('Channel ID')}: {preview.data.channel_id}
                </p>
                <p>
                  {t('Status')}: {preview.data.status} /{' '}
                  {preview.data.billing_status}
                </p>
              </div>
            ) : null}
            {preview.data ? (
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor='agent-order-id'>
                    {t('Verified paid shop order ID')}
                  </FieldLabel>
                  <Input
                    id='agent-order-id'
                    maxLength={128}
                    value={orderID}
                    onChange={(event) => setOrderID(event.target.value)}
                  />
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'Enter only this task’s allocated paid revenue and costs, not the entire order amount.'
                    )}
                  </p>
                </Field>
                <Field>
                  <FieldLabel htmlFor='agent-cost-evidence'>
                    {t('Upstream cost evidence ID')}
                  </FieldLabel>
                  <Input
                    id='agent-cost-evidence'
                    maxLength={128}
                    value={costEvidenceID}
                    onChange={(event) => setCostEvidenceID(event.target.value)}
                  />
                </Field>
                <div className='grid gap-3 sm:grid-cols-3'>
                  <Field>
                    <FieldLabel htmlFor='agent-sale'>
                      {t('Sale amount for this task (CNY)')}
                    </FieldLabel>
                    <Input
                      id='agent-sale'
                      inputMode='decimal'
                      value={sale}
                      onChange={(event) => setSale(event.target.value)}
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor='agent-cost'>
                      {t('Upstream cost for this task (CNY)')}
                    </FieldLabel>
                    <Input
                      id='agent-cost'
                      inputMode='decimal'
                      value={cost}
                      onChange={(event) => setCost(event.target.value)}
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor='agent-extra-cost'>
                      {t('Other variable costs (CNY)')}
                    </FieldLabel>
                    <Input
                      id='agent-extra-cost'
                      inputMode='decimal'
                      value={extraCost}
                      onChange={(event) => setExtraCost(event.target.value)}
                    />
                  </Field>
                </div>
                <Button
                  disabled={!canCreate || create.isPending}
                  onClick={() => create.mutate()}
                >
                  {t('Record verified reward')}
                </Button>
              </FieldGroup>
            ) : null}
          </CardContent>
        </Card>
        <Card className='mt-4'>
          <CardHeader>
            <CardTitle>{t('Reward ledger')}</CardTitle>
          </CardHeader>
          <CardContent className='flex flex-col gap-3'>
            {(rewards.data?.items ?? []).map((item) => (
              <div
                key={item.id}
                className='flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3 text-sm'
              >
                <div>
                  <p className='font-medium'>
                    {item.model_name} · {t('Task')} #{item.task_record_id}
                  </p>
                  <p className='text-muted-foreground text-xs'>
                    {t('Customer ID')} {item.customer_id} · {t('Referrer ID')}{' '}
                    {item.referrer_id}
                  </p>
                  <p className='text-muted-foreground text-xs'>
                    {t('Verified paid shop order ID')}: {item.source_order_id} ·{' '}
                    {t('Upstream cost evidence ID')}: {item.cost_evidence_id}
                  </p>
                  <p className='text-muted-foreground text-xs'>
                    {t('Sale')} ¥{(item.sale_units / 10000).toFixed(4)} ·{' '}
                    {t('Cost')} ¥
                    {(
                      (item.cost_units +
                        item.extra_cost_units +
                        item.fee_units) /
                      10000
                    ).toFixed(4)}
                  </p>
                  {item.payout_reference ? (
                    <p className='text-muted-foreground text-xs'>
                      {t('External payout reference')}: {item.payout_reference}
                    </p>
                  ) : null}
                  {item.status === 'reversed' && item.paid_at > 0 ? (
                    <p className='text-destructive text-xs'>
                      {t(
                        'Previously paid reward was reversed; recover the external payout.'
                      )}
                    </p>
                  ) : null}
                </div>
                <div className='flex flex-wrap items-center gap-2'>
                  <span className='font-semibold tabular-nums'>
                    ¥{(item.reward_units / 10000).toFixed(4)}
                  </span>
                  <Badge variant='secondary'>
                    {rewardStatusLabels[item.status]}
                  </Badge>
                  {item.status === 'pending' ? (
                    <>
                      <Input
                        aria-label={t('External payout reference')}
                        className='w-44'
                        maxLength={128}
                        placeholder={t('External payout reference')}
                        value={payoutRefs[item.id] ?? ''}
                        onChange={(event) =>
                          setPayoutRefs((current) => ({
                            ...current,
                            [item.id]: event.target.value,
                          }))
                        }
                      />
                      <Button
                        size='sm'
                        disabled={
                          markPaid.isPending ||
                          !(payoutRefs[item.id] ?? '').trim() ||
                          item.eligible_at > nowSeconds
                        }
                        onClick={() => markPaid.mutate(item)}
                      >
                        {t('Record payout')}
                      </Button>
                    </>
                  ) : null}
                </div>
              </div>
            ))}
            {!rewards.isLoading && !rewards.data?.items.length ? (
              <p className='text-muted-foreground text-sm'>
                {t('No verified rewards yet')}
              </p>
            ) : null}
            <div className='flex items-center gap-2'>
              <Button
                size='sm'
                variant='outline'
                disabled={page === 1}
                onClick={() => setPage((value) => value - 1)}
              >
                {t('Previous')}
              </Button>
              <span className='text-sm'>{page}</span>
              <Button
                size='sm'
                variant='outline'
                disabled={page * 50 >= (rewards.data?.total ?? 0)}
                onClick={() => setPage((value) => value + 1)}
              >
                {t('Next')}
              </Button>
            </div>
          </CardContent>
        </Card>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
