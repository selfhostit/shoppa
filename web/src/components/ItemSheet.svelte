<script lang="ts">
  import { OTHER, sectionLabel } from '../lib/catalogue';
  import { t } from '../lib/i18n';
  import type { Item } from '../lib/list';
  import { shop } from '../lib/shop.svelte';
  import PickProduct from './PickProduct.svelte';
  import Sheet from './Sheet.svelte';

  // Pick the exact product for a usual item, change an item's name, quantity
  // and note, or take it off the list. A typed item can be kept as one of the
  // household's usual items.
  let { item, onclose }: { item: Item; onclose: () => void } = $props();

  // svelte-ignore state_referenced_locally
  const shown = shop.nameOf(item);
  // svelte-ignore state_referenced_locally
  let name = $state(shown);
  // svelte-ignore state_referenced_locally
  let quantity = $state(item.quantity);
  // svelte-ignore state_referenced_locally
  let note = $state(item.note);

  let emoji = $state('');
  let section = $state(OTHER);

  async function save(e: SubmitEvent) {
    e.preventDefault();
    const change: Partial<Item> = { quantity: quantity.trim(), note: note.trim() };
    // a new name makes it something else than the generic it was
    if (name.trim() !== shown) Object.assign(change, { name: name.trim(), generic: '' });
    await shop.save(item, change);
    if (!shop.error) onclose();
  }

  async function remember(e: SubmitEvent) {
    e.preventDefault();
    await shop.remember(item, emoji, section);
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
  {#if item.generic && !item.checked}
    <div class="mb-6"><PickProduct {item} onpicked={onclose} /></div>
  {/if}
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

  {#if !item.generic}
    <form class="card mt-6 flex flex-col gap-3" onsubmit={remember}>
      <div>
        <h3 class="font-semibold">{t('item.remember')}</h3>
        <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('item.remember.hint')}</p>
      </div>
      <div class="flex gap-2">
        <label class="flex w-20 flex-col gap-1.5">
          <span class="text-sm font-medium">{t('item.remember.emoji')}</span>
          <input bind:value={emoji} maxlength="16" class="text-center text-xl" placeholder="🛒" />
        </label>
        <label class="flex flex-1 flex-col gap-1.5">
          <span class="text-sm font-medium">{t('item.remember.section')}</span>
          <select bind:value={section}>
            {#each shop.sections as s (s.key)}
              <option value={s.key}>{s.emoji} {sectionLabel(s, shop.lang)}</option>
            {/each}
          </select>
        </label>
      </div>
      <button class="btn-secondary">{t('item.remember.button')}</button>
    </form>
  {/if}
</Sheet>
