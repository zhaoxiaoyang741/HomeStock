import axios, { AxiosError, AxiosInstance } from 'axios';

export interface Category {
  id: string;
  tenant_id: string;
  name: string;
}

export interface MaterialSummary {
  id: string;
  tenant_id: string;
  name: string;
  spec: string;
  category_id: string;
  category: Category | null;
  default_unit: string;
  status: string;
  total_quantity: number;
  lot_count: number;
  nearest_expire_at: string | null;
  locations: string[];
}

export interface MaterialDetail extends MaterialSummary {}

export interface StockLot {
  id: string;
  tenant_id: string;
  material_id: string;
  material: MaterialDetail | null;
  quantity_on_hand: number;
  unit: string;
  location: string;
  expire_at: string | null;
  purchased_at: string | null;
  received_at: string;
  notes: string;
  status: string;
}

export interface ConsumedLot {
  lot_id: string;
  consumed_quantity: number;
  remaining_quantity: number;
  location: string;
  expire_at: string | null;
}

export interface ConsumeMaterialResponse {
  material: MaterialDetail;
  requested_quantity: number;
  unit: string;
  consumed_lots: ConsumedLot[];
}

export interface ListMaterialsResponse {
  materials: MaterialSummary[];
  total: number;
}

export interface ListStockLotsResponse {
  lots: StockLot[];
  total: number;
}

export interface InboundStockLotPayload {
  material_id?: string;
  name?: string;
  spec?: string;
  category_id?: string;
  quantity: number;
  unit?: string;
  location?: string;
  expire_at?: string;
  purchased_at?: string;
  notes?: string;
}

export interface UpdateStockLotPayload {
  expire_at?: string;
  purchased_at?: string;
  notes?: string;
  location?: string;
}

export interface AdjustStockLotPayload {
  target_quantity: number;
  reason?: string;
  remark?: string;
}

export interface HealthCheckResponse {
  status: string;
}

export interface HealthCheckResult {
  ok: boolean;
  baseUrl: string;
  status?: string;
  error?: string;
}

export interface UserHeaders {
  userName?: string;
  userID?: string;
  channel?: string;
}

export interface InventoryClient {
  inboundStockLot(params: InboundStockLotPayload): Promise<StockLot>;
  listMaterials(filter: { category_id?: string; keyword?: string }): Promise<ListMaterialsResponse>;
  getMaterial(id: string): Promise<MaterialDetail>;
  consumeMaterial(id: string, params: { quantity: number; reason?: string }): Promise<ConsumeMaterialResponse>;
  listStockLots(filter: {
    material_id?: string;
    category_id?: string;
    location?: string;
    status?: string;
    keyword?: string;
    expiring_soon?: boolean;
  }): Promise<ListStockLotsResponse>;
  updateStockLot(id: string, params: UpdateStockLotPayload): Promise<StockLot>;
  adjustStockLot(id: string, params: AdjustStockLotPayload): Promise<StockLot>;
  findMaterialByName(name: string): Promise<MaterialSummary | null>;
  findLotsByMaterialName(name: string): Promise<StockLot[]>;
  findCategoryByName(name: string): Promise<Category | null>;
  ensureCategoryByName(name: string): Promise<Category>;
  checkHealth(): Promise<HealthCheckResult>;
}

function normalizedText(value: string | null | undefined): string {
  return (value ?? '').trim().toLowerCase();
}

function sortMatchScore(target: string, candidate: string): number {
  if (candidate === target) {
    return 0;
  }
  if (candidate.startsWith(target)) {
    return 1;
  }
  if (candidate.includes(target)) {
    return 2;
  }
  if (target.includes(candidate)) {
    return 3;
  }
  return Number.MAX_SAFE_INTEGER;
}

function formatAxiosError(err: unknown): string {
  if (!axios.isAxiosError(err)) {
    return err instanceof Error ? err.message : String(err);
  }

  const responseMessage =
    typeof err.response?.data === 'object' &&
    err.response?.data !== null &&
    'error' in err.response.data &&
    typeof (err.response.data as { error?: unknown }).error === 'string'
      ? (err.response.data as { error: string }).error
      : undefined;

  if (responseMessage) {
    return responseMessage;
  }

  return err.message;
}

export class InventoryAPIClient implements InventoryClient {
  private client: AxiosInstance;
  private readonly baseURL: string;

