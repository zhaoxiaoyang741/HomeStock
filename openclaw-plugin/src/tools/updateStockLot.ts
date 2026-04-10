import { InventoryClient, StockLot } from '../utils/apiClient.js';
import {
  coerceNumericValue,
  coerceStringValue,
  formatLocalDate,
  isRawCommandEnvelope,
  normalizeDateInput,
  parseSlashCommandArgs,
} from '../utils/runtime.js';

export interface UpdateStockLotParams {
  name: string;
  quantity?: number;
  expire_at?: string;
  notes?: string;
  location?: string;
}

export function parseUpdateStockLotParams(input: UpdateStockLotParams | unknown): UpdateStockLotParams {
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
  const result: UpdateStockLotParams = { name };

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
  if (lot.location) {
    parts.push(lot.location);
  }
  return parts.join(' / ');
}

export async function updateStockLot(
  params: UpdateStockLotParams | unknown,
  client: InventoryClient,
): Promise<string> {
  try {
    const parsed = parseUpdateStockLotParams(params);
    const name = parsed.name?.trim();
    if (!name) {
      return '缺少物料名称。';
    }
    if (parsed.quantity !== undefined && parsed.quantity < 0) {
      return '新的库存数量不能小于 0。';
    }

    const hasUpdates =
      parsed.quantity !== undefined ||
      parsed.expire_at !== undefined ||
      parsed.notes !== undefined ||
      parsed.location !== undefined;
    if (!hasUpdates) {
      return '至少提供一个要更新的字段：quantity、expire_at、notes 或 location。';
    }

    const lots = await client.findLotsByMaterialName(name);
    if (lots.length === 0) {
      return `未找到“${name}”，请先查询库存确认物料名称。`;
    }
    if (lots.length > 1) {
      const choices = lots
        .map((lot) => {
          const locationText = lot.location ? `，位置：${lot.location}` : '';
          const expireText = lot.expire_at ? `，到期：${formatLocalDate(lot.expire_at)}` : '';
          return `• ${lot.material?.name ?? name}${lot.material?.spec ? ` / ${lot.material.spec}` : ''}，库存：${lot.quantity_on_hand}${lot.unit}${locationText}${expireText}`;
        })
        .join('\n');
      return `“${name}”当前存在多个批次，暂不自动猜测。\n${choices}\n请先在 Web 中选择具体批次后再修改。`;
    }

    let updatedLot = lots[0];
    const changes: string[] = [];

    if (parsed.quantity !== undefined) {
      updatedLot = await client.adjustStockLot(updatedLot.id, {
        target_quantity: parsed.quantity,
        reason: 'manual adjust',
        remark: parsed.notes?.trim() || undefined,
      });
      changes.push(`库存：${updatedLot.quantity_on_hand}${updatedLot.unit}`);
    }

    const hasMetadataUpdates =
      parsed.expire_at !== undefined || parsed.notes !== undefined || parsed.location !== undefined;
    if (hasMetadataUpdates) {
      updatedLot = await client.updateStockLot(updatedLot.id, {
        expire_at:
          parsed.expire_at === undefined
            ? undefined
            : parsed.expire_at.trim() === ''
            ? ''
            : normalizeDateInput(parsed.expire_at),
        notes: parsed.notes === undefined ? undefined : parsed.notes.trim(),
        location: parsed.location === undefined ? undefined : parsed.location.trim(),
      });

      if (parsed.location !== undefined) {
        changes.push(`位置：${updatedLot.location || '已清空'}`);
      }
      if (parsed.expire_at !== undefined) {
        changes.push(`过期日期：${updatedLot.expire_at ? formatLocalDate(updatedLot.expire_at) : '已清空'}`);
      }
      if (parsed.notes !== undefined) {
        changes.push(`备注：${updatedLot.notes || '已清空'}`);
      }
    }

    if (parsed.quantity !== undefined && !hasMetadataUpdates) {
      return `已调整批次 **${lotLabel(updatedLot)}** 的库存为 ${updatedLot.quantity_on_hand}${updatedLot.unit}`;
    }

    return `已更新批次 **${lotLabel(updatedLot)}**\n${changes.map((change) => `• ${change}`).join('\n')}`;
  } catch (err) {
    return `更新批次失败：${(err as Error).message}`;
  }
}
