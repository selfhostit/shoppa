import { describe, expect, it } from 'vitest';
import {
  bySection,
  choiceKey,
  exactly,
  genericName,
  guess,
  genericsFor,
  guessLang,
  iconChoices,
  sectionsFor,
  suggest,
  type Generic,
  type Section,
} from './catalogue';
import bundled from '../../../backend/catalogue/generics.json';
import { icons } from './icons';
import type { Kind } from './kinds';
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
    name_el: '',
    aliases: '',
    custom: false,
    ...over,
  }) as Generic;

const generics = [
  g('milk', 'Milk', 'Melk', 'Lait', { aliases: 'halfvolle melk', name_el: 'Γάλα' }),
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
    // Greek, with or without the accent
    expect(suggest(generics, 'γαλ', 'el')[0].slug).toBe('milk');
    expect(exactly(generics, 'γάλα', 'el')?.slug).toBe('milk');
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
    expect(genericName(generics[0], 'el')).toBe('Γάλα');
    expect(genericName(generics[1], 'el')).toBe('Chocolate milk');
    expect(genericName(g('x', '', 'Witloof', ''), 'fr')).toBe('Witloof');
  });

  it('guesses the language from the phone', () => {
    expect(guessLang(['fr-BE', 'nl'])).toBe('fr');
    expect(guessLang(['de-DE', 'en-GB'])).toBe('en');
    expect(guessLang(['de-DE'])).toBe('nl');
    expect(guessLang(['el-GR', 'en'])).toBe('el');
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

describe('guess', () => {
  const food = [
    ...generics,
    g('yoghurt', 'Yoghurt', 'Yoghurt', 'Yaourt', { aliases: 'yogurt', emoji: '🥣' }),
    g('greek-yoghurt', 'Greek yoghurt', 'Griekse yoghurt', 'Yaourt grec'),
    g('chicken', 'Chicken', 'Kip', 'Poulet', { section: 'meat', emoji: '🍗' }),
    g('chicken-breast', 'Chicken breast', 'Kipfilet (vers)', 'Blanc de poulet', {
      section: 'meat',
    }),
    g('chicken-thighs', 'Chicken thighs', 'Kippendijen', 'Cuisses de poulet', { section: 'meat' }),
    g('chickpeas', 'Chickpeas', 'Kikkererwten', 'Pois chiches', { section: 'pantry' }),
    g('eggs', 'Eggs', 'Eieren', 'Œufs', { name_el: 'Αυγά' }),
  ];
  const kind = (typed: string, lang: 'en' | 'nl' | 'fr' | 'el' = 'en') =>
    guess(food, typed, lang)?.slug;

  it('finds the item in a longer name, in any case', () => {
    expect(kind('Yogurt lactose free')).toBe('yoghurt');
    expect(kind('LACTOSE FREE YOGHURT')).toBe('yoghurt');
    expect(kind('yaourt nature', 'fr')).toBe('yoghurt');
  });

  it('prefers the most specific item', () => {
    expect(kind('chicken breast small')).toBe('chicken-breast');
    expect(kind('chicken wings')).toBe('chicken');
    expect(kind('greek yoghurt 0%')).toBe('greek-yoghurt');
    expect(kind('chocolate milk for the kids')).toBe('chocolate-milk');
  });

  it('takes plurals and singulars, accents and compounds', () => {
    expect(kind('yogurts')).toBe('yoghurt');
    expect(kind('chicken thigh')).toBe('chicken-thighs');
    expect(kind('egg')).toBe('eggs');
    expect(kind('αυγα βιολογικα', 'el')).toBe('eggs');
    expect(kind('aardbeienyoghurt', 'nl')).toBe('yoghurt');
  });

  it("doesn't guess from parts of words", () => {
    expect(kind('chickpeas in a tin')).toBe('chickpeas');
    expect(kind('kipcurry', 'nl')).toBeUndefined();
    expect(kind('lactose free')).toBeUndefined();
    expect(kind('')).toBeUndefined();
  });
});

describe('choices', () => {
  it('keeps a typed name in any case, and the exact thing otherwise', () => {
    const item = { product: '', generic: '', name: 'Chicken  Breast, small' };
    expect(choiceKey(item)).toBe('chicken breast small');
    expect(choiceKey({ ...item, name: 'chicken breast small' })).toBe('chicken breast small');
    expect(choiceKey({ ...item, generic: 'g1' })).toBe('generic:g1');
    expect(choiceKey({ ...item, generic: 'g1', product: 'p1' })).toBe('product:p1');
  });

  it('keeps a typed name per kind of list', () => {
    const item = { product: '', generic: '', name: 'Lamp' };
    expect(choiceKey(item, 'groceries')).toBe('lamp');
    expect(choiceKey(item, 'home')).toBe('home:lamp');
    expect(choiceKey({ ...item, generic: 'g1' }, 'home')).toBe('generic:g1');
  });

  it("offers the kind of list's icons first, each once, then every other", () => {
    const sections = [
      { key: 'dairy', kind: 'groceries', icon: 'milk', en: 'Dairy', nl: '', fr: '', el: '' },
      { key: 'home-beds', kind: 'home', icon: 'bed', en: 'Beds', nl: '', fr: '', el: '' },
    ] as Section[];
    const choices = iconChoices(
      sections,
      [
        g('apples', 'Apples', 'Appels', 'Pommes', { kind: 'groceries', icon: 'apple' }),
        g('sofa', 'Sofa', 'Zetel', 'Canapé', { kind: 'home', icon: 'sofa' }),
        g('mine', 'Mine', 'Mijn', 'Le mien', { kind: 'home', icon: 'cat', custom: true }),
        g('old', 'Old', 'Oud', 'Vieux', { kind: 'home', icon: '', emoji: '🦄' }),
      ],
      'home',
    );
    expect(choices.slice(0, 2)).toEqual(['bed', 'sofa']);
    expect(new Set(choices).size).toBe(choices.length);
    expect(choices).toContain('apple');
    expect(choices).not.toContain('🦄');
    expect(choices.length).toBe(Object.keys(icons).length);
  });
});

describe('kinds of list', () => {
  const catalogue = bundled as unknown as {
    sections: Section[];
    items: {
      slug: string;
      kind?: Kind;
      section: string;
      icon: string;
      aliases?: string;
      en: string;
      nl: string;
      fr: string;
      el: string;
    }[];
  };
  const all = catalogue.items.map((it) =>
    g(it.slug, it.en, it.nl, it.fr, {
      kind: it.kind ?? 'groceries',
      section: it.section,
      icon: it.icon,
      name_el: it.el,
      aliases: it.aliases ?? '',
    }),
  );

  it('has an icon for every section and item', () => {
    for (const s of catalogue.sections) expect(icons[s.icon], s.key).toBeDefined();
    for (const it of catalogue.items) expect(icons[it.icon], it.slug).toBeDefined();
  });

  it('gives each kind its own sections, Other last', () => {
    const home = sectionsFor(catalogue.sections, 'home').map((s) => s.key);
    expect(home[0]).toBe('home-furniture');
    expect(home.at(-1)).toBe('other');
    expect(home).not.toContain('dairy');
    expect(sectionsFor(catalogue.sections, 'groceries')[0].key).toBe('produce');
  });

  it('puts furniture typed in Greek in its section', () => {
    const home = genericsFor(all, 'home');
    const section = (typed: string) => guess(home, typed, 'el')?.section;
    expect(section('Αποθηκευτικό έπιπλο')).toBe('home-furniture');
    expect(section('Έπιπλο για είσοδο, παπουτσοθήκη')).toBe('home-furniture');
    expect(section('Χαλί διάδρομος για κουζίνα')).toBe('home-textiles');
    expect(section('Έπιλο μικρό μπάνιο νεροχύτη')).toBe('home-bathroom');
    expect(section('Κουρτίνες σαλόνι')).toBe('home-textiles');
  });

  it('only suggests what fits the kind of list', () => {
    const top = (kind: Kind, typed: string) =>
      suggest(genericsFor(all, kind), typed, 'en', 1)[0]?.slug;
    expect(top('home', 'sofa')).toBe('home-sofa');
    expect(top('groceries', 'sofa')).toBeUndefined();
    expect(top('camping', 'tent')).toBe('camp-tent');
    expect(top('trip', 'passport')).toBe('trip-passport');
  });
});
