<template>
  <BaseDialog :show="show" :title="title" width="wide" @close="emit('close')">
    <div class="space-y-3">
      <p
        v-if="showAcceptButton && requireScrollToEnd && !hasReachedBottom"
        class="text-xs text-amber-600 dark:text-amber-400"
      >
        {{ t('auth.userAgreementScrollHint') }}
      </p>

      <div
        ref="contentRef"
        class="max-h-[60vh] overflow-y-auto rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900"
        @scroll="updateScrollState"
      >
        <article class="agreement-content" v-html="renderedContent"></article>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.close') }}
        </button>
        <button
          v-if="showAcceptButton"
          type="button"
          class="btn btn-primary"
          :disabled="!canAccept"
          @click="handleAccept"
        >
          {{ t('auth.userAgreementAccept') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = withDefaults(
  defineProps<{
    show: boolean
    title: string
    content: string
    showAcceptButton?: boolean
    requireScrollToEnd?: boolean
  }>(),
  {
    showAcceptButton: false,
    requireScrollToEnd: false
  }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'accept'): void
}>()

const { t } = useI18n()

marked.setOptions({
  gfm: true,
  breaks: true
})

const contentRef = ref<HTMLElement | null>(null)
const hasReachedBottom = ref(false)

const renderedContent = computed(() => {
  const html = marked.parse(props.content || '') as string
  return DOMPurify.sanitize(html)
})

const canAccept = computed(() => {
  if (!props.showAcceptButton) return false
  if (!props.requireScrollToEnd) return true
  return hasReachedBottom.value
})

function updateScrollState(): void {
  const element = contentRef.value
  if (!element) {
    hasReachedBottom.value = false
    return
  }

  const maxScrollTop = element.scrollHeight - element.clientHeight
  if (maxScrollTop <= 4) {
    hasReachedBottom.value = true
    return
  }

  hasReachedBottom.value = element.scrollTop + element.clientHeight >= element.scrollHeight - 4
}

function handleAccept(): void {
  if (!canAccept.value) return
  emit('accept')
}

watch(
  () => [props.show, props.content] as const,
  async ([show]) => {
    if (!show) {
      hasReachedBottom.value = false
      return
    }

    await nextTick()
    updateScrollState()
  }
)
</script>

<style scoped>
.agreement-content {
  color: rgb(31 41 55);
  font-size: 0.95rem;
  line-height: 1.7;
  word-break: break-word;
}

.agreement-content :deep(h1),
.agreement-content :deep(h2),
.agreement-content :deep(h3),
.agreement-content :deep(h4) {
  margin-top: 1.25rem;
  margin-bottom: 0.75rem;
  font-weight: 700;
  color: rgb(17 24 39);
}

.agreement-content :deep(h1) {
  font-size: 1.5rem;
}

.agreement-content :deep(h2) {
  font-size: 1.25rem;
}

.agreement-content :deep(h3) {
  font-size: 1.1rem;
}

.agreement-content :deep(p),
.agreement-content :deep(ul),
.agreement-content :deep(ol),
.agreement-content :deep(blockquote) {
  margin: 0.75rem 0;
}

.agreement-content :deep(ul),
.agreement-content :deep(ol) {
  padding-left: 1.25rem;
}

.agreement-content :deep(li) {
  margin: 0.35rem 0;
}

.agreement-content :deep(a) {
  color: rgb(8 145 178);
  text-decoration: underline;
}

.agreement-content :deep(code) {
  border-radius: 0.375rem;
  background: rgb(229 231 235);
  padding: 0.1rem 0.35rem;
  font-size: 0.875em;
}

.agreement-content :deep(pre) {
  overflow-x: auto;
  border-radius: 0.75rem;
  background: rgb(17 24 39);
  padding: 0.875rem;
  color: rgb(243 244 246);
}

.agreement-content :deep(pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
}

.agreement-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 1rem 0;
}

.agreement-content :deep(th),
.agreement-content :deep(td) {
  border: 1px solid rgb(209 213 219);
  padding: 0.625rem;
  text-align: left;
}

.dark .agreement-content {
  color: rgb(226 232 240);
}

.dark .agreement-content :deep(h1),
.dark .agreement-content :deep(h2),
.dark .agreement-content :deep(h3),
.dark .agreement-content :deep(h4) {
  color: rgb(255 255 255);
}

.dark .agreement-content :deep(code) {
  background: rgb(31 41 55);
}

.dark .agreement-content :deep(th),
.dark .agreement-content :deep(td) {
  border-color: rgb(75 85 99);
}
</style>
