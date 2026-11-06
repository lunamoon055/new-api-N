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
import { useState, useMemo, useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import {
  EMPTY_NOTIFICATION_READ_STATE,
  getNotificationViewerKey,
  NOTIFICATION_STORAGE_KEY,
  useNotificationStore,
} from '@/stores/notification-store'
import { getNotificationFeed } from '@/lib/api'
import {
  getAnnouncementKey,
  getNoticeKey,
  getPublishedNotifications,
  getRecentUnreadNotifications,
  getSnapshotAnnouncements,
  NOTIFICATION_REFRESH_EVENT,
  NOTIFICATION_REFRESH_MS,
  NOTIFICATION_REFRESH_STORAGE_KEY,
  type NotificationSnapshot,
  type NotificationTab,
} from '@/lib/notifications'

interface NotificationSession {
  viewerKey: string
  kind: 'automatic' | 'manual'
  tab: NotificationTab
  snapshot: NotificationSnapshot
}

/** A dialog owns an immutable batch. Feed updates never acknowledge unseen items. */
export function useNotifications(options?: { autoOpenRecent?: boolean }) {
  const { t } = useTranslation()
  const autoOpenRecent = options?.autoOpenRecent ?? false
  const viewerKey = useAuthStore((state) =>
    getNotificationViewerKey(state.auth.user?.id)
  )
  const [session, setSession] = useState<NotificationSession | null>(null)
  const sessionRef = useRef<NotificationSession | null>(null)
  const manualOpeningRef = useRef<string | null>(null)
  const refreshRef = useRef<(() => void) | null>(null)
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['notifications'],
    queryFn: getNotificationFeed,
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
  })
  const read = useNotificationStore(
    (state) => state.readByViewer[viewerKey] ?? EMPTY_NOTIFICATION_READ_STATE
  )
  const published = useMemo(
    () => (data ? getPublishedNotifications(data) : null),
    [data]
  )
  const unreadCounts = useMemo(() => {
    const notice =
      published?.notice.content &&
      !read.readNoticeKeys.includes(getNoticeKey(published.notice))
        ? 1
        : 0
    const readKeys = new Set(read.readAnnouncementKeys)
    const announcements =
      published?.announcements.filter(
        (item) => !readKeys.has(getAnnouncementKey(item))
      ).length ?? 0
    return { notice, announcements, total: notice + announcements }
  }, [published, read])

  useEffect(() => {
    let active = true
    let refreshing = false
    const isVisible = () => document.visibilityState !== 'hidden'
    const refresh = async () => {
      if (!active || refreshing || !isVisible() || !navigator.onLine) return
      refreshing = true
      try {
        const result = await refetch({ cancelRefetch: false })
        if (
          !active ||
          !autoOpenRecent ||
          !result.isSuccess ||
          !result.data ||
          !isVisible() ||
          manualOpeningRef.current === viewerKey ||
          sessionRef.current?.viewerKey === viewerKey
        )
          return
        // Do not interrupt model details or another active dialog. The
        // next poll/focus/entry retries while unread versions remain queued.
        if (
          document.querySelector(
            '[role="dialog"]:not([hidden]), [role="alertdialog"]:not([hidden])'
          )
        )
          return
        const snapshot = getRecentUnreadNotifications(
          result.data,
          useNotificationStore.getState().getReadState(viewerKey)
        )
        if (!snapshot) return
        const next: NotificationSession = {
          viewerKey,
          kind: 'automatic',
          tab: 'announcements',
          snapshot,
        }
        sessionRef.current = next
        setSession(next)
      } finally {
        refreshing = false
      }
    }
    const refreshNow = () => {
      void refresh()
    }
    const onStorage = (event: StorageEvent) => {
      if (event.key === NOTIFICATION_STORAGE_KEY)
        void useNotificationStore.persist.rehydrate()
      if (event.key === NOTIFICATION_REFRESH_STORAGE_KEY) refreshNow()
    }
    refreshRef.current = refreshNow
    refreshNow()
    const timer = autoOpenRecent
      ? window.setInterval(refreshNow, NOTIFICATION_REFRESH_MS)
      : undefined
    window.addEventListener('focus', refreshNow)
    window.addEventListener('online', refreshNow)
    document.addEventListener('visibilitychange', refreshNow)
    window.addEventListener(NOTIFICATION_REFRESH_EVENT, refreshNow)
    window.addEventListener('storage', onStorage)
    return () => {
      active = false
      refreshRef.current = null
      if (timer !== undefined) window.clearInterval(timer)
      window.removeEventListener('focus', refreshNow)
      window.removeEventListener('online', refreshNow)
      document.removeEventListener('visibilitychange', refreshNow)
      window.removeEventListener(NOTIFICATION_REFRESH_EVENT, refreshNow)
      window.removeEventListener('storage', onStorage)
    }
  }, [autoOpenRecent, refetch, viewerKey])

  const current = session?.viewerKey === viewerKey ? session : null
  const acknowledgeTab = (
    snapshot: NotificationSnapshot,
    tab: NotificationTab
  ) => {
    const visible =
      tab === 'notice'
        ? { notice: snapshot.notice, announcements: [] }
        : {
            notice: { content: '', published_at: 0 },
            announcements: snapshot.announcements,
          }
    useNotificationStore.getState().acknowledgeSnapshot(viewerKey, visible)
  }
  const closeDialog = (today = false) => {
    const closing = sessionRef.current
    if (!closing || closing.viewerKey !== viewerKey) return
    const snapshot =
      closing.kind === 'automatic'
        ? closing.snapshot
        : {
            notice:
              closing.tab === 'notice'
                ? closing.snapshot.notice
                : { content: '', published_at: 0 },
            announcements:
              closing.tab === 'announcements'
                ? closing.snapshot.announcements
                : [],
          }
    useNotificationStore
      .getState()
      .acknowledgeSnapshot(viewerKey, snapshot, today, data?.server_time)
    sessionRef.current = null
    setSession(null)
    // Fresh publications received while the old batch was open stay unread.
    refreshRef.current?.()
  }
  const openDialog = async (tab?: NotificationTab) => {
    manualOpeningRef.current = viewerKey
    try {
      const result = await refetch({ cancelRefetch: false })
      if (
        !refreshRef.current ||
        getNotificationViewerKey(useAuthStore.getState().auth.user?.id) !==
          viewerKey
      )
        return
      const feed = result.isSuccess ? result.data : data
      if (!feed) {
        toast.error(t('Unable to load notifications. Please try again.'))
        return
      }
      const snapshot = getPublishedNotifications(feed)
      const noticeUnread =
        snapshot.notice.content &&
        !useNotificationStore
          .getState()
          .getReadState(viewerKey)
          .readNoticeKeys.includes(getNoticeKey(snapshot.notice))
      const target =
        tab ??
        (snapshot.notice.content &&
        (noticeUnread || snapshot.announcements.length === 0)
          ? 'notice'
          : 'announcements')
      const next: NotificationSession = {
        viewerKey,
        kind: 'manual',
        tab: target,
        snapshot,
      }
      sessionRef.current = next
      setSession(next)
      acknowledgeTab(snapshot, target)
    } finally {
      if (manualOpeningRef.current === viewerKey)
        manualOpeningRef.current = null
    }
  }
  const setActiveTab = (tab: NotificationTab) => {
    const previous = sessionRef.current
    if (!previous || previous.viewerKey !== viewerKey) return
    const next = { ...previous, tab }
    sessionRef.current = next
    setSession(next)
    acknowledgeTab(next.snapshot, tab)
  }
  return {
    notice: current?.snapshot.notice.content ?? published?.notice.content ?? '',
    announcements:
      current?.kind === 'automatic'
        ? getSnapshotAnnouncements(current.snapshot)
        : (current?.snapshot.announcements ?? published?.announcements ?? []),
    recentOnly: current?.kind === 'automatic',
    loading: !current && isLoading,
    unreadCount: unreadCounts.total,
    unreadNoticeCount: unreadCounts.notice,
    unreadAnnouncementsCount: unreadCounts.announcements,
    dialogOpen: current !== null,
    setDialogOpen: (open: boolean) => {
      if (!open) closeDialog()
    },
    activeTab: current?.tab ?? 'notice',
    setActiveTab,
    openDialog,
    closeDialog: () => closeDialog(),
    closeToday: () => closeDialog(true),
  }
}
