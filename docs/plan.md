# Shoppa: architecture and build plan

Shoppa is a shared shopping list for a household, like Bring!, that runs as one
container on your own hardware and installs on a phone. It is made for Belgium: an
item can be linked to the exact product, from Open Food Facts, so whoever does the
shopping knows which milk to buy and sees its photo.

## Generic items and exact products

Most of a list is generic: "melk", "bananen", "brood". Those come from a small
catalogue that ships with the app, each with a Dutch, French and English name, a
section of the store and an emoji. Typing searches it and Open Food Facts at once,
so "melk" shows the generic item first, then milks you can pick by brand, size and
photo. Free text always works too.

A generic item can be **pinned** to a product: "melk" becomes "Boni halfvolle melk
1L". The household's choices are remembered, so the next time someone types "melk"
the pinned product comes first. A generic can have more than one pin, ordered by how
often each was bought.

Scanning the barcode of an empty package puts that exact product on the list.

## The list

- A household has one or more lists, synced live between phones.
- Each item has a quantity ("2×") and a note ("the big one").
- Ticking an item moves it to **recently bought**, where one tap puts it back on the
  list with its product and note. A list holds one record per thing, so adding
  "melk" again brings back the same item instead of a second one.
- Every tick is also kept as a purchase, from day one. Version 2 learns usual items
  and each store's aisle order from that history.

## Offline

The list opens and can be ticked off in a shop with no signal. Changes wait on the
phone and are sent when the server is reachable again, the way Splitwiz's outbox
works, with one change: the phone gives new items a real PocketBase id when it makes
them, so nothing needs renaming after the sync. Ticking is the same whoever does it
first, so two phones ticking the same item never conflict, and a tick made offline
keeps the time it was made (up to a week back). The service worker keeps product
photos, so the list looks the same offline. Signing out forgets the copy on the phone.

## Notifications

"Anna added 3 items to Groceries." The server collects the items a person adds to a
list and sends one notification after two quiet minutes, or five minutes at most.
Each list has its own notification tag, so a newer one replaces the older one on the
phone instead of piling up.

## Food data

- **Open Food Facts:** products sold in Belgium, the Netherlands and Greece, from the
  monthly export, so search and barcodes work without the internet. Shoppa keeps names
  (NL/FR/EN), brand, package size, categories and the small image URL, and no
  nutrients. Unlike Troye it also keeps products without nutrition data, since you
  shop for them all the same. A barcode that isn't in the export is looked up live in
  Open Food Facts, then Open Products Facts and Open Beauty Facts for things that
  aren't food, and kept.
- **Search in your own words.** Belgian products are named in Dutch or French, so a
  product is also found by Open Food Facts' English category tags
  (`en:semi-skimmed-milks`: "milk" finds "Boni halfvolle melk"), and a word that
  names a catalogue item searches its names in the other languages too ("cheese"
  finds "kaas" and "fromage"). Greek works as typed, with or without accents.
- **Product photos stay home.** The server downloads a product's small photo the
  first time a phone shows it and stores it with the product in `pb_data`. Phones
  only ever talk to Shoppa.
- **Generic catalogue:** about 300 items written by hand, food and household, in
  `backend/catalogue/generics.json`. Each lists Open Food Facts categories, so a
  generic can suggest products before anyone has pinned one. NEVO can't be
  redistributed and its names ("Melk halfvol pasteur") are too detailed for a list.
- Every request to Open Food Facts says who is asking, in the User-Agent, as they ask.
- Credits, in the app and the README: "Open Food Facts, https://world.openfoodfacts.org
  (Open Database Licence)" for the data, and "Product photos: Open Food Facts
  contributors, CC BY-SA 3.0" for the images.

### Shared with Troye: `selfhostit/foods`

Troye already reads the Open Food Facts export and looks up barcodes. Rather than
copying that code, the parts that don't depend on an app's own tables move to a Go
module, `github.com/selfhostit/foods`, that both apps import:

- `Food`, with image URL, package size and categories added
- reading the export for a set of countries, and live barcode lookups, with each
  app's own User-Agent; the "has calories" filter becomes a separate check Troye calls
- `Normalize`, the plural stemming and the name scoring for search
- the USDA and NEVO readers

Each app keeps its own collections, how it stores them, and its own ranking: Troye
ranks by what you logged, Shoppa by your pins and purchases. Troye switches to the
module in a PR of its own that changes no behavior.

## Data model

