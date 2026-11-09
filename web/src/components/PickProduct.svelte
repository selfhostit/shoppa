<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '../lib/i18n';
  import type { Item } from '../lib/list';
  import { productLine, productName, type Product } from '../lib/products';
  import { status } from '../lib/stores/status.svelte';
  import { catalogue } from '../lib/stores/catalogue.svelte';
  import { pins } from '../lib/stores/pins.svelte';
  import { searchProducts } from '../lib/stores/search';
  import Thumb from './Thumb.svelte';

  // "Melk": which one? The products pinned for this generic item first, then
  // a search, starting with the item's own name. Picking one makes the item
  // that product and remembers it for next time.
  let { item, onpicked }: { item: Item; onpicked: () => void } = $props();

  // svelte-ignore state_referenced_locally
  const generic = item.generic;
  // svelte-ignore state_referenced_locally
  const name = catalogue.nameOf(item);
  let query = $state('');
  let found = $state<Product[]>([]);
  let searched = $state(false);

  const pinned = $derived(pins.pinnedFor(generic));
  const others = $derived(found.filter((p) => !pinned.some((q) => q.id === p.id)));

  async function search(q: string) {
    found = await searchProducts(q || name);
    searched = true;
  }

  onMount(() => {
    search('');
  });

  $effect(() => {
    const q = query.trim();
    if (!q) return;
    const timer = setTimeout(() => search(q), 250);
    return () => clearTimeout(timer);
  });

  async function pick(p: Product) {
    await pins.pin(item, p);
    if (!status.error) onpicked();
  }
</script>

<section class="card flex flex-col gap-3 p-4">
  <div>
    <h3 class="font-semibold">{t('pin.which')}</h3>
    <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('pin.hint', { name })}</p>
  </div>

  <ul class="-mx-1 flex flex-col">
    {#each pinned as p (p.id)}
      <li class="flex items-center">
        {@render row(p, true)}
        <button
          class="grid size-10 shrink-0 place-items-center rounded-full text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-800"
          onclick={() => pins.unpin(generic, p)}
          aria-label={t('pin.forget', { name: productName(p, catalogue.lang) })}
        >
          <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2"
            ><path d="M6 6l12 12M18 6L6 18" stroke-linecap="round" /></svg
          >
        </button>
      </li>
    {/each}
  </ul>

  <input
    type="search"
    bind:value={query}
    placeholder={t('pin.search')}
    aria-label={t('pin.search')}
    autocomplete="off"
  />

  <ul class="-mx-1 flex flex-col">
    {#each others as p (p.id)}
      <li class="flex">{@render row(p, false)}</li>
    {:else}
      {#if searched}
        <li class="px-1 py-2 text-sm text-zinc-500 dark:text-zinc-400">
          {pinned.length ? t('pin.noother') : t('pin.none')}
        </li>
      {/if}
    {/each}
  </ul>
</section>

{#snippet row(p: Product, mine: boolean)}
  <button
    class="flex min-w-0 flex-1 items-center gap-3 rounded-xl px-1 py-2 text-left hover:bg-zinc-50 dark:hover:bg-zinc-800/60"
    onclick={() => pick(p)}
  >
    <Thumb product={p} />
    <span class="flex min-w-0 flex-1 flex-col">
      <span class="truncate font-medium">{productName(p, catalogue.lang)}</span>
      <span class="truncate text-sm text-zinc-500 dark:text-zinc-400">{productLine(p)}</span>
    </span>
    {#if mine}
      <span class="shrink-0 text-xs font-medium text-brand-700 dark:text-brand-400"
        >★ {t('pin.usual')}</span
      >
    {/if}
  </button>
{/snippet}
