import type { RecordModel } from 'pocketbase';
import type { Kind } from './kinds';

export interface List extends RecordModel {
  name: string;
  /** what the list is for; groceries when empty */
  kind: Kind | '';
  emoji: string;
  sort: number;
  archived: boolean;
}

export interface Item extends RecordModel {
  list: string;
  name: string;
  /** the generic item it is, if any: then it shows in each person's language */
  generic: string;
  /** the exact product, if any */
  product: string;
  quantity: string;
  note: string;
  checked: boolean;
  checked_at: string;
  checked_by: string;
  added_by: string;
  created: string;
}

/** How many recently bought items show under the list. */
export const RECENT = 24;

/**
 * Splits what was typed into the add bar into a name and a quantity:
 * "2 melk", "2x melk", "melk x2" and "melk 2×" are two of "melk".
 */
export function parseEntry(text: string): { name: string; quantity: string } {
  const s = text.trim().replace(/\s+/g, ' ');
  const before = s.match(/^(\d{1,3})\s*[x×*]?\s+(.+)$/i) ?? s.match(/^(\d{1,3})[x×*](.+)$/i);
  if (before) return { name: before[2].trim(), quantity: `${before[1]}×` };
  const after = s.match(/^(.+?)\s+(?:[x×*]\s*(\d{1,3})|(\d{1,3})\s*[x×*])$/i);
  if (after) return { name: after[1].trim(), quantity: `${after[2] ?? after[3]}×` };
  return { name: s, quantity: '' };
}

/** What makes two items the same thing, as the server sees it: "Mélk" is "melk". */
export function itemKey(name: string): string {
  return name
    .normalize('NFD')
    .replace(/\p{Mn}/gu, '')
    .toLowerCase()
    .replace(/ß/g, 'ss')
    .replace(/æ/g, 'ae')
    .replace(/œ/g, 'oe')
    .replace(/[^\p{L}\p{N}]+/gu, ' ')
    .trim();
}

/** The items still to buy, in the order they were added. */
export function toBuy(items: Item[]): Item[] {
  return items
    .filter((i) => !i.checked)
    .sort((a, b) => a.created.localeCompare(b.created) || a.id.localeCompare(b.id));
}

/** The items bought most recently first, at most `limit`. */
export function recentlyBought(items: Item[], limit = RECENT): Item[] {
  return items
    .filter((i) => i.checked)
    .sort((a, b) => b.checked_at.localeCompare(a.checked_at) || a.name.localeCompare(b.name))
    .slice(0, limit);
}

/** The lists in their order, archived ones left out. */
export function listsInOrder(lists: List[]): List[] {
  return lists
    .filter((l) => !l.archived)
    .sort((a, b) => a.sort - b.sort || a.created.localeCompare(b.created));
}

/** Replaces or adds a record by id, or removes it, as realtime events arrive. */
export function upsert<T extends RecordModel>(records: T[], record: T, remove = false): T[] {
  const rest = records.filter((r) => r.id !== record.id);
  return remove ? rest : [...rest, record];
}
