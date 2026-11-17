<script lang="ts">
  import { t } from '../lib/i18n';
  import { errorMessage, pb } from '../lib/pb';
  import { session } from '../lib/session.svelte';
  import People from './People.svelte';

  // Who is signed in, changing your own password, and for the owner the People sheet.
  const user = $derived(session.user!);

  let people = $state(false);
  let open = $state(false);
  let current = $state('');
  let password = $state('');
  let again = $state('');
  let busy = $state(false);
  let error = $state('');
  let done = $state(false);

  async function change(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    if (password !== again) {
      error = t('account.password.mismatch');
      return;
    }
    busy = true;
    try {
      await pb.collection('users').update(user.id, {
        oldPassword: current,
        password,
        passwordConfirm: again,
      });
    } catch (err) {
      const data = (err as { response?: { data?: Record<string, unknown> } }).response?.data;
      error = data?.oldPassword
        ? t('account.password.wrong')
        : data?.password
          ? t('account.password.short')
          : errorMessage(err);
      busy = false;
      return;
    }
    // a new password signs out every device, this one too: sign in again with it
    try {
      await session.login(user.email, password);
      current = password = again = '';
      open = false;
      done = true;
    } catch {
      pb.authStore.clear();
    } finally {
      busy = false;
    }
  }
</script>

<section class="card flex flex-col gap-3 text-sm text-zinc-600 dark:text-zinc-400">
  <p>{t('me.signedin', { email: user.email })}{session.isOwner ? ` ${t('me.owner')}` : ''}.</p>
  <div class="flex flex-wrap gap-2">
    {#if session.isOwner}
      <button class="btn-secondary px-3 py-2 text-sm" onclick={() => (people = true)}
        >{t('account.people')}</button
      >
    {/if}
    <button
      class="btn-secondary px-3 py-2 text-sm"
      onclick={() => ((open = !open), (done = false), (error = ''))}
      aria-expanded={open}>{t('account.password')}</button
    >
  </div>
  {#if done}<p role="status">{t('account.password.done')}</p>{/if}
  {#if open}
    <form class="flex flex-col gap-4 text-zinc-900 dark:text-zinc-100" onsubmit={change}>
      <label>
        {t('account.password.current')}
        <input type="password" autocomplete="current-password" bind:value={current} required />
      </label>
      <label>
        {t('account.password.new')}
        <input
          type="password"
          autocomplete="new-password"
          bind:value={password}
          required
          minlength="8"
          placeholder={t('account.password.placeholder')}
        />
      </label>
      <label>
        {t('account.password.again')}
        <input
          type="password"
          autocomplete="new-password"
          bind:value={again}
          required
          minlength="8"
        />
      </label>
      {#if session.isOwner}
        <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('account.password.admin')}</p>
      {/if}
      {#if error}<p class="text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>{/if}
      <button class="btn-primary" disabled={busy}
        >{busy ? t('account.password.saving') : t('account.password')}</button
      >
    </form>
  {/if}
  {#if session.isOwner}
    <p><a class="text-brand-600 underline dark:text-brand-400" href="/_/">{t('me.admin')}</a></p>
  {/if}
</section>

{#if people}
  <People onclose={() => (people = false)} />
{/if}
