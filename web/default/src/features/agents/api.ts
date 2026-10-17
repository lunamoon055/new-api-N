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
import { api } from '@/lib/api'

export type AgentStatus = 'pending' | 'approved' | 'rejected' | 'revoked'

export interface AgentApplication {
  id: number
  user_id: number
  username?: string
  status: AgentStatus
  note: string
  review_note: string
  reviewer_id: number
  applied_at: number
  reviewed_at: number
}

interface ApiResponse<T> {
  success: boolean
  message?: string
  data: T
}

export interface AgentCenterData {
  application: AgentApplication | null
  agent_code: string
  promotion_code: string
  customer_count: number
  reward_pending_units: number
  reward_paid_units: number
}

export interface ReferralReward {
  id: number
  task_record_id: number
  customer_id: number
  referrer_id: number
  referral_kind: string
  source_order_id: string
  cost_evidence_id: string
  channel_id: number
  model_name: string
  sale_units: number
  fee_units: number
  cost_units: number
  extra_cost_units: number
  margin_units: number
  reward_units: number
  platform_units: number
  status: 'pending' | 'paid' | 'reversed'
  eligible_at: number
  created_at: number
  paid_at: number
  payout_reference: string
}

export interface AgentRewardSummary {
  id: number
  model_name: string
  reward_units: number
  status: 'pending' | 'paid' | 'reversed'
  eligible_at: number
  created_at: number
}

export interface AgentTaskPreview {
  id: number
  task_id: string
  user_id: number
  channel_id: number
  model_name: string
  quota: number
  status: string
  billing_status: string
  finish_time: number
  inviter_id: number
  referral_kind: string
}

export async function getAgentCenter(): Promise<AgentCenterData> {
  const response =
    await api.get<ApiResponse<AgentCenterData>>('/api/user/agent')
  if (!response.data?.success || !response.data.data) {
    throw new Error('Agent service unavailable')
  }
  return response.data.data
}

export async function applyForAgent(note: string): Promise<void> {
  await api.post('/api/user/agent/apply', { note })
}

export async function getAgentApplications(status: string, page: number) {
  const response = await api.get<
    ApiResponse<{ items: AgentApplication[]; total: number }>
  >('/api/user/agent/applications', { params: { status, page } })
  return response.data.data
}

export async function reviewAgentApplication(
  id: number,
  decision: 'approved' | 'rejected' | 'revoked',
  note: string
): Promise<void> {
  await api.post(`/api/user/agent/applications/${id}/review`, {
    decision,
    note,
  })
}

export async function getMyReferralRewards() {
  const response = await api.get<
    ApiResponse<{ items: AgentRewardSummary[]; total: number }>
  >('/api/user/agent/rewards')
  return response.data.data
}

export async function getAdminReferralRewards(status: string, page: number) {
  const response = await api.get<
    ApiResponse<{ items: ReferralReward[]; total: number }>
  >('/api/user/agent/admin/rewards', { params: { status, page } })
  return response.data.data
}

export async function getAgentTaskPreview(
  id: number
): Promise<AgentTaskPreview> {
  const response = await api.get<ApiResponse<AgentTaskPreview>>(
    `/api/user/agent/admin/task/${id}`
  )
  return response.data.data
}

export async function searchAgentTasks(
  taskID: string
): Promise<AgentTaskPreview[]> {
  const response = await api.get<ApiResponse<AgentTaskPreview[]>>(
    '/api/user/agent/admin/tasks/search',
    { params: { task_id: taskID } }
  )
  return response.data.data
}

export async function createReferralReward(input: {
  task_record_id: number
  source_order_id: string
  cost_evidence_id: string
  sale_units: number
  cost_units: number
  extra_cost_units: number
}): Promise<ReferralReward> {
  const response = await api.post<ApiResponse<ReferralReward>>(
    '/api/user/agent/admin/rewards',
    input
  )
  return response.data.data
}

export async function markReferralRewardPaid(
  id: number,
  payoutReference: string
): Promise<void> {
  await api.post(`/api/user/agent/admin/rewards/${id}/paid`, {
    payout_reference: payoutReference,
  })
}
