import test from 'node:test';
import assert from 'node:assert/strict';

import { daysUntilExpiry, normalizeDateInput, parseSlashCommandArgs, resolveBaseUrl } from '../src/utils/runtime.js';

test('normalizeDateInput converts YYYY-MM-DD to RFC3339 at local end of day', () => {
  const iso = normalizeDateInput('2026-04-30');
  const date = new Date(iso);

  assert.equal(date.getFullYear(), 2026);
  assert.equal(date.getMonth(), 3);
  assert.equal(date.getDate(), 30);
  assert.equal(date.getHours(), 23);
  assert.equal(date.getMinutes(), 59);
});

test('normalizeDateInput rejects invalid dates', () => {
  assert.throws(() => normalizeDateInput('2026-02-30'), /日期无效/);
  assert.throws(() => normalizeDateInput('2026\/04\/30'), /YYYY-MM-DD/);
});

test('daysUntilExpiry rounds up to the next full day', () => {
  const now = new Date('2026-04-07T00:00:00.000Z');
  assert.equal(daysUntilExpiry('2026-04-08T12:00:00.000Z', now), 2);
});

test('parseSlashCommandArgs keeps quoted strings together', () => {
  const args = parseSlashCommandArgs('土豆 5个 "冰箱 上层"');
  assert.deepEqual(args, ['土豆', '5个', '冰箱 上层']);
});

test('resolveBaseUrl respects config over env over default', () => {
  const original = process.env.INVENTORY_API_URL;

  process.env.INVENTORY_API_URL = 'http://env.example';
  assert.equal(resolveBaseUrl({ baseUrl: 'http://config.example' }), 'http://config.example');
  assert.equal(resolveBaseUrl({}), 'http://env.example');

  delete process.env.INVENTORY_API_URL;
  assert.equal(resolveBaseUrl({}), 'http://localhost:8888');

  if (original) {
    process.env.INVENTORY_API_URL = original;
  } else {
    delete process.env.INVENTORY_API_URL;
  }
});
