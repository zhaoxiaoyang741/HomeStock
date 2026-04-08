import { InventoryClient, Item } from '../utils/apiClient.js';
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
  force_create?: boolean;
  merge_with_id?: string;
}

const KNOWN_LOCATIONS = ['冰箱', '冷冻层', '冷藏室', '厨柜', '橱柜', '厨房', '储物间', '浴室', '阳台'];
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
      force_create: record.force_create === true,
      merge_with_id: coerceStringValue(record.merge_with_id),
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

function buildSuccessMessage(item: Item): string {
  let response = `✅ 已添加 **${item.name}** ${item.quantity}${item.unit}`;
  if (item.location) {
    response += `，存放在【${item.location}】`;
  }
  if (item.category?.name) {
    response += `，分类：${item.category.name}`;
  }
  if (item.expire_at) {
    const daysLeft = daysUntilExpiry(item.expire_at);
    response += `\n📅 过期日期：${formatLocalDate(item.expire_at)}（还剩 ${daysLeft} 天）`;
  }
  if (item.notes) {
    response += `\n📝 备注：${item.notes}`;
  }
  return response;
}

function buildMergePrompt(existing: Item, incoming: AddItemParams, mergedQty: number): string {
  const existingLabel = existing.spec
    ? `${existing.name}（${existing.spec}）`
    : existing.name;
  const incomingQty = incoming.quantity ?? 1;
  const unit = existing.unit;
  const locationNote = existing.location ? `，存放在【${existing.location}】` : '';

  return [
    `⚠️ 发现相似物料`,
    ``,
    `库存中已有「${existingLabel}」${existing.quantity}${unit}${locationNote}。`,
    ``,
    `您想要：`,
    `1. **合并**：库存更新为 ${mergedQty}${unit}（现有 ${existing.quantity} + 新增 ${incomingQty}）`,
    `2. **新建**：创建一条独立记录`,
    ``,
    `请回复"合并"或"新建"。`,
    ``,
    `[TOOL_HINT] 用户选择合并时：再次调用 add_item，参数加上 merge_with_id="${existing.id}" 且 quantity=${mergedQty}`,
    `[TOOL_HINT] 用户选择新建时：再次调用 add_item，参数加上 force_create=true`,
  ].join('\n');
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

    // Merge path: update existing item directly, no dedup check needed
    if (parsed.merge_with_id) {
      const mergedQty = parsed.quantity;
      if (mergedQty === undefined || mergedQty <= 0) {
        return '❌ 合并时必须提供合并后的总数量（quantity）。';
      }
      const updated = await client.updateItem(parsed.merge_with_id, { quantity: mergedQty });
      return `✅ 已合并「**${updated.name}**」，当前库存：${updated.quantity}${updated.unit}${updated.location ? `，存放在【${updated.location}】` : ''}`;
    }

    // Dedup check — skip when force_create is set
    if (!parsed.force_create) {
      const spec = parsed.spec?.trim();
      const { items: similar, total } = await client.findSimilarItems(name, spec);
      if (total > 0) {
        const existing = similar[0];
        const mergedQty = existing.quantity + (parsed.quantity ?? 1);
        return buildMergePrompt(existing, parsed, mergedQty);
      }
    }

    const category = parsed.category?.trim();
    const categoryRecord = category ? await client.ensureCategoryByName(category) : null;
    const expireAt = parsed.expire_at ? normalizeDateInput(parsed.expire_at) : undefined;

    const item = await client.addItem({
      name,
      quantity: parsed.quantity,
      unit: parsed.unit?.trim() || undefined,
      spec: parsed.spec?.trim() || undefined,
      location: parsed.location?.trim() || undefined,
      category_id: categoryRecord?.id,
      expire_at: expireAt,
      notes: parsed.notes?.trim() ?? undefined,
    });

    return buildSuccessMessage(item);
  } catch (err) {
    return `❌ 添加失败：${(err as Error).message}`;
  }
}
