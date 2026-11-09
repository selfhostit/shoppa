<script lang="ts">
  import { onMount } from 'svelte';
  import { guessLang } from '../lib/catalogue';
  import { t } from '../lib/i18n';
  import { errorMessage } from '../lib/pb';
  import { openShared, tickShared, type SharedItem, type SharedList } from '../lib/shares';

  // A list someone shared with a guest: no account, only the link. It looks
  // again every so often, since a guest gets no live updates.
  let { token }: { token: string } = $props();

  const lang = guessLang(navigator.languages ?? [navigator.language]);
  let list = $state<SharedList | null>(null);
  let error = $state('');

  const open = $derived(list?.items.filter((i) => !i.checked) ?? []);
  const bought = $derived(list?.items.filter((i) => i.checked) ?? []);

  async function load() {
    try {
      list = await openShared(token, lang);
      error = '';
    } catch (err) {
      error = (err as { status?: number }).status === 404 ? t('shared.gone') : errorMessage(err);
    }
  }

  async function tick(item: SharedItem) {
    if (!list?.can_tick) return;
    item.checked = !item.checked;
    try {
      await tickShared(token, item.id, item.checked);
    } catch (err) {
      item.checked = !item.checked;
      error = errorMessage(err);
    }
  }

  onMount(() => {
    load();
    const timer = setInterval(load, 20_000);
    const onVisible = () => document.visibilityState === 'visible' && load();
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
    };
  });
</script>

<main class="mx-auto flex min-h-dvh max-w-md flex-col gap-4 px-4 pt-6 pb-10">
  {#if error}
    <p class="text-center" role="alert">{error}</p>
  {/if}
  {#if list}
    <header>
      <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('shared.by')}</p>
      <h1 class="text-2xl font-bold tracking-tight">{list.name}</h1>
    </header>
    {#if open.length}
      <ul class="card divide-y divide-zinc-100 p-0 dark:divide-zinc-800">
        {#each open as item (item.id)}
          {@render row(item)}
        {/each}
      </ul>
    {:else}
      <p class="text-center text-zinc-500">{t('list.empty.title')}</p>
    {/if}
    {#if bought.length}
      <h2 class="font-semibold">{t('shared.bought')}</h2>
      <ul class="card divide-y divide-zinc-100 p-0 opacity-70 dark:divide-zinc-800">
        {#each bought as item (item.id)}
          {@render row(item)}
        {/each}
      </ul>
    {/if}
  {/if}
</main>

{#snippet row(item: SharedItem)}
  <li class="flex items-center gap-3 px-4 py-3">
    {#if list?.can_tick}
      <input
        type="checkbox"
        class="size-6 shrink-0 accent-brand-600"
        checked={item.checked}
        onchange={() => tick(item)}
        aria-label={t('list.tick', { name: item.name })}
      />
    {/if}
    <span class="flex min-w-0 flex-1 flex-col">
      <span class="font-medium {item.checked ? 'line-through' : ''}"
        >{item.name}
        {#if item.quantity}<span class="text-sm text-zinc-500"> · {item.quantity}</span>{/if}</span
      >
      {#if item.note}<span class="truncate text-sm text-zinc-500">{item.note}</span>{/if}
    </span>
  </li>
{/snippet}
