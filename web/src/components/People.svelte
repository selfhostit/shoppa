<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { t } from '../lib/i18n';
  import { errorMessage, pb, type User } from '../lib/pb';
  import { generatePassword, inviteText, personError } from '../lib/people';
  import { session } from '../lib/session.svelte';
  import Sheet from './Sheet.svelte';

  // The owner adds, resets and removes people here. Only the owner gets a button
  // for it (Account.svelte), and the users collection's API rules hold for
  // everyone else anyway.
  let { onclose }: { onclose: () => void } = $props();

  let people = $state<User[]>([]);
  let loading = $state(true);
  let error = $state('');

  let name = $state('');
  let email = $state('');
  let busy = $state(false);
  let formError = $state('');

  // what to send to the person just added or reset; the password is never shown again
  let invite = $state<{ text: string; title: string } | null>(null);
  let copied = $state(false);
  let card = $state<HTMLElement>();
  let textarea = $state<HTMLTextAreaElement>();
  const canShare = typeof navigator.share === 'function';

  const label = (p: { name: string; email: string }) => p.name || p.email;

  async function load() {
    try {
      people = await pb.collection('users').getFullList<User>({ sort: 'created' });
      error = '';
    } catch (err) {
      error = errorMessage(err);
    } finally {
      loading = false;
    }
  }

  onMount(load);

  async function show(person: { name: string; email: string }, password: string, title: string) {
    invite = {
      title,
      text: inviteText({
        appName: document.title,
        address: location.origin,
        email: person.email,
        password,
        name: person.name,
      }),
    };
    copied = false;
    await tick();
    card?.scrollIntoView({ block: 'start' });
  }

  async function add(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    formError = '';
    const password = generatePassword();
    const person = { name: name.trim(), email: email.trim().toLowerCase() };
    try {
      await pb.collection('users').create({
        ...person,
        role: 'member',
        verified: true,
        password,
        passwordConfirm: password,
      });
      name = email = '';
      await load();
      await show(person, password, t('people.added', { name: label(person) }));
    } catch (err) {
      formError = personError(err);
    } finally {
      busy = false;
    }
  }

  async function reset(person: User) {
    if (!confirm(t('people.reset.confirm', { name: label(person) }))) return;
    error = '';
    const password = generatePassword();
    try {
      await pb.collection('users').update(person.id, { password, passwordConfirm: password });
      await show(person, password, t('people.reset.title', { name: label(person) }));
    } catch (err) {
      error = personError(err);
    }
  }

  async function remove(person: User) {
    if (!confirm(t('people.remove.confirm', { name: label(person) }))) return;
    error = '';
    try {
      await pb.collection('users').delete(person.id);
      await load();
    } catch (err) {
      error = errorMessage(err, t('people.remove.failed', { name: label(person) }));
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(invite!.text);
      copied = true;
    } catch {
      // no clipboard over plain http: select the text so a long press copies it
      textarea?.select();
    }
  }

  async function share() {
    try {
      await navigator.share({ text: invite!.text });
    } catch {
      // closed the share sheet
    }
  }
</script>

<Sheet title={t('people.title')} {onclose}>
  <div class="flex flex-col gap-4">
    {#if invite}
      <section
        bind:this={card}
        class="card flex scroll-mt-4 flex-col gap-3 border-brand-500 dark:border-brand-400"
      >
        <h3 class="font-semibold">{invite.title}</h3>
        <p class="text-sm text-zinc-600 dark:text-zinc-400">{t('people.send')}</p>
        <textarea
          bind:this={textarea}
          readonly
          rows="8"
          class="text-sm"
          value={invite.text}
          onfocus={(e) => e.currentTarget.select()}></textarea>
        <div class="flex gap-2">
          <button class="btn-primary flex-1" onclick={copy}
            >{copied ? t('people.copied') : t('people.copy')}</button
          >
          {#if canShare}
            <button class="btn-secondary flex-1" onclick={share}>{t('people.share')}</button>
          {/if}
          <button class="btn-secondary flex-1" onclick={() => (invite = null)}
            >{t('people.done')}</button
          >
        </div>
      </section>
    {/if}

    <section class="card flex flex-col gap-3">
      <h3 class="font-semibold">{t('people.everyone')}</h3>
      {#if loading}
        <p class="text-sm text-zinc-500">{t('people.loading')}</p>
      {:else}
        <ul class="divide-y divide-zinc-200 dark:divide-zinc-800">
          {#each people as person (person.id)}
            {@const you = person.id === session.user?.id}
            <li class="flex flex-col gap-2 py-3 first:pt-0 last:pb-0">
              <div class="min-w-0">
                <p class="truncate font-medium">
                  {label(person)}
                  {#if you}<span class="text-sm font-normal text-zinc-500">{t('people.you')}</span
                    >{/if}
                </p>
                <p class="truncate text-sm text-zinc-500 dark:text-zinc-400">
                  {[
                    person.name && person.email,
                    person.role === 'owner' ? t('people.owner') : t('people.member'),
                  ]
                    .filter(Boolean)
                    .join(' · ')}
                </p>
              </div>
              {#if !you && person.role !== 'owner'}
                <div class="flex gap-2">
                  <button class="btn-secondary px-3 py-2 text-sm" onclick={() => reset(person)}
                    >{t('people.reset')}</button
                  >
                  <button
                    class="btn-secondary px-3 py-2 text-sm text-red-600 dark:text-red-400"
                    onclick={() => remove(person)}>{t('people.remove')}</button
                  >
                </div>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
      {#if error}<p class="text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>{/if}
    </section>

    <section class="card">
      <h3 class="font-semibold">{t('people.add')}</h3>
      <p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">{t('people.add.hint')}</p>
      <form class="mt-4 flex flex-col gap-4" onsubmit={add}>
        <label
          >{t('people.name')} <input autocomplete="off" bind:value={name} maxlength="100" /></label
        >
        <label>
          {t('people.email')}
          <input type="email" autocomplete="off" bind:value={email} required />
        </label>
        {#if formError}
          <p class="text-sm text-red-600 dark:text-red-400" role="alert">{formError}</p>
        {/if}
        <button class="btn-primary" disabled={busy}
          >{busy ? t('people.adding') : t('people.add.button')}</button
        >
      </form>
    </section>
  </div>
</Sheet>
