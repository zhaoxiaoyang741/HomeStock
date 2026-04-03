import { InventoryAPIClient } from '../utils/apiClient';

export async function addItem(
  params: {
    name: string;
    quantity?: number;
    unit?: string;
    location?: string;
    notes?: string;
  },
  client: InventoryAPIClient
): Promise<string> {
  try {
    const item = await client.addItem(params);

    let response = `✅ 已添加 **${item.name}** ${item.quantity}${item.unit}`;
    if (item.location) response += `，存放在【${item.location}】`;
    if (item.expire_at) {
      const expireDate = new Date(item.expire_at);
      const daysLeft = Math.ceil((expireDate.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
      response += `\n📅 预计到期：${expireDate.toLocaleDateString('zh-CN')}（还剩 ${daysLeft} 天）`;
    }
    return response;
  } catch (err) {
    return `❌ 添加失败：${(err as Error).message}`;
  }
}
