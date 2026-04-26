<script lang="ts">
  import { onMount } from 'svelte'
  import type { PingSample } from '../lib/types'

  export let title: string
  export let samples: PingSample[] = []
  export let mode: 'rtt' | 'loss'

  let canvas: HTMLCanvasElement

  onMount(() => {
    const resize = () => drawGraph()
    window.addEventListener('resize', resize)
    drawGraph()
    return () => window.removeEventListener('resize', resize)
  })

  $: if (canvas) {
    drawGraph()
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
    const start = now - 5 * 60 * 1000
    const visible = samples.filter((sample) => sample.time >= start)

    if (visible.length === 0) {
      ctx.fillStyle = text
      ctx.font = '12px system-ui, sans-serif'
      ctx.fillText('No samples yet', 12, height / 2)
      return
    }

    const values = visible.map((sample) => (mode === 'rtt' ? sample.rttMs : sample.loss))
    const maxValue = mode === 'loss' ? 100 : Math.max(1, ...values) * 1.18

    const points = visible.map((sample) => {
      const x = ((sample.time - start) / (now - start)) * width
      const value = mode === 'rtt' ? sample.rttMs : sample.loss
      const y = height - Math.min(1, Math.max(0, value / maxValue)) * (height - 18) - 8
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
    ctx.fillText('5m ago', 8, height - 8)
    ctx.fillText('now', width - 28, height - 8)
  }
</script>

<section class="graph-card">
  <div class="graph-title">
    <span>{title}</span>
    <small>{samples.length} samples</small>
  </div>
  <canvas bind:this={canvas}></canvas>
</section>
