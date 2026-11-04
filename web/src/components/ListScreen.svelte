<script lang="ts">
  import { onMount } from 'svelte';
  import { bySection, sectionLabel, suggest, type Generic } from '../lib/catalogue';
  import { t, tn } from '../lib/i18n';
  import { itemKey, parseEntry, recentlyBought, toBuy, type Item } from '../lib/list';
  import type { Product } from '../lib/products';
  import { productLine, productName } from '../lib/products';
  import { usualFor } from '../lib/pins';
  import { shop } from '../lib/shop.svelte';
  import ItemSheet from './ItemSheet.svelte';
  import Thumb from './Thumb.svelte';
  import ListPicker from './ListPicker.svelte';
  import ScanSheet from './ScanSheet.svelte';

  // The open list: add what you need at the top, tick it off in the shop, and
  // put recently bought things back with one tap.
  let text = $state('');
  let editing = $state<Item | null>(null);
  let picking = $state(false);
  let scanning = $state(false);
  let input: HTMLInputElement;

  const open = $derived(toBuy(shop.items));
  const groups = $derived(bySection(open, shop.sections, (i) => shop.sectionOf(i)));
  const recent = $derived(recentlyBought(shop.items));
  const typed = $derived(parseEntry(text));
  const suggestions = $derived(suggest(shop.generics, typed.name, shop.lang));
  // what is already on the list doesn't need suggesting
  const onList = $derived(new Set(open.map((i) => i.generic).filter(Boolean)));
  const typedIsGeneric = $derived(
    suggestions.some((g) => itemKey(shop.nameOfGeneric(g)) === itemKey(typed.name)),
  );

  // products for what is typed, fetched a moment after typing stops
  let found = $state<Product[]>([]);
  $effect(() => {
    const q = typed.name;
    if (q.length < 2) {
      found = [];
      return;
    }
    const timer = setTimeout(async () => (found = await shop.searchProducts(q)), 250);
    return () => clearTimeout(timer);
  });
  const onListProducts = $derived(new Set(open.map((i) => i.product).filter(Boolean)));
  // the household's own choices for what is suggested come first
  const usual = $derived(usualFor(shop.pins, suggestions));
  const others = $derived(found.filter((p) => !usual.some((u) => u.id === p.id)));

  onMount(() => {
    shop.start();
    return () => shop.stop();
  });

  async function add(e: SubmitEvent) {
    e.preventDefault();
    await addThis();
  }

  async function addThis(generic?: Generic, product?: Product) {
    const was = text;
    text = '';
    found = [];
    await shop.add(was, generic, product);
    if (shop.error) text = was;
    input.focus();
  }
</script>

