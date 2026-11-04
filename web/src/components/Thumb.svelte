<script lang="ts">
  import { photoUrl, type Product } from '../lib/products';

  // An item's picture: its product's photo, kept on the server, or else its
  // emoji.
  let {
    product,
    emoji = '',
    size = 'size-10',
  }: { product?: Product; emoji?: string; size?: string } = $props();

  let failed = $state(false);
  const src = $derived(product && !failed ? photoUrl(product) : '');
</script>

{#if src}
  <img
    {src}
    alt=""
    loading="lazy"
    class="{size} shrink-0 rounded-lg bg-white object-contain p-0.5 ring-1 ring-zinc-200 dark:ring-zinc-700"
    onerror={() => (failed = true)}
  />
{:else if emoji}
  <span class="grid {size} shrink-0 place-items-center text-xl" aria-hidden="true">{emoji}</span>
{/if}
