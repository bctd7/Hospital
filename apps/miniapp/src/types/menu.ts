export type MenuTone = "blue" | "cyan" | "orange";

export interface MenuEntry {
  id: string;
  title: string;
  description?: string;
  symbol: string;
  tone: MenuTone;
  route: string;
}
