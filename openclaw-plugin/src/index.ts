import { Type } from '@sinclair/typebox';
import { definePluginEntry } from 'openclaw/plugin-sdk/plugin-entry';

import { addItem } from './tools/addItem.js';
import { queryItems } from './tools/queryItems.js';
import { removeItem } from './tools/removeItem.js';
import { updateItem } from './tools/updateItem.js';
import { createClient, resolveBaseUrl, textResult } from './utils/runtime.js';

export default definePluginEntry({
  id: 'home-inventory',
  name: 'Home Inventory',
  description: '家用物料管理系统，通过自然语言管理家中食材和日用品。',
  register(api) {
    const client = createClient(resolveBaseUrl(api.pluginConfig), { channel: 'feishu' });

    api.registerTool({
      name: 'add_item',
      description: '向库存中新增一个入库批次。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称，如：土豆、酱油、洗洁精' }),
        quantity: Type.Optional(Type.Number({ description: '数量，默认 1' })),
        unit: Type.Optional(Type.String({ description: '单位，如：个、克、瓶、袋' })),
        spec: Type.Optional(Type.String({ description: '规格，如：500ml、1kg、大瓶装' })),
        location: Type.Optional(Type.String({ description: '存放位置，如：冰箱、厨房、储物间' })),
        category: Type.Optional(Type.String({ description: '分类名称，如：蔬菜、水果、调料、日用品' })),
        expire_at: Type.Optional(Type.String({ description: '过期日期，格式 YYYY-MM-DD' })),
        notes: Type.Optional(Type.String({ description: '备注信息' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await addItem(params, client));
      },
    });

    api.registerTool({
      name: 'remove_item',
      description: '从物料总库存中消耗指定数量，后端会按先过期先出自动分配批次。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称' }),
        quantity: Type.Optional(Type.Number({ description: '消耗数量，不填则默认消耗全部库存' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await removeItem(params, client));
      },
    });

    api.registerTool({
      name: 'query_items',
      description: '查询物料汇总或临期批次，可按位置、分类、关键字过滤。',
      parameters: Type.Object({
        location: Type.Optional(Type.String({ description: '存放位置，如：冰箱、厨房' })),
        category: Type.Optional(Type.String({ description: '分类名称，如：蔬菜、调料' })),
        expiring_soon: Type.Optional(Type.Boolean({ description: '是否只看 7 天内到期的批次' })),
        keyword: Type.Optional(Type.String({ description: '按名称或规格关键字搜索' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await queryItems(params, client));
      },
    });

    api.registerTool({
      name: 'update_item',
      description: '更新单个批次的信息；若同一物料存在多个批次，工具会提示用户到 Web 中选择批次。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称（用于查找）' }),
        quantity: Type.Optional(Type.Number({ description: '新的批次库存' })),
        expire_at: Type.Optional(Type.String({ description: '新过期日期，格式 YYYY-MM-DD；传空字符串表示清空' })),
        notes: Type.Optional(Type.String({ description: '新备注；传空字符串表示清空' })),
        location: Type.Optional(Type.String({ description: '新存放位置；传空字符串表示清空' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await updateItem(params, client));
      },
    });
  },
});
