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
import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  CheckCircle2,
  Coins,
  ListChecks,
  XCircle,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { formatNumber, formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { computeTimeRange } from '@/lib/time'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'
import { getFailedTaskCount } from '@/features/dashboard/api'
import { getDefaultDays } from '@/features/dashboard/lib/filters'
import { calculateDashboardStats } from '@/features/dashboard/lib/stats'
import type {
  DashboardFilters,
  QuotaDataItem,
} from '@/features/dashboard/types'

type MetricTone = 'success' | 'destructive' | 'warning'

interface RequestResultOverviewProps {
  data: QuotaDataItem[]
  filters?: DashboardFilters
  loading?: boolean
  error?: boolean
}

interface ResultMetricProps {
  icon: LucideIcon
  label: string
  value: string
  tone: MetricTone
  loading?: boolean
  error?: boolean
  className?: string
}

const METRIC_TONE_CLASSES: Record<MetricTone, { icon: string; value: string }> =
  {
    success: {
      icon: 'border-success/30 bg-success/10 text-success',
      value: 'text-success',
    },
    destructive: {
      icon: 'border-destructive/30 bg-destructive/10 text-destructive',
      value: 'text-destructive',
    },
    warning: {
      icon: 'border-warning/30 bg-warning/10 text-warning',
      value: 'text-warning',
    },
  }

function ResultMetric(props: ResultMetricProps) {
  const Icon = props.icon
  const toneClasses = METRIC_TONE_CLASSES[props.tone]

  return (
    <div
      className={cn(
        'flex min-h-20 items-center gap-3 px-3 py-3 sm:px-5',
        props.className
      )}
    >
      <div
        className={cn(
          'flex size-8 shrink-0 items-center justify-center rounded-full border',
          toneClasses.icon
        )}
      >
        <Icon className='size-4' aria-hidden='true' />
      </div>
      <div className='min-w-0'>
        <div className='text-muted-foreground truncate text-xs font-medium'>
          {props.label}
        </div>
        {props.loading ? (
          <Skeleton className='mt-1.5 h-6 w-16' />
        ) : (
          <div
            className={cn(
              'mt-0.5 font-mono text-lg font-bold tracking-tight tabular-nums sm:text-xl',
              props.error ? 'text-muted-foreground' : toneClasses.value
            )}
            aria-live='polite'
          >
            {props.error ? '--' : props.value}
          </div>
        )}
      </div>
    </div>
  )
}

export function RequestResultOverview(props: RequestResultOverviewProps) {
  const { t } = useTranslation()
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isAdmin = Boolean(userRole && userRole >= ROLE.ADMIN)
  const startTime = props.filters?.start_timestamp?.getTime()
  const endTime = props.filters?.end_timestamp?.getTime()
  const granularity = props.filters?.time_granularity
  const username = isAdmin ? props.filters?.username?.trim() : undefined

  const requestParams = useMemo(() => {
    const timeRange = computeTimeRange(
      getDefaultDays(granularity),
      startTime === undefined ? undefined : new Date(startTime),
      endTime === undefined ? undefined : new Date(endTime)
    )
    return {
      ...timeRange,
      ...(username ? { username } : {}),
    }
  }, [endTime, granularity, startTime, username])

  const failureCountQuery = useQuery({
    queryKey: [
      'dashboard',
      'model-result-overview',
      'task-failure-count',
      isAdmin ? 'admin' : 'self',
      requestParams,
    ],
    queryFn: () => getFailedTaskCount(requestParams, isAdmin),
    staleTime: 60 * 1000,
    retry: false,
  })

  const successStats = useMemo(
    () => calculateDashboardStats(props.data),
    [props.data]
  )

  return (
    <section
      className='overflow-hidden rounded-lg border'
      aria-label={t('Call Results')}
    >
      <div className='grid grid-cols-2 lg:grid-cols-4'>
        <div className='flex min-h-20 items-center gap-3 px-3 py-3 sm:px-5'>
          <div className='text-muted-foreground flex size-8 shrink-0 items-center justify-center rounded-full border'>
            <ListChecks className='size-4' aria-hidden='true' />
          </div>
          <div className='min-w-0'>
            <div className='truncate text-sm font-semibold'>
              {t('Call Results')}
            </div>
            <div className='text-muted-foreground mt-1 truncate text-xs'>
              {t('Current Filter Range')}
            </div>
          </div>
        </div>

        <ResultMetric
          icon={CheckCircle2}
          label={t('Successful Count')}
          value={formatNumber(successStats.totalCount)}
          tone='success'
          loading={props.loading}
          error={props.error}
          className='border-l'
        />
        <ResultMetric
          icon={XCircle}
          label={t('Failed Count')}
          value={formatNumber(failureCountQuery.data ?? 0)}
          tone='destructive'
          loading={failureCountQuery.isPending}
          error={failureCountQuery.isError}
          className='border-t lg:border-t-0 lg:border-l'
        />
        <ResultMetric
          icon={Coins}
          label={t('Successful Consumption')}
          value={formatQuota(successStats.totalQuota)}
          tone='warning'
          loading={props.loading}
          error={props.error}
          className='border-t border-l lg:border-t-0'
        />
      </div>
    </section>
  )
}
