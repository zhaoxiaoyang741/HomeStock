import test from 'node:test';
import assert from 'node:assert/strict';

import { checkHomeStockService } from '../src/tools/checkHomeStockService.js';
import { consumeMaterial } from '../src/tools/consumeMaterial.js';
import { inboundStock } from '../src/tools/inboundStock.js';
import { queryInventory } from '../src/tools/queryInventory.js';
import { updateStockLot } from '../src/tools/updateStockLot.js';
import type {
  Category,
  HealthCheckResult,
  InventoryClient,
  MaterialSummary,
  StockLot,
} from '../src/utils/apiClient.js';

function createMaterial(overrides: Partial<MaterialSummary> = {}): MaterialSummary {
  return {
    id: 'mat-1',
    tenant_id: 'default',
    name: '土豆',
    spec: '500g',
    category_id: 'cat-1',
    category: { id: 'cat-1', tenant_id: 'default', name: '蔬菜' },
    default_unit: '袋',
    status: 'active',
    total_quantity: 5,
    lot_count: 2,
    nearest_expire_at: '2026-04-30T15:59:59.999Z',
    locations: ['冰箱'],
    ...overrides,
  };
}

function createLot(overrides: Partial<StockLot> = {}): StockLot {
  return {
    id: 'lot-1',
    tenant_id: 'default',
    material_id: 'mat-1',
    material: createMaterial() as unknown as never,
    quantity_on_hand: 5,
    unit: '袋',
    location: '冰箱',
    expire_at: '2026-04-30T15:59:59.999Z',
    purchased_at: '2026-04-10T00:00:00.000Z',
    received_at: '2026-04-10T00:00:00.000Z',
    notes: '',
    status: 'active',
    ...overrides,
  };
}

function createHealthCheckResult(overrides: Partial<HealthCheckResult> = {}): HealthCheckResult {
  return {
    ok: true,
    baseUrl: 'http://inventory.example',
    status: 'ok',
    ...overrides,
  };
}

function createClient(overrides: Partial<InventoryClient> = {}): InventoryClient {
  const categories: Category[] = [
    { id: 'cat-1', tenant_id: 'default', name: '蔬菜' },
    { id: 'cat-2', tenant_id: 'default', name: '调料' },
  ];

  return {
    async inboundStockLot(params) {
      return createLot({
        material: createMaterial({
          name: params.name ?? '土豆',
          spec: params.spec ?? '500g',
          category_id: params.category_id ?? 'cat-1',
          category: params.category_id ? { id: params.category_id, tenant_id: 'default', name: '蔬菜' } : null,
          default_unit: params.unit ?? '袋',
        }) as unknown as never,
        quantity_on_hand: params.quantity,
        unit: params.unit ?? '袋',
        location: params.location ?? '',
        expire_at: params.expire_at ?? null,
        notes: params.notes ?? '',
      });
    },
    async listMaterials() {
      return {
        materials: [
          createMaterial(),
          createMaterial({
            id: 'mat-2',
            name: '酱油',
            spec: '500ml',
            category_id: 'cat-2',
            category: { id: 'cat-2', tenant_id: 'default', name: '调料' },
            default_unit: '瓶',
            total_quantity: 2,
            lot_count: 1,
            locations: ['厨房'],
            nearest_expire_at: '2026-04-08T15:59:59.999Z',
          }),
        ],
        total: 2,
      };
    },
    async getMaterial(id) {
      return createMaterial({ id });
    },
    async consumeMaterial(id, params) {
      return {
        material: createMaterial({ id, total_quantity: Math.max(0, 5 - params.quantity) }),
        requested_quantity: params.quantity,
        unit: '袋',
        consumed_lots: [
          {
            lot_id: 'lot-1',
            consumed_quantity: params.quantity,
            remaining_quantity: Math.max(0, 5 - params.quantity),
            location: '冰箱',
            expire_at: '2026-04-30T15:59:59.999Z',
          },
        ],
      };
    },
    async listStockLots() {
      return {
        lots: [
          createLot(),
          createLot({
            id: 'lot-2',
            material_id: 'mat-2',
            material: createMaterial({
              id: 'mat-2',
              name: '酱油',
              spec: '500ml',
              category_id: 'cat-2',
              category: { id: 'cat-2', tenant_id: 'default', name: '调料' },
              default_unit: '瓶',
              total_quantity: 2,
              lot_count: 1,
              locations: ['厨房'],
              nearest_expire_at: '2026-04-08T15:59:59.999Z',
            }) as unknown as never,
            quantity_on_hand: 2,
            unit: '瓶',
            location: '厨房',
            expire_at: '2026-04-08T15:59:59.999Z',
          }),
        ],
        total: 2,
      };
    },
    async updateStockLot(id, params) {
      return createLot({
        id,
        expire_at: params.expire_at === undefined ? '2026-04-30T15:59:59.999Z' : params.expire_at || null,
        notes: params.notes ?? '',
        location: params.location ?? '冰箱',
      });
    },
    async adjustStockLot(id, params) {
      return createLot({
        id,
        quantity_on_hand: params.target_quantity,
      });
    },
    async findMaterialByName(name) {
      return name.includes('土豆') ? createMaterial() : null;
    },
    async findLotsByMaterialName(name) {
      return name.includes('土豆') ? [createLot()] : [];
    },
    async findCategoryByName(name) {
      return categories.find((category) => category.name === name) ?? null;
    },
    async ensureCategoryByName(name) {
      const existing = categories.find((category) => category.name === name);
      if (existing) {
        return existing;
      }
      const created = { id: `cat-${categories.length + 1}`, tenant_id: 'default', name };
      categories.push(created);
      return created;
    },
    async checkHealth() {
      return createHealthCheckResult();
    },
    ...overrides,
  };
}

