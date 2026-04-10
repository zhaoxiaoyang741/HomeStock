import { InventoryClient, StockLot } from '../utils/apiClient.js';
import {
  coerceNumericValue,
  coerceStringValue,
  formatLocalDate,
  isRawCommandEnvelope,
  normalizeDateInput,
  parseSlashCommandArgs,
} from '../utils/runtime.js';

export interface UpdateItemParams {
  name: string;
  quantity?: number;
  expire_at?: string;
  notes?: string;
  location?: string;
}

export function parseUpdateItemParams(input: UpdateItemParams | unknown): UpdateItemParams {
  if (!isRawCommandEnvelope(input)) {
    const record = (input ?? {}) as Record<string, unknown>;
    return {
      name: coerceStringValue(record.name) ?? '',
      quantity: coerceNumericValue(record.quantity),
      expire_at:
        coerceStringValue(record.expire_at) ??
        coerceStringValue(record.expiry_date) ??
        coerceStringValue(record.expires_at),
      notes: coerceStringValue(record.notes),
      location: coerceStringValue(record.location),
    };
  }

  const tokens = parseSlashCommandArgs(input.command ?? '');
  if (tokens.length === 0) {
    return { name: '' };
  }

  const [name, ...rest] = tokens;
  const result: UpdateItemParams = { name };

  for (const token of rest) {
    const quantityMatch = token.match(/^(\d+(?:\.\d+)?)([\u4e00-\u9fa5a-zA-Z]+)?$/);
    if (quantityMatch && result.quantity === undefined) {
      result.quantity = Number(quantityMatch[1]);
      continue;
    }

    if (/^\d{4}-\d{2}-\d{2}$/.test(token) && result.expire_at === undefined) {
      result.expire_at = token;
      continue;
    }

    if (result.location === undefined) {
      result.location = token;
      continue;
    }

    result.notes = result.notes ? `${result.notes} ${token}` : token;
  }

  return result;
}

function lotLabel(lot: StockLot): string {
  const parts = [lot.material?.name ?? '物料'];
  if (lot.material?.spec) {
    parts.push(lot.material.spec);
  }
  parts.push(`#${lot.id.slice(0, 8)}`);
  return parts.join(' / ');
}

export async function updateItem(
  params: UpdateItemParams | unknown,
  client: InventoryClient
): Promise<string> {
  try {
    const parsed = parseUpdateItemParams(params);
    const name = parsed.name?.trim();
    if (!name) {
      return '❌ 缺少物料名称。';
    }
    if (parsed.quantity !== undefined && parsed.quantity < 0) {
      return '❌ 新数量不能小于 0。';
    }

    const hasUpdates =
      parsed.quantity !== undefined ||
      parsed.expire_at !== undefined ||
      parsed.notes !== undefined ||
      parsed.location !== undefined;

    if (!hasUpdates) {
      return '❌ 至少提供一个要更新的字段：quantity、expire_at、notes 或 location。';
    }

    const lots = await client.findLotsByMaterialName(name);
    if (lots.length === 0) {
      return `❓ 未找到「${name}」，请先查询库存确认名称。`;
    }
    if (lots.length > 1) {
      const choices = lots
        .map((lot) => `• ${lotLabel(lot)}，库存 ${lot.quantity_on_hand}${lot.unit}${lot.location ? `，位置 ${lot.location}` : ''}`)
        .join('\n');
      return `⚠️ 「${name}」存在多个批次，暂不自动选择。\n${choices}\n请先在 Web 中选择具体批次后再修改。`;
    }

    const lot = lots[0];
    if (parsed.quantity !== undefined) {
      const adjusted = await client.adjustStockLot(lot.id, {
        target_quantity: parsed.quantity,
        reason: 'manual adjust',
        remark: parsed.notes?.trim() || undefined,
      });
      return `✅ 已调整 **${lotLabel(adjusted)}** 的库存为 ${adjusted.quantity_on_hand}${adjusted.unit}`;
    }

    const updated = await client.updateStockLot(lot.id, {
      expire_at:
        parsed.expire_at === undefined
          ? undefined
          : parsed.expire_at.trim() === ''
          ? ''
          : normalizeDateInput(parsed.expire_at),
      notes: parsed.notes === undefined ? undefined : parsed.notes.trim(),
      location: parsed.location === undefined ? undefined : parsed.location.trim(),
    });

    const changes: string[] = [];
    if (parsed.location !== undefined) {
      changes.push(`位置：${updated.location || '已清空'}`);
    }
    if (parsed.expire_at !== undefined) {
      changes.push(`过期日期：${updated.expire_at ? formatLocalDate(updated.expire_at) : '已清空'}`);
    }
    if (parsed.notes !== undefined) {
      changes.push(`备注：${updated.notes || '已清空'}`);
    }

    return `✅ 已更新 **${lotLabel(updated)}**\n${changes.map(change => `• ${change}`).join('\n')}`;
  } catch (err) {
    return `❌ 更新失败：${(err as Error).message}`;
  }
}
