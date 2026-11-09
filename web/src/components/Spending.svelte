<script lang="ts">
  import { t } from '../lib/i18n';
  import { pb } from '../lib/pb';
  import { money, monthName, type MonthSpend } from '../lib/spend';
  import { catalogue } from '../lib/stores/catalogue.svelte';

  // What the household spent per month and list, from the prices entered on
  // bought items. Shown once there is a price to add up.
  const months = pb.send<MonthSpend[]>('/api/app/spend', {}).catch(() => []);
</script>

{#await months then months}
  {#if months.length}
    <section class="card flex flex-col gap-3" aria-labelledby="spend">
      <div>
        <h2 id="spend" class="font-semibold">{t('spend.title')}</h2>
        <p class="text-sm text-zinc-500 dark:text-zinc-400">{t('spend.hint')}</p>
      </div>
      <ul class="flex flex-col gap-3">
        {#each months as m (m.month)}
          <li>
            <div class="flex items-baseline justify-between font-medium">
              <span>{monthName(m.month, catalogue.lang)}</span>
              <span>{money(m.total, catalogue.lang)}</span>
            </div>
            <ul class="text-sm text-zinc-500 dark:text-zinc-400">
              {#each m.lists as l (l.list)}
                <li class="flex justify-between">
                  <span>{l.name}</span><span>{money(l.total, catalogue.lang)}</span>
                </li>
              {/each}
            </ul>
          </li>
        {/each}
      </ul>
    </section>
  {/if}
{/await}
