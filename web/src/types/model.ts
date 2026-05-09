export interface ModelConfig {
  model_name: string
  model: string
  provider: 'openai' | 'ollama'
  enabled: boolean
  api_key: string
  api_base: string
}

export interface ModelListData {
  models: ModelConfig[]
  active_model: string
  last_reload_time: string
}

export interface UpdateModelPayload {
  model_name: string
  enabled?: boolean
  model?: string
  provider?: 'openai' | 'ollama'
  api_key?: string
  api_base?: string
}

export interface SwapModelPayload {
  model_name: string
}
