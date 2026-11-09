<script lang="ts">
  import { sectionLabel, type Section } from '../lib/catalogue';
  import { t } from '../lib/i18n';
  import { kinds } from '../lib/kinds';
  import { status } from '../lib/stores/status.svelte';
  import { household } from '../lib/stores/household.svelte';
  import { catalogue } from '../lib/stores/catalogue.svelte';
  import { choices } from '../lib/stores/choices.svelte';
  import IconPicker from './IconPicker.svelte';
  import Sheet from './Sheet.svelte';

  // Pick another icon for a section of the open list, for the whole household.
  let { section, onclose }: { section: Section | undefined; onclose: () => void } = $props();

  const name = $derived(sectionLabel(section, catalogue.lang) || t('list.other'));
  const kind = $derived(t(kinds.find((k) => k.id === household.kind)!.label));
  const picked = $derived(
    choices.sectionIcons.some((s) => s.key === `${household.kind}:${section?.key ?? 'other'}`),
  );

  async function pick(icon: string) {
    await choices.pickSectionIcon(section, icon);
    if (!status.error) onclose();
  }
</script>

<Sheet title={t('section.title', { name })} {onclose}>
  <div class="flex flex-col gap-4">
    <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('section.hint', { name, kind })}</p>
    <IconPicker
      current={choices.sectionIcon(section)}
      reset={picked ? t('section.own') : ''}
      tall
      onpick={pick}
    />
    {#if status.error}
      <p class="text-sm text-red-600 dark:text-red-400" role="alert">{status.error}</p>
    {/if}
  </div>
</Sheet>
