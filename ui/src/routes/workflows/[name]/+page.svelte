<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import {
    getWorkflow,
    listExecutions,
    listExecutionStates,
    type WorkflowGraph,
    type FlowEntry,
    type Execution,
    type StateRecord,
  } from '$lib/api';
  import type { WSEvent, StateTransitionPayload } from '$lib/ws';

  const name = $derived($page.params.name as string);

  // ---- state ----
  let graph = $state<WorkflowGraph | null>(null);
  let executions = $state<Execution[]>([]);
  let states = $state<StateRecord[]>([]);
  let selectedExecId = $state<string | null>(null);
  let expandedNode = $state<string | null>(null);
  let loading = $state(true);
  let error = $state('');

  let sseSource: EventSource | null = null;

  // ---- layout constants ----
  const NODE_W = 160;
  const NODE_H = 60;
  const GAP_X = 80;
  const GAP_Y = 72;
  const PAD = 24;

  interface LayoutNode {
    name: string;
    x: number;
    y: number;
    isCatch: boolean;
    triggerName?: string;
  }

  interface LayoutEdge {
    x1: number; y1: number;
    x2: number; y2: number;
    dashed: boolean;
    label?: string;
  }

  function buildLayout(flow: FlowEntry[]): { nodes: LayoutNode[]; edges: LayoutEdge[]; svgW: number; svgH: number } {
    if (!flow.length) return { nodes: [], edges: [], svgW: 0, svgH: 0 };

    const flowMap = new Map(flow.map((f) => [f.name, f]));
    const mainPath: string[] = [];
    const visited = new Set<string>();

    let cur: string | undefined = flow[0].name;
    while (cur && !visited.has(cur)) {
      visited.add(cur);
      mainPath.push(cur);
      const entry = flowMap.get(cur);
      if (!entry || entry.is_end || !entry.next) break;
      cur = entry.next;
    }

    // Find catch nodes reachable via .catch but not in the main path.
    const catchEntries: Array<{ name: string; trigger: string }> = [];
    const mainSet = new Set(mainPath);
    for (const entry of flow) {
      if (entry.catch && !mainSet.has(entry.catch)) {
        catchEntries.push({ name: entry.catch, trigger: entry.name });
      }
    }

    const nodes: LayoutNode[] = [];
    const mainY = PAD + 20;

    mainPath.forEach((nodeName, i) => {
      nodes.push({ name: nodeName, x: PAD + i * (NODE_W + GAP_X), y: mainY, isCatch: false });
    });

    const mainNodeMap = new Map(nodes.map((n) => [n.name, n]));

    catchEntries.forEach(({ name: catchName, trigger }) => {
      const trigNode = mainNodeMap.get(trigger);
      nodes.push({
        name: catchName,
        x: trigNode ? trigNode.x : PAD,
        y: mainY + NODE_H + GAP_Y,
        isCatch: true,
        triggerName: trigger,
      });
    });

    const nodeMap = new Map(nodes.map((n) => [n.name, n]));

    // Build edges.
    const edges: LayoutEdge[] = [];

    // Next edges (horizontal arrows along main path).
    for (let i = 0; i < mainPath.length - 1; i++) {
      const src = mainNodeMap.get(mainPath[i])!;
      const dst = mainNodeMap.get(mainPath[i + 1])!;
      edges.push({
        x1: src.x + NODE_W,
        y1: src.y + NODE_H / 2,
        x2: dst.x,
        y2: dst.y + NODE_H / 2,
        dashed: false,
      });
    }

    // Catch edges (vertical dashed arrows).
    for (const { name: catchName, trigger } of catchEntries) {
      const src = mainNodeMap.get(trigger);
      const dst = nodeMap.get(catchName);
      if (src && dst) {
        edges.push({
          x1: src.x + NODE_W / 2,
          y1: src.y + NODE_H,
          x2: dst.x + NODE_W / 2,
          y2: dst.y,
          dashed: true,
          label: 'error',
        });
      }
    }

    const hasCatch = catchEntries.length > 0;
    const svgW = PAD * 2 + mainPath.length * NODE_W + Math.max(0, mainPath.length - 1) * GAP_X;
    const svgH = PAD * 2 + NODE_H + (hasCatch ? GAP_Y + NODE_H : 0) + 40 + 20;

    return { nodes, edges, svgW, svgH };
  }

  // ---- status helpers ----
  function statusFor(nodeName: string): string {
    return states.find((s) => s.state_name === nodeName)?.status ?? '';
  }

  function nodeColors(status: string): { fill: string; stroke: string; text: string } {
    switch (status.toLowerCase()) {
      case 'running':   return { fill: '#eff6ff', stroke: '#1d4ed8', text: '#1d4ed8' };
      case 'completed': return { fill: '#f0fdf4', stroke: '#15803d', text: '#15803d' };
      case 'failed':    return { fill: '#fef2f2', stroke: '#b91c1c', text: '#b91c1c' };
      default:          return { fill: '#fafaf9', stroke: '#d4d2cb', text: '#1a1916' };
    }
  }

  function stateFor(nodeName: string): StateRecord | null {
    return states.find((s) => s.state_name === nodeName) ?? null;
  }

  // ---- SSE ----
  function connectSSE(execId: string) {
    sseSource?.close();
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('kflow_token') : null;
    const url = `/api/v1/executions/${execId}/stream${token ? `?token=${encodeURIComponent(token)}` : ''}`;
    sseSource = new EventSource(url);
    sseSource.onmessage = (e) => {
      try {
        const ev = JSON.parse(e.data as string) as WSEvent;
        if (ev.type !== 'state_transition') return;
        const p = ev.payload as StateTransitionPayload;
        const idx = states.findIndex((s) => s.state_name === p.state_name);
        if (idx >= 0) {
          states[idx] = { ...states[idx], status: p.to_status as StateRecord['status'], error: p.error ?? '' };
          states = [...states];
        }
      } catch {
        // malformed — ignore
      }
    };
  }

  // ---- data loading ----
  async function loadExecStates(execId: string) {
    states = await listExecutionStates(execId);
    connectSSE(execId);
  }

  async function load() {
    loading = true;
    error = '';
    try {
      [graph, executions] = await Promise.all([
        getWorkflow(name),
        listExecutions({ workflow: name, limit: 20 }),
      ]);
      selectedExecId =
        executions.find((e) => e.status === 'Running')?.id ??
        executions[0]?.id ??
        null;
      if (selectedExecId) await loadExecStates(selectedExecId);
    } catch (e) {
      const err = e as { error: string };
      error = err.error ?? 'Failed to load workflow';
    } finally {
      loading = false;
    }
  }

  async function onExecChange(e: Event) {
    const id = (e.target as HTMLSelectElement).value;
    selectedExecId = id || null;
    if (id) await loadExecStates(id);
    else states = [];
  }

  function shortId(id: string): string {
    return id.length > 14 ? `${id.slice(0, 8)}…` : id;
  }

  onMount(load);
  onDestroy(() => sseSource?.close());

  const layout = $derived(graph ? buildLayout(graph.flow) : { nodes: [], edges: [], svgW: 0, svgH: 0 });
