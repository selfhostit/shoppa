<script lang="ts">
  import { onMount } from 'svelte';
  import { t, tn } from '../lib/i18n';
  import { recentlyBought, toBuy, type Item } from '../lib/list';
  import { shop } from '../lib/shop.svelte';
  import ItemSheet from './ItemSheet.svelte';
  import ListPicker from './ListPicker.svelte';

  // The open list: add what you need at the top, tick it off in the shop, and
  // put recently bought things back with one tap.
  let text = $state('');
  let editing = $state<Item | null>(null);
  let picking = $state(false);
  let input: HTMLInputElement;

  const open = $derived(toBuy(shop.items));
  const recent = $derived(recentlyBought(shop.items));

  onMount(() => {
    shop.start();
    return () => shop.stop();
  });

  async function add(e: SubmitEvent) {
    e.preventDefault();
    const typed = text;
    text = '';
    await shop.add(typed);
    if (shop.error) text = typed;
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

{#if shop.error}
  <p class="text-sm text-red-600 dark:text-red-400" role="alert">{shop.error}</p>
{/if}

{#if open.length}
  <ul class="card divide-y divide-zinc-100 p-0 dark:divide-zinc-800">
    {#each open as item (item.id)}
      <li class="flex items-center gap-1 pr-3">
        <button
          class="group grid size-14 shrink-0 place-items-center"
          onclick={() => shop.tick(item)}
          aria-label={t('list.tick', { name: item.name })}
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
          class="flex min-w-0 flex-1 flex-col items-start py-3 text-left"
          onclick={() => (editing = item)}
          aria-label={t('list.edit', { name: item.name })}
        >
          <span class="flex w-full items-baseline gap-2">
            <span class="truncate text-[1.05rem] font-medium">{item.name}</span>
            {#if item.quantity}
              <span
                class="shrink-0 rounded-full bg-brand-500/15 px-2 py-0.5 text-xs font-semibold text-brand-700 dark:text-brand-400"
                >{item.quantity}</span
              >
            {/if}
          </span>
          {#if item.note}
            <span class="truncate text-sm text-zinc-500 dark:text-zinc-400">{item.note}</span>
          {/if}
        </button>
        {#if item.added_by && shop.people.length > 1}
          <span
            class="shrink-0 text-xs text-zinc-400"
            title={t('list.by', { name: shop.nameOf(item.added_by) })}
            >{shop.nameOf(item.added_by)}</span
          >
        {/if}
      </li>
    {/each}
  </ul>
{:else if !shop.loading && shop.list}
  <section class="flex flex-col items-center gap-1 px-6 py-6 text-center">
    <img
      src={recent.length ? '/brand/karel-cheer.svg' : '/brand/karel.svg'}
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
          <button class="chip" onclick={() => shop.putBack(item)}>{item.name}</button>
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
