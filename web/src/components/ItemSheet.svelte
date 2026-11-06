<script lang="ts">
  import { sectionLabel, standardIcons } from '../lib/catalogue';
  import { t } from '../lib/i18n';
  import type { Item } from '../lib/list';
  import { shop } from '../lib/shop.svelte';
  import PickProduct from './PickProduct.svelte';
  import Sheet from './Sheet.svelte';

  // Pick the exact product for a usual item, change an item's name, quantity
  // and note, pick its section and icon, or take it off the list. A typed item
  // can be kept as one of the household's usual items.
  let { item, onclose }: { item: Item; onclose: () => void } = $props();

  // svelte-ignore state_referenced_locally
  const shown = shop.nameOf(item);
  // svelte-ignore state_referenced_locally
  let name = $state(shown);
  // svelte-ignore state_referenced_locally
  let quantity = $state(item.quantity);
  // svelte-ignore state_referenced_locally
  let note = $state(item.note);

  let pickingIcon = $state(false);
  const icons = $derived(standardIcons(shop.sections, shop.generics));
  const choice = $derived(shop.choiceOf(item));

  async function pickIcon(emoji: string) {
    pickingIcon = false;
    await shop.pick(item, { emoji });
  }

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
    await shop.remember(item, shop.emojiOf(item), shop.sectionOf(item));
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

  <section class="card mt-6 flex flex-col gap-3" aria-labelledby="item-look">
    <div>
      <h3 id="item-look" class="font-semibold">{t('item.look')}</h3>
      <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('item.look.hint', { name: shown })}</p>
    </div>
    <div class="flex gap-2">
      <div class="flex w-20 flex-col gap-1.5">
        <span class="text-sm font-medium">{t('item.icon')}</span>
        <button
          type="button"
          class="grid h-[3.125rem] place-items-center rounded-xl border border-zinc-300 bg-white text-2xl dark:border-zinc-700 dark:bg-zinc-900"
          aria-label={t('item.icon.change')}
          aria-expanded={pickingIcon}
          onclick={() => (pickingIcon = !pickingIcon)}>{shop.emojiOf(item) || '🛒'}</button
        >
      </div>
      <label class="flex flex-1 flex-col gap-1.5">
        <span class="text-sm font-medium">{t('item.section')}</span>
        <select
          value={shop.sectionOf(item)}
          onchange={(e) => shop.pick(item, { section: e.currentTarget.value })}
        >
          {#each shop.sections as s (s.key)}
            <option value={s.key}>{s.emoji} {sectionLabel(s, shop.lang)}</option>
          {/each}
        </select>
      </label>
    </div>
    {#if pickingIcon}
      <div class="flex flex-col gap-2">
        {#if choice?.emoji}
          <button type="button" class="chip self-start" onclick={() => pickIcon('')}
            >{t('item.icon.auto')}</button
          >
        {/if}
        <div
          class="grid max-h-56 grid-cols-7 gap-1 overflow-y-auto"
          role="listbox"
          aria-label={t('item.icon')}
        >
          {#each icons as icon (icon)}
            <button
              type="button"
              role="option"
              aria-selected={shop.emojiOf(item) === icon}
              aria-label={t('item.icon.pick', { icon })}
              class="grid aspect-square place-items-center rounded-lg text-2xl hover:bg-zinc-100 dark:hover:bg-zinc-800 {shop.emojiOf(
                item,
              ) === icon
                ? 'bg-brand-500/15 ring-2 ring-brand-500'
                : ''}"
              onclick={() => pickIcon(icon)}>{icon}</button
            >
          {/each}
        </div>
      </div>
    {/if}
  </section>

  {#if !item.generic}
    <form class="card mt-6 flex flex-col gap-3" onsubmit={remember}>
      <div>
        <h3 class="font-semibold">{t('item.remember')}</h3>
        <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('item.remember.hint')}</p>
      </div>
      <button class="btn-secondary">{t('item.remember.button')}</button>
    </form>
  {/if}
</Sheet>
