<script lang="ts">
  import { onMount } from 'svelte';
  import { listWorkflows } from '$lib/api';

  let workflows = $state<string[]>([]);
  let loading = $state(true);
  let error = $state('');

  onMount(async () => {
    try {
      workflows = await listWorkflows();
    } catch (e) {
      const err = e as { error: string };
      error = err.error ?? 'Failed to load workflows';
    } finally {
      loading = false;
    }
  });
</script>

<div class="p-8 max-w-4xl">
  <h1 class="text-3xl font-medium">Workflows</h1>

  {#if loading}
    <p class="empty">Loading…</p>
  {:else if error}
    <p class="empty text-red-600">{error}</p>
  {:else if workflows.length === 0}
    <p class="empty">No workflows registered. Run <code>kflow.Dispatch(wf)</code> to register a workflow.</p>
  {:else}
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each workflows as name (name)}
          <tr onclick={() => (window.location.href = `/workflows/${encodeURIComponent(name)}`)}>
            <td><code>{name}</code></td>
            <td class="text-right">
              <a
                href="/workflows/{encodeURIComponent(name)}"
                class="text-xs text-accent hover:underline"
                onclick={(e) => e.stopPropagation()}
              >View graph →</a>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>
