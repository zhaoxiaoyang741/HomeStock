import { useEffect, useState, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Brain, CheckCircle2, Loader2, Save, Pencil, RefreshCw } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import { getModelList, updateModelConfig, swapActiveModel } from '@/api/models'
import type { ModelConfig } from '@/types/model'

interface ModelEditState {
  model: string
  apiKey: string
  editingKey: boolean
  apiBase: string
  enabled: boolean
}

export function ModelConfigSection() {
  const { t } = useTranslation('settings')
  const [modelList, setModelList] = useState<ModelConfig[]>([])
  const [activeModel, setActiveModel] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [editStates, setEditStates] = useState<Record<string, ModelEditState>>({})
  const [saving, setSaving] = useState<Record<string, boolean>>({})
  const [swapping, setSwapping] = useState<Record<string, boolean>>({})
  const [saveSuccess, setSaveSuccess] = useState<Record<string, string>>({})
  const [swapSuccess, setSwapSuccess] = useState('')

  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const initEditState = useCallback((models: ModelConfig[]) => {
    setEditStates((prev) => {
      const next = { ...prev }
      for (const m of models) {
        if (!next[m.model_name]) {
          next[m.model_name] = {
            model: m.model,
            apiKey: '',
            editingKey: false,
            apiBase: m.api_base,
            enabled: m.enabled,
          }
        }
      }
      return next
    })
  }, [])

  const fetchModelList = useCallback(async () => {
    try {
      const data = await getModelList()
      setModelList(data.models)
      setActiveModel(data.active_model)
      initEditState(data.models)
      setError('')
    } catch (err) {
      if (!loading) {
        setError(err instanceof Error ? err.message : t('modelFetchFailed'))
      }
    } finally {
      setLoading(false)
    }
  }, [loading, t, initEditState])

  useEffect(() => {
    void fetchModelList()

    pollingRef.current = setInterval(() => {
      void fetchModelList()
    }, 10000)

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current)
      }
    }
  }, [fetchModelList])

  function getEditState(modelName: string): ModelEditState {
    return editStates[modelName] ?? { model: '', apiKey: '', editingKey: false, apiBase: '', enabled: false }
  }

  function updateEditState(modelName: string, partial: Partial<ModelEditState>) {
    setEditStates((prev) => ({
      ...prev,
      [modelName]: { ...(prev[modelName] ?? getEditState(modelName)), ...partial },
    }))
  }

  async function handleSave(modelName: string) {
    setSaving((prev) => ({ ...prev, [modelName]: true }))
    setError('')
    setSaveSuccess((prev) => ({ ...prev, [modelName]: '' }))
    try {
      const state = getEditState(modelName)
      await updateModelConfig({
        model_name: modelName,
        enabled: state.enabled,
        model: state.model || undefined,
        api_key: state.apiKey || undefined,
        api_base: state.apiBase || undefined,
      })
      setSaveSuccess((prev) => ({ ...prev, [modelName]: t('modelSaveSuccess') }))
      updateEditState(modelName, { apiKey: '', editingKey: false })
      await fetchModelList()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('modelSaveFailed'))
    } finally {
      setSaving((prev) => ({ ...prev, [modelName]: false }))
    }
  }

  async function handleSwap(modelName: string) {
    setSwapping((prev) => ({ ...prev, [modelName]: true }))
    setError('')
    setSwapSuccess('')
    try {
      await swapActiveModel({ model_name: modelName })
      setSwapSuccess(t('modelApplySuccess'))
      await fetchModelList()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('modelApplyFailed'))
    } finally {
      setSwapping((prev) => ({ ...prev, [modelName]: false }))
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="w-6 h-6 text-on-surface-variant animate-spin" />
      </div>
    )
  }

  if (modelList.length === 0) {
    return (
      <div className="text-center space-y-2 py-16">
        <Brain className="w-12 h-12 text-on-surface-variant/40 mx-auto" />
        <p className="text-sm text-on-surface-variant">{t('modelEmpty')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Error message */}
      {error && (
        <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">
          {error}
        </div>
      )}

      {/* Swap success message */}
      {swapSuccess && (
        <div className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">
          {swapSuccess}
        </div>
      )}

      {modelList.map((mc) => {
        const state = getEditState(mc.model_name)
        const isActive = mc.model_name === activeModel
        const isSaving = saving[mc.model_name] ?? false
        const isSwapping = swapping[mc.model_name] ?? false
        const successMsg = saveSuccess[mc.model_name] ?? ''

        return (
          <Card
            key={mc.model_name}
            className={cn(
              'rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm transition-all',
              isActive && 'ring-1 ring-primary/30',
            )}
          >
            <CardHeader className="pb-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <Brain className="w-5 h-5 text-primary" />
                  <div>
                    <CardTitle className="text-lg font-bold tracking-tight">{mc.model_name}</CardTitle>
                    <CardDescription>
                      <Badge variant="outline" className="text-xs">
                        {mc.provider === 'ollama' ? t('modelProviderOllama') : t('modelProviderOpenAI')}
                      </Badge>
                    </CardDescription>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  {isActive && (
                    <Badge className="bg-primary/10 text-primary border-primary/20 text-xs gap-1">
                      <CheckCircle2 className="w-3 h-3" />
                      {t('modelActive')}
                    </Badge>
                  )}
                  <div className={cn(
                    'w-3 h-3 rounded-full',
                    isActive ? 'bg-primary' : 'bg-on-surface-variant/30',
                  )} />
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-5">
              <Separator />

              {/* Success message for this model */}
              {successMsg && (
                <div className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">
                  {successMsg}
                </div>
              )}

              {/* Enable toggle */}
              <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
                <div>
                  <p className="text-sm font-medium text-on-surface">{t('modelEnableLabel')}</p>
                  <p className="text-xs text-on-surface-variant">{t('modelEnableHint')}</p>
                </div>
                <Checkbox
                  checked={state.enabled}
                  onCheckedChange={(checked) => updateEditState(mc.model_name, { enabled: checked === true })}
                />
              </div>

              {/* Model ID */}
              <div className="space-y-1.5">
                <Label htmlFor={`model-${mc.model_name}-id`}>{t('modelIdLabel')}</Label>
                <Input
                  id={`model-${mc.model_name}-id`}
                  placeholder={mc.model}
                  value={state.model}
                  onChange={(e) => updateEditState(mc.model_name, { model: e.target.value })}
                />
              </div>

              {/* API Key */}
              <div className="space-y-1.5">
                <Label htmlFor={`model-${mc.model_name}-key`}>{t('modelApiKeyLabel')}</Label>
                {state.editingKey ? (
                  <>
                    <Input
                      id={`model-${mc.model_name}-key`}
                      type="password"
                      placeholder={mc.api_key ? '········' : t('modelApiKeyPlaceholder')}
                      value={state.apiKey}
                      onChange={(e) => updateEditState(mc.model_name, { apiKey: e.target.value })}
                    />
                    <p className="text-xs text-on-surface-variant">{t('modelApiKeyHint')}</p>
                  </>
                ) : (
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-on-surface-variant">{mc.api_key || t('modelApiKeyNotSet')}</span>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => updateEditState(mc.model_name, { editingKey: true })}
                    >
                      <Pencil className="w-3.5 h-3.5 mr-1" />
                      {t('modelChangeKey')}
                    </Button>
                  </div>
                )}
              </div>

              {/* API Base */}
              <div className="space-y-1.5">
                <Label htmlFor={`model-${mc.model_name}-base`}>{t('modelApiBaseLabel')}</Label>
                <Input
                  id={`model-${mc.model_name}-base`}
                  placeholder={mc.api_base || t('modelApiBaseHint')}
                  value={state.apiBase}
                  onChange={(e) => updateEditState(mc.model_name, { apiBase: e.target.value })}
                />
                <p className="text-xs text-on-surface-variant">{t('modelApiBaseHint')}</p>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-3">
                <Button
                  size="sm"
                  onClick={() => void handleSave(mc.model_name)}
                  disabled={isSaving}
                >
                  {isSaving ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="mr-2 h-4 w-4" />
                  )}
                  {isSaving ? t('modelSaving') : t('common:save')}
                </Button>
                {!isActive && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => void handleSwap(mc.model_name)}
                    disabled={isSwapping || !state.enabled}
                  >
                    {isSwapping ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <RefreshCw className="mr-2 h-4 w-4" />
                    )}
                    {isSwapping ? t('modelApplying') : t('modelApply')}
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        )
      })}

      {/* Refresh button */}
      <div className="flex justify-center">
        <Button
          variant="outline"
          size="sm"
          onClick={() => { setLoading(true); void fetchModelList() }}
          disabled={loading}
        >
          <RefreshCw className={cn('mr-2 h-4 w-4', loading ? 'animate-spin' : '')} />
          {t('common:refresh')}
        </Button>
      </div>
    </div>
  )
}
