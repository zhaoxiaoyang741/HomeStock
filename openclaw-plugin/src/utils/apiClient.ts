import axios, { AxiosInstance } from 'axios';

export interface Category {
  id: string;
  tenant_id: string;
  name: string;
}

export interface Item {
  id: string;
  tenant_id: string;
  name: string;
  category_id: string;
  category: Category | null;
  quantity: number;
  unit: string;
  location: string;
  expire_at: string | null;
  purchased_at?: string;
  notes: string;
}

export interface ListItemsResponse {
  items: Item[];
  total: number;
}

export interface ListCategoriesResponse {
  categories: Category[];
  total: number;
}

export interface AddItemPayload {
  name: string;
  quantity?: number;
  unit?: string;
  location?: string;
  category_id?: string;
  expire_at?: string;
  notes?: string;
}

export interface UpdateItemPayload {
  quantity?: number;
  expire_at?: string;
  notes?: string;
  location?: string;
}

export interface InventoryClient {
  addItem(params: AddItemPayload): Promise<Item>;
  listItems(filter: { location?: string; category_id?: string }): Promise<ListItemsResponse>;
  searchByKeyword(keyword: string): Promise<Item[]>;
  updateItem(id: string, params: UpdateItemPayload): Promise<Item>;
  deleteItem(id: string): Promise<void>;
  findItemByName(name: string): Promise<Item | null>;
  findCategoryByName(name: string): Promise<Category | null>;
  ensureCategoryByName(name: string): Promise<Category>;
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

export class InventoryAPIClient implements InventoryClient {
  private client: AxiosInstance;

  constructor(baseURL: string, tenantID: string = 'default') {
    this.client = axios.create({
      baseURL,
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': tenantID,
      },
      timeout: 10000,
    });
  }

  async addItem(params: AddItemPayload): Promise<Item> {
    const payload: Record<string, unknown> = {
      name: params.name,
      quantity: params.quantity ?? 1,
      unit: params.unit ?? '个',
      location: params.location ?? '',
      notes: params.notes ?? '',
    };

    if (params.category_id) {
      payload.category_id = params.category_id;
    }
    if (params.expire_at) {
      payload.expire_at = params.expire_at;
    }

    const { data } = await this.client.post<Item>('/api/v1/items', payload);
    return data;
  }

  async listItems(filter: { location?: string; category_id?: string }): Promise<ListItemsResponse> {
    const params: Record<string, string> = {};
    if (filter.location) {
      params.location = filter.location;
    }
    if (filter.category_id) {
      params.category_id = filter.category_id;
    }

    const { data } = await this.client.get<ListItemsResponse>('/api/v1/items', { params });
    return data;
  }

  async searchByKeyword(keyword: string): Promise<Item[]> {
    const query = normalizedText(keyword);
    if (!query) {
      const { items } = await this.listItems({});
      return items;
    }

    const { items } = await this.listItems({});
    return items.filter(item => normalizedText(item.name).includes(query));
  }

  async deleteItem(id: string): Promise<void> {
    await this.client.delete(`/api/v1/items/${id}`);
  }

  async updateItem(id: string, params: UpdateItemPayload): Promise<Item> {
    const payload: Record<string, unknown> = {};
    if (params.quantity !== undefined) {
      payload.quantity = params.quantity;
    }
    if (params.expire_at !== undefined) {
      payload.expire_at = params.expire_at;
    }
    if (params.notes !== undefined) {
      payload.notes = params.notes;
    }
    if (params.location !== undefined) {
      payload.location = params.location;
    }

    const { data } = await this.client.put<Item>(`/api/v1/items/${id}`, payload);
    return data;
  }

  async listCategories(): Promise<ListCategoriesResponse> {
    const { data } = await this.client.get<ListCategoriesResponse>('/api/v1/categories');
    return data;
  }

  async createCategory(name: string): Promise<Category> {
    const { data } = await this.client.post<Category>('/api/v1/categories', {
      name: name.trim(),
    });
    return data;
  }

  async findCategoryByName(name: string): Promise<Category | null> {
    const query = normalizedText(name);
    if (!query) {
      return null;
    }

    const { categories } = await this.listCategories();
    const matches = categories
      .map(category => ({
        category,
        score: sortMatchScore(query, normalizedText(category.name)),
      }))
      .filter(entry => entry.score < Number.MAX_SAFE_INTEGER)
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

  async findItemByName(name: string): Promise<Item | null> {
    const query = normalizedText(name);
    if (!query) {
      return null;
    }

    const { items } = await this.listItems({});
    const matches = items
      .map(item => ({
        item,
        score: sortMatchScore(query, normalizedText(item.name)),
      }))
      .filter(entry => entry.score < Number.MAX_SAFE_INTEGER)
      .sort((left, right) => left.score - right.score || left.item.name.length - right.item.name.length);

    return matches[0]?.item ?? null;
  }
}
