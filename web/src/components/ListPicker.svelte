<script lang="ts">
  import { t } from '../lib/i18n';
  import { GROCERIES, kindIcon, kindOfList, kinds, type Kind } from '../lib/kinds';
  import { shop } from '../lib/shop.svelte';
  import { status } from '../lib/stores/status.svelte';
  import { household } from '../lib/stores/household.svelte';
  import Glyph from './Glyph.svelte';
  import ShareList from './ShareList.svelte';
  import Sheet from './Sheet.svelte';

  // Switch between the household's lists, start a new one for groceries,
  // furniture, camping or a trip, or rename, change or delete the open one.
  let { onclose }: { onclose: () => void } = $props();

  let newName = $state('');
  let newKind = $state<Kind>(GROCERIES);
  // the list a new one starts from, with all its items; empty for none
  let fromId = $state('');
  const from = $derived(household.ordered.find((l) => l.id === fromId));
  // svelte-ignore state_referenced_locally
  let rename = $state(household.list?.name ?? '');

  async function pick(id: string) {
    await shop.open(id);
    onclose();
  }

  async function create(e: SubmitEvent) {
    e.preventDefault();
    await shop.createList(newName, newKind, from);
    if (!status.error) onclose();
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (household.list) await household.rename(household.list, rename);
  }

  async function remove() {
    const list = household.list;
    if (!list || !confirm(t('lists.delete.confirm', { name: list.name }))) return;
    await shop.deleteList(list);
    if (!status.error) onclose();
  }
</script>

<Sheet title={t('lists.title')} {onclose}>
  <div class="flex flex-col gap-6">
    <ul class="card divide-y divide-zinc-100 p-0 dark:divide-zinc-800">
      {#each household.ordered as list (list.id)}
        <li>
          <button
            class="flex w-full items-center gap-3 px-4 py-3.5 text-left"
            onclick={() => pick(list.id)}
            aria-current={list.id === household.list?.id ? 'true' : undefined}
          >
            <span class="text-brand-700 dark:text-brand-400"
              ><Glyph icon={kindIcon(kindOfList(list))} class="size-6" /></span
            >
            <span class="flex-1 font-medium">{list.name}</span>
            {#if list.id === household.list?.id}
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
      <select bind:value={fromId} aria-label={t('lists.from')}>
        <option value="">{t('lists.from.empty')}</option>
        {#each household.ordered as list (list.id)}
          <option value={list.id}>{t('lists.from.list', { name: list.name })}</option>
        {/each}
      </select>
      {#if !from}
        {@render kindChips(newKind, (k) => (newKind = k), t('lists.kind'))}
      {/if}
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

    {#if household.list}
      <form class="flex flex-col gap-1.5" onsubmit={save}>
        <label for="rename-list" class="text-sm font-medium">{t('lists.rename')}</label>
        <div class="flex gap-2">
          <input id="rename-list" bind:value={rename} required maxlength="60" />
          <button
            class="btn-secondary shrink-0"
            disabled={!rename.trim() || rename.trim() === household.list.name}
            >{t('item.save')}</button
          >
        </div>
      </form>
      <div class="flex flex-col gap-1.5">
        <span class="text-sm font-medium">{t('lists.kind.this')}</span>
        {@render kindChips(
          household.kind,
          (k) => household.list && household.setKind(household.list, k),
          t('lists.kind.this'),
        )}
        <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('lists.kind.hint')}</p>
      </div>
      <ShareList list={household.list} />
      {#if household.ordered.length > 1}
        <button class="btn-secondary text-red-600 dark:text-red-400" onclick={remove}
          >{t('lists.delete')}</button
        >
      {:else}
        <p class="text-center text-sm text-zinc-500 dark:text-zinc-400">{t('lists.last')}</p>
      {/if}
    {/if}

    {#if status.error}
      <p class="text-sm text-red-600 dark:text-red-400" role="alert">{status.error}</p>
    {/if}
  </div>
</Sheet>

{#snippet kindChips(current: Kind, pick: (kind: Kind) => void, label: string)}
  <div class="flex flex-wrap gap-2" role="radiogroup" aria-label={label}>
    {#each kinds as k (k.id)}
      <button
        type="button"
        class="chip {current === k.id ? 'chosen' : ''}"
        role="radio"
        aria-checked={current === k.id}
        onclick={() => pick(k.id)}><Glyph icon={k.icon} class="size-4" />{t(k.label)}</button
      >
    {/each}
  </div>
{/snippet}
