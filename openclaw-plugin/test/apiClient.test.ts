import test from 'node:test';
import assert from 'node:assert/strict';

import { InventoryAPIClient } from '../src/utils/apiClient.js';

test('InventoryAPIClient unwraps paged result envelopes for materials, lots, and categories', async () => {
  const client = new InventoryAPIClient('http://inventory.example');
  const fakeHttp = {
    async get(url: string) {
      if (url === '/api/v1/materials') {
        return {
          data: {
            code: 0,
            message: 'ok',
            data: {
              items: [{ id: 'mat-1', name: '牛奶', spec: '500ml', locations: null, total_quantity: 1, default_unit: '件' }],
              total: 1,
            },
          },
        };
      }

      if (url === '/api/v1/stock-lots') {
        return {
          data: {
            code: 0,
            message: 'ok',
            data: {
              items: [{ id: 'lot-1', quantity_on_hand: 2, unit: '瓶', location: '冰箱' }],
              total: 1,
            },
          },
        };
      }

      if (url === '/api/v1/categories') {
        return {
          data: {
            code: 0,
            message: 'ok',
            data: {
              items: [{ id: 'cat-1', tenant_id: 'default', name: '调料' }],
              total: 1,
            },
          },
        };
      }

      throw new Error(`unexpected GET ${url}`);
    },
  };

  (client as unknown as { client: unknown }).client = fakeHttp;

  const materials = await client.listMaterials({ keyword: '牛奶' });
  const lots = await client.listStockLots({ location: '冰箱' });
  const categories = await client.listCategories();

  assert.equal(materials.total, 1);
  assert.equal(materials.materials[0]?.name, '牛奶');
  assert.deepEqual(materials.materials[0]?.locations, []);
  assert.equal(lots.total, 1);
  assert.equal(lots.lots[0]?.location, '冰箱');
  assert.equal(categories.total, 1);
  assert.equal(categories.categories[0]?.name, '调料');
});

test('InventoryAPIClient unwraps result envelopes for single objects and health checks', async () => {
  const client = new InventoryAPIClient('http://192.168.18.13:8888');
  const fakeHttp = {
    async get(url: string) {
      if (url === '/api/v1/materials/mat-1') {
        return {
          data: {
            code: 0,
            message: 'ok',
            data: { id: 'mat-1', name: '牛奶', spec: '500ml', locations: ['冰箱'] },
          },
        };
      }

      if (url === '/api/v1/health') {
        return {
          data: {
            code: 0,
            message: 'ok',
            data: { status: 'ok' },
          },
        };
      }

      throw new Error(`unexpected GET ${url}`);
    },
    async post(url: string) {
      if (url === '/api/v1/categories') {
        return {
          data: {
            code: 0,
            message: 'ok',
            data: { id: 'cat-1', tenant_id: 'default', name: '调料' },
          },
        };
      }

      throw new Error(`unexpected POST ${url}`);
    },
    async put(url: string) {
      if (url === '/api/v1/stock-lots/lot-1') {
        return {
          data: {
            code: 0,
            message: 'ok',
            data: { id: 'lot-1', location: '冰箱', quantity_on_hand: 2 },
          },
        };
      }

      throw new Error(`unexpected PUT ${url}`);
    },
  };

  (client as unknown as { client: unknown }).client = fakeHttp;

  const material = await client.getMaterial('mat-1');
  const category = await client.createCategory('调料');
  const lot = await client.updateStockLot('lot-1', { location: '冰箱' });
  const health = await client.checkHealth();

  assert.equal(material.id, 'mat-1');
  assert.equal(category.name, '调料');
  assert.equal(lot.id, 'lot-1');
  assert.equal(health.ok, true);
  assert.equal(health.baseUrl, 'http://192.168.18.13:8888');
});
