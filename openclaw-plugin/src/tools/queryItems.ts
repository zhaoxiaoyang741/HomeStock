import { InventoryClient, Item } from '../utils/apiClient.js';
import {
  coerceStringValue,
  daysUntilExpiry,
  formatLocalDate,
  isRawCommandEnvelope,
  parseSlashCommandArgs,
} from '../utils/runtime.js';

export interface QueryItemsParams {
  location?: string;
  category?: string;
  expiring_soon?: boolean;
  keyword?: string;
}

const EXPIRING_KEYWORDS = new Set(['expiring', '快过期', '即将过期', '过期', '临期']);

export function parseQueryItemsParams(input: QueryItemsParams | unknown): QueryItemsParams {
  if (!isRawCommandEnvelope(input)) {
    const record = (input ?? {}) as Record<string, unknown>;
    return {
      location: coerceStringValue(record.location),
      category: coerceStringValue(record.category),
      keyword: coerceStringValue(record.keyword) ?? coerceStringValue(record.name),
      expiring_soon:
        typeof record.expiring_soon === 'boolean'
          ? record.expiring_soon
          : coerceStringValue(record.expiring_soon) === 'true',
    };
  }

  const tokens = parseSlashCommandArgs(input.command ?? '');
  const result: QueryItemsParams = {};

  for (const token of tokens) {
    if (EXPIRING_KEYWORDS.has(token)) {
      result.expiring_soon = true;
      continue;
    }

    if (!result.location) {
      result.location = token;
      continue;
    }

    if (!result.category) {
      result.category = token;
      continue;
    }

    if (!result.keyword) {
      result.keyword = token;
    }
  }

  return result;
}

function filterExpiringSoon(items: Item[]): Item[] {
  return items.filter(item => {
    if (!item.expire_at) {
      return false;
    }

    const daysLeft = daysUntilExpiry(item.expire_at);
    return !Number.isNaN(daysLeft) && daysLeft <= 7;
  });
}

function filterByKeyword(items: Item[], keyword?: string): Item[] {
  const query = keyword?.trim().toLowerCase();
  if (!query) {
    return items;
  }

  return items.filter(item => item.name.toLowerCase().includes(query));
}

function buildLine(item: Item, includeLocation: boolean): string {
  let line = `• ${item.name} ${item.quantity}${item.unit}`;
  if (includeLocation && item.location) {
    line += `（${item.location}）`;
  }
  if (item.category?.name) {
    line += ` [${item.category.name}]`;
  }

  if (item.expire_at) {
    const daysLeft = daysUntilExpiry(item.expire_at);
    if (daysLeft <= 0) {
      line += ` 🔴已过期（${formatLocalDate(item.expire_at)}）`;
    } else if (daysLeft <= 3) {
      line += ` 🟡${formatLocalDate(item.expire_at)}到期，还剩${daysLeft}天`;
    } else {
      line += ` 📅${formatLocalDate(item.expire_at)}到期`;
    }
  }

  return line;
}

export async function queryItems(
  params: QueryItemsParams | unknown,
  client: InventoryClient
): Promise<string> {
  try {
    const parsed = parseQueryItemsParams(params);
    const categoryName = parsed.category?.trim();
    const category = categoryName ? await client.findCategoryByName(categoryName) : null;

    if (categoryName && !category) {
      return `📦 当前没有找到分类「${categoryName}」的库存记录。`;
    }

    const { items } = await client.listItems({
      location: parsed.location?.trim() || undefined,
      category_id: category?.id,
    });

    let filtered = filterByKeyword(items, parsed.keyword);
    if (parsed.expiring_soon) {
      filtered = filterExpiringSoon(filtered);
    }

    if (filtered.length === 0) {
      const scopeParts = [parsed.location?.trim(), categoryName, parsed.keyword?.trim()].filter(Boolean);
      const scope = scopeParts.length > 0 ? `「${scopeParts.join(' / ')}」` : '当前条件';
      return `📦 ${scope}下没有匹配的库存记录。`;
    }

    const titleParts = ['📦 库存查询结果'];
    if (parsed.location?.trim()) {
      titleParts.push(`位置：${parsed.location.trim()}`);
    }
    if (categoryName) {
      titleParts.push(`分类：${categoryName}`);
    }
    if (parsed.keyword?.trim()) {
      titleParts.push(`关键字：${parsed.keyword.trim()}`);
    }
    if (parsed.expiring_soon) {
      titleParts.push('仅看 7 天内到期');
    }

    const lines = filtered.map(item => buildLine(item, !parsed.location?.trim()));
    return `${titleParts.join('｜')}（共 ${filtered.length} 项）\n${lines.join('\n')}`;
  } catch (err) {
    return `❌ 查询失败：${(err as Error).message}`;
  }
}
