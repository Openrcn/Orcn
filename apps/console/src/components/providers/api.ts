export interface ArrayItemSchema {
  type: "object" | "string" | "number" | "boolean";
  properties?: Record<string, { type: string; label?: string; placeholder?: string }>;
  required?: string[];
}

export interface ProviderField {
  key: string;
  name: string;
  description?: string;
  type: "text" | "password" | "number" | "boolean" | "select" | "file" | "json_array" | "string_array" | "number_array" | string;
  required: boolean;
  secret: boolean;
  default?: string;
  placeholder?: string;
  options?: string[];
  array_item_schema?: ArrayItemSchema;
}

export interface ProviderCapability {
  id: string;
  name: string;
  description: string;
  type: string;
  has_volume_support?: boolean;
  requires_regions?: boolean;
  requires_vpc?: boolean;
  features?: string[];
  schema?: ProviderField[];
}

export interface ProviderConnection {
  ID: string;
  OrganizationID: string;
  Provider: string;
  Name: string;
  Status: string;
  Config: string;
  CredentialSecretID?: string | null;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface Machine {
  ID: string;
  ProviderConnectionID: string;
  Name: string;
  Endpoint: string;
  ProxyEndpoint: string;
  TokenSecretID: string | null;
  Status: string;
  LastSeen: string | null;
  CPUArch: string | null;
  CPUCores: number;
  MemoryGB: number;
  GPUModel: string | null;
  GPUVRAMGB: number;
  StorageGB: number;
  Labels: string | null;
  CreatedAt: string;
  UpdatedAt: string;
}

const CONNECTIONS = "/api/v1/provider-connections";

async function parse<T>(res: Response): Promise<T> {
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    const message =
      (data && (data.error || data.message)) || `Request failed (${res.status})`;
    throw new Error(message);
  }
  return data as T;
}

export async function fetchProviders(): Promise<ProviderCapability[]> {
  const res = await fetch("/api/v1/providers");
  const data = await parse<{ providers?: ProviderCapability[] }>(res);
  return data.providers || [];
}

export async function fetchSchemas(): Promise<Record<string, ProviderField[]>> {
  const res = await fetch(`${CONNECTIONS}/schema`);
  if (!res.ok) return {};
  return (await res.json().catch(() => ({}))) as Record<string, ProviderField[]>;
}

export async function fetchSchema(provider: string): Promise<ProviderField[]> {
  const res = await fetch(
    `${CONNECTIONS}/schema?provider=${encodeURIComponent(provider)}`,
  );
  const data = await parse<{ fields?: ProviderField[] }>(res);
  return data.fields || [];
}

export async function fetchConnections(): Promise<ProviderConnection[]> {
  const res = await fetch(CONNECTIONS);
  const data = await parse<ProviderConnection[]>(res);
  return Array.isArray(data) ? data : [];
}

function buildBody(
  provider: string,
  fields: ProviderField[],
  values: Record<string, unknown>,
  files: Record<string, File>,
  name?: string,
): { body: BodyInit; headers: Record<string, string> } {
  const useForm = Object.keys(files).length > 0;

  if (useForm) {
    const fd = new FormData();
    fd.append("provider", provider);
    if (name) fd.append("name", name);
    fields.forEach((f) => {
      const file = files[f.key];
      if (file) {
        fd.append(f.key, file);
        return;
      }
      const v = values[f.key];
      if (v === undefined || v === null || v === "") return;
      fd.append(f.key, String(v));
    });
    return { body: fd, headers: {} };
  }

  const obj: Record<string, unknown> = { provider };
  if (name) obj.name = name;
  fields.forEach((f) => {
    const v = values[f.key];
    if (v === undefined || v === null || v === "") return;
    obj[f.key] = v;
  });
  return {
    body: JSON.stringify(obj),
    headers: { "Content-Type": "application/json" },
  };
}

export async function verifyConnectionInput(
  provider: string,
  fields: ProviderField[],
  values: Record<string, unknown>,
  files: Record<string, File>,
): Promise<{ verified: boolean; error?: string }> {
  const { body, headers } = buildBody(provider, fields, values, files);
  const res = await fetch(`${CONNECTIONS}/verify`, {
    method: "POST",
    headers,
    body,
  });
  return parse<{ verified: boolean; error?: string }>(res);
}

export async function createConnection(
  provider: string,
  name: string,
  fields: ProviderField[],
  values: Record<string, unknown>,
  files: Record<string, File>,
): Promise<ProviderConnection> {
  const { body, headers } = buildBody(provider, fields, values, files, name);
  const res = await fetch(CONNECTIONS, { method: "POST", headers, body });
  return parse<ProviderConnection>(res);
}

export async function fetchConnection(id: string): Promise<ProviderConnection> {
  const res = await fetch(`${CONNECTIONS}/${id}`);
  return parse<ProviderConnection>(res);
}

export async function updateConnection(
  id: string,
  provider: string,
  name: string,
  fields: ProviderField[],
  values: Record<string, unknown>,
  files: Record<string, File>,
): Promise<ProviderConnection> {
  const { body, headers } = buildBody(provider, fields, values, files, name);
  const res = await fetch(`${CONNECTIONS}/${id}`, {
    method: "PUT",
    headers,
    body,
  });
  return parse<ProviderConnection>(res);
}

export async function reverifyConnection(
  id: string,
): Promise<{ verified: boolean; status?: string }> {
  const res = await fetch(`${CONNECTIONS}/${id}/verify`, { method: "POST" });
  return parse<{ verified: boolean; status?: string }>(res);
}

export async function deleteConnection(id: string): Promise<void> {
  const res = await fetch(`${CONNECTIONS}/${id}`, { method: "DELETE" });
  await parse<unknown>(res);
}

export async function fetchConnectionConfig(id: string): Promise<{ config: Record<string, any>; secret: Record<string, any> }> {
  const res = await fetch(`${CONNECTIONS}/${id}/config`);
  return parse<{ config: Record<string, any>; secret: Record<string, any> }>(res);
}

export function providerLabel(id: string): string {
  return id
    .replace(/[_-]+/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase())
    .trim();
}

export async function fetchConnectionMachines(connectionId: string): Promise<{ machines: Machine[] }> {
  const res = await fetch(`${CONNECTIONS}/${connectionId}/machines`);
  return parse<{ machines: Machine[] }>(res);
}

export async function createMachine(
  connectionId: string,
  data: { name: string; endpoint: string; proxy_endpoint?: string; token: string },
): Promise<Machine> {
  const res = await fetch(`${CONNECTIONS}/${connectionId}/machines`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  return parse<Machine>(res);
}

export async function updateMachine(
  connectionId: string,
  machineId: string,
  data: { name?: string; endpoint?: string; proxy_endpoint?: string; token?: string },
): Promise<Machine> {
  const res = await fetch(`${CONNECTIONS}/${connectionId}/machines/${machineId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  return parse<Machine>(res);
}

export async function deleteMachine(connectionId: string, machineId: string): Promise<void> {
  const res = await fetch(`${CONNECTIONS}/${connectionId}/machines/${machineId}`, {
    method: "DELETE",
  });
  await parse<unknown>(res);
}
