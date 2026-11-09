import type { Lang } from './catalogue';

/** What one list cost in a month. */
export type ListSpend = { list: string; name: string; total: number; priced: number };

/** What the household spent in a month: GET /api/app/spend, newest first. */
export type MonthSpend = { month: string; total: number; lists: ListSpend[] };

/** "€12.50", in the way your language writes it. */
export function money(amount: number, lang: Lang | string): string {
  return new Intl.NumberFormat(lang, { style: 'currency', currency: 'EUR' }).format(amount);
}

/** "November 2026" for "2026-11". */
export function monthName(month: string, lang: Lang | string): string {
  const [y, m] = month.split('-').map(Number);
  return new Intl.DateTimeFormat(lang, { month: 'long', year: 'numeric' }).format(
    new Date(y, m - 1, 1),
  );
}

/** What was typed in a price field, as a number: "3,20" is 3.2; empty or odd is 0. */
export function parsePrice(text: string): number {
  const n = Number(text.trim().replace(/^€\s*/, '').replace(',', '.'));
  return Number.isFinite(n) && n > 0 ? Math.round(n * 100) / 100 : 0;
}
