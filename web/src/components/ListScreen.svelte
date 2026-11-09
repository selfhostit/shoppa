<script lang="ts">
  import { onMount } from 'svelte';
  import { bySection, sectionLabel, suggest, type Generic, type Section } from '../lib/catalogue';
  import { t, tn } from '../lib/i18n';
  import { GROCERIES, kindIcon } from '../lib/kinds';
  import { itemKey, parseEntry, recentlyBought, splitEntries, toBuy, type Item } from '../lib/list';
  import type { Product } from '../lib/products';
  import { productLine, productName } from '../lib/products';
  import { usualFor } from '../lib/pins';
  import { shop } from '../lib/shop.svelte';
  import { status } from '../lib/stores/status.svelte';
  import { household } from '../lib/stores/household.svelte';
  import { catalogue } from '../lib/stores/catalogue.svelte';
  import { choices } from '../lib/stores/choices.svelte';
  import { items } from '../lib/stores/items.svelte';
  import { pins } from '../lib/stores/pins.svelte';
  import { trips } from '../lib/stores/trips.svelte';
  import { searchProducts } from '../lib/stores/search';
  import Glyph from './Glyph.svelte';
  import ItemRow from './ItemRow.svelte';
  import ItemSheet from './ItemSheet.svelte';
  import TripBar from './TripBar.svelte';
  import SectionSheet from './SectionSheet.svelte';
  import Thumb from './Thumb.svelte';
  import ListPicker from './ListPicker.svelte';
  import ScanSheet from './ScanSheet.svelte';

  // The open list: add what you need at the top, tick it off in the shop, and
  // put recently bought things back with one tap.
  let text = $state('');
  let editing = $state<Item | null>(null);
  let picking = $state(false);
  let scanning = $state(false);
  // the section whose icon is being picked; null for Other
  let iconFor = $state<{ section: Section | undefined } | null>(null);
  let input: HTMLInputElement;

  const open = $derived(toBuy(items.items));
  const trip = $derived(trips.current);
  const groups = $derived(bySection(open, catalogue.kindSections, (i) => choices.sectionOf(i)));
  // products, barcodes and usual brands are for groceries
  const groceries = $derived(household.kind === GROCERIES);
  const recent = $derived(recentlyBought(items.items));
  // "milk, bread, eggs" or a pasted list is several items
  const entries = $derived(splitEntries(text));
  const many = $derived(entries.length > 1);
  const typed = $derived(parseEntry(many ? '' : text));
  const suggestions = $derived(suggest(catalogue.kindGenerics, typed.name, catalogue.lang));
  // what is already on the list doesn't need suggesting
  const onList = $derived(new Set(open.map((i) => i.generic).filter(Boolean)));
  const typedIsGeneric = $derived(
    suggestions.some((g) => itemKey(catalogue.nameOfGeneric(g)) === itemKey(typed.name)),
  );

  // products for what is typed, fetched a moment after typing stops
  let found = $state<Product[]>([]);
  $effect(() => {
    const q = typed.name;
    if (q.length < 2 || !groceries) {
      found = [];
      return;
    }
    const timer = setTimeout(async () => (found = await searchProducts(q)), 250);
    return () => clearTimeout(timer);
  });
  const onListProducts = $derived(new Set(open.map((i) => i.product).filter(Boolean)));
  // the household's own choices for what is suggested come first
  const usual = $derived(usualFor(pins.pins, suggestions));
  const others = $derived(found.filter((p) => !usual.some((u) => u.id === p.id)));

  onMount(() => {
    shop.start();
    return () => shop.stop();
  });

  async function add(e: SubmitEvent) {
    e.preventDefault();
    if (many) await addAll(entries);
    else await addThis();
  }

  /** A pasted list goes on at once, one item per line; an input would join the lines. */
  async function paste(e: ClipboardEvent) {
    const pasted = splitEntries(e.clipboardData?.getData('text') ?? '');
    if (pasted.length < 2) return;
    e.preventDefault();
    await addAll(pasted);
  }

  async function addAll(all: string[]) {
    const was = text;
    text = '';
    found = [];
    await items.addMany(all);
    if (status.error) text = was;
    input.focus();
  }

  async function addThis(generic?: Generic, product?: Product) {
    const was = text;
    text = '';
    found = [];
    await items.add(was, generic, product);
    if (status.error) text = was;
    input.focus();
  }
</script>

