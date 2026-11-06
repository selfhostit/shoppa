<script lang="ts">
  import { photoUrl, type Product } from '../lib/products';
  import { t } from '../lib/i18n';
  import PhotoViewer from './PhotoViewer.svelte';

  // An item's picture: its product's photo, kept on the server, or else its
  // emoji. Given a name, a tap on the photo shows it larger.
  let {
    product,
    emoji = '',
    size = 'size-10',
    zoom = '',
  }: { product?: Product; emoji?: string; size?: string; zoom?: string } = $props();

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
{:else if emoji}
  <span class="grid {size} shrink-0 place-items-center text-xl" aria-hidden="true">{emoji}</span>
{/if}
