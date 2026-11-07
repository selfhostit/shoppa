import type { Key } from './i18n';

/**
 * What a list is for. Each kind has its own sections and catalogue items
 * (backend/catalogue/generics.json), so a sofa on a furniture list goes under
 * Furniture. Products, barcodes and usual brands are only for groceries.
 */
export type Kind = 'groceries' | 'home' | 'camping' | 'trip';

export const GROCERIES: Kind = 'groceries';

export const kinds: { id: Kind; icon: string; label: Key }[] = [
  { id: 'groceries', icon: 'shopping-cart', label: 'kind.groceries' },
  { id: 'home', icon: 'sofa', label: 'kind.home' },
  { id: 'camping', icon: 'tent', label: 'kind.camping' },
  { id: 'trip', icon: 'luggage', label: 'kind.trip' },
];

/** A list's kind: groceries for one made before there were kinds. */
export function kindOfList(list: { kind?: string } | undefined): Kind {
  return kinds.find((k) => k.id === list?.kind)?.id ?? GROCERIES;
}

export function kindIcon(kind: Kind): string {
  return kinds.find((k) => k.id === kind)?.icon ?? 'shopping-cart';
}
