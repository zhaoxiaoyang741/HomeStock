export interface SchedulerStatus {
  state: 'idle' | 'running'
  last_run_at: string | null
  next_run_at: string | null
  last_result: 'success' | 'failed' | ''
  last_error?: string
}
