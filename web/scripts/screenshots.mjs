// Takes the phone screenshots for the README and pull requests, in light and
// dark, from a fresh app filled with a household's list. CI runs it against
// the built app; to run it yourself:
//
//   make build && dist/shoppa serve --dir=/tmp/shoppa-shots &
//   cd web && npm i --no-save playwright && npx playwright install chromium
//   node scripts/screenshots.mjs ../docs/images/screens
import { mkdirSync } from 'node:fs';
import { chromium } from 'playwright';

const base = process.env.SHOPPA_URL ?? 'http://127.0.0.1:8090';
const out = process.argv[2] ?? 'screenshots';
const password = 'screenshots123';
mkdirSync(out, { recursive: true });

async function api(path, { token, body, method = body ? 'POST' : 'GET' } = {}) {
  const res = await fetch(base + path, {
    method,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: token } : {}) },
    body: body && JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`${method} ${path}: ${res.status} ${await res.text()}`);
  return res.status === 204 ? null : res.json();
}

const signIn = (email) =>
  api('/api/collections/users/auth-with-password', { body: { identity: email, password } });

// a household of two, with a week's shopping; Anna sees item names in Dutch,
// Ben in French
await api('/api/app/setup', { body: { name: 'Anna', email: 'anna@example.com', password } });
const owner = await signIn('anna@example.com');
const anna = owner.token;
await api('/api/collections/users/records', {
  token: anna,
  body: {
    name: 'Ben',
    email: 'ben@example.com',
    password,
    passwordConfirm: password,
    role: 'member',
  },
});
const member = await signIn('ben@example.com');
const ben = member.token;
for (const [who, lang] of [
  [owner, 'nl'],
  [member, 'fr'],
]) {
  await api(`/api/collections/users/records/${who.record.id}`, {
    token: who.token,
    method: 'PATCH',
    body: { lang },
  });
}

const [groceries] = (await api('/api/collections/lists/records', { token: anna })).items;
const add = (token, list, name, extra = {}) =>
  api('/api/app/items', { token, body: { list, name, ...extra } });
const tick = (token, item) =>
  api(`/api/collections/items/records/${item.id}`, {
    token,
    method: 'PATCH',
    body: { checked: true },
  });

for (const [name, by] of [
  ['Eieren', anna],
  ['Pommes', ben],
  ['Pasta', anna],
  ['Huile d’olive', ben],
  ['Wc-papier', anna],
  ['Yoghurt', ben],
]) {
  await tick(by, await add(by, groceries.id, name));
}
await add(anna, groceries.id, 'Melk', { quantity: '2×' });
await add(anna, groceries.id, 'Bananen');
await add(ben, groceries.id, 'Pain complet', { note: 'Sliced, please' });
await add(ben, groceries.id, 'Fromage', { note: 'Jonge Gouda in plakjes' });
await add(anna, groceries.id, 'Tomaten', { quantity: '500 g' });
await add(ben, groceries.id, 'Café en grains');
await add(anna, groceries.id, 'Afwasmiddel');
await add(anna, groceries.id, 'Kattengras');

// a few well-known products, looked up in Open Food Facts the way a scan is;
// the ones it knows go on the list with their photo
const found = [];
for (const code of ['5449000000996', '3017620422003', '5410126116953', '5411188112709']) {
  try {
    found.push(await api(`/api/app/barcode/${code}`, { token: anna }));
  } catch (err) {
    console.warn(`${code}: ${err.message}`);
  }
}
for (const p of found.slice(0, 3)) await add(anna, groceries.id, '', { product: p.id });

// the household's usual chocolate spread is Nutella: pinned once, from a
// "choco" item on a scratch list
const nutella = found.find((p) => /nutella/i.test(p.name));
if (nutella) {
  const scratch = await api('/api/collections/lists/records', {
    token: anna,
    body: { name: 'Scratch', sort: 9 },
  });
  const choco = await add(anna, scratch.id, 'choco');
  await api(`/api/app/items/${choco.id}/pin`, { token: anna, body: { product: nutella.id } });
  await api(`/api/collections/lists/records/${scratch.id}`, { token: anna, method: 'DELETE' });
  // and Ben wants choco again, without saying which
  await add(ben, groceries.id, 'pâte à tartiner');
}

const bbq = await api('/api/collections/lists/records', {
  token: anna,
  body: { name: 'Weekend BBQ', emoji: '🔥', sort: 1 },
});
for (const name of ['Worsten', 'Houtskool', 'Maïs']) await tick(ben, await add(ben, bbq.id, name));

