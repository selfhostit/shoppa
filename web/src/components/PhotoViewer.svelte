<script lang="ts">
  import { t } from '../lib/i18n';
  import { photoUrl, type Product } from '../lib/products';

  // A product's photo as large as the screen allows, over everything else. A
  // tap anywhere, the close button or Escape closes it.
  let { product, name, onclose }: { product: Product; name: string; onclose: () => void } =
    $props();

  let large = $state(true);
  const src = $derived(photoUrl(product) + (large ? '?size=large' : ''));
  let close: HTMLButtonElement;

  $effect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onclose();
    addEventListener('keydown', onKey);
    document.body.style.overflow = 'hidden';
    close.focus();
    return () => {
      removeEventListener('keydown', onKey);
      document.body.style.overflow = '';
    };
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 z-30 flex flex-col bg-black/90 px-4 pt-[max(1rem,env(safe-area-inset-top))] pb-[max(1.5rem,env(safe-area-inset-bottom))]"
  role="dialog"
  aria-modal="true"
  aria-label={t('photo.title', { name })}
  tabindex="-1"
  onclick={onclose}
>
  <header class="flex items-center gap-2 text-white">
    <button
      bind:this={close}
      class="-ml-2 rounded-full p-2 hover:bg-white/10"
      aria-label={t('close')}
      onclick={onclose}
    >
      <svg viewBox="0 0 24 24" class="size-6" fill="none" stroke="currentColor" stroke-width="2"
        ><path d="M6 6l12 12M18 6L6 18" stroke-linecap="round" /></svg
      >
    </button>
    <h2 class="truncate text-lg font-semibold">{name}</h2>
  </header>
  <div class="flex min-h-0 flex-1 items-center justify-center py-4">
    <img
      {src}
      alt={name}
      class="max-h-full w-full max-w-md rounded-2xl bg-white object-contain p-3"
      onerror={() => (large = false)}
    />
  </div>
</div>
