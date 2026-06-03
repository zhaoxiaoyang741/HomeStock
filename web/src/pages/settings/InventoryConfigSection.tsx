import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Package, Loader2, Save } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { getInventoryConfig, updateInventoryConfig } from '@/api/inventory-settings'
import type { InventoryConfig } from '@/api/inventory-settings'

export function InventoryConfigSection() {
  const { t } = useTranslation('settings')
  const [config, setConfig] = useState<InventoryConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState('')

  // Editable fields
  const [defaultLocation, setDefaultLocation] = useState('')
  const [defaultExpiryDays, setDefaultExpiryDays] = useState(0)
  const [defaultQuantity, setDefaultQuantity] = useState(1)
  const [dueSoonDays, setDueSoonDays] = useState(7)
  const [trackPrice, setTrackPrice] = useState(false)
  const [trackOpened, setTrackOpened] = useState(false)
  const [autoAddShoppingList, setAutoAddShoppingList] = useState(false)
  const [nluDefaultQuantity, setNluDefaultQuantity] = useState(1)
  const [nluAutoSelectThreshold, setNluAutoSelectThreshold] = useState(0.85)
  const [nluAutoSelectLead, setNluAutoSelectLead] = useState(0.15)

  const fetchConfig = async () => {
    try {
      const cfg = await getInventoryConfig()
      setConfig(cfg)
      setDefaultLocation(cfg.default_location)
      setDefaultExpiryDays(cfg.default_expiry_days)
      setDefaultQuantity(cfg.default_quantity)
      setDueSoonDays(cfg.due_soon_days)
      setTrackPrice(cfg.track_price)
      setTrackOpened(cfg.track_opened)
      setAutoAddShoppingList(cfg.auto_add_shopping_list)
      setNluDefaultQuantity(cfg.nlu_default_quantity)
      setNluAutoSelectThreshold(cfg.nlu_auto_select_threshold)
      setNluAutoSelectLead(cfg.nlu_auto_select_lead)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('invConfigLoadFailed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchConfig()
  }, [])

  async function handleSave() {
    setSaving(true)
    setError('')
    setSaveSuccess('')
    try {
      const updated = await updateInventoryConfig({
        default_location: defaultLocation,
        default_expiry_days: defaultExpiryDays,
        default_quantity: defaultQuantity,
        due_soon_days: dueSoonDays,
        track_price: trackPrice,
        track_opened: trackOpened,
        auto_add_shopping_list: autoAddShoppingList,
        nlu_default_quantity: nluDefaultQuantity,
        nlu_auto_select_threshold: nluAutoSelectThreshold,
        nlu_auto_select_lead: nluAutoSelectLead,
      })
      setConfig(updated)
      setSaveSuccess(t('invConfigSaveSuccess'))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('invConfigSaveFailed'))
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="w-6 h-6 text-on-surface-variant animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-3">
            <Package className="w-5 h-5 text-primary" />
            <div>
              <CardTitle className="text-lg font-bold tracking-tight">{t('invSectionTitle')}</CardTitle>
              <CardDescription>{t('invSectionDescription')}</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-5">
          <Separator />

          {/* Error message */}
          {error && (
            <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">
              {error}
            </div>
          )}

          {/* Success message */}
          {saveSuccess && (
            <div className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">
              {saveSuccess}
            </div>
          )}

          {/* ── Inbound Defaults ── */}
          <h4 className="text-sm font-medium text-on-surface">{t('invDefaultsTitle')}</h4>

          {/* Default location */}
          <div className="space-y-1.5">
            <Label htmlFor="inv-default-location">{t('invDefaultLocation')}</Label>
            <Input
              id="inv-default-location"
              placeholder={t('invDefaultLocationHint')}
              value={defaultLocation}
              onChange={(e) => setDefaultLocation(e.target.value)}
            />
          </div>

          {/* Default expiry days */}
          <div className="space-y-1.5">
            <Label htmlFor="inv-default-expiry">{t('invDefaultExpiryDays')}</Label>
            <Input
              id="inv-default-expiry"
              type="number"
              min={-1}
              value={defaultExpiryDays}
              onChange={(e) => setDefaultExpiryDays(parseInt(e.target.value) || 0)}
            />
            <p className="text-xs text-on-surface-variant">{t('invDefaultExpiryDaysHint')}</p>
          </div>

          {/* Default quantity */}
          <div className="space-y-1.5">
            <Label htmlFor="inv-default-qty">{t('invDefaultQuantity')}</Label>
            <Input
              id="inv-default-qty"
              type="number"
              min={1}
              step={0.5}
              value={defaultQuantity}
              onChange={(e) => setDefaultQuantity(parseFloat(e.target.value) || 1)}
            />
          </div>

          {/* Due soon days */}
          <div className="space-y-1.5">
            <Label htmlFor="inv-due-soon">{t('invDueSoonDays')}</Label>
            <Input
              id="inv-due-soon"
              type="number"
              min={1}
              value={dueSoonDays}
              onChange={(e) => setDueSoonDays(parseInt(e.target.value) || 7)}
            />
            <p className="text-xs text-on-surface-variant">{t('invDueSoonDaysHint')}</p>
          </div>

          <Separator />

          {/* ── Feature Toggles ── */}
          <h4 className="text-sm font-medium text-on-surface">{t('invFeatureToggles')}</h4>

          {/* Track price */}
          <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
            <div>
              <p className="text-sm font-medium text-on-surface">{t('invTrackPrice')}</p>
              <p className="text-xs text-on-surface-variant">{t('invTrackPriceHint')}</p>
            </div>
            <Checkbox
              checked={trackPrice}
              onCheckedChange={(checked) => setTrackPrice(checked === true)}
            />
          </div>

          {/* Track opened */}
          <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
            <div>
              <p className="text-sm font-medium text-on-surface">{t('invTrackOpened')}</p>
              <p className="text-xs text-on-surface-variant">{t('invTrackOpenedHint')}</p>
            </div>
            <Checkbox
              checked={trackOpened}
              onCheckedChange={(checked) => setTrackOpened(checked === true)}
            />
          </div>

          {/* Auto add shopping list */}
          <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
            <div>
              <p className="text-sm font-medium text-on-surface">{t('invAutoAddShoppingList')}</p>
              <p className="text-xs text-on-surface-variant">{t('invAutoAddShoppingListHint')}</p>
            </div>
            <Checkbox
              checked={autoAddShoppingList}
              onCheckedChange={(checked) => setAutoAddShoppingList(checked === true)}
            />
          </div>

          <Separator />

          {/* ── NLU Settings ── */}
          <h4 className="text-sm font-medium text-on-surface">{t('invNluSettings')}</h4>

          {/* NLU default quantity */}
          <div className="space-y-1.5">
            <Label htmlFor="inv-nlu-default-qty">{t('invNluDefaultQuantity')}</Label>
            <Input
              id="inv-nlu-default-qty"
              type="number"
              min={1}
              step={0.5}
              value={nluDefaultQuantity}
              onChange={(e) => setNluDefaultQuantity(parseFloat(e.target.value) || 1)}
            />
            <p className="text-xs text-on-surface-variant">{t('invNluDefaultQuantityHint')}</p>
          </div>

          {/* NLU auto-select threshold */}
          <div className="space-y-1.5">
            <Label htmlFor="inv-nlu-threshold">{t('invNluThreshold')}</Label>
            <Input
              id="inv-nlu-threshold"
              type="number"
              min={0}
              max={1}
              step={0.05}
              value={nluAutoSelectThreshold}
              onChange={(e) => setNluAutoSelectThreshold(parseFloat(e.target.value) || 0.85)}
            />
            <p className="text-xs text-on-surface-variant">{t('invNluThresholdHint')}</p>
          </div>

          {/* NLU auto-select lead */}
          <div className="space-y-1.5">
            <Label htmlFor="inv-nlu-lead">{t('invNluLead')}</Label>
            <Input
              id="inv-nlu-lead"
              type="number"
              min={0}
              max={1}
              step={0.05}
              value={nluAutoSelectLead}
              onChange={(e) => setNluAutoSelectLead(parseFloat(e.target.value) || 0.15)}
            />
            <p className="text-xs text-on-surface-variant">{t('invNluLeadHint')}</p>
          </div>

          {/* Save button */}
          <Button
            size="sm"
            onClick={() => void handleSave()}
            disabled={saving}
          >
            {saving ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            {saving ? t('modelSaving') : t('common:save')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
