import { InventoryClient } from '../utils/apiClient.js';

export async function checkHomeStockService(client: InventoryClient): Promise<string> {
  const health = await client.checkHealth();

  if (health.ok) {
    return `HomeStock 服务正常，当前地址为 ${health.baseUrl}。`;
  }

  return `HomeStock 服务不可用，当前地址为 ${health.baseUrl}。健康检查失败：${health.error ?? '未知错误'}。`;
}