  constructor(baseURL: string, tenantID: string = 'default', userHeaders: UserHeaders = {}) {
    this.baseURL = baseURL.replace(/\/+$/, '');
    this.client = axios.create({
      baseURL: this.baseURL,
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': tenantID,
        ...(userHeaders.userName ? { 'X-User-Name': userHeaders.userName } : {}),
        ...(userHeaders.userID ? { 'X-User-ID': userHeaders.userID } : {}),
        'X-Channel': userHeaders.channel ?? 'feishu',
      },
      timeout: 10000,
    });
  }

  async inboundStockLot(params: InboundStockLotPayload): Promise<StockLot> {
    const payload: Record<string, unknown> = {
      quantity: params.quantity,
    };
    if (params.material_id) payload.material_id = params.material_id;
    if (params.name) payload.name = params.name;
    if (params.spec) payload.spec = params.spec;
    if (params.category_id) payload.category_id = params.category_id;
    if (params.unit) payload.unit = params.unit;
    if (params.location) payload.location = params.location;
    if (params.expire_at) payload.expire_at = params.expire_at;
    if (params.purchased_at) payload.purchased_at = params.purchased_at;
    if (params.notes) payload.notes = params.notes;

    const { data } = await this.client.post<StockLot>('/api/v1/stock-lots/inbound', payload);
    return data;
  }

  async listMaterials(filter: { category_id?: string; keyword?: string }): Promise<ListMaterialsResponse> {
    const params: Record<string, string> = {};
    if (filter.category_id) params.category_id = filter.category_id;
    if (filter.keyword) params.keyword = filter.keyword;
    const { data } = await this.client.get<ListMaterialsResponse>('/api/v1/materials', { params });
    return data;
  }

  async getMaterial(id: string): Promise<MaterialDetail> {
    const { data } = await this.client.get<MaterialDetail>(`/api/v1/materials/${id}`);
    return data;
  }

  async consumeMaterial(id: string, params: { quantity: number; reason?: string }): Promise<ConsumeMaterialResponse> {
    const { data } = await this.client.post<ConsumeMaterialResponse>(`/api/v1/materials/${id}/consume`, params);
    return data;
  }

  async listStockLots(filter: {
    material_id?: string;
    category_id?: string;
    location?: string;
    status?: string;
    keyword?: string;
    expiring_soon?: boolean;
  }): Promise<ListStockLotsResponse> {
    const params: Record<string, string> = {};
    if (filter.material_id) params.material_id = filter.material_id;
    if (filter.category_id) params.category_id = filter.category_id;
    if (filter.location) params.location = filter.location;
    if (filter.status) params.status = filter.status;
    if (filter.keyword) params.keyword = filter.keyword;
    if (filter.expiring_soon) params.expiring_soon = 'true';
    const { data } = await this.client.get<ListStockLotsResponse>('/api/v1/stock-lots', { params });
    return data;
  }

  async updateStockLot(id: string, params: UpdateStockLotPayload): Promise<StockLot> {
    const { data } = await this.client.put<StockLot>(`/api/v1/stock-lots/${id}`, params);
    return data;
  }

  async adjustStockLot(id: string, params: AdjustStockLotPayload): Promise<StockLot> {
    const { data } = await this.client.post<StockLot>(`/api/v1/stock-lots/${id}/adjust`, params);
    return data;
  }

  async listCategories(): Promise<{ categories: Category[]; total: number }> {
    const { data } = await this.client.get<{ categories: Category[]; total: number }>('/api/v1/categories');
    return data;
  }

  async createCategory(name: string): Promise<Category> {
    const { data } = await this.client.post<Category>('/api/v1/categories', { name: name.trim() });
    return data;
  }

  async findCategoryByName(name: string): Promise<Category | null> {
    const query = normalizedText(name);
    if (!query) return null;

    const { categories } = await this.listCategories();
    const matches = categories
      .map((category) => ({
        category,
        score: sortMatchScore(query, normalizedText(category.name)),
      }))
      .filter((entry) => entry.score < Number.MAX_SAFE_INTEGER)
      .sort((left, right) => left.score - right.score || left.category.name.length - right.category.name.length);

    return matches[0]?.category ?? null;
  }

  async ensureCategoryByName(name: string): Promise<Category> {
    const trimmed = name.trim();
    const existing = await this.findCategoryByName(trimmed);
    if (existing) {
      return existing;
    }
    return this.createCategory(trimmed);
  }

  async findMaterialByName(name: string): Promise<MaterialSummary | null> {
    const query = normalizedText(name);
    if (!query) return null;

    const { materials } = await this.listMaterials({ keyword: name });
    const matches = materials
      .map((material) => ({
        material,
        score: sortMatchScore(query, normalizedText(material.name)),
      }))
      .filter((entry) => entry.score < Number.MAX_SAFE_INTEGER)
      .sort((left, right) => left.score - right.score || left.material.name.length - right.material.name.length);

    return matches[0]?.material ?? null;
  }

  async findLotsByMaterialName(name: string): Promise<StockLot[]> {
    const material = await this.findMaterialByName(name);
    if (!material) {
      return [];
    }
    const { lots } = await this.listStockLots({ material_id: material.id });
    return lots;
  }

  async checkHealth(): Promise<HealthCheckResult> {
    try {
      const { data } = await this.client.get<HealthCheckResponse>('/api/v1/health');
      return {
        ok: data.status === 'ok',
        baseUrl: this.baseURL,
        status: data.status,
      };
    } catch (err) {
      return {
        ok: false,
        baseUrl: this.baseURL,
        error: formatAxiosError(err),
      };
    }
  }
}
