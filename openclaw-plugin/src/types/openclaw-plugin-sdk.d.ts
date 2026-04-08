declare module 'openclaw/plugin-sdk/plugin-entry' {
  export interface ToolRegistration {
    name: string;
    description: string;
    parameters: unknown;
    execute(toolCallId: string, params: unknown): Promise<unknown> | unknown;
  }

  export interface PluginApi {
    pluginConfig?: { baseUrl?: unknown };
    registerTool(tool: ToolRegistration): void;
  }

  export interface PluginEntry {
    id: string;
    name: string;
    description?: string;
    register(api: PluginApi): void;
  }

  export function definePluginEntry(entry: PluginEntry): PluginEntry;
}
