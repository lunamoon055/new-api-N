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
export type NotificationTab = 'notice' | 'announcements'
export type Announcement = Record<string, unknown>

export interface NotificationFeed {
  server_time: number // milliseconds; eligibility uses the server clock
  notice: { content: string; published_at: number } // publication in seconds
  announcements: Announcement[]
}

export interface NotificationSnapshot {
  notice: NotificationFeed['notice']
  announcements: Announcement[]
}

export interface NotificationReadState {
  readNoticeKeys: string[]
  readAnnouncementKeys: string[]
  closedUntilDate: string | null
  closedNotificationKeys: string[]
}

export const NOTIFICATION_REFRESH_MS = 15_000
export const NOTIFICATION_REFRESH_EVENT = 'notification-feed-refresh'
export const NOTIFICATION_REFRESH_STORAGE_KEY = 'notification-feed-version'
const DAY_MS = 24 * 60 * 60 * 1000

function hashString(input: string): string {
  let hash = 0
  for (let i = 0; i < input.length; i += 1) {
    hash = (hash << 5) - hash + input.charCodeAt(i)
    hash |= 0
  }
  return hash.toString(36)
}

export function getAnnouncementKey(item: Announcement): string {
  const text = (key: string) =>
    typeof item[key] === 'string' ? item[key].trim() : ''
  const revision = hashString(
    JSON.stringify({
      publishDate: text('publishDate'),
      content: text('content'),
      extra: text('extra'),
      type: text('type'),
      title: text('title'),
      link: text('link'),
    })
  )
  return item.id !== undefined && item.id !== null
    ? `id:${item.id}:${revision}`
    : `hash:${revision}`
}

export function getNoticeKey(notice: NotificationFeed['notice']): string {
  return `notice:${notice.published_at || 0}:${hashString(notice.content.trim())}`
}

export function getStatusAnnouncements(
  status: Record<string, unknown> | null | undefined
): Announcement[] {
  if (!status?.announcements_enabled || !Array.isArray(status.announcements))
    return []
  return status.announcements
    .filter(
      (item): item is Announcement =>
        item !== null &&
        typeof item === 'object' &&
        typeof item.content === 'string' &&
        item.content.trim() !== ''
    )
    .sort((a, b) => publicationTime(b) - publicationTime(a))
}

function publicationTime(item: Announcement): number {
  if (typeof item.publishDate !== 'string') return 0
  const time = Date.parse(item.publishDate)
  return Number.isFinite(time) ? time : 0
}

export function getPublishedNotifications(
  feed: NotificationFeed
): NotificationSnapshot {
  const announcements = getStatusAnnouncements({
    announcements_enabled: true,
    announcements: feed.announcements,
  }).filter((item) => publicationTime(item) <= feed.server_time)
  const notice = {
    ...feed.notice,
    content: (feed.notice?.content || '').trim(),
  }
  if (notice.published_at * 1000 > feed.server_time) notice.content = ''
  return { notice, announcements: announcements.map((item) => ({ ...item })) }
}

export function getRecentUnreadNotifications(
  feed: NotificationFeed,
  read: NotificationReadState
): NotificationSnapshot | null {
  const published = getPublishedNotifications(feed)
  const now = feed.server_time
  const isRecent = (time: number) =>
    Number.isFinite(time) && time > 0 && time <= now && now - time < DAY_MS
  const closedKeys = new Set(
    read.closedUntilDate === new Date(now).toDateString()
      ? read.closedNotificationKeys
      : []
  )
  const noticeKey = getNoticeKey(published.notice)
  const noticeUnread =
    published.notice.content &&
    isRecent(published.notice.published_at * 1000) &&
    !read.readNoticeKeys.includes(noticeKey) &&
    !closedKeys.has(noticeKey)
  const readAnnouncements = new Set(read.readAnnouncementKeys)
  const announcements = published.announcements.filter((item) => {
    const key = getAnnouncementKey(item)
    return (
      isRecent(publicationTime(item)) &&
      !readAnnouncements.has(key) &&
      !closedKeys.has(key)
    )
  })
  if (!noticeUnread && announcements.length === 0) return null
  return {
    notice: noticeUnread ? published.notice : { content: '', published_at: 0 },
    announcements,
  }
}

export function getSnapshotKeys(snapshot: NotificationSnapshot): {
  noticeKeys: string[]
  announcementKeys: string[]
} {
  return {
    noticeKeys: snapshot.notice.content ? [getNoticeKey(snapshot.notice)] : [],
    announcementKeys: snapshot.announcements.map(getAnnouncementKey),
  }
}

export function getSnapshotAnnouncements(
  snapshot: NotificationSnapshot
): Announcement[] {
  const items = [...snapshot.announcements]
  if (snapshot.notice.content) {
    items.push({
      content: snapshot.notice.content,
      type: 'ongoing',
      publishDate:
        snapshot.notice.published_at > 0
          ? new Date(snapshot.notice.published_at * 1000).toISOString()
          : undefined,
    })
  }
  return items.sort((a, b) => publicationTime(b) - publicationTime(a))
}

// Same-tab event plus a content-free cross-tab signal. Polling remains the
// fallback for different browsers/devices and unavailable browser storage.
export function requestNotificationRefresh(): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new Event(NOTIFICATION_REFRESH_EVENT))
  try {
    window.localStorage.setItem(
      NOTIFICATION_REFRESH_STORAGE_KEY,
      `${Date.now()}:${Math.random()}`
    )
  } catch {
    /* refresh still works in this tab */
  }
}
