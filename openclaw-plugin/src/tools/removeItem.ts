import { InventoryAPIClient } from '../utils/apiClient';

export async function removeItem(
  params: { name: string; quantity?: number },
  client: InventoryAPIClient
): Promise<string> {
  try {
    const item = await client.findItemByName(params.name);
    if (!item) {
      return `❓ 未找到「${params.name}」，请检查库存中是否有该物料。`;
    }

    const removeQty = params.quantity ?? item.quantity;

    if (removeQty >= item.quantity) {
      await client.deleteItem(item.id);
      return `✅ 已从库存中移除 **${item.name}**（${item.location || '未指定位置'}）`;
    } else {
      const newQty = item.quantity - removeQty;
      await client.updateItem(item.id, { quantity: newQty });
      return `✅ 已消耗 **${item.name}** ${removeQty}${item.unit}，剩余 ${newQty}${item.unit}`;
    }
  } catch (err) {
    return `❌ 操作失败：${(err as Error).message}`;
  }
}
