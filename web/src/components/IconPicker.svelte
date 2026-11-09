<script lang="ts">
  import { iconChoices } from '../lib/catalogue';
  import { t } from '../lib/i18n';
  import { household } from '../lib/stores/household.svelte';
  import { catalogue } from '../lib/stores/catalogue.svelte';
  import Glyph from './Glyph.svelte';

  // A grid of line icons to pick from, the open list's own first. With a
  // reset label, a button above it goes back to the automatic one.
  let {
    current,
    reset = '',
    tall = false,
    onpick,
  }: { current: string; reset?: string; tall?: boolean; onpick: (icon: string) => void } = $props();

  const icons = $derived(iconChoices(catalogue.sections, catalogue.generics, household.kind));
</script>

<div class="flex flex-col gap-2">
  {#if reset}
    <button type="button" class="chip self-start" onclick={() => onpick('')}>{reset}</button>
  {/if}
  <div
    class="grid grid-cols-6 gap-1 p-1 sm:grid-cols-8 {tall ? '' : 'max-h-72 overflow-y-auto'}"
    role="listbox"
    aria-label={t('item.icon')}
  >
    {#each icons as icon (icon)}
      <button
        type="button"
        role="option"
        aria-selected={current === icon}
        aria-label={t('item.icon.pick', { icon: icon.replace(/-/g, ' ') })}
        title={icon.replace(/-/g, ' ')}
        class="grid aspect-square place-items-center rounded-lg text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800 {current ===
        icon
          ? 'bg-brand-500/15 text-brand-700 ring-2 ring-brand-500 dark:text-brand-400'
          : ''}"
        onclick={() => onpick(icon)}><Glyph {icon} class="size-6" /></button
      >
    {/each}
  </div>
</div>
