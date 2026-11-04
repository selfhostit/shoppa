import type { RecordModel } from 'pocketbase';
import type { Generic, Lang } from './catalogue';

export interface Product extends RecordModel {
  barcode: string;
  name: string;
  name_nl: string;
  name_fr: string;
  brand: string;
  quantity: string;
  categories: string;
  image_url: string;
}

/** A product's name in the language, or its main name. */
export function productName(p: Product, lang: Lang): string {
  return (lang === 'nl' || lang === 'fr' ? p[`name_${lang}`] : '') || p.name;
}

/** "Boni · 1 L": what tells two products with the same name apart. */
export function productLine(p: Product): string {
  return [p.brand, p.quantity].filter(Boolean).join(' · ');
}

/** The photo, kept on the server the first time it's shown. */
export function photoUrl(p: Product): string {
  return p.image_url ? `/api/app/products/${p.id}/photo` : '';
}

/**
 * The catalogue item a product is a kind of, from Open Food Facts' categories:
 * Boni halfvolle melk (en:semi-skimmed-milks) is a Melk. The most specific
 * category wins, since Open Food Facts lists them from broad to narrow.
 */
export function kindOf(p: Product, generics: Generic[]): Generic | undefined {
  const tags = p.categories.split(',').map((t) => t.trim());
  for (const tag of tags.reverse()) {
    if (!tag) continue;
    const g = generics.find((g) => g.off_categories.split(',').some((t) => t.trim() === tag));
    if (g) return g;
  }
  return undefined;
}
