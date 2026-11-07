<script lang="ts">
  import { icons } from '../lib/icons';

  // A line icon by its name (lib/icons.ts), or an emoji picked before there
  // were icons, shown as it is.
  let { icon, class: cls = 'size-5' }: { icon: string; class?: string } = $props();
</script>

{#if icons[icon]}
  <svg
    viewBox="0 0 24 24"
    class="shrink-0 {cls}"
    fill="none"
    stroke="currentColor"
    stroke-width="1.75"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true">{@html icons[icon]}</svg
  >
{:else if icon}
  <span class="grid shrink-0 place-items-center leading-none {cls}" aria-hidden="true">{icon}</span>
{/if}
