import { InventoryAPIClient } from '../utils/apiClient';

export async function queryItems(
  params: {
    location?: string;
    category?: string;
    expiring_soon?: boolean;
  },
  client: InventoryAPIClient
): Promise<string> {
  try {
    const { items, total } = await client.listItems(params);

    if (total === 0) {
      const scope = params.location ? `【${params.location}】中` : '家里';
      return `📦 ${scope}目前没有库存记录。`;
    }

    const title = params.expiring_soon
      ? `⚠️ 即将过期的物料（共 ${total} 项）：\n`
      : params.location
      ? `📦 【${params.location}】库存（共 ${total} 项）：\n`
      : `📦 全部库存（共 ${total} 项）：\n`;

    const lines = items.map(item => {
      let line = `• ${item.name} ${item.quantity}${item.unit}`;
      if (item.location && !params.location) line += `（${item.location}）`;
      if (item.expire_at) {
        const expireDate = new Date(item.expire_at);
        const daysLeft = Math.ceil((expireDate.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
        if (daysLeft <= 0) {
          line += ` 🔴已过期`;
        } else if (daysLeft <= 3) {
          line += ` 🟡还剩${daysLeft}天`;
        } else {
          line += ` 📅${expireDate.toLocaleDateString('zh-CN')}到期`;
        }
      }
      return line;
    });

    return title + lines.join('\n');
  } catch (err) {
    return `❌ 查询失败：${(err as Error).message}`;
  }
}
