<script lang="ts">
  import { sectionLabel } from '../lib/catalogue';
  import { t, tn } from '../lib/i18n';
  import { GROCERIES } from '../lib/kinds';
  import type { Item } from '../lib/list';
  import { status } from '../lib/stores/status.svelte';
  import { household } from '../lib/stores/household.svelte';
  import { catalogue } from '../lib/stores/catalogue.svelte';
  import { choices } from '../lib/stores/choices.svelte';
  import { items } from '../lib/stores/items.svelte';
  import { STAPLE_DAYS, staples } from '../lib/stores/staples.svelte';
  import Glyph from './Glyph.svelte';
  import IconPicker from './IconPicker.svelte';
  import PickProduct from './PickProduct.svelte';
  import Sheet from './Sheet.svelte';

  // Pick the exact product for a usual item, change an item's name, quantity
  // and note, pick its section and icon, or take it off the list. A typed item
  // can be kept as one of the household's usual items.
  let { item, onclose }: { item: Item; onclose: () => void } = $props();

  // svelte-ignore state_referenced_locally
  const shown = catalogue.nameOf(item);
  // svelte-ignore state_referenced_locally
  let name = $state(shown);
  // svelte-ignore state_referenced_locally
  let quantity = $state(item.quantity);
  // svelte-ignore state_referenced_locally
  let note = $state(item.note);

  let pickingIcon = $state(false);
  const choice = $derived(choices.choiceOf(item));

  async function pickIcon(icon: string) {
    pickingIcon = false;
    await choices.pick(item, { icon });
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    const change: Partial<Item> = { quantity: quantity.trim(), note: note.trim() };
    // a new name makes it something else than the generic it was
    if (name.trim() !== shown) Object.assign(change, { name: name.trim(), generic: '' });
    await items.save(item, change);
    if (!status.error) onclose();
  }

  async function remember(e: SubmitEvent) {
    e.preventDefault();
    await items.remember(item, choices.iconOf(item), choices.sectionOf(item));
    if (!status.error) onclose();
  }

  async function remove() {
    await items.remove(item);
    if (!status.error) onclose();
  }

  async function putBack() {
    await items.putBack(item);
    if (!status.error) onclose();
  }
</script>

<Sheet title={t('item.title')} {onclose}>
  {#if item.generic && !item.checked && household.kind === GROCERIES}
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
    {#if status.error}
      <p class="text-sm text-red-600 dark:text-red-400" role="alert">{status.error}</p>
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
          class="grid h-[3.125rem] place-items-center rounded-xl border border-zinc-300 bg-white text-2xl text-brand-700 dark:border-zinc-700 dark:bg-zinc-900 dark:text-brand-400"
          aria-label={t('item.icon.change')}
          aria-expanded={pickingIcon}
          onclick={() => (pickingIcon = !pickingIcon)}
          ><Glyph icon={choices.iconOf(item)} class="size-7" /></button
        >
      </div>
      <label class="flex flex-1 flex-col gap-1.5">
        <span class="text-sm font-medium">{t('item.section')}</span>
        <select
          value={choices.sectionOf(item)}
          onchange={(e) => choices.pick(item, { section: e.currentTarget.value })}
        >
          {#each catalogue.kindSections as s (s.key)}
            <option value={s.key}>{sectionLabel(s, catalogue.lang)}</option>
          {/each}
        </select>
      </label>
    </div>
    {#if pickingIcon}
      <IconPicker
        current={choices.iconOf(item)}
        reset={choice?.icon || choice?.emoji ? t('item.icon.auto') : ''}
        onpick={pickIcon}
      />
    {/if}
  </section>

  <section class="card mt-6 flex flex-col gap-3" aria-labelledby="item-repeat">
    <div>
      <h3 id="item-repeat" class="font-semibold">{t('staple.title')}</h3>
      <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('staple.hint', { name: shown })}</p>
    </div>
    <select
      value={staples.of(item)?.every ?? 0}
      onchange={(e) => staples.set(item, Number(e.currentTarget.value))}
      aria-label={t('staple.title')}
    >
      <option value={0}>{t('staple.never')}</option>
      {#each STAPLE_DAYS as days (days)}
        <option value={days}>{tn('staple.weeks', days / 7)}</option>
      {/each}
    </select>
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
