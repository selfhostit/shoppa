<script lang="ts">
  import { sectionLabel, type Section } from '../lib/catalogue';
  import { t } from '../lib/i18n';
  import { kinds } from '../lib/kinds';
  import { shop } from '../lib/shop.svelte';
  import IconPicker from './IconPicker.svelte';
  import Sheet from './Sheet.svelte';

  // Pick another icon for a section of the open list, for the whole household.
  let { section, onclose }: { section: Section | undefined; onclose: () => void } = $props();

  const name = $derived(sectionLabel(section, shop.lang) || t('list.other'));
  const kind = $derived(t(kinds.find((k) => k.id === shop.kind)!.label));
  const picked = $derived(
    shop.sectionIcons.some((s) => s.key === `${shop.kind}:${section?.key ?? 'other'}`),
  );

  async function pick(icon: string) {
    await shop.pickSectionIcon(section, icon);
    if (!shop.error) onclose();
  }
</script>

<Sheet title={t('section.title', { name })} {onclose}>
  <div class="flex flex-col gap-4">
    <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('section.hint', { name, kind })}</p>
    <IconPicker
      current={shop.sectionIcon(section)}
      reset={picked ? t('section.own') : ''}
      tall
      onpick={pick}
    />
    {#if shop.error}
      <p class="text-sm text-red-600 dark:text-red-400" role="alert">{shop.error}</p>
    {/if}
  </div>
</Sheet>
