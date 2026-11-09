import type { RecordModel } from 'pocketbase';
import { pb } from './pb';

// A link that shows one list to someone outside the household, with no
// account (backend/app/shares.go).

export interface Share extends RecordModel {
  list: string;
  token: string;
  can_tick: boolean;
}

/** An item as a guest sees it. */
export type SharedItem = {
  id: string;
  name: string;
  quantity: string;
  note: string;
  checked: boolean;
};

/** A list as a guest sees it. */
export type SharedList = { name: string; kind: string; can_tick: boolean; items: SharedItem[] };

const SHARED = '/shared/';

/** The address a share opens, to send to the guest. */
export function shareUrl(share: Share, origin = location.origin): string {
  return origin + SHARED + share.token;
}

/** The token in a guest's address, or '' when this isn't one. */
export function sharedToken(path = location.pathname): string {
  return path.startsWith(SHARED) ? path.slice(SHARED.length).split('/')[0] : '';
}

export function sharesOf(list: string): Promise<Share[]> {
  return pb
    .collection<Share>('shares')
    .getFullList({ filter: pb.filter('list = {:list}', { list }), sort: 'created' });
}

export function share(list: string, canTick: boolean): Promise<Share> {
  return pb.collection<Share>('shares').create({ list, can_tick: canTick });
}

export function unshare(share: Share): Promise<boolean> {
  return pb.collection('shares').delete(share.id);
}

/** What a guest's link shows, with item names in their language. */
export function openShared(token: string, lang: string): Promise<SharedList> {
  return pb.send<SharedList>(`/api/app/shared/${encodeURIComponent(token)}`, {
    query: { lang },
  });
}

/** Ticks an item off for a guest, or puts it back. */
export function tickShared(token: string, item: string, checked: boolean): Promise<void> {
  return pb.send(`/api/app/shared/${encodeURIComponent(token)}/items/${item}`, {
    method: 'POST',
    body: { checked },
  });
}
