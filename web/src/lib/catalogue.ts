import type { RecordModel } from 'pocketbase';
import { itemKey, type Item } from './list';

export type Lang = 'en' | 'nl' | 'fr';
export const langs: { id: Lang; label: string }[] = [
  { id: 'nl', label: 'Nederlands' },
  { id: 'fr', label: 'Français' },
  { id: 'en', label: 'English' },
];

export interface Section {
  key: string;
  emoji: string;
  en: string;
  nl: string;
  fr: string;
}

export interface Generic extends RecordModel {
  slug: string;
  section: string;
  emoji: string;
  name_en: string;
  name_nl: string;
  name_fr: string;
  aliases: string;
  custom: boolean;
}

export const OTHER = 'other';

/** The language to show item names in before someone picks one: their phone's, else Dutch. */
export function guessLang(languages: readonly string[]): Lang {
  for (const l of languages) {
    const code = l.slice(0, 2).toLowerCase();
    if (code === 'nl' || code === 'fr' || code === 'en') return code;
  }
  return 'nl';
}

/** A generic's name in the language, or in another when it has none there. */
export function genericName(g: Generic, lang: Lang): string {
  return g[`name_${lang}`] || g.name_en || g.name_nl || g.name_fr;
}

export function sectionLabel(s: Section | undefined, lang: Lang): string {
  return s ? s[lang] || s.en : '';
}

const namesOf = (g: Generic, lang: Lang) => [
  { text: genericName(g, lang), own: true },
  ...[g.name_en, g.name_nl, g.name_fr, ...g.aliases.split(';')].map((text) => ({
    text,
    own: false,
  })),
];

/**
 * The generics that match what is being typed, best first: a whole name before
 * the start of a name before the start of a word, names in your own language
 * before others, your household's own before the catalogue, and short names.
 */
export function suggest(generics: Generic[], typed: string, lang: Lang, limit = 6): Generic[] {
  const q = itemKey(typed);
  if (!q) return [];
  const scored: { g: Generic; score: number }[] = [];
  for (const g of generics) {
    let best = 0;
    for (const { text, own } of namesOf(g, lang)) {
      const name = itemKey(text);
      if (!name) continue;
      let s = 0;
      if (name === q) s = 100;
      else if (name.startsWith(q)) s = 70;
      else if (` ${name}`.includes(` ${q}`)) s = 50;
      else if (q.length >= 3 && name.includes(q)) s = 20;
      if (s && own) s += 5;
      best = Math.max(best, s);
    }
    if (best)
      scored.push({ g, score: best + (g.custom ? 3 : 0) - genericName(g, lang).length / 100 });
  }
  return scored
    .sort((a, b) => b.score - a.score)
    .slice(0, limit)
    .map((s) => s.g);
}

/** The generic named exactly what was typed, if there is one. */
export function exactly(generics: Generic[], typed: string, lang: Lang): Generic | undefined {
  const [top] = suggest(generics, typed, lang, 1);
  if (!top) return undefined;
  const q = itemKey(typed);
  return namesOf(top, lang).some(({ text }) => itemKey(text) === q) ? top : undefined;
}

/** The items grouped by store section, in the sections' order; items without one go last. */
export function bySection(
  items: Item[],
  sections: Section[],
  sectionOf: (item: Item) => string,
): { section: Section | undefined; items: Item[] }[] {
  const groups = new Map<string, Item[]>();
  for (const item of items) {
    const key = sectionOf(item) || OTHER;
    groups.set(key, [...(groups.get(key) ?? []), item]);
  }
  const order = [...sections.map((s) => s.key)];
  if (!order.includes(OTHER)) order.push(OTHER);
  return [...groups.keys()]
    .sort((a, b) => {
      const ia = order.indexOf(a) < 0 ? order.indexOf(OTHER) : order.indexOf(a);
      const ib = order.indexOf(b) < 0 ? order.indexOf(OTHER) : order.indexOf(b);
      return ia - ib;
    })
    .map((key) => ({ section: sections.find((s) => s.key === key), items: groups.get(key)! }));
}
