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
  'list.done.body': 'Everything is in the cart. Carty will keep the bananas warm.',
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

  'offline.title': 'Offline. You can still tick things off; they sync when the server is back.',
  'offline.waiting_one':
    'Offline. {n} change waits on this phone and syncs when the server is back.',
  'offline.waiting_other':
    'Offline. {n} changes wait on this phone and sync when the server is back.',

  'scan.title': 'Scan a barcode',
  'scan.open': 'Scan a barcode',
  'scan.point': 'Point the camera at the barcode of the empty package.',
  'scan.nocamera':
    'This browser can’t read barcodes with the camera. Type the number under the barcode instead.',
  'scan.camerafail': 'Couldn’t open the camera. Type the number under the barcode instead.',
  'scan.number': 'Barcode number',
  'scan.lookup': 'Look up',
  'scan.looking': 'Looking up {code}…',
  'scan.added': 'On the list',
  'scan.again': 'Scan another',
  'scan.done': 'Done',
  'scan.unknown':
    'Open Food Facts doesn’t know {code} yet. Type what it is, and it goes on the list as text.',
  'scan.unknown.placeholder': 'What is it?',
  'scan.unknown.add': 'Add',

  'pin.usual': 'Your usual',
  'pin.which': 'Which one?',
  'pin.hint': 'Pick the one you buy. Next time, “{name}” suggests it first.',
  'pin.search': 'Search products',
  'pin.forget': 'Forget {name} as your usual',
  'pin.none': 'No products found for that.',
  'pin.noother': 'No other products found.',

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
  'item.look': 'Section and icon',
  'item.look.hint': 'Kept for {name} every time it is added, on everyone’s phone.',
  'item.section': 'Section',
  'item.icon': 'Icon',
  'item.icon.change': 'Change the icon',
  'item.icon.auto': 'Automatic',
  'item.icon.pick': 'Use {icon}',
  'item.remember': 'Keep it as a usual item',
  'item.remember.hint': 'It is suggested when you type, with this section and icon.',
  'item.remember.button': 'Keep it',

  'kind.groceries': 'Groceries',
  'kind.home': 'Furniture & home',
  'kind.camping': 'Camping',
  'kind.trip': 'Trips & outings',

  'section.title': 'Icon for {name}',
  'section.hint': 'Shown for {name} on every “{kind}” list, on everyone’s phone.',
  'section.change': 'Change the icon for {name}',
  'section.own': 'Its own icon',

  'photo.show': 'See the photo of {name}',
  'photo.title': 'Photo of {name}',

  'lists.title': 'Lists',
  'lists.switch': 'Switch list',
  'lists.new': 'New list',
  'lists.new.placeholder': 'Weekend, party, DIY…',
  'lists.create': 'Create',
  'lists.rename': 'Rename',
  'lists.kind': 'What is it for?',
  'lists.kind.hint': 'Each kind of list has its own sections and icons.',
  'lists.kind.this': 'This list is for',
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
  'me.theme': 'Look',
  'me.theme.hint': 'Light or dark, or like the rest of your phone. Just on this phone.',
  'me.theme.system': 'Like my phone',
  'me.theme.light': 'Light',
  'me.theme.dark': 'Dark',
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
  'me.icons': 'Icons from',
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