<header class="flex items-center justify-between gap-3">
  <button
    class="-ml-2 flex min-w-0 items-center gap-2 rounded-xl px-2 py-1 text-left hover:bg-zinc-100 dark:hover:bg-zinc-900"
    onclick={() => (picking = true)}
    aria-label={t('lists.switch')}
  >
    <span
      class="grid size-9 shrink-0 place-items-center rounded-xl bg-brand-600 text-white"
      aria-hidden="true"><Glyph icon={kindIcon(household.kind)} class="size-5" /></span
    >
    <span class="truncate text-2xl font-bold tracking-tight">{household.list?.name ?? ''}</span>
    <svg
      viewBox="0 0 24 24"
      class="size-5 shrink-0 text-zinc-400"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      ><path d="M7 10l5 5 5-5" stroke-linecap="round" stroke-linejoin="round" /></svg
    >
  </button>
  {#if household.list}
    <span class="shrink-0 text-sm text-zinc-500 dark:text-zinc-400"
      >{tn('list.items', open.length)}</span
    >
  {/if}
</header>

<div class="relative">
  <form class="flex gap-2" onsubmit={add}>
    <input
      bind:this={input}
      bind:value={text}
      onpaste={paste}
      placeholder={t('list.add')}
      aria-label={t('list.add')}
      maxlength="500"
      enterkeyhint="done"
      autocomplete="off"
      disabled={!household.list}
    />
    {#if !text.trim() && groceries}
      <button
        type="button"
        class="btn-secondary shrink-0 px-3.5"
        onclick={() => (scanning = true)}
        disabled={!household.list}
        aria-label={t('scan.open')}
      >
        <svg
          viewBox="0 0 24 24"
          class="size-5"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          aria-hidden="true"
          ><path
            d="M4 8V5a1 1 0 0 1 1-1h3M16 4h3a1 1 0 0 1 1 1v3M20 16v3a1 1 0 0 1-1 1h-3M8 20H5a1 1 0 0 1-1-1v-3M8 8v8M11 8v8M14 8v8M17 8v8"
          /></svg
        >
      </button>
    {/if}
    <button
      class="btn-primary shrink-0 px-4"
      disabled={!text.trim()}
      aria-label={t('list.add.button')}
    >
      <svg viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2.5"
        ><path d="M12 5v14M5 12h14" stroke-linecap="round" /></svg
      >
    </button>
  </form>

  {#if many}
    <ul
      class="card absolute inset-x-0 top-full z-10 mt-2 p-0 shadow-lg"
      aria-label={t('list.suggestions')}
    >
      <li>
        <button
          class="flex w-full items-center gap-3 px-4 py-3 text-left"
          onclick={() => addAll(entries)}
        >
          <span class="w-7 text-center text-xl text-zinc-400" aria-hidden="true">+</span>
          <span class="min-w-0 flex-1">
            <span class="font-medium">{tn('list.addmany', entries.length)}</span>
            <span class="block truncate text-sm text-zinc-500">{entries.join(' · ')}</span>
          </span>
        </button>
      </li>
    </ul>
  {:else if typed.name}
    <ul
      class="card absolute inset-x-0 top-full z-10 mt-2 max-h-[70dvh] divide-y divide-zinc-100 overflow-y-auto p-0 shadow-lg dark:divide-zinc-800"
      aria-label={t('list.suggestions')}
    >
      {#each usual as p (p.id)}
        {@render productRow(p, true)}
      {/each}
      {#each suggestions as g (g.id)}
        <li>
          <button
            class="flex w-full items-center gap-3 px-4 py-3 text-left"
            onclick={() => addThis(g)}
          >
            <span class="grid w-7 place-items-center text-brand-700 dark:text-brand-400"
              ><Glyph icon={g.icon || g.emoji} class="size-6" /></span
            >
            <span class="flex-1">
              <span class="font-medium">{catalogue.nameOfGeneric(g)}</span>
              {#if typed.quantity}<span class="text-sm text-zinc-500">
                  · {typed.quantity}</span
                >{/if}
            </span>
            {#if onList.has(g.id)}
              <span class="text-xs text-brand-700 dark:text-brand-400">{t('list.onlist')}</span>
            {:else}
              <span class="text-xs text-zinc-400"
                >{sectionLabel(
                  catalogue.kindSections.find((s) => s.key === g.section),
                  catalogue.lang,
                )}</span
              >
            {/if}
          </button>
        </li>
      {/each}
      {#if others.length}
        <li
          class="bg-zinc-50 px-4 pt-3 pb-1.5 text-xs font-semibold tracking-wide text-zinc-500 uppercase dark:bg-zinc-950/40 dark:text-zinc-400"
        >
          {t('list.products')}
        </li>
        {#each others as p (p.id)}
          {@render productRow(p, false)}
        {/each}
      {/if}
      {#if !typedIsGeneric}
        <li>
          <button
            class="flex w-full items-center gap-3 px-4 py-3 text-left"
            onclick={() => addThis()}
          >
            <span class="w-7 text-center text-xl text-zinc-400" aria-hidden="true">+</span>
            <span class="flex-1">{t('list.addtyped', { name: typed.name })}</span>
          </button>
        </li>
      {/if}
    </ul>
  {/if}
</div>

<!-- going shopping is news only for a household of more than one -->
{#if household.list && household.people.length > 1}<TripBar />{/if}

{#if status.offline}
  <p
    class="rounded-xl bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:bg-amber-950/60 dark:text-amber-200"
    role="status"
  >
    {items.outbox.length ? tn('offline.waiting', items.outbox.length) : t('offline.title')}
  </p>
{/if}
{#if status.error}
  <p class="text-sm text-red-600 dark:text-red-400" role="alert">{status.error}</p>
{/if}

{#each groups as group (group.section?.key ?? 'other')}
  {@const label = sectionLabel(group.section, catalogue.lang) || t('list.other')}
  <section>
    <h2 class="mb-1.5 px-1">
      <!-- a tap picks another icon for the section -->
      <button
        class="-mx-1 flex items-center gap-1.5 rounded-lg px-1 py-0.5 text-xs font-semibold tracking-wide text-zinc-500 uppercase hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-900"
        onclick={() => (iconFor = { section: group.section })}
        aria-label={t('section.change', { name: label })}
      >
        <Glyph icon={choices.sectionIcon(group.section)} class="size-4 text-sm" />
        <!-- in its own language, so Greek capitals lose their accents -->
        <span lang={group.section ? catalogue.lang : 'en'}>{label}</span>
      </button>
    </h2>
    <ul class="card divide-y divide-zinc-100 p-0 dark:divide-zinc-800">
      {#each group.items as item (item.id)}
        <ItemRow {item} onedit={() => (editing = item)} {trip} />
      {/each}
    </ul>
  </section>
{/each}

{#if !open.length && !status.loading && household.list}
  <section class="flex flex-col items-center gap-1 px-6 py-6 text-center">
    <img
      src={recent.length ? '/brand/carty-cheer.svg' : '/brand/carty.svg'}
      alt=""
      class="mb-2 size-32"
    />
    <h2 class="text-lg font-semibold">
      {recent.length ? t('list.done.title') : t('list.empty.title')}
    </h2>
    <p class="text-sm text-zinc-500 dark:text-zinc-400">
      {recent.length ? t('list.done.body') : t('list.empty.body')}
    </p>
  </section>
{/if}

{#if recent.length}
  <section class="mt-2">
    <h2 class="font-semibold">{t('list.recent')}</h2>
    <p class="mb-3 text-sm text-zinc-500 dark:text-zinc-400">{t('list.recent.hint')}</p>
    <ul class="flex flex-wrap gap-2">
      {#each recent as item (item.id)}
        <li>
          <button class="chip" onclick={() => items.putBack(item)}>
            <Glyph icon={choices.iconOf(item)} class="size-4 text-brand-700 dark:text-brand-400" />
            {catalogue.nameOf(item)}</button
          >
        </li>
      {/each}
    </ul>
  </section>
{/if}

{#if editing}
  <ItemSheet item={editing} onclose={() => (editing = null)} />
{/if}
{#if picking}
  <ListPicker onclose={() => (picking = false)} />
{/if}
{#if scanning}
  <ScanSheet onclose={() => (scanning = false)} />
{/if}
{#if iconFor}
  <SectionSheet section={iconFor.section} onclose={() => (iconFor = null)} />
{/if}

{#snippet productRow(p: Product, mine: boolean)}
  <li>
    <button
      class="flex w-full items-center gap-3 px-4 py-2.5 text-left"
      onclick={() => addThis(undefined, p)}
    >
      <Thumb product={p} />
      <span class="flex min-w-0 flex-1 flex-col">
        <span class="truncate font-medium">{productName(p, catalogue.lang)}</span>
        <span class="truncate text-sm text-zinc-500 dark:text-zinc-400">{productLine(p)}</span>
      </span>
      {#if mine || onListProducts.has(p.id)}
        <span class="flex shrink-0 flex-col items-end text-xs text-brand-700 dark:text-brand-400">
          {#if mine}<span class="font-medium">★ {t('pin.usual')}</span>{/if}
          {#if onListProducts.has(p.id)}<span>{t('list.onlist')}</span>{/if}
        </span>
      {/if}
    </button>
  </li>
{/snippet}
