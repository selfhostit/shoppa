import { describe, expect, it } from 'vitest';
import {
  bySection,
  exactly,
  genericName,
  guessLang,
  suggest,
  type Generic,
  type Section,
} from './catalogue';
import type { Item } from './list';

const g = (slug: string, en: string, nl: string, fr: string, over: Partial<Generic> = {}) =>
  ({
    id: slug,
    slug,
    section: 'dairy',
    emoji: '🥛',
    name_en: en,
    name_nl: nl,
    name_fr: fr,
    aliases: '',
    custom: false,
    ...over,
  }) as Generic;

const generics = [
  g('milk', 'Milk', 'Melk', 'Lait', { aliases: 'halfvolle melk' }),
  g('chocolate-milk', 'Chocolate milk', 'Chocomelk', 'Lait chocolaté'),
  g('buttermilk', 'Buttermilk', 'Karnemelk', 'Babeurre'),
  g('melon', 'Melon', 'Meloen', 'Melon', { section: 'produce' }),
  g('salad-mix', 'Salad mix', 'Slamix', 'Mélange de salades', { section: 'produce' }),
  g('grapes', 'Grapes', 'Druiven', 'Raisins', { section: 'produce' }),
  g('raisins', 'Raisins', 'Rozijnen', 'Raisins secs', { section: 'produce' }),
];

describe('suggest', () => {
  it('puts a whole name first, then names that start with what is typed', () => {
    expect(suggest(generics, 'melk', 'nl').map((x) => x.slug)).toEqual([
      'milk',
      'chocolate-milk',
      'buttermilk',
    ]);
    // equally good: the shorter name first
    expect(
      suggest(generics, 'mel', 'nl')
        .map((x) => x.slug)
        .slice(0, 2),
    ).toEqual(['milk', 'melon']);
  });

  it('ranks matches in other languages after your own', () => {
    // the French "mélange" starts with mel too, but a Dutch speaker means melk
    expect(suggest(generics, 'mel', 'nl').map((x) => x.slug)).toEqual([
      'milk',
      'melon',
      'chocolate-milk',
      'buttermilk',
      'salad-mix',
    ]);
    expect(
      suggest(generics, 'mel', 'fr')
        .map((x) => x.slug)
        .slice(0, 2),
    ).toEqual(['melon', 'salad-mix']);
  });

  it('finds names in every language, and other names', () => {
    expect(suggest(generics, 'lait', 'nl')[0].slug).toBe('milk');
    expect(suggest(generics, 'halfvolle', 'en')[0].slug).toBe('milk');
    expect(suggest(generics, 'xyz', 'en')).toEqual([]);
    expect(suggest(generics, '  ', 'en')).toEqual([]);
  });

  it('prefers a name in your own language when two clash', () => {
    expect(exactly(generics, 'raisins', 'fr')?.slug).toBe('grapes');
    expect(exactly(generics, 'raisins', 'en')?.slug).toBe('raisins');
    expect(exactly(generics, 'melk', 'en')?.slug).toBe('milk');
    expect(exactly(generics, 'mel', 'nl')).toBeUndefined();
  });

  it('puts your household’s own items first', () => {
    const own = g('custom-melk-abcd', 'Melk van de boer', 'Melk van de boer', 'Melk van de boer', {
      custom: true,
    });
    expect(suggest([...generics, own], 'melk v', 'nl')[0].slug).toBe('custom-melk-abcd');
  });
});

describe('names and sections', () => {
  it('shows a name in your language, else another', () => {
    expect(genericName(generics[0], 'fr')).toBe('Lait');
    expect(genericName(g('x', '', 'Witloof', ''), 'fr')).toBe('Witloof');
  });

  it('guesses the language from the phone', () => {
    expect(guessLang(['fr-BE', 'nl'])).toBe('fr');
    expect(guessLang(['de-DE', 'en-GB'])).toBe('en');
    expect(guessLang(['de-DE'])).toBe('nl');
  });

  it('groups the list by section in store order', () => {
    const sections = [
      { key: 'produce', emoji: '🥦', en: 'Fruit', nl: 'Fruit', fr: 'Fruits' },
      { key: 'dairy', emoji: '🥛', en: 'Dairy', nl: 'Zuivel', fr: 'Laitiers' },
      { key: 'other', emoji: '🛒', en: 'Other', nl: 'Overige', fr: 'Autres' },
    ] as Section[];
    const item = (name: string, section: string) =>
      ({ id: name, name, section }) as unknown as Item;
    const groups = bySection(
      [
        item('Melk', 'dairy'),
        item('Kattengras', ''),
        item('Appels', 'produce'),
        item('Kaas', 'dairy'),
      ],
      sections,
      (i) => (i as unknown as { section: string }).section,
    );
    expect(groups.map((gr) => [gr.section?.key, gr.items.map((i) => i.name)])).toEqual([
      ['produce', ['Appels']],
      ['dairy', ['Melk', 'Kaas']],
      ['other', ['Kattengras']],
    ]);
  });
});
