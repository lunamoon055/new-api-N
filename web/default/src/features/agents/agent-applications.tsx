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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  getAgentApplications,
  reviewAgentApplication,
  type AgentApplication,
} from './api'

export function AgentApplications() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [status, setStatus] = useState('pending')
  const [page, setPage] = useState(1)
  const [notes, setNotes] = useState<Record<number, string>>({})
  const applications = useQuery({
    queryKey: ['agent', 'applications', status, page],
    queryFn: () => getAgentApplications(status, page),
  })
  const review = useMutation({
    mutationFn: (request: {
      item: AgentApplication
      decision: 'approved' | 'rejected' | 'revoked'
    }) =>
      reviewAgentApplication(
        request.item.id,
        request.decision,
        notes[request.item.id] ?? ''
      ),
    onSuccess: () => {
      toast.success(t('Application reviewed'))
      void queryClient.invalidateQueries({
        queryKey: ['agent', 'applications'],
      })
    },
  })
  const statusLabels = {
    pending: t('Pending review'),
    approved: t('Approved'),
    rejected: t('Rejected'),
    revoked: t('Revoked'),
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Agent applications')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t(
          'Only approved applicants can use agent status and its reward rate.'
        )}
      </SectionPageLayout.Description>
      <SectionPageLayout.Content>
        <p className='text-muted-foreground mb-4 text-sm'>
          {t(
            'Set the old inviter signup reward to zero in System Settings before approving agents.'
          )}
        </p>
        <ToggleGroup
          variant='outline'
          className='mb-4 flex-wrap'
          value={[status]}
          onValueChange={(values) => {
            if (values[0]) {
              setStatus(values[0])
              setPage(1)
            }
          }}
        >
          {(['pending', 'approved', 'rejected', 'revoked'] as const).map(
            (value) => (
              <ToggleGroupItem key={value} value={value}>
                {statusLabels[value]}
              </ToggleGroupItem>
            )
          )}
        </ToggleGroup>
        {applications.isLoading ? <p>{t('Loading...')}</p> : null}
        <div className='flex flex-col gap-3'>
          {(applications.data?.items ?? []).map((item) => (
            <Card key={item.id}>
              <CardHeader className='flex flex-row flex-wrap items-center justify-between gap-2'>
                <CardTitle className='text-base'>
                  {item.username || `#${item.user_id}`}
                </CardTitle>
                <span className='text-muted-foreground text-xs'>
                  {new Date(item.applied_at * 1000).toLocaleString()}
                </span>
              </CardHeader>
              <CardContent className='flex flex-col gap-3'>
                <p className='text-sm whitespace-pre-wrap'>{item.note}</p>
                {item.review_note ? (
                  <p className='text-muted-foreground text-sm'>
                    {t('Review note')}: {item.review_note}
                  </p>
                ) : null}
                {item.status === 'pending' || item.status === 'approved' ? (
                  <div className='flex flex-wrap gap-2'>
                    <Input
                      aria-label={t('Review note')}
                      className='min-w-52 flex-1'
                      maxLength={500}
                      placeholder={t('Review note')}
                      value={notes[item.id] ?? ''}
                      onChange={(event) =>
                        setNotes((current) => ({
                          ...current,
                          [item.id]: event.target.value,
                        }))
                      }
                    />
                    {item.status === 'pending' ? (
                      <>
                        <Button
                          size='sm'
                          disabled={review.isPending}
                          onClick={() =>
                            review.mutate({ item, decision: 'approved' })
                          }
                        >
                          {t('Approve')}
                        </Button>
                        <Button
                          size='sm'
                          variant='outline'
                          disabled={
                            review.isPending || !(notes[item.id] ?? '').trim()
                          }
                          onClick={() =>
                            review.mutate({ item, decision: 'rejected' })
                          }
                        >
                          {t('Reject')}
                        </Button>
                      </>
                    ) : (
                      <Button
                        size='sm'
                        variant='outline'
                        disabled={
                          review.isPending || !(notes[item.id] ?? '').trim()
                        }
                        onClick={() =>
                          review.mutate({ item, decision: 'revoked' })
                        }
                      >
                        {t('Revoke')}
                      </Button>
                    )}
                  </div>
                ) : null}
              </CardContent>
            </Card>
          ))}
          {!applications.isLoading && !applications.data?.items.length ? (
            <p className='text-muted-foreground text-sm'>
              {t('No applications')}
            </p>
          ) : null}
        </div>
        <div className='mt-4 flex items-center gap-2'>
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
            disabled={page * 50 >= (applications.data?.total ?? 0)}
            onClick={() => setPage((value) => value + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
