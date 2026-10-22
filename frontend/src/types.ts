export interface User {
  id: number
  first_name: string
  last_name: string
  email: string
  age: number
  created_at: string
  updated_at: string
}

export interface UserRecord {
  id: number
  first_name: string
  last_name: string
  email: string
  age: number | null
}

export interface UserDifference {
  email: string
  status: 'missing_in_yugabyte' | 'missing_in_postgres' | 'data_mismatch'
  postgres_user?: UserRecord
  yugabyte_user?: UserRecord
  different_fields?: string[]
}

export interface CompareResponse {
  success: boolean
  differences_count: number
  differences: UserDifference[]
}

export interface OutboxItem {
  id: number
  entity_type: string
  entity_key: string
  operation: string
  status: string
  attempts: number
  next_retry_at: string
  last_error: string | null
  created_at: string
  updated_at: string
}

export interface OutboxResponse {
  success: boolean
  count: number
  data: OutboxItem[]
}

export interface SyncMissingResponse {
  success: boolean
  message: string
  source: string
  target: string
  inserted: number
  skipped: number
  failed: number
  failures: { email: string; reason: string }[]
}

export interface RepairResponse {
  success: boolean
  message: string
  source: string
  target: string
  inserted: number
  updated: number
  unchanged: number
  deleted: number
  delete_extra_enabled: boolean
  failed: number
  failures: { email: string; reason: string }[]
}