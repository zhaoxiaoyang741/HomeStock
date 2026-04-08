import test from 'node:test';
import assert from 'node:assert/strict';

import { addItem } from '../src/tools/addItem.js';
import { queryItems } from '../src/tools/queryItems.js';
import { removeItem } from '../src/tools/removeItem.js';
import { updateItem } from '../src/tools/updateItem.js';
import type { Category, InventoryClient, Item } from '../src/utils/apiClient.js';

function createItem(overrides: Partial<Item> = {}): Item {
  return {
    id: 'item-1',
    tenant_id: 'default',
    name: '土豆',
    category_id: 'cat-1',
    category: { id: 'cat-1', tenant_id: 'default', name: '蔬菜' },
    quantity: 5,
    unit: '个',
    location: '冰箱',
    expire_at: '2026-04-30T15:59:59.999Z',
    notes: '',
    ...overrides,
  };
}

function createClient(overrides: Partial<InventoryClient> = {}): InventoryClient {
  const categories: Category[] = [
    { id: 'cat-1', tenant_id: 'default', name: '蔬菜' },
    { id: 'cat-2', tenant_id: 'default', name: '调料' },
  ];

  return {
    async addItem(params) {
      return createItem({
        name: params.name,
        quantity: params.quantity ?? 1,
        unit: params.unit ?? '个',
        location: params.location ?? '',
        category_id: params.category_id ?? '',
        category: params.category_id ? { id: params.category_id, tenant_id: 'default', name: '蔬菜' } : null,
        expire_at: params.expire_at ?? null,
        notes: params.notes ?? '',
      });
    },
    async listItems() {
      return {
        items: [
          createItem(),
          createItem({
            id: 'item-2',
            name: '酱油',
            category_id: 'cat-2',
            category: { id: 'cat-2', tenant_id: 'default', name: '调料' },
            quantity: 2,
            unit: '瓶',
            location: '厨柜',
            expire_at: '2026-04-08T15:59:59.999Z',
          }),
        ],
        total: 2,
      };
    },
    async searchByKeyword(keyword) {
      return keyword.includes('土') ? [createItem()] : [];
    },
    async updateItem(id, params) {
      return createItem({
        id,
        quantity: params.quantity ?? 5,
        expire_at: params.expire_at === undefined ? '2026-04-30T15:59:59.999Z' : params.expire_at || null,
        notes: params.notes ?? '',
        location: params.location ?? '冰箱',
      });
    },
    async deleteItem() {
      return undefined;
    },
    async findItemByName(name) {
      return name.includes('土豆') ? createItem() : null;
    },
    async findCategoryByName(name) {
      return categories.find(category => category.name === name) ?? null;
    },
    async ensureCategoryByName(name) {
      const existing = categories.find(category => category.name === name);
      if (existing) {
        return existing;
      }

      const created = { id: `cat-${categories.length + 1}`, tenant_id: 'default', name };
      categories.push(created);
      return created;
    },
    ...overrides,
  };
}

test('addItem auto-creates categories and normalizes date input', async () => {
  let ensuredCategory = '';
  let recordedExpireAt = '';

  const client = createClient({
    async ensureCategoryByName(name) {
      ensuredCategory = name;
      return { id: 'cat-new', tenant_id: 'default', name };
    },
    async addItem(params) {
      recordedExpireAt = params.expire_at ?? '';
      return createItem({
        name: params.name,
        category_id: params.category_id ?? '',
        category: { id: params.category_id ?? 'cat-new', tenant_id: 'default', name: '调料' },
        expire_at: params.expire_at ?? null,
      });
    },
  });

  const result = await addItem(
    { name: '酱油', category: '调料', expire_at: '2026-04-30' },
    client
  );

  assert.equal(ensuredCategory, '调料');
  assert.match(recordedExpireAt, /^2026-04-30T/);
  assert.match(result, /分类：调料/);
});

test('addItem accepts model-shaped arguments with quantity strings and expiry_date alias', async () => {
  let payload: Record<string, unknown> | undefined;
  const client = createClient({
    async addItem(params) {
      payload = params as unknown as Record<string, unknown>;
      return createItem({
        name: params.name,
        quantity: params.quantity ?? 1,
        unit: params.unit ?? '个',
        location: params.location ?? '',
        expire_at: params.expire_at ?? null,
        category: { id: 'cat-3', tenant_id: 'default', name: '食材' },
      });
    },
    async ensureCategoryByName(name) {
      return { id: 'cat-3', tenant_id: 'default', name };
    },
  });

  const result = await addItem(
    {
      name: '草莓',
      quantity: '5个',
      location: '冰箱',
      expiry_date: '2026-04-30',
      category: '食材',
    },
    client
  );

  assert.equal(payload?.quantity, 5);
  assert.match(String(payload?.expire_at), /^2026-04-30T/);
  assert.match(result, /已添加/);
});

test('queryItems filters by category, keyword, and expiring soon locally', async () => {
  const client = createClient();
  const result = await queryItems(
    { category: '调料', keyword: '酱', expiring_soon: true },
    client
  );

  assert.match(result, /酱油/);
  assert.doesNotMatch(result, /土豆/);
});

test('removeItem updates quantity for partial consumption', async () => {
  let updatedQuantity = 0;
  const client = createClient({
    async updateItem(id, params) {
      updatedQuantity = params.quantity ?? 0;
      return createItem({ id, quantity: updatedQuantity });
    },
  });

  const result = await removeItem({ name: '土豆', quantity: 2 }, client);
  assert.equal(updatedQuantity, 3);
  assert.match(result, /剩余 3个/);
});

test('updateItem rejects empty updates', async () => {
  const client = createClient();
  const result = await updateItem({ name: '土豆' }, client);
  assert.match(result, /至少提供一个要更新的字段/);
});

test('updateItem clears fields when empty strings are provided', async () => {
  const client = createClient({
    async updateItem(id, params) {
      return createItem({
        id,
        quantity: params.quantity ?? 5,
        expire_at: params.expire_at === '' ? null : params.expire_at ?? null,
        notes: params.notes ?? '',
        location: params.location ?? '',
      });
    },
  });

  const result = await updateItem(
    { name: '土豆', expire_at: '', notes: '', location: '' },
    client
  );

  assert.match(result, /过期日期：已清空/);
  assert.match(result, /备注：已清空/);
  assert.match(result, /位置：已清空/);
});
