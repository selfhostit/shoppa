<script lang="ts">
  import { sectionLabel, type Section } from '../lib/catalogue';
  import { t } from '../lib/i18n';
  import { catalogue } from '../lib/stores/catalogue.svelte';
  import { choices } from '../lib/stores/choices.svelte';
  import Glyph from './Glyph.svelte';

  // A section's icon and name over its items; a tap picks another icon for it.
  let {
    section,
    onpick,
  }: { section: Section | undefined; onpick: (section: Section | undefined) => void } = $props();

  const label = $derived(sectionLabel(section, catalogue.lang) || t('list.other'));
</script>

<h2 class="mb-1.5 px-1">
  <button
    class="-mx-1 flex items-center gap-1.5 rounded-lg px-1 py-0.5 text-xs font-semibold tracking-wide text-zinc-500 uppercase hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-900"
    onclick={() => onpick(section)}
    aria-label={t('section.change', { name: label })}
  >
    <Glyph icon={choices.sectionIcon(section)} class="size-4 text-sm" />
    <!-- in its own language, so Greek capitals lose their accents -->
    <span lang={section ? catalogue.lang : 'en'}>{label}</span>
  </button>
</h2>
