<script lang="ts">
  import { onMount } from 'svelte'
  import { getConnections, getHistory, getInitialState, getStatus, isPaused, pause, resume } from './lib/api'
  import Graph from './components/Graph.svelte'
  import type { Connection, PingSample, SortField } from './lib/types'

  let connections: Connection[] = []
  let history: PingSample[] = []
  let selectedKey = ''
  let filter = ''
  let sortField: SortField = 'app'
  let sortAsc = true
  let paused = false
  let pingEnabled = true
  let loading = true
  let scanning = false
  let ready = false
  let lastUpdated = ''
  let error = ''

  $: selected = connections.find((connection) => connection.key === selectedKey) ?? null

  onMount(() => {
    let disposed = false

    async function init() {
      try {
        const state = await getInitialState()
        filter = state.filter
        paused = state.paused
        pingEnabled = state.pingEnabled
        scanning = state.scanning
        ready = state.ready
        error = state.lastError
        await refresh()
      } catch (err) {
        error = err instanceof Error ? err.message : String(err)
      } finally {
        loading = false
      }
    }

    const timer = window.setInterval(async () => {
      if (!disposed) {
        paused = await isPaused()
        const status = await getStatus()
        scanning = status.scanning
        ready = status.ready
        if (status.lastError) error = status.lastError
        if (!paused) {
          await refresh()
        }
      }
    }, 1500)

    init()

    return () => {
      disposed = true
      window.clearInterval(timer)
    }
  })

  async function refresh() {
    try {
      const next = await getConnections(filter, sortField, sortAsc)
      connections = next
      const status = await getStatus()
      scanning = status.scanning
      ready = status.ready

      if (!selectedKey && next.length > 0) {
        selectedKey = next[0].key
      }
      if (selectedKey && !next.some((connection) => connection.key === selectedKey)) {
        selectedKey = next[0]?.key ?? ''
      }
      history = selectedKey ? await getHistory(selectedKey) : []
      lastUpdated = new Date().toLocaleTimeString()
      error = status.lastError
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
    }
  }

  async function selectConnection(key: string) {
    selectedKey = key
    history = await getHistory(key)
  }

  function setSort(field: SortField) {
    if (sortField === field) {
      sortAsc = !sortAsc
    } else {
      sortField = field
      sortAsc = true
    }
    refresh()
  }

  async function togglePause() {
    paused = paused ? await resume() : await pause()
  }

  function sortLabel(field: SortField) {
    if (sortField !== field) return ''
    return sortAsc ? ' ↑' : ' ↓'
  }

  function pingClass(value: number) {
    if (value <= 0) return 'muted'
    if (value < 50) return 'good'
    if (value < 150) return 'warn'
    return 'bad'
  }

  function lossClass(value: number) {
    if (value < 1) return 'good'
    if (value < 10) return 'warn'
    return 'bad'
  }

  function fmtPing(value: number) {
    return value > 0 ? `${value.toFixed(1)}ms` : '-'
  }

  function fmtLoss(value: number) {
    return `${value.toFixed(0)}%`
  }
</script>

<main class="shell">
  <section class="hero">
    <div>
      <p class="eyebrow">Realtime connection monitor</p>
      <h1>Ping Tracker</h1>
    </div>
    <div class="stats">
      <div>
        <span>{connections.length}</span>
        <small>connections</small>
      </div>
      <div>
        <span>{paused ? 'Paused' : scanning ? 'Scanning' : ready ? 'Live' : 'Starting'}</span>
        <small>state</small>
      </div>
      <div>
        <span>{lastUpdated || '-'}</span>
        <small>updated</small>
      </div>
    </div>
  </section>

  <section class="toolbar">
    <input
      aria-label="Filter by app name"
      bind:value={filter}
      oninput={() => refresh()}
      placeholder="Filter by app name"
      spellcheck="false"
    />
    <button onclick={() => refresh()}>Refresh</button>
    <button class:active={paused} onclick={togglePause}>{paused ? 'Resume UI' : 'Pause UI'}</button>
    <span class="hint">TCP probes: {pingEnabled ? 'enabled' : 'disabled'}</span>
  </section>

  {#if error}
    <section class="error">{error}</section>
  {/if}

  <section class="content">
    <div class="table-card">
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th><button onclick={() => setSort('pid')}>PID{sortLabel('pid')}</button></th>
              <th><button onclick={() => setSort('app')}>App{sortLabel('app')}</button></th>
              <th><button onclick={() => setSort('ping')}>Ping{sortLabel('ping')}</button></th>
              <th><button onclick={() => setSort('loss')}>Loss{sortLabel('loss')}</button></th>
              <th>Dir</th>
              <th>Proto</th>
              <th>Local</th>
              <th>Remote</th>
              <th><button onclick={() => setSort('state')}>State{sortLabel('state')}</button></th>
              <th><button onclick={() => setSort('tx')}>TX{sortLabel('tx')}</button></th>
              <th><button onclick={() => setSort('rx')}>RX{sortLabel('rx')}</button></th>
            </tr>
          </thead>
          <tbody>
            {#if loading}
              <tr><td colspan="11" class="empty">Loading connections...</td></tr>
            {:else if !ready}
              <tr><td colspan="11" class="empty">Starting tracker. Connections will appear after the first scan...</td></tr>
            {:else if connections.length === 0}
              <tr><td colspan="11" class="empty">No active connections found.</td></tr>
            {:else}
              {#each connections as connection (connection.key)}
                <tr class:selected={connection.key === selectedKey} onclick={() => selectConnection(connection.key)}>
                  <td>{connection.pid}</td>
                  <td class="app-name">{connection.appName}</td>
                  <td class={pingClass(connection.pingMs)}>{fmtPing(connection.pingMs)}</td>
                  <td class={lossClass(connection.loss)}>{fmtLoss(connection.loss)}</td>
                  <td><span class="pill">{connection.direction}</span></td>
                  <td>{connection.protocol}</td>
                  <td class="endpoint">{connection.local}</td>
                  <td class="endpoint">{connection.remote}</td>
                  <td>{connection.state}</td>
                  <td>{connection.txRate}</td>
                  <td>{connection.rxRate}</td>
                </tr>
              {/each}
            {/if}
          </tbody>
        </table>
      </div>
    </div>

    <aside class="detail-card">
      {#if selected}
        <div class="detail-head">
          <div>
            <p class="eyebrow">Selected connection</p>
            <h2>{selected.appName}</h2>
          </div>
          <span class="state">{selected.state}</span>
        </div>

        <dl class="metrics">
          <div>
            <dt>Ping</dt>
            <dd class={pingClass(selected.pingMs)}>{fmtPing(selected.pingMs)}</dd>
          </div>
          <div>
            <dt>Loss</dt>
            <dd class={lossClass(selected.loss)}>{selected.loss.toFixed(1)}%</dd>
          </div>
          <div>
            <dt>PID</dt>
            <dd>{selected.pid}</dd>
          </div>
          <div>
            <dt>Age</dt>
            <dd>{selected.age}</dd>
          </div>
        </dl>

        <div class="endpoints">
          <span>{selected.local}</span>
          <span>{selected.direction}</span>
          <span>{selected.remote}</span>
        </div>

        <Graph title="Ping latency" samples={history} mode="rtt" />
        <Graph title="Packet loss" samples={history} mode="loss" />
      {:else}
        <div class="empty-detail">Select a connection to show ping history.</div>
      {/if}
    </aside>
  </section>
</main>
