<script lang="ts">
  import { t } from '../lib/i18n';
  import type { List } from '../lib/list';
  import { share, shareUrl, sharesOf, unshare, type Share } from '../lib/shares';
  import { run } from '../lib/stores/status.svelte';

  // Links that show this list to someone outside the household, with no
  // account: to look at, or to tick things off too.
  let { list }: { list: List } = $props();

  let shares = $state<Share[]>([]);
  let copied = $state('');

  // loaded quietly: the list sheet works without them
  $effect(() => {
    const id = list.id;
    sharesOf(id)
      .then((found) => id === list.id && (shares = found))
      .catch(() => (shares = []));
  });

  async function add(canTick: boolean) {
    await run(async () => {
      const s = await share(list.id, canTick);
      shares = [...shares, s];
      await send(s);
    });
  }

  /** The phone's share sheet where there is one, the clipboard otherwise. */
  async function send(s: Share) {
    const url = shareUrl(s);
    try {
      if (navigator.share) {
        await navigator.share({ title: list.name, url });
        return;
      }
    } catch {
      // closed, or not allowed: copy it instead
    }
    await navigator.clipboard?.writeText(url);
    copied = s.id;
  }

  async function stop(s: Share) {
    await run(async () => {
      await unshare(s);
      shares = shares.filter((x) => x.id !== s.id);
    });
  }
</script>

<div class="flex flex-col gap-2">
  <div>
    <span class="text-sm font-medium">{t('share.title')}</span>
    <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('share.hint')}</p>
  </div>
  {#each shares as s (s.id)}
    <div class="flex items-center gap-2 text-sm">
      <span class="flex-1">{s.can_tick ? t('share.tick') : t('share.view')}</span>
      <button class="btn-secondary px-3 py-1.5 text-sm" onclick={() => send(s)}
        >{copied === s.id ? t('share.copied') : t('share.send')}</button
      >
      <button
        class="btn-secondary px-3 py-1.5 text-sm text-red-600 dark:text-red-400"
        onclick={() => stop(s)}>{t('share.stop')}</button
      >
    </div>
  {/each}
  <div class="flex gap-2">
    <button class="btn-secondary flex-1 text-sm" onclick={() => add(false)}
      >{t('share.new.view')}</button
    >
    <button class="btn-secondary flex-1 text-sm" onclick={() => add(true)}
      >{t('share.new.tick')}</button
    >
  </div>
</div>
