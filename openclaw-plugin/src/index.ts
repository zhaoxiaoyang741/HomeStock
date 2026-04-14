import { Type } from '@sinclair/typebox';
import { definePluginEntry } from 'openclaw/plugin-sdk/plugin-entry';

import { checkHomeStockService } from './tools/checkHomeStockService.js';
import { consumeMaterial } from './tools/consumeMaterial.js';
import { inboundStock } from './tools/inboundStock.js';
import { queryInventory } from './tools/queryInventory.js';
import { updateStockLot } from './tools/updateStockLot.js';
import { createClient, resolveBaseUrl, textResult } from './utils/runtime.js';

export default definePluginEntry({
  id: 'home-inventory',
  name: 'Home Inventory',
  description: '家庭物料管理系统，通过自然语言管理家中食材和日用品。',
  register(api) {
    const client = createClient(resolveBaseUrl(api.pluginConfig), { channel: 'feishu' });

    api.registerTool({
      name: 'inbound_stock',
      description: '新增一条入库批次，可填写物料名、规格、位置、分类、过期日期和备注。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称，如：土豆、酱油、洗洁精' }),
        quantity: Type.Optional(Type.Number({ description: '数量，默认 1' })),
        unit: Type.Optional(Type.String({ description: '单位，如：个、克、瓶、袋' })),
        spec: Type.Optional(Type.String({ description: '规格，如：500ml、1kg、大瓶装' })),
        location: Type.Optional(Type.String({ description: '存放位置，如：冰箱、厨房、储物间' })),
        category: Type.Optional(Type.String({ description: '分类名称，如：蔬菜、调料、日用品' })),
        expire_at: Type.Optional(Type.String({ description: '过期日期，格式 YYYY-MM-DD' })),
        notes: Type.Optional(Type.String({ description: '备注信息' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await inboundStock(params, client));
      },
    });

    api.registerTool({
      name: 'consume_material',
      description: '按物料消耗库存，后端会按批次自动分配扣减。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称' }),
        quantity: Type.Optional(Type.Number({ description: '消耗数量，不填则默认消耗当前全部库存' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await consumeMaterial(params, client));
      },
    });

    api.registerTool({
      name: 'query_inventory',
      description:
        '查询当前库存汇总或临期批次，适用于“目前冰箱里有什么”“现在还有什么调料”“还有没有土豆”“回复某人：冰箱里还有什么”“查一下快过期的”这类问题。',
      parameters: Type.Object({
        location: Type.Optional(Type.String({ description: '存放位置，如：冰箱、厨房' })),
        category: Type.Optional(Type.String({ description: '分类名称，如：蔬菜、调料' })),
        expiring_soon: Type.Optional(Type.Boolean({ description: '是否只看 7 天内到期的批次' })),
        keyword: Type.Optional(Type.String({ description: '按名称或规格关键字搜索' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await queryInventory(params, client));
      },
    });

    api.registerTool({
      name: 'check_homestock_service',
      description: '检查当前 HomeStock 后端服务是否可用，并返回插件实际使用的后端地址。',
      parameters: Type.Object({}),
      async execute() {
        return textResult(await checkHomeStockService(client));
      },
    });

    api.registerTool({
      name: 'update_stock_lot',
      description: '更新单个批次的库存或元数据；若同一物料有多个批次，会提示先明确批次。',
      parameters: Type.Object({
        name: Type.String({ description: '物料名称，用于查找对应批次' }),
        quantity: Type.Optional(Type.Number({ description: '新的批次库存' })),
        expire_at: Type.Optional(Type.String({ description: '新的过期日期，格式 YYYY-MM-DD；传空字符串表示清空' })),
        notes: Type.Optional(Type.String({ description: '新的备注；传空字符串表示清空' })),
        location: Type.Optional(Type.String({ description: '新的存放位置；传空字符串表示清空' })),
      }),
      async execute(_toolCallId, params) {
        return textResult(await updateStockLot(params, client));
      },
    });
  },
});
