<script lang="ts">
  import { onMount } from 'svelte';
  import { errorMessage, pb } from './lib/pb';
  import { session } from './lib/session.svelte';
  import Home from './components/Home.svelte';
  import Login from './components/Login.svelte';
  import Setup from './components/Setup.svelte';
  import Shared from './components/Shared.svelte';
  import { sharedToken } from './lib/shares';

  // A guest's link shows only the list shared with them.
  const guest = sharedToken();

  // Signed-out visitors see the setup screen on a new install, the sign-in form otherwise.
  // Anyone already signed in means setup is done.
  let setupNeeded = $state<boolean | null>(session.user ? false : null);
  let error = $state('');

  async function checkSetup() {
    error = '';
    try {
      setupNeeded = (await pb.send<{ needed: boolean }>('/api/app/setup', {})).needed;
    } catch (err) {
      error = errorMessage(err);
    }
  }

  onMount(() => {
    if (guest) return;
    if (session.user) session.refresh();
    else checkSetup();
  });
</script>

{#if guest}
  <Shared token={guest} />
{:else if session.user}
  <Home />
{:else if error}
  <main class="mx-auto flex min-h-dvh max-w-sm flex-col items-center justify-center gap-4 px-6">
    <p class="text-center" role="alert">{error}</p>
    <button class="btn-secondary" onclick={checkSetup}>Try again</button>
  </main>
{:else if setupNeeded}
  <Setup />
{:else if setupNeeded === false}
  <Login />
{/if}
