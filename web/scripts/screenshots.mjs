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

const signIn = async (email) =>
  (await api('/api/collections/users/auth-with-password', { body: { identity: email, password } }))
    .token;

// a household of two, with a week's shopping
await api('/api/app/setup', { body: { name: 'Anna', email: 'anna@example.com', password } });
const anna = await signIn('anna@example.com');
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
const ben = await signIn('ben@example.com');

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
  ['Appels', ben],
  ['Pasta', anna],
  ['Olijfolie', ben],
  ['Wc-papier', anna],
  ['Yoghurt', ben],
]) {
  await tick(by, await add(by, groceries.id, name));
}
await add(anna, groceries.id, 'Halfvolle melk', { quantity: '2×' });
await add(anna, groceries.id, 'Bananen');
await add(ben, groceries.id, 'Volkoren brood', { note: 'Sliced, please' });
await add(ben, groceries.id, 'Jonge kaas', { note: 'The Gouda in slices' });
await add(anna, groceries.id, 'Tomaten', { quantity: '500 g' });
await add(ben, groceries.id, 'Koffiebonen');
await add(anna, groceries.id, 'Afwasmiddel');

const bbq = await api('/api/collections/lists/records', {
  token: anna,
  body: { name: 'Weekend BBQ', emoji: '🔥', sort: 1 },
});
for (const name of ['Worstjes', 'Houtskool', 'Maïs']) await tick(ben, await add(ben, bbq.id, name));

const browser = await chromium.launch();
for (const scheme of ['light', 'dark']) {
  const page = await browser.newPage({
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 2,
    colorScheme: scheme,
  });
  const shot = (name) =>
    page.screenshot({ path: `${out}/${name}${scheme === 'dark' ? '-dark' : ''}.png` });

  await page.goto(base);
  await page.getByLabel('Email').fill('anna@example.com');
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.getByText('Halfvolle melk').waitFor();
  await page.evaluate(() => document.fonts.ready);
  await shot('list');

  await page.getByRole('button', { name: 'Change Jonge kaas' }).click();
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

  await page.goto(`${base}/#me`);
  await page.getByText('Hi Anna').waitFor();
  await shot('me');
  await page.close();
}
await browser.close();
console.log(`Screenshots in ${out}`);