const browser = await chromium.launch();
for (const scheme of ['light', 'dark']) {
  const page = await browser.newPage({
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 2,
    colorScheme: scheme,
  });
  // product photos are downloaded by the server the first time they're shown
  const photos = () =>
    page
      .waitForFunction(() => [...document.images].every((i) => i.complete), null, {
        timeout: 15000,
      })
      .catch(() => console.warn('some photos did not load'));
  const shot = async (name) => {
    await photos();
    await page.screenshot({ path: `${out}/${name}${scheme === 'dark' ? '-dark' : ''}.png` });
  };

  page.on('response', (res) => {
    if (res.url().includes('/photo') && res.status() >= 400)
      console.warn(`photo ${res.status()}: ${res.url()}`);
  });
  await page.goto(base);
  await page.getByLabel('Email').fill('anna@example.com');
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.getByRole('button', { name: 'Tick off Melk' }).waitFor();
  await page.evaluate(() => document.fonts.ready);
  await shot('list');

  await page.getByLabel('Add an item').fill('mel');
  await page.getByRole('list', { name: 'Suggestions' }).waitFor();
  await shot('suggest');
  if (found.length) {
    // English finds the Belgian products through their categories
    await page.getByLabel('Add an item').fill('spread');
    await page
      .getByText('Products')
      .waitFor({ timeout: 5000 })
      .catch(() => {});
    await page.waitForTimeout(800);
    await shot('products');
  }
  if (nutella) {
    // the usual comes first
    await page.getByLabel('Add an item').fill('choco');
    await page.getByText('Your usual').first().waitFor();
    await shot('usual');
  }
  await page.getByLabel('Add an item').fill('');

  if (found.length) {
    // scanning: no camera here, so the number under the barcode is typed
    await page.getByRole('button', { name: 'Scan a barcode' }).click();
    await page.getByLabel('Barcode number').fill(found.at(-1).barcode);
    await page.getByRole('button', { name: 'Look up' }).click();
    await page.getByRole('status').waitFor();
    await shot('scan');
    await page.getByRole('button', { name: 'Done' }).click();
  }

  if (nutella) {
    // which one? the usual first, then the rest
    await page.getByRole('button', { name: 'Change Choco' }).click();
    await page.getByText('Which one?').waitFor();
    await page.waitForTimeout(600);
    await shot('pick');
    await page.getByRole('button', { name: 'Close' }).click();
  }

  await page.getByRole('button', { name: 'Change Kaas' }).click();
  await page.getByRole('dialog').waitFor();
  await shot('item');
  await page.getByRole('button', { name: 'Close' }).click();

  await page.getByRole('button', { name: 'Switch list' }).click();
  await page.getByRole('dialog').waitFor();
  await shot('lists');
  await page.getByRole('button', { name: /Weekend BBQ/ }).click();
  await page.getByText('All done!').waitFor();
  await shot('done');
  // back to the groceries for the next scheme
  await page.getByRole('button', { name: 'Switch list' }).click();
  await page.getByRole('button', { name: /Groceries/ }).click();

  // in a shop with no signal: the list still opens, and ticks and new items
  // wait on the phone until the server is back
  await page.getByRole('button', { name: 'Tick off Melk' }).waitFor();
  await page.waitForTimeout(500);
  await page.context().setOffline(true);
  await page.reload();
  await page.getByRole('button', { name: 'Tick off Afwasmiddel' }).click();
  await page.getByLabel('Add an item').fill('Brood');
  await page.getByLabel('Add an item').press('Enter');
  await page.getByText('2 changes wait on this phone').waitFor();
  await page.getByLabel('Add an item').blur();
  await page.evaluate(() => scrollTo(0, 0));
  await shot('offline');
  await page.context().setOffline(false);
  await page.evaluate(() => dispatchEvent(new Event('online')));
  await page.getByText('wait on this phone').waitFor({ state: 'detached' });
  // they reached the server; put the list back as it was for the next scheme
  const synced = (
    await api(`/api/collections/items/records?perPage=200&filter=list='${groceries.id}'`, {
      token: anna,
    })
  ).items;
  const dish = synced.find((i) => i.name === 'Afwasmiddel');
  const bread = synced.find((i) => i.name === 'Brood');
  if (!dish?.checked || !bread) throw new Error('the changes made offline did not sync');
  await api(`/api/collections/items/records/${dish.id}`, {
    token: anna,
    method: 'PATCH',
    body: { checked: false },
  });
  await api(`/api/collections/items/records/${bread.id}`, { token: anna, method: 'DELETE' });

  await page.goto(`${base}/#me`);
  await page.getByText('Hi Anna').waitFor();
  await shot('me');
  await page.close();
}

// the same list on Ben's phone, in French
const page = await browser.newPage({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2 });
await page.goto(base);
await page.getByLabel('Email').fill('ben@example.com');
await page.getByLabel('Password').fill(password);
await page.getByRole('button', { name: 'Sign in' }).click();
await page.getByRole('button', { name: 'Tick off Lait' }).first().waitFor();
await page.screenshot({ path: `${out}/list-fr.png` });

// and in Greek
await api(`/api/collections/users/records/${member.record.id}`, {
  token: ben,
  method: 'PATCH',
  body: { lang: 'el' },
});
await page.reload();
await page.getByRole('button', { name: 'Tick off Γάλα' }).waitFor();
await page.waitForFunction(() => [...document.images].every((i) => i.complete)).catch(() => {});
await page.screenshot({ path: `${out}/list-el.png` });
await page.close();
await browser.close();
console.log(`Screenshots in ${out}`);
