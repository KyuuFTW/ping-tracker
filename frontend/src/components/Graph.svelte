<script lang="ts">
  import { onMount, tick } from 'svelte'
  import type { PingSample } from '../lib/types'

  export let title: string
  export let samples: PingSample[] = []
  export let mode: 'rtt' | 'loss'
  export let scaleKey = ''

  let canvas: HTMLCanvasElement
  let summary = ''
  let lastDrawKey = ''
  let lastScaleKey = ''
  let rttMaxScale = 100
  let pendingFrame = 0

  onMount(() => {
    const resize = () => scheduleDraw(drawKey(samples, mode, scaleKey), true)
    window.addEventListener('resize', resize)
    scheduleDraw(drawKey(samples, mode, scaleKey), true)
    return () => window.removeEventListener('resize', resize)
  })

  $: if (scaleKey !== lastScaleKey) {
    lastScaleKey = scaleKey
    rttMaxScale = 100
    lastDrawKey = ''
  }

  $: if (canvas && drawKey(samples, mode, scaleKey) !== lastDrawKey) {
    scheduleDraw(drawKey(samples, mode, scaleKey))
  }

  async function scheduleDraw(key: string, force = false) {
    if (!canvas) return
    if (!force && key === lastDrawKey) return

    lastDrawKey = key
    await tick()

    if (pendingFrame) {
      cancelAnimationFrame(pendingFrame)
    }

    pendingFrame = requestAnimationFrame(() => {
      pendingFrame = 0
      drawGraph()
    })
  }

  function drawGraph() {
    const ctx = canvas?.getContext('2d')
    if (!ctx) return

    const ratio = window.devicePixelRatio || 1
    const rect = canvas.getBoundingClientRect()
    canvas.width = Math.max(1, Math.floor(rect.width * ratio))
    canvas.height = Math.max(1, Math.floor(rect.height * ratio))

    ctx.setTransform(ratio, 0, 0, ratio, 0, 0)

    const width = rect.width
    const height = rect.height
    ctx.clearRect(0, 0, width, height)

    const grid = 'rgba(148, 163, 184, 0.18)'
    const text = 'rgba(203, 213, 225, 0.72)'
    const line = mode === 'rtt' ? '#38bdf8' : '#fb7185'
    const fill = mode === 'rtt' ? 'rgba(56, 189, 248, 0.14)' : 'rgba(251, 113, 133, 0.14)'

    ctx.strokeStyle = grid
    ctx.lineWidth = 1
    for (let i = 1; i < 4; i++) {
      const y = (height / 4) * i
      ctx.beginPath()
      ctx.moveTo(0, y)
      ctx.lineTo(width, y)
      ctx.stroke()
    }

    const now = Date.now()
    const windowStart = now - 5 * 60 * 1000
    const visible = samples.filter((sample) => sample.time >= windowStart)

    if (visible.length === 0) {
      summary = 'No samples'
      ctx.fillStyle = text
      ctx.font = '12px system-ui, sans-serif'
      ctx.fillText('No samples yet', 12, height / 2)
      return
    }

    const values = visible.map((sample) => (mode === 'rtt' ? sample.rttMs : sample.loss))
    const minSampleTime = Math.min(...visible.map((sample) => sample.time))
    const maxSampleTime = Math.max(...visible.map((sample) => sample.time), now)
    const start = Math.max(windowStart, minSampleTime)
    const end = Math.max(start + 1000, maxSampleTime)

    const minRaw = Math.min(...values)
    const maxRaw = Math.max(...values)
    const avg = values.reduce((sum, value) => sum + value, 0) / values.length
    summary = mode === 'loss'
      ? `avg ${avg.toFixed(1)}% · min ${minRaw.toFixed(1)}% · max ${maxRaw.toFixed(1)}%`
      : `avg ${avg.toFixed(1)}ms · min ${minRaw.toFixed(1)}ms · max ${maxRaw.toFixed(1)}ms`
    if (mode === 'rtt' && maxRaw > rttMaxScale) {
      rttMaxScale = roundScale(maxRaw)
    }

    const minValue = 0
    const maxValue = mode === 'loss' ? 100 : rttMaxScale

    const points = visible.map((sample) => {
      const x = ((sample.time - start) / (end - start)) * width
      const value = mode === 'rtt' ? sample.rttMs : sample.loss
      const y = height - Math.min(1, Math.max(0, (value - minValue) / (maxValue - minValue))) * (height - 18) - 8
      return [x, y] as const
    })

    ctx.fillStyle = fill
    ctx.strokeStyle = line
    ctx.lineWidth = 2
    ctx.beginPath()
    points.forEach(([x, y], index) => {
      if (index === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    })
    ctx.stroke()

    ctx.lineTo(points[points.length - 1][0], height)
    ctx.lineTo(points[0][0], height)
    ctx.closePath()
    ctx.fill()

    ctx.fillStyle = text
    ctx.font = '11px system-ui, sans-serif'
    ctx.fillText(mode === 'loss' ? '100%' : `${maxValue.toFixed(0)}ms`, 8, 14)
    ctx.fillText(mode === 'loss' ? '0%' : '0ms', 8, height - 22)
    ctx.fillText(formatAge(now - start), 8, height - 8)
    ctx.fillText('now', width - 28, height - 8)
  }

  function drawKey(nextSamples: PingSample[], nextMode: 'rtt' | 'loss', nextScaleKey: string) {
    const last = nextSamples[nextSamples.length - 1]
    return `${nextScaleKey}:${nextMode}:${nextSamples.length}:${last?.time ?? 0}:${last?.rttMs ?? 0}:${last?.loss ?? 0}`
  }

  function roundScale(value: number) {
    const padded = Math.max(100, value * 1.1)
    if (padded <= 250) return Math.ceil(padded / 25) * 25
    if (padded <= 1000) return Math.ceil(padded / 100) * 100
    return Math.ceil(padded / 500) * 500
  }

  function formatAge(ms: number) {
    const seconds = Math.max(1, Math.round(ms / 1000))
    if (seconds < 60) return `${seconds}s ago`
    return `${Math.round(seconds / 60)}m ago`
  }
</script>

<section class="graph-card">
  <div class="graph-title">
    <span>{title}</span>
    <small>{samples.length} samples</small>
  </div>
  <canvas bind:this={canvas}></canvas>
  <div class="graph-summary">{summary}</div>
</section>
