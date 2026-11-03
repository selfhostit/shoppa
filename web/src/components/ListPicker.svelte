<script lang="ts">
  import { t } from '../lib/i18n';
  import { shop } from '../lib/shop.svelte';
  import Sheet from './Sheet.svelte';

  // Switch between the household's lists, start a new one, or rename or
  // delete the open one.
  let { onclose }: { onclose: () => void } = $props();

  let newName = $state('');
  // svelte-ignore state_referenced_locally
  let rename = $state(shop.list?.name ?? '');

  async function pick(id: string) {
    await shop.open(id);
    onclose();
  }

  async function create(e: SubmitEvent) {
    e.preventDefault();
    await shop.createList(newName);
    if (!shop.error) onclose();
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (shop.list) await shop.renameList(shop.list, rename);
  }

  async function remove() {
    const list = shop.list;
    if (!list || !confirm(t('lists.delete.confirm', { name: list.name }))) return;
    await shop.deleteList(list);
    if (!shop.error) onclose();
  }
</script>

<Sheet title={t('lists.title')} {onclose}>
  <div class="flex flex-col gap-6">
    <ul class="card divide-y divide-zinc-100 p-0 dark:divide-zinc-800">
      {#each shop.ordered as list (list.id)}
        <li>
          <button
            class="flex w-full items-center gap-3 px-4 py-3.5 text-left"
            onclick={() => pick(list.id)}
            aria-current={list.id === shop.list?.id ? 'true' : undefined}
          >
            <span class="text-xl" aria-hidden="true">{list.emoji || '🛒'}</span>
            <span class="flex-1 font-medium">{list.name}</span>
            {#if list.id === shop.list?.id}
              <svg
                viewBox="0 0 24 24"
                class="size-5 text-brand-600 dark:text-brand-400"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
                ><path
                  d="M5 12.5l4.5 4.5L19 7.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                /></svg
              >
            {/if}
          </button>
        </li>
      {/each}
    </ul>

    <form class="flex flex-col gap-1.5" onsubmit={create}>
      <label for="new-list" class="text-sm font-medium">{t('lists.new')}</label>
      <div class="flex gap-2">
        <input
          id="new-list"
          bind:value={newName}
          required
          maxlength="60"
          placeholder={t('lists.new.placeholder')}
        />
        <button class="btn-primary shrink-0" disabled={!newName.trim()}>{t('lists.create')}</button>
      </div>
    </form>

    {#if shop.list}
      <form class="flex flex-col gap-1.5" onsubmit={save}>
        <label for="rename-list" class="text-sm font-medium">{t('lists.rename')}</label>
        <div class="flex gap-2">
          <input id="rename-list" bind:value={rename} required maxlength="60" />
          <button
            class="btn-secondary shrink-0"
            disabled={!rename.trim() || rename.trim() === shop.list.name}>{t('item.save')}</button
          >
        </div>
      </form>
      {#if shop.ordered.length > 1}
        <button class="btn-secondary text-red-600 dark:text-red-400" onclick={remove}
          >{t('lists.delete')}</button
        >
      {:else}
        <p class="text-center text-sm text-zinc-500 dark:text-zinc-400">{t('lists.last')}</p>
      {/if}
    {/if}

    {#if shop.error}
      <p class="text-sm text-red-600 dark:text-red-400" role="alert">{shop.error}</p>
    {/if}
  </div>
</Sheet>
