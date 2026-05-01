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
  duration: { type: Number, default: 1500 },
  interval: { type: Number, default: 50 },
  trigger: { type: Boolean, default: false },
})

const emit = defineEmits(['done'])

const CHAR_POOL = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+-=[]{}|;:,.<>?/~`'
const COLORS = ['#38bdf8', '#22d3ee', '#818cf8', '#c084fc', '#34d399', '#fbbf24']

const displayChars = ref([])
let timer = null
let round = 0

function randomChar() {
  return CHAR_POOL[Math.floor(Math.random() * CHAR_POOL.length)]
}

function randomColor() {
  return COLORS[Math.floor(Math.random() * COLORS.length)]
}

function reset() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
  round = 0
  displayChars.value = props.text.split('').map(() => ({
    ch: randomChar(),
    locked: false,
    color: randomColor(),
  }))
}

function start() {
  reset()
  const totalRounds = Math.ceil(props.duration / props.interval)

  timer = setInterval(() => {
    round++
    const progress = round / totalRounds
    const progressSq = progress * progress

    displayChars.value = displayChars.value.map((item, i) => {
      if (item.locked) return item

      // 锁定概率：基础概率 × 位置因子（前面的字符更容易先锁定）× 随机扰动
      // 空格直接跳过随机字符，保持空格
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

      // 最后一轮强制锁定
      if (round >= totalRounds) {
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

    if (round >= totalRounds || displayChars.value.every(c => c.locked)) {
      clearInterval(timer)
      timer = null
      // 确保最终完全一致
      displayChars.value = props.text.split('').map(ch => ({
        ch,
        locked: true,
        color: null,
      }))
      emit('done')
    }
  }, props.interval)
}

function charStyle(char) {
  if (!char.locked && char.color) {
    return { color: char.color }
  }
  return {}
}

watch(() => props.trigger, (val) => {
  if (val) {
    // 延迟一点点开始，等展开动画开始后再启动文字解码
    setTimeout(() => start(), 80)
  } else {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    // 折叠时直接显示原文
    displayChars.value = props.text.split('').map(ch => ({
      ch,
      locked: true,
      color: null,
    }))
  }
}, { immediate: true })

onUnmounted(() => {
  if (timer) clearInterval(timer)
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
