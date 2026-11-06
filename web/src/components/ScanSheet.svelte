<script lang="ts">
  import { t } from '../lib/i18n';
  import { errorMessage } from '../lib/pb';
  import { productLine, productName, type Product } from '../lib/products';
  import { shop } from '../lib/shop.svelte';
  import BarcodeScanner from './BarcodeScanner.svelte';
  import Sheet from './Sheet.svelte';
  import Thumb from './Thumb.svelte';

  // Scan the empty package at home and it goes on the list as that exact
  // product. One after another: "Scan another" opens the camera again.
  let { onclose }: { onclose: () => void } = $props();

  type State =
    | { kind: 'scan' }
    | { kind: 'looking'; code: string }
    | { kind: 'added'; product: Product }
    | { kind: 'unknown'; code: string }
    | { kind: 'error'; message: string };
  let step = $state<State>({ kind: 'scan' });
  let name = $state('');
  // a new scanner each time, so the camera starts again
  let round = $state(0);

  async function found(code: string) {
    step = { kind: 'looking', code };
    try {
      const product = await shop.lookup(code);
      if (!product) {
        name = '';
        step = { kind: 'unknown', code };
        return;
      }
      await shop.add('', undefined, product);
      step = shop.error ? { kind: 'error', message: shop.error } : { kind: 'added', product };
    } catch (err) {
      step = { kind: 'error', message: errorMessage(err) };
    }
  }

  async function addTyped(e: SubmitEvent) {
    e.preventDefault();
    await shop.add(name);
    if (!shop.error) again();
  }

  function again() {
    round++;
    step = { kind: 'scan' };
  }
</script>

<Sheet title={t('scan.title')} {onclose}>
  {#if step.kind === 'scan'}
    {#key round}
      <BarcodeScanner onfound={found} />
    {/key}
  {:else if step.kind === 'looking'}
    <div class="flex flex-col items-center gap-3 py-10 text-center">
      <img src="/brand/carty-thinking.svg" alt="" class="size-28" />
      <p class="text-zinc-500 dark:text-zinc-400">{t('scan.looking', { code: step.code })}</p>
    </div>
  {:else if step.kind === 'added'}
    <div class="card flex items-center gap-4" role="status">
      <Thumb product={step.product} size="size-16" zoom={productName(step.product, shop.lang)} />
      <div class="min-w-0 flex-1">
        <p class="line-clamp-2 font-semibold">{productName(step.product, shop.lang)}</p>
        <p class="truncate text-sm text-zinc-500 dark:text-zinc-400">
          {productLine(step.product)}
        </p>
        <p class="mt-1 text-sm font-medium text-brand-700 dark:text-brand-400">
          ✓ {t('scan.added')}
        </p>
      </div>
    </div>
    <div class="mt-4 flex flex-col gap-2">
      <button class="btn-primary" onclick={again}>{t('scan.again')}</button>
      <button class="btn-secondary" onclick={onclose}>{t('scan.done')}</button>
    </div>
  {:else if step.kind === 'unknown'}
    <form class="flex flex-col gap-3" onsubmit={addTyped}>
      <p class="text-zinc-600 dark:text-zinc-400">{t('scan.unknown', { code: step.code })}</p>
      <div class="flex gap-2">
        <input
          bind:value={name}
          required
          maxlength="120"
          placeholder={t('scan.unknown.placeholder')}
          aria-label={t('scan.unknown.placeholder')}
        />
        <button class="btn-primary shrink-0" disabled={!name.trim()}>{t('scan.unknown.add')}</button
        >
      </div>
      <button type="button" class="btn-secondary" onclick={again}>{t('scan.again')}</button>
    </form>
  {:else}
    <p class="text-red-600 dark:text-red-400" role="alert">{step.message}</p>
    <button class="btn-secondary mt-4 w-full" onclick={again}>{t('scan.again')}</button>
  {/if}
</Sheet>
