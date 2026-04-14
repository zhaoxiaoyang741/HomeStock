import { InventoryClient, MaterialSummary, StockLot } from '../utils/apiClient.js';
import {
  coerceStringValue,
  daysUntilExpiry,
  formatLocalDate,
  isRawCommandEnvelope,
  parseSlashCommandArgs,
} from '../utils/runtime.js';

export interface QueryInventoryParams {
  location?: string;
  category?: string;
  expiring_soon?: boolean;
  keyword?: string;
}

const EXPIRING_KEYWORDS = new Set(['expiring', '快过期', '即将过期', '过期', '临期']);

export function parseQueryInventoryParams(input: QueryInventoryParams | unknown): QueryInventoryParams {
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
  const result: QueryInventoryParams = {};

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

function filterMaterialByKeyword(materials: MaterialSummary[], keyword?: string): MaterialSummary[] {
  const query = keyword?.trim().toLowerCase();
  if (!query) {
    return materials;
  }

  return materials.filter((material) => `${material.name} ${material.spec}`.toLowerCase().includes(query));
}

function materialLocations(material: MaterialSummary): string[] {
  return Array.isArray(material.locations) ? material.locations : [];
}

function hasMaterialStock(material: MaterialSummary): boolean {
  return Number.isFinite(material.total_quantity) && material.total_quantity > 0;
}

function hasLotStock(lot: StockLot): boolean {
  return Number.isFinite(lot.quantity_on_hand) && lot.quantity_on_hand > 0;
}

function buildMaterialLine(material: MaterialSummary): string {
  let line = `• ${material.name}`;
  if (material.spec) {
    line += ` / ${material.spec}`;
  }
  line += ` ${material.total_quantity}${material.default_unit}`;
  const locations = materialLocations(material);
  if (locations.length > 0) {
    line += ` [${locations.join('、')}]`;
  }
  if (material.nearest_expire_at) {
    const daysLeft = daysUntilExpiry(material.nearest_expire_at);
    line += ` 最近到期：${formatLocalDate(material.nearest_expire_at)}（${daysLeft}天）`;
  }
  return line;
}

function buildLotLine(lot: StockLot): string {
  let line = `• ${lot.material?.name ?? '物料'} ${lot.quantity_on_hand}${lot.unit}`;
  if (lot.material?.spec) {
    line += ` / ${lot.material.spec}`;
  }
  if (lot.location) {
    line += ` [${lot.location}]`;
  }
  if (lot.expire_at) {
    const daysLeft = daysUntilExpiry(lot.expire_at);
    if (daysLeft <= 0) {
      line += ` 已过期（${formatLocalDate(lot.expire_at)}）`;
    } else {
      line += ` ${formatLocalDate(lot.expire_at)}到期，还剩${daysLeft}天`;
    }
  }
  return line;
}

async function explainServiceState(client: InventoryClient, queryError: unknown): Promise<string> {
  const health = await client.checkHealth();
  const commandHint = '如需走确定性兜底，请直接使用 /query_inventory，并尽量带上位置、分类或关键字。';

  if (health.ok) {
    return `HomeStock 服务在线（${health.baseUrl}），但库存查询失败：${(queryError as Error).message}。${commandHint}`;
  }

  return `HomeStock 服务当前不可用（${health.baseUrl}）。健康检查失败：${health.error ?? '未知错误'}。库存查询未执行成功。${commandHint}`;
}

export async function queryInventory(
  params: QueryInventoryParams | unknown,
  client: InventoryClient,
): Promise<string> {
  try {
    const parsed = parseQueryInventoryParams(params);
    const categoryName = parsed.category?.trim();
    const category = categoryName ? await client.findCategoryByName(categoryName) : null;

    if (categoryName && !category) {
      return `当前没有找到分类“${categoryName}”的库存记录。`;
    }

    if (parsed.expiring_soon) {
      const { lots } = await client.listStockLots({
        category_id: category?.id,
        location: parsed.location?.trim() || undefined,
        keyword: parsed.keyword?.trim() || undefined,
        expiring_soon: true,
      });
      const activeLots = lots.filter(hasLotStock);

      if (activeLots.length === 0) {
        return '当前没有临期批次。';
      }

      return `临期批次（共 ${activeLots.length} 条）\n${activeLots.map(buildLotLine).join('\n')}`;
    }

    const { materials } = await client.listMaterials({
      category_id: category?.id,
      keyword: parsed.keyword?.trim() || undefined,
    });

    let filtered = filterMaterialByKeyword(materials, parsed.keyword).filter(hasMaterialStock);
    if (parsed.location?.trim()) {
      const location = parsed.location.trim();
      filtered = filtered.filter((material) => materialLocations(material).includes(location));
    }

    if (filtered.length === 0) {
      return '当前条件下没有匹配的库存记录。';
    }

    return `物料汇总（共 ${filtered.length} 项）\n${filtered.map(buildMaterialLine).join('\n')}`;
  } catch (err) {
    return explainServiceState(client, err);
  }
}
