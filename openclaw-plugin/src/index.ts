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
      description: '向库存中添加一个物料（食材或日用品）。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称，如：土豆、酱油、洗洁精' }),
        quantity: Type.Optional(Type.Number({ description: '数量，默认 1' })),
        unit: Type.Optional(Type.String({ description: '单位，如：个、克、瓶、袋' })),
        spec: Type.Optional(Type.String({ description: '规格，如：500ml、1kg、大瓶装' })),
        location: Type.Optional(Type.String({ description: '存放位置，如：冰箱、厨柜、冷冻层' })),
        category: Type.Optional(Type.String({ description: '分类名称，如：蔬菜、水果、调料、日用品' })),
        expire_at: Type.Optional(Type.String({ description: '过期日期，格式 YYYY-MM-DD' })),
        notes: Type.Optional(Type.String({ description: '备注信息' })),
        force_create: Type.Optional(Type.Boolean({ description: '为 true 时跳过重复检查，直接创建新条目' })),
        merge_with_id: Type.Optional(Type.String({ description: '合并时填入已有物料的 ID，quantity 须填合并后总量；设置后直接更新该物料数量，不新建记录' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await addItem(params, client));
      },
    });

    api.registerTool({
      name: 'remove_item',
      description: '从库存中消耗或删除物料。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称' }),
        quantity: Type.Optional(Type.Number({ description: '消耗数量，不填则全部删除' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await removeItem(params, client));
      },
    });

    api.registerTool({
      name: 'query_items',
      description: '查询家中库存物料，可按位置、分类、关键字和临期状态过滤。',
      parameters: Type.Object({
        location: Type.Optional(Type.String({ description: '存放位置，如：冰箱、厨柜' })),
        category: Type.Optional(Type.String({ description: '分类名称，如：蔬菜、调料' })),
        expiring_soon: Type.Optional(Type.Boolean({ description: '是否只看 7 天内到期的物料' })),
        keyword: Type.Optional(Type.String({ description: '按名称关键字搜索' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await queryItems(params, client));
      },
    });

    api.registerTool({
      name: 'update_item',
      description: '修改已有物料的数量、过期时间、备注或存放位置。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称（用于查找）' }),
        quantity: Type.Optional(Type.Number({ description: '新数量' })),
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
