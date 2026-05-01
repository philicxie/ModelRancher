<template>
  <span class="text-decode">
    <span
      v-for="(char, i) in displayChars"
      :key="i"
      class="decode-char"
      :class="{ 'locked': char.locked }"
      :style="charStyle(char)"
    >{{ char.ch }}</span>
  </span>
</template>

<script setup>
import { ref, watch, onUnmounted } from 'vue'

const props = defineProps({
  text: { type: String, default: '' },
  duration: { type: Number, default: 2000 },
  trigger: { type: Boolean, default: false },
})

const emit = defineEmits(['done'])

const CHAR_POOL = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+-=[]{}|;:,.<>?/~`'
const COLORS = ['#38bdf8', '#22d3ee', '#818cf8', '#c084fc', '#34d399', '#fbbf24']

const displayChars = ref([])
let timer = null
let elapsed = 0
let round = 0

function randomChar() {
  return CHAR_POOL[Math.floor(Math.random() * CHAR_POOL.length)]
}

function randomColor() {
  return COLORS[Math.floor(Math.random() * COLORS.length)]
}

function reset() {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
  elapsed = 0
  round = 0
  displayChars.value = props.text.split('').map(() => ({
    ch: randomChar(),
    locked: false,
    color: randomColor(),
  }))
}

function tick() {
  round++
  const progress = Math.min(elapsed / props.duration, 1)
  const progressSq = progress * progress

  displayChars.value = displayChars.value.map((item, i) => {
    if (item.locked) return item

    const isSpace = props.text[i] === ' '
    const posFactor = 1 + (props.text.length - 1 - i) / props.text.length * 0.6
    const lockChance = progressSq * posFactor * 0.85
    const rand = Math.random()

    if (!isSpace && rand < lockChance) {
      return {
        ch: props.text[i],
        locked: true,
        color: null,
      }
    }

    if (elapsed >= props.duration) {
      return {
        ch: props.text[i],
        locked: true,
        color: null,
      }
    }

    return {
      ch: isSpace ? ' ' : randomChar(),
      locked: false,
      color: randomColor(),
    }
  })

  if (elapsed >= props.duration || displayChars.value.every(c => c.locked)) {
    timer = null
    displayChars.value = props.text.split('').map(ch => ({
      ch,
      locked: true,
      color: null,
    }))
    emit('done')
    return
  }

  // interval 从 40ms 逐渐增长到 120ms
  const nextDelay = 40 + progress * 80
  elapsed += nextDelay

  timer = setTimeout(tick, nextDelay)
}

function start() {
  reset()
  tick()
}

function charStyle(char) {
  if (!char.locked && char.color) {
    return { color: char.color }
  }
  return {}
}

watch(() => props.trigger, (val) => {
  if (val) {
    setTimeout(() => start(), 80)
  } else {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
    displayChars.value = props.text.split('').map(ch => ({
      ch,
      locked: true,
      color: null,
    }))
  }
}, { immediate: true })

onUnmounted(() => {
  if (timer) clearTimeout(timer)
})
</script>

<style scoped>
.text-decode {
  display: inline-flex;
  font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', 'Consolas', 'Monaco', monospace;
  font-variant-numeric: tabular-nums;
}

.decode-char {
  display: inline-block;
  transition: color 0.15s ease;
}

.decode-char.locked {
  color: inherit;
}
</style>
