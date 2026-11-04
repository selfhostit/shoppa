<script lang="ts">
  import { onMount } from 'svelte';
  import type { RecordModel } from 'pocketbase';
  import { t } from '../lib/i18n';
  import { errorMessage, pb } from '../lib/pb';
  import { session } from '../lib/session.svelte';

  // Where the products come from, how many there are and when they were last
  // updated. The owner can update them now instead of waiting for the 10th.
  interface Import extends RecordModel {
    products: number;
    imported: string;
    running: boolean;
    error: string;
  }

  let status = $state<Import | null>(null);
  let error = $state('');

  async function load() {
    try {
      status = await pb.collection<Import>('product_imports').getFirstListItem('name = "off"');
    } catch {
      status = null;
    }
  }

  onMount(() => {
    load();
    let stop: (() => void) | undefined;
    pb.collection<Import>('product_imports')
      .subscribe('*', ({ record }) => (status = record))
      .then((s) => (stop = s))
      .catch(() => {});
    return () => stop?.();
  });

  async function update() {
    error = '';
    try {
      await pb.send('/api/app/products/import', { method: 'POST' });
      if (status) status = { ...status, running: true };
    } catch (err) {
      error = errorMessage(err);
    }
  }
</script>

<section class="card flex flex-col gap-3 text-sm">
  <div>
    <h2 class="text-base font-semibold">{t('products.title')}</h2>
    <p class="text-zinc-500 dark:text-zinc-400">
      {#if status?.running}
        {t('products.running')}
      {:else if status?.imported}
        {t('products.count', {
          n: status.products.toLocaleString(),
          date: new Date(status.imported.replace(' ', 'T')).toLocaleDateString(),
        })}
      {:else}
        {t('products.none')}
      {/if}
    </p>
    {#if status?.error}
      <p class="mt-1 text-red-600 dark:text-red-400">{status.error}</p>
    {/if}
  </div>
  {#if session.isOwner}
    <button class="btn-secondary" disabled={status?.running} onclick={update}
      >{t('products.update')}</button
    >
  {/if}
  {#if error}<p class="text-red-600 dark:text-red-400" role="alert">{error}</p>{/if}
  <p class="text-xs text-zinc-500 dark:text-zinc-400">
    {t('products.credit')}
    <a class="underline" href="https://world.openfoodfacts.org" target="_blank" rel="noreferrer"
      >Open Food Facts</a
    >
    {t('products.licence')}
  </p>
</section>
