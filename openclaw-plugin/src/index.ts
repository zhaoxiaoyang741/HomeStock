import { InventoryAPIClient } from './utils/apiClient';
import { addItem } from './tools/addItem';
import { removeItem } from './tools/removeItem';
import { queryItems } from './tools/queryItems';

const BACKEND_URL = process.env.INVENTORY_API_URL ?? 'http://192.168.18.13:8080';

// OpenClaw 插件接口（@openclaw/plugin-sdk 尚未发布，本地定义）
export abstract class Plugin {
  abstract handleTool(toolName: string, params: Record<string, unknown>): Promise<string>;
}

export default class HomeInventoryPlugin extends Plugin {
  private client: InventoryAPIClient;

  constructor() {
    super();
    this.client = new InventoryAPIClient(BACKEND_URL);
  }

  async handleTool(toolName: string, params: Record<string, unknown>): Promise<string> {
    switch (toolName) {
      case 'add_item':
        return addItem(params as Parameters<typeof addItem>[0], this.client);

      case 'remove_item':
        return removeItem(params as Parameters<typeof removeItem>[0], this.client);

      case 'query_items':
        return queryItems(params as Parameters<typeof queryItems>[0], this.client);

      default:
        return `未知 Tool: ${toolName}`;
    }
  }
}
