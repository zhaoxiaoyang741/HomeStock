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

    const material = await client.findMaterialByName(name);
    if (!material) {
      return `❓ 未找到「${name}」，请先查询库存确认名称。`;
    }

    const quantity = parsed.quantity ?? material.total_quantity;
    const result = await client.consumeMaterial(material.id, {
      quantity,
      reason: 'manual consume',
    });

    const lotSummary = result.consumed_lots
      .map((lot) => `• ${lot.lot_id} 消耗 ${lot.consumed_quantity}${result.unit}，剩余 ${lot.remaining_quantity}${result.unit}`)
      .join('\n');

    return `✅ 已消耗 **${material.name}** ${quantity}${result.unit}\n${lotSummary}`;
  } catch (err) {
    return `❌ 操作失败：${(err as Error).message}`;
  }
}