| Collection | Fields |
|---|---|
| `lists` | name, emoji, sort, archived |
| `items` | list, name, generic, product, quantity, note, checked, checked_at, checked_by, added_by |
| `purchases` | item, list, generic, product, by, at |
| `generics` | slug, name_en, name_nl, name_fr, aliases, section, emoji, off_categories, custom |
| `pins` | generic, product, uses, last_used |
| `products` | barcode, name, name_nl, name_fr, brand, quantity, categories, image_url, thumb, search, source |
| `product_imports` | name, version, products, imported, running, error |

One install is one household: everyone signed in sees every list. Each person picks
the language product names show in (`users.lang`); the interface starts in English
with its texts in one place, ready for translations.

## Hosting

The app runs on the Synology DS220+, set up like Flowy and Troye:

- `deploy/install.sh` over SSH, on port **8093**, next to Splitwiz (8090),
  Flowy (8091) and Troye (8092).
- `https://shoppa.<name>.synology.me` through the DSM reverse proxy with the
  existing Let's Encrypt wildcard certificate, so both phones reach it in the shop
  with no VPN app, and push and app install work (they need HTTPS).
- Tailscale stays an option for keeping it off the internet.
- The Open Food Facts export (about a gigabyte, streamed) is read on the 10th of
  each month at 04:30, a week after Troye's, so the NAS never does both at once. A new
  install reads it a minute after it first starts (`SHOPPA_AUTO_IMPORT=off` skips that).

## Build plan

Each step is one PR that leaves the app working.

1. Turn the template into Shoppa: rename, remove the demo, and this plan.
2. Brand: Carty the cart, the icon and colors.
3. Lists and items: lists, adding by free text, quantity and note, ticking off,
   recently bought, live sync, purchases, and the interface texts in one place.
4. Generic catalogue: the bundled items, search, sections and emoji, and adding your
   own.
5. The `selfhostit/foods` module, and Troye's switch to it (a PR in Troye).
6. Products: the monthly import, mixed search results, product photos kept at home,
   live barcode lookups, the credits, and each person's language for product names.
7. Barcode scanning (the browser's own `BarcodeDetector`, which Troye uses).
8. Pins: pin a generic to a product, and pinned products first.
9. Offline: the outbox, and photos kept by the service worker.
10. Notifications, grouped per list and person.
11. README with phone screenshots, the org profile and the website.

### Version 2

- Stores and aisle order: tag items for Colruyt, Delhaize, Aldi, Lidl, Carrefour and
  others, sort the list by a store's sections (set by dragging), and later learn the
  order from how items are ticked off.
- Mealie: add a recipe's ingredients to a list, with a Mealie API token.
- Troye: kcal, Nutri-Score and nutrition on a product card when Troye is installed.
- Usual items and suggestions from purchase history.

## Progress

**Now:** 1.9 push, grouped
**Next:** 1.10 README, screenshots, org profile, website
**Blocked:** nothing

### 0. Agree before code
- [x] 0.1 Read plasto, Troye, Splitwiz, Flowy
- [x] 0.2 Name and mascot chosen: Shoppa, with Carty the cart (#8)
- [x] 0.3 Data model agreed (#7)
- [x] 0.4 Plan for sharing the foods package agreed: `selfhostit/food` (#7)
- [x] 0.5 This plan written (#7)

### 1. Version 1
- [x] 1.1 Rename plasto, strip the demo, deploy skeleton (#7)
- [x] 1.2 Shared food module; Troye still passes its tests (food#1, troye#32)
- [x] 1.3 Lists and items, realtime sync (#10)
- [x] 1.4 Search: generic catalogue + OFF products (#11, #12)
- [x] 1.5 Thumbnails kept on the NAS, with credits (#12)
- [x] 1.6 Barcode scan, pin a generic item to a product (#14, #15)
- [x] 1.7 Check off, recently bought (#10)
- [x] 1.8 Offline (#17)
- [ ] 1.9 Push, grouped (#PR)
- [ ] 1.10 README, screenshots, org profile, website (#PR)
- [ ] 1.11 Running on the Synology, used for a week

### 2. Version 2
- [ ] 2.1 Stores and aisle order
- [ ] 2.2 Mealie: add a recipe's ingredients
- [ ] 2.3 Troye: nutrition on product cards
- [ ] 2.4 Usual items and suggestions

### Decisions
- 2026-11-04: images are downloaded on first use and kept in pb_data
- 2026-11-04: the app is called Shoppa; the mascot is Carty (short for Mr. Shopping Carty), renamed from Karel

### Later
- (ideas found along the way)
