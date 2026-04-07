import path from 'path';
import fs from 'fs';
import { createRequire } from 'module';
import { InventoryAPIClient } from './utils/apiClient';
import { addItem } from './tools/addItem';
import { removeItem } from './tools/removeItem';
import { queryItems } from './tools/queryItems';

// 通过 process.execPath 推导 openclaw 安装路径，无需 shell 命令
// node 二进制: .nvm/versions/node/vX.Y.Z/bin/node
// openclaw:   .nvm/versions/node/vX.Y.Z/lib/node_modules/openclaw
function loadOpenclawSdk() {
  const candidates = [
    path.resolve(path.dirname(process.execPath), '..', 'lib', 'node_modules', 'openclaw', 'package.json'),
    '/usr/local/lib/node_modules/openclaw/package.json',
    '/usr/lib/node_modules/openclaw/package.json',
  ];

  for (const openclawPkg of candidates) {
    if (fs.existsSync(openclawPkg)) {
      const req = createRequire(openclawPkg);
      return {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        definePluginEntry: (req('openclaw/plugin-sdk/plugin-entry') as any).definePluginEntry,
      };
    }
  }
  throw new Error(`openclaw not found, searched: ${candidates.join(', ')}`);
}

const { definePluginEntry } = loadOpenclawSdk();

function textResult(text: string) {
  return { content: [{ type: 'text' as const, text }], details: text };
}

// 解析斜杠命令原始文本 → add_item 参数
// 格式：<名称> [数量+单位] [位置]   e.g. "土豆 5个 冰箱"
function parseAddItemCommand(raw: string): { name: string; quantity?: number; unit?: string; location?: string } {
  const tokens = raw.trim().split(/\s+/).filter(Boolean);
  if (tokens.length === 0) return { name: '' };

  const result: { name: string; quantity?: number; unit?: string; location?: string } = { name: '' };
  const nameParts: string[] = [];

  for (const token of tokens) {
    const qtyMatch = token.match(/^(\d+(?:\.\d+)?)(.*)$/);
    if (qtyMatch && result.quantity === undefined) {
      result.quantity = parseFloat(qtyMatch[1]);
      result.unit = qtyMatch[2] || '个';
    } else if (result.quantity !== undefined && result.location === undefined) {
      result.location = token;
    } else {
      nameParts.push(token);
    }
  }
  result.name = nameParts.join('');
  return result;
}

// 解析斜杠命令原始文本 → remove_item 参数
// 格式：<名称> [数量]   e.g. "土豆 2"
function parseRemoveItemCommand(raw: string): { name: string; quantity?: number } {
  const tokens = raw.trim().split(/\s+/).filter(Boolean);
  if (tokens.length === 0) return { name: '' };

  const last = tokens[tokens.length - 1];
  const numMatch = last.match(/^(\d+(?:\.\d+)?)$/);
  if (numMatch && tokens.length > 1) {
    return { name: tokens.slice(0, -1).join(''), quantity: parseFloat(numMatch[1]) };
  }
  return { name: tokens.join('') };
}

// 解析斜杠命令原始文本 → query_items 参数
// 格式：[位置] [分类] [expiring]   e.g. "冰箱 蔬菜" 或 "expiring"
function parseQueryItemsCommand(raw: string): { location?: string; category?: string; expiring_soon?: boolean } {
  const tokens = raw.trim().split(/\s+/).filter(Boolean);
  const result: { location?: string; category?: string; expiring_soon?: boolean } = {};
  const remaining = tokens.filter(t => {
    if (t === 'expiring' || t === '过期' || t === '快过期') { result.expiring_soon = true; return false; }
    return true;
  });
  if (remaining[0]) result.location = remaining[0];
  if (remaining[1]) result.category = remaining[1];
  return result;
}

module.exports = definePluginEntry({
  id: 'home-inventory',
  name: 'home-inventory',
  description: '家用物料管理系统 - 通过自然语言管理家中食材和日用品',
  register(api: any) {
    const baseUrl =
      (api.pluginConfig?.baseUrl as string | undefined) ??
      process.env.INVENTORY_API_URL ??
      'http://localhost:8080';
    const client = new InventoryAPIClient(baseUrl);

    api.registerTool({
      name: 'add_item',
      label: '添加物料',
      description: '向库存中添加一个物料（食材/日用品）。用户说"帮我添加X"、"买了X"、"放了X个Y"时调用。',
      parameters: {
        type: 'object',
        properties: {
          name:     { type: 'string', description: '物料名称，如：土豆、酱油' },
          quantity: { type: 'number', description: '数量，默认1' },
          unit:     { type: 'string', description: '单位，如：个、瓶、袋，默认个' },
          location: { type: 'string', description: '存放位置，如：冰箱、厨柜' },
        },
        required: ['name'],
      },
      execute: async (_toolCallId: string, params: any) => {
        const p = params.command !== undefined
          ? parseAddItemCommand(params.command)
          : params;
        return textResult(await addItem(p, client));
      },
    });

    api.registerTool({
      name: 'remove_item',
      label: '消耗物料',
      description: '从库存中消耗或删除一个物料。用户说"用了X"、"吃了X"、"X吃完了"时调用。',
      parameters: {
        type: 'object',
        properties: {
          name:     { type: 'string', description: '物料名称' },
          quantity: { type: 'number', description: '消耗数量，不填则全部删除' },
        },
        required: ['name'],
      },
      execute: async (_toolCallId: string, params: any) => {
        const p = params.command !== undefined
          ? parseRemoveItemCommand(params.command)
          : params;
        return textResult(await removeItem(p, client));
      },
    });

    api.registerTool({
      name: 'query_items',
      label: '查询库存',
      description: '查询家中库存物料。用户说"还有什么"、"冰箱里有什么"、"快过期的"时调用。',
      parameters: {
        type: 'object',
        properties: {
          location:      { type: 'string',  description: '按位置过滤，如：冰箱' },
          category:      { type: 'string',  description: '按分类过滤' },
          expiring_soon: { type: 'boolean', description: '只看即将过期的物料' },
        },
      },
      execute: async (_toolCallId: string, params: any) => {
        const p = params.command !== undefined
          ? parseQueryItemsCommand(params.command)
          : params;
        return textResult(await queryItems(p, client));
      },
    });
  },
});
