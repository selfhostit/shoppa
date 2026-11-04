// Every text the interface shows, in one place, so the app can be translated.
// Use t('key') in components, with {name}-style values filled in from vars.
// A key ending in _one / _other is picked by count with tn().

export const en = {
  close: 'Close',
  'tab.list': 'List',
  'tab.me': 'Me',

  'list.add': 'Add an item',
  'list.add.button': 'Add',
  'list.items_one': '{n} item',
  'list.items_other': '{n} items',
  'list.empty.title': 'Nothing on the list',
  'list.empty.body': 'Add what you need above. It shows up on every phone at once.',
  'list.done.title': 'All done!',
  'list.done.body': 'Everything is in the cart. Karel will keep the bananas warm.',
  'list.recent': 'Recently bought',
  'list.recent.hint': 'Tap to put it back on the list.',
  'list.tick': 'Tick off {name}',
  'list.edit': 'Change {name}',
  'list.by': 'by {name}',
  'list.suggestions': 'Suggestions',
  'list.onlist': 'On the list',
  'list.addtyped': 'Add “{name}”',
  'list.other': 'Other',
  'list.products': 'Products',

  'item.title': 'Item',
  'item.name': 'Name',
  'item.quantity': 'How many',
  'item.quantity.placeholder': '2×, 500 g, a big bag…',
  'item.note': 'Note',
  'item.note.placeholder': 'The big one, the ripe ones…',
  'item.save': 'Save',
  'item.remove': 'Remove from the list',
  'item.bought': 'Bought',
  'item.back': 'Put back on the list',
  'item.remember': 'Keep it as a usual item',
  'item.remember.hint': 'Give it an emoji and a section, and it is suggested when you type.',
  'item.remember.emoji': 'Emoji',
  'item.remember.section': 'Section',
  'item.remember.button': 'Keep it',

  'lists.title': 'Lists',
  'lists.switch': 'Switch list',
  'lists.new': 'New list',
  'lists.new.placeholder': 'Weekend, party, DIY…',
  'lists.create': 'Create',
  'lists.rename': 'Rename',
  'lists.delete': 'Delete this list',
  'lists.delete.confirm': 'Delete {name} and everything on it? This can’t be undone.',
  'lists.last': 'A household needs at least one list.',

  'me.hi': 'Hi {name}',
  'me.signout': 'Sign out',
  'me.install': 'Install Shoppa for quick access from your home screen.',
  'me.install.button': 'Install',
  'me.install.ios': 'To install Shoppa, tap Share in Safari, then Add to Home Screen.',
  'me.lang': 'Item names in',
  'me.lang.hint': 'Everyone sees the items in their own language.',
  'me.signedin': 'Signed in as {email}',

  'products.title': 'Products',
  'products.count': '{n} products from Belgium, the Netherlands and Greece, updated {date}.',
  'products.none':
    'The products from Belgium, the Netherlands and Greece are being downloaded for the first time. This takes a while.',
  'products.running': 'Updating the products from Open Food Facts…',
  'products.update': 'Update now',
  'products.credit': 'Product names and photos come from',
  'products.licence': '(data: Open Database Licence; photos by its contributors: CC BY-SA 3.0).',
  'me.owner': '(owner)',
  'me.admin': 'Manage people and data on the admin page.',
  'me.version': 'Version {version}',
} as const;

export type Key = keyof typeof en;

type Plural<K> = K extends `${infer B}_one` ? B : never;
export type PluralKey = Plural<Key>;

/** The text for key, with {name}-style values filled in. */
export function t(key: Key, vars: Record<string, string | number> = {}): string {
  return en[key].replace(/\{(\w+)\}/g, (all, name: string) =>
    name in vars ? String(vars[name]) : all,
  );
}

const plurals = new Intl.PluralRules('en');

/** The text for a count: key_one or key_other, with {n} filled in. */
export function tn(key: PluralKey, n: number, vars: Record<string, string | number> = {}): string {
  const form = plurals.select(n) === 'one' ? 'one' : 'other';
  return t(`${key}_${form}` as Key, { n, ...vars });
}
