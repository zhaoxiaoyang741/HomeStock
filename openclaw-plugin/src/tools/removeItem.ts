import { InventoryClient } from '../utils/apiClient.js';
import { coerceNumericValue, coerceStringValue, isRawCommandEnvelope, parseSlashCommandArgs } from '../utils/runtime.js';

export interface RemoveItemParams {
  name: string;
  quantity?: number;
}

export function parseRemoveItemParams(input: RemoveItemParams | unknown): RemoveItemParams {
  if (!isRawCommandEnvelope(input)) {
    const record = (input ?? {}) as Record<string, unknown>;
    return {
      name: coerceStringValue(record.name) ?? '',
      quantity: coerceNumericValue(record.quantity),
    };
  }

  const tokens = parseSlashCommandArgs(input.command ?? '');
  if (tokens.length === 0) {
    return { name: '' };
  }

  const lastToken = tokens[tokens.length - 1];
  if (/^\d+(?:\.\d+)?$/.test(lastToken) && tokens.length > 1) {
    return {
      name: tokens.slice(0, -1).join(' '),
      quantity: Number(lastToken),
    };
  }

  return { name: tokens.join(' ') };
}

export async function removeItem(
  params: RemoveItemParams | unknown,
  client: InventoryClient
): Promise<string> {
  try {
    const parsed = parseRemoveItemParams(params);
    const name = parsed.name?.trim();
    if (!name) {
      return '❌ 缺少物料名称。';
    }
    if (parsed.quantity !== undefined && parsed.quantity <= 0) {
      return '❌ 消耗数量必须大于 0。';
    }

    const item = await client.findItemByName(name);
    if (!item) {
      return `❓ 未找到「${name}」，请先查询库存确认名称。`;
    }

    const removeQuantity = parsed.quantity ?? item.quantity;

    if (removeQuantity >= item.quantity) {
      await client.deleteItem(item.id);
      return `✅ 已从库存中移除 **${item.name}**（${item.location || '未指定位置'}）`;
    }

    const updated = await client.updateItem(item.id, {
      quantity: item.quantity - removeQuantity,
    });
    return `✅ 已消耗 **${item.name}** ${removeQuantity}${item.unit}，剩余 ${updated.quantity}${updated.unit}`;
  } catch (err) {
    return `❌ 操作失败：${(err as Error).message}`;
  }
}