<header class="flex items-center justify-between gap-3">
  <button
    class="-ml-2 flex min-w-0 items-center gap-2 rounded-xl px-2 py-1 text-left hover:bg-zinc-100 dark:hover:bg-zinc-900"
    onclick={() => (picking = true)}
    aria-label={t('lists.switch')}
  >
    <span class="text-2xl" aria-hidden="true">{shop.list?.emoji || '🛒'}</span>
    <span class="truncate text-2xl font-bold tracking-tight">{shop.list?.name ?? ''}</span>
    <svg
      viewBox="0 0 24 24"
      class="size-5 shrink-0 text-zinc-400"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      ><path d="M7 10l5 5 5-5" stroke-linecap="round" stroke-linejoin="round" /></svg
    >
  </button>
  {#if shop.list}
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
      placeholder={t('list.add')}
      aria-label={t('list.add')}
      maxlength="120"
      enterkeyhint="done"
      autocomplete="off"
      disabled={!shop.list}
    />
    {#if !text.trim()}
      <button
        type="button"
        class="btn-secondary shrink-0 px-3.5"
        onclick={() => (scanning = true)}
        disabled={!shop.list}
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

  {#if typed.name}
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
            <span class="w-7 text-center text-xl" aria-hidden="true">{g.emoji}</span>
            <span class="flex-1">
              <span class="font-medium">{shop.nameOfGeneric(g)}</span>
              {#if typed.quantity}<span class="text-sm text-zinc-500">
                  · {typed.quantity}</span
                >{/if}
            </span>
            {#if onList.has(g.id)}
              <span class="text-xs text-brand-700 dark:text-brand-400">{t('list.onlist')}</span>
            {:else}
              <span class="text-xs text-zinc-400"
                >{sectionLabel(
                  shop.sections.find((s) => s.key === g.section),
                  shop.lang,
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

{#if shop.error}
  <p class="text-sm text-red-600 dark:text-red-400" role="alert">{shop.error}</p>
{/if}

{#each groups as group (group.section?.key ?? 'other')}
  <section>
    <h2
      class="mb-1.5 flex items-center gap-1.5 px-1 text-xs font-semibold tracking-wide text-zinc-500 uppercase dark:text-zinc-400"
    >
      <span aria-hidden="true">{group.section?.emoji ?? '🛒'}</span>
      <!-- in its own language, so Greek capitals lose their accents -->
      <span lang={group.section ? shop.lang : 'en'}
        >{sectionLabel(group.section, shop.lang) || t('list.other')}</span
      >
    </h2>
    <ul class="card divide-y divide-zinc-100 p-0 dark:divide-zinc-800">
      {#each group.items as item (item.id)}
        {@const name = shop.nameOf(item)}
        <li class="flex items-center gap-1 pr-3">
          <button
            class="group grid size-14 shrink-0 place-items-center"
            onclick={() => shop.tick(item)}
            aria-label={t('list.tick', { name })}
          >
            <span
              class="grid size-7 place-items-center rounded-full border-2 border-zinc-300 text-transparent transition group-hover:border-brand-500 group-active:bg-brand-500 group-active:text-white dark:border-zinc-600"
            >
              <svg
                viewBox="0 0 24 24"
                class="size-4"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
                ><path
                  d="M5 12.5l4.5 4.5L19 7.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                /></svg
              >
            </span>
          </button>
          <button
            class="flex min-w-0 flex-1 items-center gap-3 py-3 text-left"
            onclick={() => (editing = item)}
            aria-label={t('list.edit', { name })}
          >
            <Thumb product={shop.productOf(item)} emoji={shop.emojiOf(item)} />
            <span class="flex min-w-0 flex-col">
              <span class="flex w-full items-baseline gap-2">
                <span class="truncate text-[1.05rem] font-medium">{name}</span>
                {#if item.quantity}
                  <span
                    class="shrink-0 rounded-full bg-brand-500/15 px-2 py-0.5 text-xs font-semibold text-brand-700 dark:text-brand-400"
                    >{item.quantity}</span
                  >
                {/if}
              </span>
              {#if shop.lineOf(item)}
                <span class="truncate text-sm text-zinc-500 dark:text-zinc-400"
                  >{shop.lineOf(item)}</span
                >
              {/if}
              {#if item.note}
                <span class="truncate text-sm text-zinc-500 dark:text-zinc-400">{item.note}</span>
              {/if}
            </span>
          </button>
          {#if item.added_by && shop.people.length > 1}
            <span
              class="shrink-0 text-xs text-zinc-400"
              title={t('list.by', { name: shop.personName(item.added_by) })}
              >{shop.personName(item.added_by)}</span
            >
          {/if}
        </li>
      {/each}
    </ul>
  </section>
{/each}

{#if !open.length && !shop.loading && shop.list}
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
          <button class="chip" onclick={() => shop.putBack(item)}>
            {#if shop.emojiOf(item)}
              <span aria-hidden="true">{shop.emojiOf(item)}</span>
            {:else}
              <svg
                viewBox="0 0 24 24"
                class="size-3.5 text-brand-600 dark:text-brand-400"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
                aria-hidden="true"><path d="M12 5v14M5 12h14" stroke-linecap="round" /></svg
              >
            {/if}
            {shop.nameOf(item)}</button
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

{#snippet productRow(p: Product, mine: boolean)}
  <li>
    <button
      class="flex w-full items-center gap-3 px-4 py-2.5 text-left"
      onclick={() => addThis(undefined, p)}
    >
      <Thumb product={p} />
      <span class="flex min-w-0 flex-1 flex-col">
        <span class="truncate font-medium">{productName(p, shop.lang)}</span>
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
