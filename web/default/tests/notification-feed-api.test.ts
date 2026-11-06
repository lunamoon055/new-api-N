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
import { afterAll, afterEach, describe, expect, it, spyOn } from 'bun:test'
import { AxiosError } from 'axios'
import { api, getNotificationFeed } from '../src/lib/api'
const mock = spyOn(api, 'get')
afterEach(() => mock.mockReset())
afterAll(() => mock.mockRestore())
describe('background notification feed failures', () => {
  it.each([401, 500, 503])('does not expose HTTP %i to global sign-out/error-page handlers', async status => {
    const failure = new AxiosError('unavailable')
    failure.response = { status } as never
    mock.mockRejectedValueOnce(failure)
    let received: unknown
    try { await getNotificationFeed() } catch (error) { received = error }
    expect(received).toBeInstanceOf(Error)
    expect(received).not.toBeInstanceOf(AxiosError)
  })
  it('can fetch a later publication after a failed request', async () => {
    mock.mockRejectedValueOnce(new Error('offline'))
    await expect(getNotificationFeed()).rejects.toThrow()
    const feed = { server_time: Date.now(), notice: { content: 'new', published_at: 0 }, announcements: [] }
    mock.mockResolvedValueOnce({ data: { success: true, data: feed } } as never)
    expect(await getNotificationFeed()).toEqual(feed)
  })
  it('rejects malformed success bodies without crashing the announcement UI', async () => {
    mock.mockResolvedValueOnce({ data: { success: true, data: {} } } as never)
    await expect(getNotificationFeed()).rejects.toThrow('Notification feed unavailable')
  })
})
