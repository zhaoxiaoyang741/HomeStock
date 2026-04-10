import { InventoryClient, StockLot } from '../utils/apiClient.js';
import {
  coerceNumericValue,
  coerceStringValue,
  daysUntilExpiry,
  formatLocalDate,
  isRawCommandEnvelope,
  normalizeDateInput,
  parseSlashCommandArgs,
} from '../utils/runtime.js';

export interface AddItemParams {
  name: string;
  quantity?: number;
  unit?: string;
  spec?: string;
  location?: string;
  category?: string;
  expire_at?: string;
  notes?: string;
}

const KNOWN_LOCATIONS = ['冰箱', '冷冻层', '冷藏室', '厨房', '储物间', '浴室', '阳台'];
const KNOWN_CATEGORIES = ['蔬菜', '水果', '肉类', '海鲜', '调料', '零食', '饮料', '日用品'];

export function parseAddItemParams(input: AddItemParams | unknown): AddItemParams {
  if (!isRawCommandEnvelope(input)) {
    const record = (input ?? {}) as Record<string, unknown>;
    return {
      name: coerceStringValue(record.name) ?? '',
      quantity: coerceNumericValue(record.quantity),
      unit: coerceStringValue(record.unit),
      spec: coerceStringValue(record.spec),
      location: coerceStringValue(record.location),
      category: coerceStringValue(record.category),
      expire_at:
        coerceStringValue(record.expire_at) ??
        coerceStringValue(record.expiry_date) ??
        coerceStringValue(record.expires_at),
      notes: coerceStringValue(record.notes),
    };
  }

  const tokens = parseSlashCommandArgs(input.command ?? '');
  const result: AddItemParams = { name: '' };

  for (const token of tokens) {
    const quantityMatch = token.match(/^(\d+(?:\.\d+)?)([\u4e00-\u9fa5a-zA-Z]+)?$/);
    if (quantityMatch && result.quantity === undefined) {
      result.quantity = Number(quantityMatch[1]);
      if (quantityMatch[2]) {
        result.unit = quantityMatch[2];
      }
      continue;
    }

    if (!result.location && KNOWN_LOCATIONS.some(location => token.includes(location))) {
      result.location = token;
      continue;
    }

    if (!result.category && KNOWN_CATEGORIES.some(category => token.includes(category))) {
      result.category = token;
      continue;
    }

    if (!result.expire_at && /^\d{4}-\d{2}-\d{2}$/.test(token)) {
      result.expire_at = token;
      continue;
    }

    if (!result.name) {
      result.name = token;
      continue;
    }

    result.notes = result.notes ? `${result.notes} ${token}` : token;
  }

  return result;
}

function buildSuccessMessage(lot: StockLot): string {
  const material = lot.material;
  let response = `✅ 已入库 **${material?.name ?? '物料'}** ${lot.quantity_on_hand}${lot.unit}`;
  if (material?.spec) {
    response += ` / ${material.spec}`;
  }
  if (lot.location) {
    response += `，位置：${lot.location}`;
  }
  if (lot.expire_at) {
    const daysLeft = daysUntilExpiry(lot.expire_at);
    response += `\n📅 过期日期：${formatLocalDate(lot.expire_at)}（还剩 ${daysLeft} 天）`;
  }
  if (lot.notes) {
    response += `\n📝 备注：${lot.notes}`;
  }
  return response;
}

export async function addItem(params: AddItemParams | unknown, client: InventoryClient): Promise<string> {
  try {
    const parsed = parseAddItemParams(params);
    const name = parsed.name?.trim();
    if (!name) {
      return '❌ 缺少物料名称。';
    }
    if (parsed.quantity !== undefined && parsed.quantity <= 0) {
      return '❌ 数量必须大于 0。';
    }

    const category = parsed.category?.trim();
    const categoryRecord = category ? await client.ensureCategoryByName(category) : null;
    const expireAt = parsed.expire_at ? normalizeDateInput(parsed.expire_at) : undefined;

    const lot = await client.inboundStockLot({
      name,
      quantity: parsed.quantity ?? 1,
      unit: parsed.unit?.trim() || undefined,
      spec: parsed.spec?.trim() || undefined,
      location: parsed.location?.trim() || undefined,
      category_id: categoryRecord?.id,
      expire_at: expireAt,
      notes: parsed.notes?.trim() || undefined,
    });

    return buildSuccessMessage(lot);
  } catch (err) {
    return `❌ 入库失败：${(err as Error).message}`;
  }
}
