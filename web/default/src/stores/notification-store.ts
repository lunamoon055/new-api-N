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
import { create } from 'zustand'
import {
  createJSONStorage,
  persist,
  type StateStorage,
} from 'zustand/middleware'
import {
  getNoticeKey,
  getSnapshotKeys,
  type NotificationReadState,
  type NotificationSnapshot,
} from '@/lib/notifications'

export const NOTIFICATION_STORAGE_KEY = 'notification-storage'
export const EMPTY_NOTIFICATION_READ_STATE: NotificationReadState = {
  readNoticeKeys: [],
  readAnnouncementKeys: [],
  closedUntilDate: null,
  closedNotificationKeys: [],
}

interface NotificationState {
  readByViewer: Record<string, NotificationReadState>
  getReadState: (viewerKey: string) => NotificationReadState
  markNoticeRead: (
    viewerKey: string,
    content: string,
    publishedAt?: number
  ) => void
  markAnnouncementsRead: (viewerKey: string, keys: string[]) => void
  acknowledgeSnapshot: (
    viewerKey: string,
    snapshot: NotificationSnapshot,
    closeToday?: boolean,
    now?: number
  ) => void
}

export function getNotificationViewerKey(userId?: number | null): string {
  return userId ? `user:${userId}` : 'guest'
}
const union = (a: string[] = [], b: string[] = []) =>
  [...new Set([...a, ...b])].sort()
const strings = (value: unknown): string[] =>
  Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []

function normalizeViewers(
  value: unknown
): Record<string, NotificationReadState> {
  if (!value || typeof value !== 'object') return {}
  const result: Record<string, NotificationReadState> = {}
  for (const [key, item] of Object.entries(value)) {
    if (!/^(guest|user:[0-9]+)$/.test(key) || !item || typeof item !== 'object')
      continue
    const state = item as Record<string, unknown>
    const readNoticeKeys = strings(state.readNoticeKeys)
    if (typeof state.lastReadNotice === 'string' && state.lastReadNotice) {
      readNoticeKeys.push(
        getNoticeKey({ content: state.lastReadNotice, published_at: 0 })
      )
    }
    result[key] = {
      readNoticeKeys: union(readNoticeKeys),
      readAnnouncementKeys: strings(state.readAnnouncementKeys),
      closedUntilDate:
        typeof state.closedUntilDate === 'string'
          ? state.closedUntilDate
          : null,
      closedNotificationKeys: strings(state.closedNotificationKeys),
    }
  }
  return result
}

export function mergeNotificationViewers(
  left: unknown,
  right: unknown
): Record<string, NotificationReadState> {
  const result = normalizeViewers(left)
  for (const [key, incoming] of Object.entries(normalizeViewers(right))) {
    const current = result[key] ?? EMPTY_NOTIFICATION_READ_STATE
    let closed = incoming
    if (
      Date.parse(current.closedUntilDate || '') >
        Date.parse(incoming.closedUntilDate || '') ||
      !incoming.closedUntilDate
    )
      closed = current
    result[key] = {
      readNoticeKeys: union(current.readNoticeKeys, incoming.readNoticeKeys),
      readAnnouncementKeys: union(
        current.readAnnouncementKeys,
        incoming.readAnnouncementKeys
      ),
      closedUntilDate: closed.closedUntilDate,
      closedNotificationKeys:
        current.closedUntilDate === incoming.closedUntilDate
          ? union(
              current.closedNotificationKeys,
              incoming.closedNotificationKeys
            )
          : closed.closedNotificationKeys,
    }
  }
  return result
}

export function createNotificationStore(storage?: StateStorage) {
  const safeStorage: StateStorage = {
    getItem: (key) => {
      try {
        return (storage ?? localStorage).getItem(key)
      } catch {
        return null
      }
    },
    setItem: (key, value) => {
      try {
        return (storage ?? localStorage).setItem(key, value)
      } catch {
        /* keep in-memory state */
      }
    },
    removeItem: (key) => {
      try {
        return (storage ?? localStorage).removeItem(key)
      } catch {
        /* keep in-memory state */
      }
    },
  }
  const readStoredViewers = () => {
    try {
      const raw = safeStorage.getItem(NOTIFICATION_STORAGE_KEY)
      return typeof raw === 'string'
        ? JSON.parse(raw)?.state?.readByViewer
        : undefined
    } catch {
      return undefined
    }
  }
  return create<NotificationState>()(
    persist(
      (set, get) => {
        const update = (
          viewer: string,
          mutate: (state: NotificationReadState) => NotificationReadState
        ) => {
          set((state) => {
            // Read/merge before writing so two tabs cannot overwrite each
            // other's acknowledgements with stale in-memory copies.
            const readByViewer = mergeNotificationViewers(
              state.readByViewer,
              readStoredViewers()
            )
            readByViewer[viewer] = mutate(
              readByViewer[viewer] ?? EMPTY_NOTIFICATION_READ_STATE
            )
            return { readByViewer }
          })
        }
        return {
          readByViewer: {},
          getReadState: (viewer) =>
            mergeNotificationViewers(get().readByViewer, readStoredViewers())[
              viewer
            ] ?? EMPTY_NOTIFICATION_READ_STATE,
          markNoticeRead: (viewer, content, publishedAt = 0) =>
            update(viewer, (state) => ({
              ...state,
              readNoticeKeys: union(state.readNoticeKeys, [
                getNoticeKey({ content, published_at: publishedAt }),
              ]),
            })),
          markAnnouncementsRead: (viewer, keys) =>
            update(viewer, (state) => ({
              ...state,
              readAnnouncementKeys: union(state.readAnnouncementKeys, keys),
            })),
          acknowledgeSnapshot: (
            viewer,
            snapshot,
            closeToday = false,
            now = Date.now()
          ) => {
            const keys = getSnapshotKeys(snapshot)
            update(viewer, (state) => ({
              ...state,
              readNoticeKeys: union(state.readNoticeKeys, keys.noticeKeys),
              readAnnouncementKeys: union(
                state.readAnnouncementKeys,
                keys.announcementKeys
              ),
              closedUntilDate: closeToday
                ? new Date(now).toDateString()
                : state.closedUntilDate,
              closedNotificationKeys: closeToday
                ? union(
                    state.closedUntilDate === new Date(now).toDateString()
                      ? state.closedNotificationKeys
                      : [],
                    [...keys.noticeKeys, ...keys.announcementKeys]
                  )
                : state.closedNotificationKeys,
            }))
          },
        }
      },
      {
        name: NOTIFICATION_STORAGE_KEY,
        version: 2,
        migrate: (value, version) => ({
          readByViewer:
            version >= 1
              ? normalizeViewers(
                  (value as { readByViewer?: unknown })?.readByViewer
                )
              : {},
        }),
        storage: createJSONStorage(() => safeStorage),
        merge: (persisted, current) => ({
          ...current,
          readByViewer: mergeNotificationViewers(
            current.readByViewer,
            (persisted as { readByViewer?: unknown })?.readByViewer
          ),
        }),
        partialize: (state) => ({ readByViewer: state.readByViewer }),
      }
    )
  )
}

export const useNotificationStore = createNotificationStore()
