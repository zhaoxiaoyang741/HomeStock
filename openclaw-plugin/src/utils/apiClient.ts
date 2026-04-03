import axios, { AxiosInstance } from 'axios';

export interface Item {
  id: string;
  name: string;
  quantity: number;
  unit: string;
  location: string;
  category: { id: string; name: string } | null;
  expire_at: string | null;
  notes: string;
}

interface ListResponse {
  items: Item[];
  total: number;
}

export class InventoryAPIClient {
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

  async addItem(params: {
    name: string;
    quantity?: number;
    unit?: string;
    location?: string;
    notes?: string;
  }): Promise<Item> {
    const { data } = await this.client.post<Item>('/api/v1/items', {
      name: params.name,
      quantity: params.quantity ?? 1,
      unit: params.unit ?? '个',
      location: params.location ?? '',
      notes: params.notes ?? '',
    });
    return data;
  }

  async listItems(filter: {
    location?: string;
    category?: string;
    expiring_soon?: boolean;
  }): Promise<ListResponse> {
    const params: Record<string, string> = {};
    if (filter.location) params.location = filter.location;
    if (filter.category) params.category = filter.category;
    if (filter.expiring_soon) params.expiring_soon = 'true';
    const { data } = await this.client.get<ListResponse>('/api/v1/items', { params });
    return data;
  }

  async deleteItem(id: string): Promise<void> {
    await this.client.delete(`/api/v1/items/${id}`);
  }

  async updateItem(id: string, params: { quantity: number }): Promise<Item> {
    const { data } = await this.client.put<Item>(`/api/v1/items/${id}`, params);
    return data;
  }

  async findItemByName(name: string): Promise<Item | null> {
    const { items } = await this.listItems({});
    return items.find(i => i.name.includes(name) || name.includes(i.name)) ?? null;
  }
}
