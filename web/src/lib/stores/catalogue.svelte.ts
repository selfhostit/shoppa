import {
  genericName,
  genericsFor,
  guess,
  guessLang,
  sectionsFor,
  type Generic,
  type Lang,
  type Section,
} from '../catalogue';
import type { Kind } from '../kinds';
import { upsert, type Item } from '../list';
import { pb } from '../pb';
import { kindOf, productLine, productName, type Product } from '../products';
import { session } from '../session.svelte';
import { household } from './household.svelte';
import { run } from './status.svelte';

// What things are: the catalogue of generic items and sections, the products
// seen on lists, and what each item is called in your language.
class Catalogue {
  generics = $state<Generic[]>([]);
  sections = $state<Section[]>([]);
  /** the products on the lists, by id, from the items' expand */
  products = $state<Record<string, Product>>({});

  /** The language item names show in for whoever is signed in. */
  get lang(): Lang {
    return session.user?.lang || guessLang(navigator.languages ?? [navigator.language]);
  }

  /** The open list's sections, in their order, Other last. */
  get kindSections(): Section[] {
    return sectionsFor(this.sections, household.kind);
  }

  private kindCache: { from: Generic[]; kind: Kind; generics: Generic[] } | null = null;

  /** The catalogue items for the open list's kind: no sofas on a grocery list. */
  get kindGenerics(): Generic[] {
    const kind = household.kind;
    const c = this.kindCache;
    if (c && c.from === this.generics && c.kind === kind) return c.generics;
    const generics = genericsFor(this.generics, kind);
    this.kindCache = { from: this.generics, kind, generics };
    return generics;
  }

  genericOf(item: Item): Generic | undefined {
    return item.generic ? this.generics.find((g) => g.id === item.generic) : undefined;
  }

  productOf(item: Item): Product | undefined {
    return item.product ? this.products[item.product] : undefined;
  }

  /**
   * The catalogue item an item is, or its product is a kind of, or else what
   * its name sounds like: "chicken breast small" is a chicken breast.
   */
  catalogueItem(item: Item): Generic | undefined {
    const p = this.productOf(item);
    const known = p ? kindOf(p, this.generics) : this.genericOf(item);
    return known ?? this.guessed(p ? productName(p, this.lang) : item.name);
  }

  private guesses = new Map<string, Generic | undefined>();
  private guessedFrom: Generic[] = [];

  private guessed(name: string): Generic | undefined {
    if (this.guessedFrom !== this.generics) {
      this.guesses.clear();
      this.guessedFrom = this.generics;
    }
    const key = `${household.kind}:${this.lang}:${name}`;
    if (!this.guesses.has(key)) this.guesses.set(key, guess(this.kindGenerics, name, this.lang));
    return this.guesses.get(key);
  }

  /**
   * An item's name: its product's or generic's in your language, or what was
   * typed.
   */
  nameOf(item: Item): string {
    const p = this.productOf(item);
    if (p) return productName(p, this.lang);
    const g = this.genericOf(item);
    return g ? genericName(g, this.lang) : item.name;
  }

  /** "Boni · 1 L" for an exact product, nothing otherwise. */
  lineOf(item: Item): string {
    const p = this.productOf(item);
    return p ? productLine(p) : '';
  }

  nameOfGeneric(g: Generic): string {
    return genericName(g, this.lang);
  }

  /** Keeps a product seen on a list. */
  keepProduct(product: Product) {
    this.products = { ...this.products, [product.id]: product };
  }

  /** Keeps the products that come along with items, and returns the items. */
  keep(items: Item[]): Item[] {
    const found = items.map((i) => i.expand?.product as Product | undefined).filter((p) => p);
    if (found.length)
      this.products = { ...this.products, ...Object.fromEntries(found.map((p) => [p!.id, p!])) };
    return items;
  }

  applyGeneric(generic: Generic, deleted = false) {
    this.generics = upsert(this.generics, generic, deleted);
  }

  /** Adds one of the household's own items to the catalogue, for the open list's kind. */
  async addGeneric(name: string, icon: string, section: string): Promise<Generic> {
    const own = /^[a-z0-9-]+$/.test(icon);
    const generic = await pb.collection<Generic>('generics').create({
      [`name_${this.lang}`]: name,
      kind: household.kind,
      ...(own ? { icon } : { emoji: icon }),
      section,
    });
    this.applyGeneric(generic);
    return generic;
  }

  async setLang(lang: Lang) {
    const user = session.user;
    if (!user) return;
    await run(() => pb.collection('users').update(user.id, { lang }));
  }
}

export const catalogue = new Catalogue();
