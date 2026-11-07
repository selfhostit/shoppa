<script lang="ts">
  import { photoUrl, type Product } from '../lib/products';
  import { t } from '../lib/i18n';
  import Glyph from './Glyph.svelte';
  import PhotoViewer from './PhotoViewer.svelte';

  // An item's picture: its product's photo, kept on the server, or else its
  // icon on a soft tile. Given a name, a tap on the photo shows it larger.
  let {
    product,
    icon = '',
    size = 'size-10',
    zoom = '',
  }: { product?: Product; icon?: string; size?: string; zoom?: string } = $props();

  let failed = $state(false);
  let viewing = $state(false);
  const src = $derived(product && !failed ? photoUrl(product) : '');
</script>

{#snippet photo()}
  <img
    {src}
    alt=""
    loading="lazy"
    class="{size} shrink-0 rounded-lg bg-white object-contain p-0.5 ring-1 ring-zinc-200 dark:ring-zinc-700"
    onerror={() => (failed = true)}
  />
{/snippet}

{#if src && zoom}
  <button
    type="button"
    class="shrink-0 cursor-zoom-in rounded-lg"
    aria-label={t('photo.show', { name: zoom })}
    onclick={() => (viewing = true)}>{@render photo()}</button
  >
  {#if viewing && product}
    <PhotoViewer {product} name={zoom} onclose={() => (viewing = false)} />
  {/if}
{:else if src}
  {@render photo()}
{:else if icon}
  <span
    class="grid {size} shrink-0 place-items-center rounded-xl bg-brand-500/10 text-xl text-brand-700 dark:bg-brand-400/10 dark:text-brand-400"
    aria-hidden="true"><Glyph {icon} class="size-6" /></span
  >
{/if}