</script>

<div class="p-8 max-w-6xl">
  <div class="mb-4 flex items-center gap-2 text-sm text-muted">
    <a href="/workflows" class="hover:text-text">Workflows</a>
    <span>/</span>
    <span class="text-text font-medium">{name}</span>
  </div>

  <h1 class="text-3xl font-medium mb-6">{name}</h1>

  {#if loading}
    <p class="empty">Loading…</p>
  {:else if error}
    <p class="empty text-red-600">{error}</p>
  {:else if graph}
    <!-- Execution selector -->
    <div class="flex items-center gap-3 mb-6">
      <label for="exec-select" class="text-sm text-muted shrink-0">Execution:</label>
      {#if executions.length === 0}
        <span class="text-sm text-muted italic">No executions yet</span>
      {:else}
        <select id="exec-select" onchange={onExecChange} value={selectedExecId ?? ''} class="text-sm">
          <option value="">— none —</option>
          {#each executions as exec (exec.id)}
            <option value={exec.id}>
              {shortId(exec.id)} · {exec.status} · {new Date(exec.created_at).toLocaleString()}
            </option>
          {/each}
        </select>
      {/if}
    </div>

    <!-- Legend -->
    <div class="flex gap-4 mb-4 text-xs text-muted items-center flex-wrap">
      <span class="font-medium text-text">Status:</span>
      <span class="badge badge-pending">Pending</span>
      <span class="badge badge-running">Running</span>
      <span class="badge badge-completed">Completed</span>
      <span class="badge badge-failed">Failed</span>
      <span class="ml-4 flex items-center gap-1.5">
        <svg width="28" height="10"><line x1="0" y1="5" x2="28" y2="5" stroke="#6d6b62" stroke-width="1.5"/><polygon points="22,2 28,5 22,8" fill="#6d6b62"/></svg>
        next
      </span>
      <span class="flex items-center gap-1.5">
        <svg width="28" height="10"><line x1="0" y1="5" x2="28" y2="5" stroke="#b91c1c" stroke-width="1.5" stroke-dasharray="4,3"/><polygon points="22,2 28,5 22,8" fill="#b91c1c"/></svg>
        catch/error
      </span>
    </div>

    <!-- DAG Graph -->
    <div class="bg-surface border border-border rounded-lg overflow-x-auto mb-6 p-2">
      {#if layout.nodes.length === 0}
        <p class="empty">No flow entries found in this workflow.</p>
      {:else}
        <svg
          width={layout.svgW}
          height={layout.svgH}
          xmlns="http://www.w3.org/2000/svg"
          style="min-width: 100%"
        >
          <defs>
            <marker id="arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
              <path d="M0,0 L0,6 L8,3 z" fill="#6d6b62"/>
            </marker>
            <marker id="arrow-err" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
              <path d="M0,0 L0,6 L8,3 z" fill="#b91c1c"/>
            </marker>
          </defs>

          <!-- Edges -->
          {#each layout.edges as edge}
            {#if edge.dashed}
              <!-- Catch edge: elbow path through midpoint -->
              {@const mx = edge.x1}
              {@const my = (edge.y1 + edge.y2) / 2}
              <path
                d="M{edge.x1},{edge.y1} L{mx},{my} L{edge.x2},{edge.y2}"
                fill="none"
                stroke="#b91c1c"
                stroke-width="1.5"
                stroke-dasharray="5,4"
                marker-end="url(#arrow-err)"
              />
              {#if edge.label}
                <text x={edge.x1 + 4} y={my - 4} font-size="10" fill="#b91c1c">{edge.label}</text>
              {/if}
            {:else}
              <!-- Next edge: straight horizontal line -->
              {@const mx = (edge.x1 + edge.x2) / 2}
              <line
                x1={edge.x1} y1={edge.y1}
                x2={edge.x2 - 7} y2={edge.y2}
                stroke="#6d6b62"
                stroke-width="1.5"
                marker-end="url(#arrow)"
              />
            {/if}
          {/each}

          <!-- Nodes -->
          {#each layout.nodes as node}
            {@const status = statusFor(node.name)}
            {@const colors = nodeColors(status)}
            {@const isRunning = status.toLowerCase() === 'running'}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <g
              transform="translate({node.x},{node.y})"
              style="cursor:pointer"
              onclick={() => { expandedNode = expandedNode === node.name ? null : node.name; }}
            >
              {#if isRunning}
                <rect
                  x="-4" y="-4"
                  width={NODE_W + 8} height={NODE_H + 8}
                  rx="10"
                  fill="none"
                  stroke={colors.stroke}
                  stroke-width="1"
                  opacity="0.4"
                >
                  <animate attributeName="opacity" values="0.4;0.1;0.4" dur="1.5s" repeatCount="indefinite"/>
                </rect>
              {/if}
              <rect
                x="0" y="0"
                width={NODE_W} height={NODE_H}
                rx="8"
                fill={colors.fill}
                stroke={colors.stroke}
                stroke-width={status ? 2 : 1}
              />
              <text
                x={NODE_W / 2} y="22"
                text-anchor="middle"
                font-size="13"
                font-weight="600"
                fill={colors.text}
                font-family="'Instrument Sans', sans-serif"
              >{node.name}</text>
              {#if status}
                <text
                  x={NODE_W / 2} y="42"
                  text-anchor="middle"
                  font-size="11"
                  fill={colors.text}
                  opacity="0.8"
                  font-family="'Instrument Sans', sans-serif"
                >{status}</text>
              {:else}
                <text
                  x={NODE_W / 2} y="42"
                  text-anchor="middle"
                  font-size="10"
                  fill="#b2afa6"
                  font-family="'Instrument Sans', sans-serif"
                >no execution</text>
              {/if}
              {#if node.isCatch}
                <rect x={NODE_W - 44} y="4" width="40" height="16" rx="4" fill="#fef2f2"/>
                <text x={NODE_W - 24} y="15" text-anchor="middle" font-size="9" fill="#b91c1c" font-family="'Instrument Sans', sans-serif">catch</text>
              {/if}
            </g>
          {/each}
        </svg>
      {/if}
    </div>

    <!-- Expanded node detail panel -->
    {#if expandedNode}
      {@const sr = stateFor(expandedNode)}
      <div class="border border-border rounded-lg bg-surface p-5 mb-6">
        <div class="flex items-center gap-3 mb-4">
          <h2 class="text-lg font-medium m-0">{expandedNode}</h2>
          {#if sr}
            <span class="badge badge-{sr.status.toLowerCase()}">{sr.status}</span>
            {#if sr.attempt > 1}
              <span class="text-xs text-muted">attempt {sr.attempt}</span>
            {/if}
          {/if}
          <button class="ml-auto text-xs text-muted" onclick={() => (expandedNode = null)}>Close ✕</button>
        </div>
        {#if sr}
          {#if sr.error}
            <p class="text-sm text-red-600 mb-3">{sr.error}</p>
          {/if}
          <div class="flex gap-4 flex-wrap">
            <div class="flex-1 min-w-[200px] flex flex-col gap-1">
              <strong class="text-sm text-muted">Input</strong>
              <pre>{JSON.stringify(sr.input, null, 2)}</pre>
            </div>
            <div class="flex-1 min-w-[200px] flex flex-col gap-1">
              <strong class="text-sm text-muted">Output</strong>
              <pre>{JSON.stringify(sr.output, null, 2)}</pre>
            </div>
          </div>
        {:else}
          <p class="text-sm text-muted">No state record for this node in the selected execution.</p>
        {/if}
      </div>
    {/if}

    <!-- States table -->
    {#if selectedExecId && states.length > 0}
      <h2 class="text-xl text-muted mb-2 border-b border-border pb-1">States</h2>
      <table>
        <thead>
          <tr>
            <th>State</th>
            <th>Status</th>
            <th>Attempt</th>
            <th>Error</th>
          </tr>
        </thead>
        <tbody>
          {#each states as s (s.state_name)}
            <tr onclick={() => { expandedNode = expandedNode === s.state_name ? null : s.state_name; }}>
              <td>{s.state_name}</td>
              <td><span class="badge badge-{s.status.toLowerCase()}">{s.status}</span></td>
              <td>{s.attempt}</td>
              <td class="text-red-600 text-xs">{s.error || '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  {/if}
</div>
