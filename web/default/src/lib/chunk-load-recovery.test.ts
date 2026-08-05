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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  clearChunkLoadRecovery,
  isChunkLoadError,
  recoverFromChunkLoadError,
} from './chunk-load-recovery'

function createEnvironment() {
  const values = new Map<string, string>()
  let reloadCount = 0

  return {
    environment: {
      location: {
        href: 'https://example.com/usage-logs/task',
        reload: () => {
          reloadCount += 1
        },
      },
      sessionStorage: {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => values.set(key, value),
        removeItem: (key: string) => values.delete(key),
      },
    },
    getReloadCount: () => reloadCount,
    values,
  }
}

describe('chunk load recovery', () => {
  test('recognizes common browser and bundler chunk load errors', () => {
    assert.equal(
      isChunkLoadError(new Error('Loading chunk 3011 failed.')),
      true
    )
    assert.equal(
      isChunkLoadError(
        new TypeError(
          'Failed to fetch dynamically imported module: https://example.com/static/js/3011.old.js'
        )
      ),
      true
    )
    assert.equal(
      isChunkLoadError({
        cause: new Error('Importing a module script failed.'),
      }),
      true
    )
    assert.equal(isChunkLoadError(new Error('API request failed')), false)
  })

  test('reloads once for the same URL inside the retry window', () => {
    const { environment, getReloadCount } = createEnvironment()
    const error = new Error('Loading chunk 3011 failed.')

    assert.equal(recoverFromChunkLoadError(error, environment, 1_000), true)
    assert.equal(recoverFromChunkLoadError(error, environment, 2_000), false)
    assert.equal(getReloadCount(), 1)

    assert.equal(recoverFromChunkLoadError(error, environment, 61_001), true)
    assert.equal(getReloadCount(), 2)
  })

  test('clears the retry marker after a successful application start', () => {
    const { environment, values, getReloadCount } = createEnvironment()
    const error = new Error('ChunkLoadError')

    assert.equal(recoverFromChunkLoadError(error, environment, 1_000), true)
    clearChunkLoadRecovery(environment.sessionStorage)
    assert.equal(values.size, 0)
    assert.equal(recoverFromChunkLoadError(error, environment, 2_000), true)
    assert.equal(getReloadCount(), 2)
  })

  test('does not reload when session storage is unavailable', () => {
    const environment = {
      location: {
        href: 'https://example.com/usage-logs/task',
        reload: () => assert.fail('reload should not be called'),
      },
      sessionStorage: {
        getItem: () => {
          throw new Error('storage denied')
        },
        setItem: () => undefined,
        removeItem: () => undefined,
      },
    }

    assert.equal(
      recoverFromChunkLoadError(new Error('ChunkLoadError'), environment),
      false
    )
  })
})
