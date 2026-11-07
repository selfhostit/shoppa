import type { RecordModel } from 'pocketbase';
import { icons } from './icons';
import { GROCERIES, type Kind } from './kinds';
import { itemKey, type Item } from './list';

export type Lang = 'en' | 'nl' | 'fr' | 'el';
export const langs: { id: Lang; label: string }[] = [
  { id: 'nl', label: 'Nederlands' },
  { id: 'fr', label: 'Français' },
  { id: 'en', label: 'English' },
  { id: 'el', label: 'Ελληνικά' },
];

export interface Section {
  key: string;
  /** the kind of list it is a section of; empty for Other, which every list has */
  kind?: Kind;
  /** a line icon's name (lib/icons.ts) */
  icon: string;
  emoji?: string;
  en: string;
  nl: string;
  fr: string;
  el: string;
}

export interface Generic extends RecordModel {
  slug: string;
  /** the kind of list it is for */
  kind: Kind;
  section: string;
  /** a line icon's name; an emoji instead for some kept before there were icons */
  icon: string;
  emoji: string;
  name_en: string;
  name_nl: string;
  name_fr: string;
  name_el: string;
  aliases: string;
  /** Open Food Facts category tags, comma-separated: how products find their section */
  off_categories: string;
  custom: boolean;
}

/** The section and icon the household picked for an item, by its key. */
export interface Choice extends RecordModel {
  key: string;
  /** empty when only the icon was picked */
  section: string;
  /** a line icon's name; empty when only the section was picked */
  icon: string;
  /** an emoji picked before there were icons */
  emoji: string;
}

/** The icon the household picked for a section of a kind of list. */
export interface SectionIcon extends RecordModel {
  /** "home:home-textiles" (sectionIconKey) */
  key: string;
  icon: string;
}

export const sectionIconKey = (kind: Kind, section: string) => `${kind}:${section}`;

/**
 * What a choice of section and icon is kept by: the exact product or
 * catalogue item, or else the name as typed, in any case and with or without
 * accents, on a kind of list: a "lamp" on a furniture list isn't the one on a
 * grocery list.
 */
export function choiceKey(
  item: Pick<Item, 'product' | 'generic' | 'name'>,
  kind: Kind = GROCERIES,
): string {
  if (item.product) return `product:${item.product}`;
  if (item.generic) return `generic:${item.generic}`;
  const name = itemKey(item.name);
  return (kind === GROCERIES ? name : `${kind}:${name}`).slice(0, 120);
}

/**
 * The icons to pick from: those of the kind of list's sections and catalogue
 * items first, then every other one.
 */
export function iconChoices(sections: Section[], generics: Generic[], kind: Kind): string[] {
  const mine = [
    ...sectionsFor(sections, kind),
    ...generics.filter((g) => g.kind === kind && !g.custom),
  ].map((x) => x.icon);
  return [...new Set([...mine, ...Object.keys(icons)])].filter((i) => i in icons);
}

/** The sections a kind of list is grouped in, in their order, Other last. */
export function sectionsFor(sections: Section[], kind: Kind): Section[] {
  const own = sections.filter((s) => s.kind === kind);
  const other = sections.find((s) => s.key === OTHER);
  return other ? [...own, other] : own;
}

/** The catalogue items for a kind of list; items kept before there were kinds are groceries. */
export function genericsFor(generics: Generic[], kind: Kind): Generic[] {
  return generics.filter((g) => (g.kind || GROCERIES) === kind);
}

export const OTHER = 'other';

/** The language to show item names in before someone picks one: their phone's, else Dutch. */
export function guessLang(languages: readonly string[]): Lang {
  for (const l of languages) {
    const code = l.slice(0, 2).toLowerCase();
    if (code === 'nl' || code === 'fr' || code === 'en' || code === 'el') return code;
  }
  return 'nl';
}

/** A generic's name in the language, or in another when it has none there. */
export function genericName(g: Generic, lang: Lang): string {
  return g[`name_${lang}`] || g.name_en || g.name_nl || g.name_fr || g.name_el;
}

export function sectionLabel(s: Section | undefined, lang: Lang): string {
  return s ? s[lang] || s.en : '';
}

const namesOf = (g: Generic, lang: Lang) => [
  { text: genericName(g, lang), own: true },
  ...[g.name_en, g.name_nl, g.name_fr, g.name_el, ...g.aliases.split(';')].map((text) => ({
    text,
    own: false,
  })),
];

/**
 * The generics that match what is being typed, best first: a whole name before
 * the start of a name before the start of a word, names in your own language
 * well before others, your household's own before the catalogue, and short
 * names.
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
      // a match in another language counts less, unless it's the whole name:
      // "mel" is the start of the French "mélange", but a Dutch speaker means melk
      if (s && own) s += 5;
      else if (s && s < 100) s *= 0.3;
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

/**
 * How well a typed word stands for a word of a catalogue name: 2 when it is
 * the word, or the word with a plural ending ("yogurts", "thigh" for
 * "thighs"), 1 when it ends in it, like Dutch compounds do ("aardbeienyoghurt"
 * is a yoghurt), and 0 otherwise.
 */
function wordMatch(typed: string, word: string): number {
  if (typed === word) return 2;
  const [long, short] = typed.length > word.length ? [typed, word] : [word, typed];
  if (short.length >= 3 && long.startsWith(short) && long.length - short.length <= 2) return 2;
  if (word.length >= 4 && typed.endsWith(word)) return 1;
  return 0;
}

/** How well the typed words hold all the words of a name, one after another. */
function wordsMatch(typed: string[], name: string[]): number {
  let best = 0;
  for (let i = 0; i + name.length <= typed.length; i++) {
    let worst = 2;
    for (let j = 0; j < name.length && worst; j++)
      worst = Math.min(worst, wordMatch(typed[i + j], name[j]));
    best = Math.max(best, worst);
  }
  return best;
}

/**
 * The catalogue item a name the household typed is most likely a kind of, for
 * its icon and section: "Yogurt lactose free" is a yoghurt and "chicken breast
 * small" a chicken breast. A name counts when all its words are among the
 * typed ones, in any case and with or without accents; the longest such name
 * is the most specific and wins, then names in your own language.
 */
export function guess(generics: Generic[], typed: string, lang: Lang): Generic | undefined {
  const words = itemKey(typed).split(' ').filter(Boolean);
  if (!words.length) return undefined;
  let best: Generic | undefined;
  let bestScore = 0;
  for (const g of generics) {
    for (const { text, own } of namesOf(g, lang)) {
      const name = itemKey(text);
      if (!name) continue;
      const match = wordsMatch(words, name.split(' '));
      if (!match) continue;
      const score = name.length * 10 + match * 3 + (own ? 2 : 0) + (g.custom ? 1 : 0);
      if (score > bestScore) [best, bestScore] = [g, score];
    }
  }
  return best;
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
