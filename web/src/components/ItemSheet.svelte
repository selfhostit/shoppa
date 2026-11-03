<script lang="ts">
  import { t } from '../lib/i18n';
  import type { Item } from '../lib/list';
  import { shop } from '../lib/shop.svelte';
  import Sheet from './Sheet.svelte';

  // Change an item's name, quantity and note, or take it off the list.
  let { item, onclose }: { item: Item; onclose: () => void } = $props();

  // svelte-ignore state_referenced_locally
  let name = $state(item.name);
  // svelte-ignore state_referenced_locally
  let quantity = $state(item.quantity);
  // svelte-ignore state_referenced_locally
  let note = $state(item.note);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    await shop.save(item, { name: name.trim(), quantity: quantity.trim(), note: note.trim() });
    if (!shop.error) onclose();
  }

  async function remove() {
    await shop.remove(item);
    if (!shop.error) onclose();
  }

  async function putBack() {
    await shop.putBack(item);
    if (!shop.error) onclose();
  }
</script>

<Sheet title={t('item.title')} {onclose}>
  <form class="flex flex-col gap-4" onsubmit={save}>
    <label class="flex flex-col gap-1.5">
      <span class="text-sm font-medium">{t('item.name')}</span>
      <input bind:value={name} required maxlength="120" />
    </label>
    <label class="flex flex-col gap-1.5">
      <span class="text-sm font-medium">{t('item.quantity')}</span>
      <input bind:value={quantity} maxlength="20" placeholder={t('item.quantity.placeholder')} />
    </label>
    <label class="flex flex-col gap-1.5">
      <span class="text-sm font-medium">{t('item.note')}</span>
      <input bind:value={note} maxlength="200" placeholder={t('item.note.placeholder')} />
    </label>
    {#if shop.error}
      <p class="text-sm text-red-600 dark:text-red-400" role="alert">{shop.error}</p>
    {/if}
    <button class="btn-primary">{t('item.save')}</button>
    {#if item.checked}
      <button type="button" class="btn-secondary" onclick={putBack}>{t('item.back')}</button>
    {/if}
    <button type="button" class="btn-secondary text-red-600 dark:text-red-400" onclick={remove}
      >{t('item.remove')}</button
    >
  </form>
</Sheet>