test('inboundStock auto-creates categories and normalizes date input', async () => {
  let ensuredCategory = '';
  let recordedExpireAt = '';

  const client = createClient({
    async ensureCategoryByName(name) {
      ensuredCategory = name;
      return { id: 'cat-new', tenant_id: 'default', name };
    },
    async inboundStockLot(params) {
      recordedExpireAt = params.expire_at ?? '';
      return createLot({
        material: createMaterial({
          name: params.name ?? '酱油',
          category_id: params.category_id ?? 'cat-new',
          category: { id: params.category_id ?? 'cat-new', tenant_id: 'default', name: '调料' },
        }) as unknown as never,
        expire_at: params.expire_at ?? null,
      });
    },
  });

  const result = await inboundStock(
    { name: '酱油', category: '调料', expire_at: '2026-04-30' },
    client,
  );

  assert.equal(ensuredCategory, '调料');
  assert.match(recordedExpireAt, /^2026-04-30T/);
  assert.match(result, /已入库/);
  assert.doesNotMatch(result, /force_create|merge_with_id|相似物料/);
});

test('queryInventory returns expiring lots when requested', async () => {
  const client = createClient();
  const result = await queryInventory(
    { category: '调料', keyword: '酱', expiring_soon: true },
    client,
  );

  assert.match(result, /临期批次/);
  assert.match(result, /酱油/);
});

test('queryInventory falls back to health check when inventory query fails', async () => {
  const client = createClient({
    async listMaterials() {
      throw new Error('inventory query failed');
    },
    async checkHealth() {
      return createHealthCheckResult({ ok: true, baseUrl: 'http://192.168.18.13:8888' });
    },
  });

  const result = await queryInventory({ location: '冰箱' }, client);

  assert.match(result, /HomeStock 服务在线/);
  assert.match(result, /192\.168\.18\.13:8888/);
});

test('consumeMaterial consumes material inventory', async () => {
  let consumedQuantity = 0;
  const client = createClient({
    async consumeMaterial(id, params) {
      consumedQuantity = params.quantity;
      return {
        material: createMaterial({ id }),
        requested_quantity: params.quantity,
        unit: '袋',
        consumed_lots: [
          {
            lot_id: 'lot-1',
            consumed_quantity: params.quantity,
            remaining_quantity: 3,
            location: '冰箱',
            expire_at: '2026-04-30T15:59:59.999Z',
          },
        ],
      };
    },
  });

  const result = await consumeMaterial({ name: '土豆', quantity: 2 }, client);
  assert.equal(consumedQuantity, 2);
  assert.match(result, /已消耗/);
  assert.match(result, /按批次自动扣减/);
});

test('updateStockLot rejects empty updates', async () => {
  const client = createClient();
  const result = await updateStockLot({ name: '土豆' }, client);
  assert.match(result, /至少提供一个要更新的字段/);
});

test('updateStockLot adjusts quantity through adjustStockLot', async () => {
  let targetQuantity = -1;
  const client = createClient({
    async adjustStockLot(id, params) {
      targetQuantity = params.target_quantity;
      return createLot({ id, quantity_on_hand: params.target_quantity });
    },
  });

  const result = await updateStockLot({ name: '土豆', quantity: 9 }, client);
  assert.equal(targetQuantity, 9);
  assert.match(result, /已调整批次/);
});

test('updateStockLot prompts when multiple lots exist', async () => {
  const client = createClient({
    async findLotsByMaterialName(name) {
      return name.includes('土豆')
        ? [createLot({ id: 'lot-1' }), createLot({ id: 'lot-2', quantity_on_hand: 2, location: '厨房' })]
        : [];
    },
  });

  const result = await updateStockLot({ name: '土豆', location: '厨房' }, client);
  assert.match(result, /存在多个批次/);
  assert.match(result, /请先在 Web 中选择具体批次/);
});

test('checkHomeStockService reports configured backend availability', async () => {
  const client = createClient({
    async checkHealth() {
      return createHealthCheckResult({ ok: false, baseUrl: 'http://192.168.18.13:8888', error: 'connect timeout' });
    },
  });

  const result = await checkHomeStockService(client);
  assert.match(result, /HomeStock 服务不可用/);
  assert.match(result, /192\.168\.18\.13:8888/);
});
