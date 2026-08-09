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
  REGISTRATION_USERNAME_ERROR_MESSAGE,
  registerFormSchema,
  sanitizeRegistrationUsername,
} from './constants'

const validRegistration = {
  email: '',
  password: 'password123',
  confirmPassword: 'password123',
}

describe('registration username validation', () => {
  test('accepts English letters, Arabic numerals, and common email characters', () => {
    for (const username of [
      'Alice',
      'alice123',
      '123456',
      'name@example.com',
      'first.last@example.com',
      'name+tag@example.com',
      'first_last',
      'first-last',
    ]) {
      const result = registerFormSchema.safeParse({
        ...validRegistration,
        username,
      })

      assert.equal(result.success, true, username)
    }
  })

  test('rejects Chinese characters, spaces, and unsupported characters', () => {
    for (const username of ['用户名', 'alice user', 'alice😀']) {
      const result = registerFormSchema.safeParse({
        ...validRegistration,
        username,
      })

      assert.equal(result.success, false, username)
      if (!result.success) {
        assert.equal(
          result.error.issues.find((issue) => issue.path[0] === 'username')
            ?.message,
          REGISTRATION_USERNAME_ERROR_MESSAGE
        )
      }
    }
  })

  test('preserves common email characters while removing unsupported input', () => {
    assert.equal(
      sanitizeRegistrationUsername('A中_1-2@example.com +😀'),
      'A_1-2@example.com+'
    )
  })
})
