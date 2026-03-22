<script lang="ts">
  import { onMount, onDestroy, type Snippet } from 'svelte';
  import { page } from '$app/stores';
  import { createWSClient } from '$lib/ws';
  import { wsEvents, wsConnected } from '$lib/wsStore';
  import '../app.css';

  let { children }: { children: Snippet } = $props();

  let client: ReturnType<typeof createWSClient>;
  let _interval: ReturnType<typeof setInterval> | undefined;

  function getToken(): string | null {
    if (typeof localStorage === 'undefined') return null;
    return localStorage.getItem('kflow_token');
  }

  onMount(() => {
    // Pick up token injected via `kflow ui <port>` URL query param
    const params = new URLSearchParams(window.location.search);
    const urlToken = params.get('token');
    if (urlToken) {
      localStorage.setItem('kflow_token', urlToken);
      params.delete('token');
      const newURL = window.location.pathname + (params.toString() ? '?' + params.toString() : '');
      history.replaceState(null, '', newURL);
    }

    client = createWSClient(
      '/api/v1/ws',
      (ev) => { wsEvents.set(ev); },
      getToken,
    );
    client.connect();

    _interval = setInterval(() => {
      wsConnected.set(client.isConnected());
    }, 1000);
  });

  onDestroy(() => {
    client?.disconnect();
    if (_interval !== undefined) clearInterval(_interval);
  });

  const navLinks = [
    {
      href: '/',
      label: 'Executions',
      icon: `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>`,
    },
    {
      href: '/services',
      label: 'Services',
      icon: `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>`,
    },
    {
      href: '/logs',
      label: 'Logs',
      icon: `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="15" y2="18"/></svg>`,
    },
  ];

  function isActive(href: string, pathname: string): boolean {
    if (href === '/') return pathname === '/';
    return pathname.startsWith(href);
  }
</script>

<div class="flex h-screen overflow-hidden bg-base">
  <!-- Sidebar -->
  <aside class="w-56 shrink-0 flex flex-col bg-surface border-r border-border">
    <!-- Brand -->
    <div class="px-5 pt-7 pb-6 flex items-center gap-2.5">
      <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
        <rect x="1" y="1" width="7" height="7" rx="1.5" fill="#0f766e"/>
        <rect x="10" y="1" width="7" height="7" rx="1.5" fill="#0f766e" opacity="0.4"/>
        <rect x="1" y="10" width="7" height="7" rx="1.5" fill="#0f766e" opacity="0.4"/>
        <rect x="10" y="10" width="7" height="7" rx="1.5" fill="#0f766e" opacity="0.2"/>
      </svg>
      <span class="text-text font-semibold tracking-tight" style="font-family: var(--font-display); font-size: 2rem; letter-spacing: -0.02em;">kflow</span>
    </div>

    <!-- Nav section label -->
    <div class="px-5 mb-2">
      <span class="text-[10px] font-semibold text-subtle uppercase tracking-widest">Navigation</span>
    </div>

    <!-- Nav links -->
    <nav class="flex-1 px-3 flex flex-col gap-0.5">
      {#each navLinks as link}
        {@const active = isActive(link.href, $page.url.pathname)}
        <a
          href={link.href}
          class="flex items-center gap-2.5 px-3 py-2 rounded text-sm transition-all duration-150
            {active
              ? 'bg-[#0f766e]/8 text-[#0f766e] font-medium border-l-2 border-[#0f766e] pl-[10px]'
              : 'text-muted hover:text-text hover:bg-raised border-l-2 border-transparent pl-[10px]'}"
          style={active ? 'font-family: var(--font-display);' : ''}
        >
          <span class="shrink-0 opacity-80">{@html link.icon}</span>
          {link.label}
        </a>
      {/each}
    </nav>

    <!-- Divider -->
    <div class="mx-5 border-t border-border mb-4"></div>

    <!-- WS status -->
    <div class="px-5 pb-5">
      {#if $wsConnected}
        <div class="flex items-center gap-2 text-xs text-emerald-700">
          <span class="relative flex h-2 w-2">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-50"></span>
            <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
          </span>
          Live
        </div>
      {:else}
        <div class="flex items-center gap-2 text-xs text-red-600">
          <span class="w-2 h-2 rounded-full bg-red-400"></span>
          Disconnected
        </div>
      {/if}
    </div>
  </aside>

  <!-- Main content -->
  <main class="flex-1 overflow-y-auto">
    {@render children()}
  </main>
</div>
