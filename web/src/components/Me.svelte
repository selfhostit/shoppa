<script lang="ts">
  import { t } from '../lib/i18n';
  import { langs, type Lang } from '../lib/catalogue';
  import { install } from '../lib/install.svelte';
  import { shop } from '../lib/shop.svelte';
  import { pb } from '../lib/pb';
  import { session } from '../lib/session.svelte';
  import Notifications from './Notifications.svelte';

  // Your account, installing the app and notifications.
  const user = $derived(session.user!);
  const version = pb
    .send<{ version: string }>('/api/app/info', {})
    .then((i) => i.version)
    .catch(() => '');
</script>

<header class="flex items-center justify-between">
  <h1 class="text-2xl font-bold tracking-tight">{t('me.hi', { name: user.name || user.email })}</h1>
  <button class="btn-secondary px-3 py-2 text-sm" onclick={() => session.logout()}
    >{t('me.signout')}</button
  >
</header>

{#if install.canPrompt}
  <section class="card flex items-center justify-between gap-4">
    <p>{t('me.install')}</p>
    <button class="btn-primary shrink-0" onclick={() => install.install()}
      >{t('me.install.button')}</button
    >
  </section>
{:else if install.needsIosSteps}
  <section class="card"><p>{t('me.install.ios')}</p></section>
{/if}

<section class="card flex flex-col gap-3">
  <div>
    <h2 class="font-semibold">{t('me.lang')}</h2>
    <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('me.lang.hint')}</p>
  </div>
  <div class="flex flex-wrap gap-2" role="radiogroup" aria-label={t('me.lang')}>
    {#each langs as l (l.id)}
      <button
        class="chip {shop.lang === l.id ? 'chosen' : ''}"
        role="radio"
        aria-checked={shop.lang === l.id}
        onclick={() => shop.setLang(l.id as Lang)}>{l.label}</button
      >
    {/each}
  </div>
</section>

<Notifications />

<section class="card text-sm text-zinc-600 dark:text-zinc-400">
  {t('me.signedin', { email: user.email })}{session.isOwner ? ` ${t('me.owner')}` : ''}.
  {#if session.isOwner}
    <a class="text-brand-600 underline dark:text-brand-400" href="/_/">{t('me.admin')}</a>
  {/if}
</section>

{#await version then v}
  {#if v}<p class="mt-auto text-center text-xs text-zinc-400">
      {t('me.version', { version: v })}
    </p>{/if}
{/await}
