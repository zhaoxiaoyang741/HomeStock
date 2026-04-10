import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';

const rootDir = process.cwd();

function read(relativePath: string): string {
  return readFileSync(path.join(rootDir, relativePath), 'utf8');
}

test('plugin index exposes only the new tool names', () => {
  const source = read('src/index.ts');

  assert.match(source, /name: 'inbound_stock'/);
  assert.match(source, /name: 'consume_material'/);
  assert.match(source, /name: 'query_inventory'/);
  assert.match(source, /name: 'update_stock_lot'/);
  assert.match(source, /name: 'check_homestock_service'/);

  assert.doesNotMatch(source, /name: 'add_item'/);
  assert.doesNotMatch(source, /name: 'remove_item'/);
  assert.doesNotMatch(source, /name: 'query_items'/);
  assert.doesNotMatch(source, /name: 'update_item'/);
});

test('README and skills only reference the new tool names', () => {
  const readme = read('README.md');
  assert.match(readme, /inbound_stock/);
  assert.match(readme, /consume_material/);
  assert.match(readme, /query_inventory/);
  assert.match(readme, /update_stock_lot/);
  assert.match(readme, /check_homestock_service/);
  assert.doesNotMatch(readme, /\badd_item\b/);
  assert.doesNotMatch(readme, /\bremove_item\b/);
  assert.doesNotMatch(readme, /\bquery_items\b/);
  assert.doesNotMatch(readme, /\bupdate_item\b/);

  const newSkillFiles = [
    ['skills/inbound_stock/SKILL.md', 'inbound_stock'],
    ['skills/consume_material/SKILL.md', 'consume_material'],
    ['skills/query_inventory/SKILL.md', 'query_inventory'],
    ['skills/update_stock_lot/SKILL.md', 'update_stock_lot'],
    ['skills/check_homestock_service/SKILL.md', 'check_homestock_service'],
  ] as const;

  for (const [relativePath, skillName] of newSkillFiles) {
    assert.equal(existsSync(path.join(rootDir, relativePath)), true);
    const source = read(relativePath);
    assert.match(source, new RegExp(`name: ${skillName}`));
    assert.match(source, new RegExp(`command-tool: ${skillName}`));
  }

  assert.equal(existsSync(path.join(rootDir, 'skills/add_item/SKILL.md')), false);
  assert.equal(existsSync(path.join(rootDir, 'skills/remove_item/SKILL.md')), false);
  assert.equal(existsSync(path.join(rootDir, 'skills/query_items/SKILL.md')), false);
  assert.equal(existsSync(path.join(rootDir, 'skills/update_item/SKILL.md')), false);
});

test('query_inventory skill covers natural fridge inventory prompts', () => {
  const source = read('skills/query_inventory/SKILL.md');

  assert.match(source, /目前冰箱里有什么/);
  assert.match(source, /现在冰箱里还有什么/);
  assert.match(source, /不要回复“我先帮你建个清单”/);
});
