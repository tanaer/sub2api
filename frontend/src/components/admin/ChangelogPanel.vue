<template>
  <div class="relative" ref="triggerRef">
    <!-- 触发按钮 -->
    <button
      @click="togglePanel"
      class="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
      :aria-label="t('admin.changelog.title')"
    >
      <Icon name="scroll" size="sm" />
    </button>

    <!-- 下拉面板 -->
    <transition name="dropdown">
      <div
        v-if="isOpen"
        class="dropdown right-0 mt-2 w-80 max-h-96 overflow-hidden flex flex-col"
      >
        <!-- 标题 -->
        <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.changelog.title') }}
          </h3>
        </div>

        <!-- 更新列表 -->
        <div class="overflow-y-auto flex-1 py-2">
          <template v-if="recentChanges.length > 0">
            <div
              v-for="(item, index) in recentChanges"
              :key="index"
              class="px-4 py-2.5 hover:bg-gray-50 dark:hover:bg-dark-800/50"
            >
              <div class="flex items-start gap-3">
                <!-- 日期 -->
                <span class="mt-0.5 flex-shrink-0 text-xs text-gray-400 dark:text-dark-400">
                  {{ formatDate(item.date) }}
                </span>

                <!-- 类型标签 -->
                <span
                  class="flex-shrink-0 rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="typeStyles[item.type]?.bg"
                >
                  {{ t(`admin.changelog.types.${item.type}`) }}
                </span>

                <!-- 内容 -->
                <span class="text-sm text-gray-700 dark:text-gray-200">
                  {{ item.content }}
                </span>
              </div>
            </div>
          </template>

          <!-- 空状态 -->
          <div v-else class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('admin.changelog.empty') }}
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import changelogData from '@/data/changelog/2026.json'

const { t } = useI18n()

const isOpen = ref(false)
const triggerRef = ref<HTMLElement | null>(null)

// 合并多年数据，按日期倒序
const allChangelog = computed(() => {
  return changelogData.sort((a, b) =>
    new Date(b.date).getTime() - new Date(a.date).getTime()
  )
})

// 最近 20 条
const recentChanges = computed(() => allChangelog.value.slice(0, 20))

// 类型样式
const typeStyles: Record<string, { bg: string }> = {
  feature: { bg: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  fix: { bg: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
  improvement: { bg: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  perf: { bg: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400' },
  docs: { bg: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300' }
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}/${date.getDate()}`
}

function togglePanel() {
  isOpen.value = !isOpen.value
}

function handleClickOutside(event: MouseEvent) {
  if (triggerRef.value && !triggerRef.value.contains(event.target as Node)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}
</style>
