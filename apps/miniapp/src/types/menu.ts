export type MenuTone = "blue" | "cyan" | "orange" | "gray";

export interface MenuEntry {
  id: string;
  title: string;
  description?: string;
  symbol: string;
  tone: MenuTone;
  route: string;
}

export interface InfoRow {
  id: string;
  title: string;
  description?: string;
  value?: string;
}
