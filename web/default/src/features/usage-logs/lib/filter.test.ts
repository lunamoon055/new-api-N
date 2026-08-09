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
import { TASK_STATUS } from '../constants'
import { buildSearchParams } from './filter'
import { buildTaskLogParams } from './utils'

describe('task log filters', () => {
  test('persists model and status in the route search state', () => {
    assert.deepEqual(
      buildSearchParams(
        {
          taskId: 'task_123',
          model: 'Seedance-2.5',
          status: TASK_STATUS.FAILURE,
        },
        'task'
      ),
      {
        filter: 'task_123',
        model: 'Seedance-2.5',
        status: TASK_STATUS.FAILURE,
      }
    )
  })

  test('maps route model and status filters to the task API query', () => {
    assert.deepEqual(
      buildTaskLogParams(
        { p: 2, page_size: 100 },
        {
          filter: 'task_123',
          model: 'Seedance-2.5',
          status: TASK_STATUS.SUCCESS,
        }
      ),
      {
        p: 2,
        page_size: 100,
        task_id: 'task_123',
        model_name: 'Seedance-2.5',
        status: TASK_STATUS.SUCCESS,
      }
    )
  })
})
