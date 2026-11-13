<script lang="ts">
  import { t } from '../lib/i18n';
  import { household } from '../lib/stores/household.svelte';
  import { trips } from '../lib/stores/trips.svelte';
  import { told, type Trip } from '../lib/trips';
  import Glyph from './Glyph.svelte';

  // "I'm going shopping": tells the others to add what they need now. While
  // someone shops, everyone sees who and where, and what was added meanwhile
  // is marked new. The shopper sees who got a notification, when starting and
  // when done.
  let starting = $state(false);
  let store = $state('');
  let input = $state<HTMLInputElement>();

  const trip = $derived(trips.current);
  const where = $derived(trip?.store ? t('trip.at', { store: trip.store }) : '');
  /** your trip that just ended, shown for a moment */
  let finished = $state<Trip>();
  let finishedTimer: ReturnType<typeof setTimeout> | undefined;

  const toldNames = (trip: Trip) => told(trip, (id) => household.personName(id));
  const toldLine = (trip: Trip) => {
    const names = toldNames(trip);
    return names ? t('trip.told', { names }) : t('trip.toldNobody');
  };

  async function end() {
    const ended = await trips.end();
    if (!ended) return;
    finished = ended;
    clearTimeout(finishedTimer);
    finishedTimer = setTimeout(() => (finished = undefined), 6000);
  }

  async function start(e: SubmitEvent) {
    e.preventDefault();
    await trips.start(store);
    starting = false;
    store = '';
  }

  $effect(() => {
    if (starting) input?.focus();
  });
</script>

{#if trip && trips.mine}
  <div
    class="flex items-center gap-3 rounded-xl bg-brand-500/10 px-3 py-2 text-sm text-brand-800 dark:text-brand-300"
    role="status"
  >
    <Glyph icon="shopping-cart" class="size-5 shrink-0" />
    <span class="flex-1">
      {t('trip.mine', { where })}
      <span class="mt-0.5 block text-xs opacity-80">{toldLine(trip)}</span>
    </span>
    <button class="btn-secondary shrink-0 px-3 py-1.5 text-sm" onclick={end}>{t('trip.end')}</button
    >
  </div>
{:else if trip}
  <div
    class="flex items-center gap-3 rounded-xl bg-brand-500/10 px-3 py-2 text-sm text-brand-800 dark:text-brand-300"
    role="status"
  >
    <Glyph icon="shopping-cart" class="size-5 shrink-0" />
    <span class="flex-1">{t('trip.other', { name: household.personName(trip.by), where })}</span>
  </div>
{:else if starting}
  <form class="flex gap-2" onsubmit={start}>
    <input
      bind:this={input}
      bind:value={store}
      maxlength="40"
      placeholder={t('trip.store')}
      aria-label={t('trip.store')}
    />
    <button class="btn-primary shrink-0">{t('trip.go')}</button>
  </form>
{:else}
  {#if finished}
    {@const names = toldNames(finished)}
    <p class="px-1 text-sm text-zinc-600 dark:text-zinc-400" role="status">
      {t('trip.done')}
      {#if names}{t('trip.told', { names })}{/if}
    </p>
  {/if}
  <button
    class="-my-1 flex items-center gap-2 self-start rounded-lg px-1 py-1 text-sm font-medium text-brand-700 hover:bg-zinc-100 dark:text-brand-400 dark:hover:bg-zinc-900"
    onclick={() => (starting = true)}
  >
    <Glyph icon="shopping-cart" class="size-4" />{t('trip.start')}
  </button>
{/if}
