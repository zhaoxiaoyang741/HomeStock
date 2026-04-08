import { InventoryAPIClient } from './apiClient.js';

export interface TextToolResult {
  content: Array<{ type: 'text'; text: string }>;
  details: string;
}

export interface RawCommandEnvelope {
  command?: string;
  commandName?: string;
  skillName?: string;
}

export function textResult(text: string): TextToolResult {
  return { content: [{ type: 'text', text }], details: text };
}

export function resolveBaseUrl(pluginConfig?: { baseUrl?: unknown }): string {
  const configured = typeof pluginConfig?.baseUrl === 'string' ? pluginConfig.baseUrl.trim() : '';
  if (configured) {
    return configured;
  }

  const fromEnv = process.env.INVENTORY_API_URL?.trim();
  if (fromEnv) {
    return fromEnv;
  }

  return 'http://localhost:8888';
}

export function createClient(baseUrl: string): InventoryAPIClient {
  return new InventoryAPIClient(baseUrl);
}

export function isRawCommandEnvelope(value: unknown): value is RawCommandEnvelope {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return typeof (value as RawCommandEnvelope).command === 'string';
}

export function parseSlashCommandArgs(raw: string): string[] {
  const matches = raw.match(/"[^"]*"|'[^']*'|\S+/g) ?? [];
  return matches
    .map(token => token.trim())
    .filter(Boolean)
    .map(token => {
      if (
        (token.startsWith('"') && token.endsWith('"')) ||
        (token.startsWith("'") && token.endsWith("'"))
      ) {
        return token.slice(1, -1);
      }

      return token;
    });
}

export function normalizeDateInput(input: string): string {
  const trimmed = input.trim();
  if (trimmed === '') {
    return '';
  }

  const match = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!match) {
    throw new Error('日期必须使用 YYYY-MM-DD 格式');
  }

  const [, yearText, monthText, dayText] = match;
  const year = Number(yearText);
  const month = Number(monthText);
  const day = Number(dayText);
  const localDate = new Date(year, month - 1, day, 23, 59, 59, 999);

  if (
    localDate.getFullYear() !== year ||
    localDate.getMonth() !== month - 1 ||
    localDate.getDate() !== day
  ) {
    throw new Error('日期无效，请检查年月日');
  }

  return localDate.toISOString();
}

export function formatLocalDate(dateLike: string): string {
  const date = new Date(dateLike);
  if (Number.isNaN(date.getTime())) {
    return dateLike;
  }

  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export function daysUntilExpiry(expireAt: string, now: Date = new Date()): number {
  const expireDate = new Date(expireAt);
  if (Number.isNaN(expireDate.getTime())) {
    return Number.NaN;
  }

  return Math.ceil((expireDate.getTime() - now.getTime()) / 86_400_000);
}

export function hasNonEmptyValue(value: unknown): boolean {
  return value !== undefined && value !== null;
}

export function coerceNumericValue(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }

  if (typeof value !== 'string') {
    return undefined;
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }

  const match = trimmed.match(/^(\d+(?:\.\d+)?)/);
  if (!match) {
    return undefined;
  }

  const parsed = Number(match[1]);
  return Number.isFinite(parsed) ? parsed : undefined;
}

export function coerceStringValue(value: unknown): string | undefined {
  return typeof value === 'string' ? value.trim() : undefined;
}
