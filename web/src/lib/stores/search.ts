import { pb } from '../pb';
import type { Product } from '../products';

// Finding products on the server: by barcode, and by what is being typed.

/** The product with this barcode, or null when nobody knows it. */
export async function lookup(code: string): Promise<Product | null> {
  try {
    return await pb.send<Product>(`/api/app/barcode/${encodeURIComponent(code)}`, {});
  } catch (err) {
    if ((err as { status?: number }).status === 404) return null;
    throw err;
  }
}

/** Products for what is being typed, best first. */
export async function searchProducts(q: string): Promise<Product[]> {
  try {
    return await pb.send<Product[]>('/api/app/search', {
      query: { q, limit: 8 },
      requestKey: 'search',
    });
  } catch {
    // a newer search took its place, or the server can't be reached
    return [];
  }
}
