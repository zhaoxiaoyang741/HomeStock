export type ScheduledTaskState = 'idle' | 'running'
export type ScheduledTaskResult = 'running' | 'success' | 'failed' | 'skipped' | ''
export type ScheduledTaskTriggerSource = 'manual' | 'scheduled'

export interface ScheduledTask {
  id: string
  code: string
  name: string
  description: string
  cron_spec: string
  enabled: boolean
  registered: boolean
  run_timeout_seconds: number
  state: ScheduledTaskState
  next_run_at: string | null
  last_run_started_at: string | null
  last_run_finished_at: string | null
  last_result: ScheduledTaskResult | string
  last_error: string
  created_at: string
  updated_at: string
}

export interface ScheduledTaskRun {
  id: string
  task_code: string
  task_name: string
  trigger_source: ScheduledTaskTriggerSource
  status: ScheduledTaskResult
  summary: string
  result_payload: string
  error_message: string
  started_at: string
  finished_at: string | null
  duration_ms: number
  triggered_by_user_name: string
  triggered_by_user_id: string
  triggered_by_channel: string
  created_at: string
}

export interface ScheduledTaskRunPage {
  items: ScheduledTaskRun[]
  total: number
  page: number
  page_size: number
}

export interface UpdateScheduledTaskPayload {
  cron_spec?: string
  enabled?: boolean
  run_timeout_seconds?: number
}
