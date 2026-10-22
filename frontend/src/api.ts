import type {
  User,
  CompareResponse,
  OutboxResponse,
  RepairResponse,
  SyncMissingResponse,
} from './types'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`HTTP ${res.status}: ${text}`)
  }
  return res.json() as Promise<T>
}

export interface CreateUserInput {
  first_name: string
  last_name: string
  email: string
  age: number
}

export interface UpdateUserInput {
  first_name: string
  last_name: string
  age: number
}

export const api = {
  health: () => fetch('/api/health').then((r) => r.text()),
  users: () => request<User[]>('/api/users'),
  usersYugabyte: () => request<User[]>('/api/users/yugabyte'),
  compare: () => request<CompareResponse>('/api/users/compare'),
  outbox: (status?: string) =>
    request<OutboxResponse>(
      `/api/sync/outbox${status ? `?status=${status}` : ''}`,
    ),
  syncMissing: () =>
    request<SyncMissingResponse>('/api/users/sync/missing', {
      method: 'POST',
    }),
  repair: (deleteExtra: boolean) =>
    request<RepairResponse>('/api/users/repair', {
      method: 'POST',
      body: JSON.stringify({ delete_extra: deleteExtra }),
    }),
  retryOutbox: (id: number) =>
    request(`/api/sync/outbox/retry?id=${id}`, { method: 'POST' }),

  createUser: (input: CreateUserInput) =>
    request('/api/users/create', {
      method: 'POST',
      body: JSON.stringify(input),
    }),

  updateUser: (id: number, input: UpdateUserInput) =>
    request(`/api/users/update?id=${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    }),

  deleteUser: (id: number) =>
    request(`/api/users/delete?id=${id}`, { method: 'DELETE' }),
}