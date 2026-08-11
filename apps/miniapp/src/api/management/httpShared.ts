export interface PagedResponse<T> {
  items: T[];
  page: number;
  page_size: number;
  total: number;
}

export function operationId(): string {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (character) => {
    const random = Math.floor(Math.random() * 16);
    const value = character === "x" ? random : (random & 0x3) | 0x8;
    return value.toString(16);
  });
}

export function queryPath(path: string, values: Record<string, unknown>): string {
  const query = Object.entries(values)
    .filter(([, value]) => value !== undefined && value !== "" && value !== "all")
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
    .join("&");
  return query ? `${path}?${query}` : path;
}
