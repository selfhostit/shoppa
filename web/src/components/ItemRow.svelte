<script lang="ts">
  import { t } from '../lib/i18n';
  import type { Item } from '../lib/list';
  import { addedDuring, type Trip } from '../lib/trips';
  import { catalogue } from '../lib/stores/catalogue.svelte';
  import { choices } from '../lib/stores/choices.svelte';
  import { household } from '../lib/stores/household.svelte';
  import { items } from '../lib/stores/items.svelte';
  import Glyph from './Glyph.svelte';
  import Thumb from './Thumb.svelte';

  // One thing still to buy: tick it off, or tap it to change it.
  let { item, onedit, trip }: { item: Item; onedit: () => void; trip?: Trip } = $props();

  const name = $derived(catalogue.nameOf(item));
</script>

<li class="flex items-center gap-1 pr-3">
  <button
    class="group grid size-14 shrink-0 place-items-center"
    onclick={() => items.tick(item)}
    aria-label={t('list.tick', { name })}
  >
    <span
      class="grid size-7 place-items-center rounded-full border-2 border-zinc-300 text-transparent transition group-hover:border-brand-500 group-active:bg-brand-500 group-active:text-white dark:border-zinc-600"
    >
      <svg viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="3"
        ><path d="M5 12.5l4.5 4.5L19 7.5" stroke-linecap="round" stroke-linejoin="round" /></svg
      >
    </span>
  </button>
  {#if catalogue.productOf(item)}
    <!-- outside the button that opens the item, so a tap on the photo shows it larger -->
    <span class="mr-2 flex shrink-0">
      <Thumb product={catalogue.productOf(item)} icon={choices.iconOf(item)} zoom={name} />
    </span>
  {/if}
  <button
    class="flex min-w-0 flex-1 items-center gap-3 py-3 text-left"
    onclick={onedit}
    aria-label={t('list.edit', { name })}
  >
    {#if !catalogue.productOf(item)}<Thumb icon={choices.iconOf(item)} />{/if}
    <span class="flex min-w-0 flex-col">
      <span class="flex w-full items-baseline gap-2">
        <span class="truncate text-[1.05rem] font-medium">{name}</span>
        {#if addedDuring(item, trip)}
          <span
            class="shrink-0 rounded-full bg-amber-400/25 px-2 py-0.5 text-xs font-semibold text-amber-800 dark:text-amber-300"
            >{t('trip.new')}</span
          >
        {/if}
        {#if item.quantity}
          <span
            class="shrink-0 rounded-full bg-brand-500/15 px-2 py-0.5 text-xs font-semibold text-brand-700 dark:text-brand-400"
            >{item.quantity}</span
          >
        {/if}
      </span>
      {#if catalogue.lineOf(item)}
        <span class="truncate text-sm text-zinc-500 dark:text-zinc-400"
          >{catalogue.lineOf(item)}</span
        >
      {/if}
      {#if item.note}
        <span class="truncate text-sm text-zinc-500 dark:text-zinc-400">{item.note}</span>
      {/if}
    </span>
  </button>
  {#if item.claimed_by}
    <span
      class="flex shrink-0 items-center gap-1 rounded-full bg-sky-500/15 px-2 py-0.5 text-xs font-semibold text-sky-800 dark:text-sky-300"
      title={t('claim.by', { name: household.personName(item.claimed_by) })}
      ><Glyph icon="hand" class="size-3.5" />{household.personName(item.claimed_by)}</span
    >
  {:else if item.added_by && household.people.length > 1}
    <span
      class="shrink-0 text-xs text-zinc-400"
      title={t('list.by', { name: household.personName(item.added_by) })}
      >{household.personName(item.added_by)}</span
    >
  {/if}
</li>
