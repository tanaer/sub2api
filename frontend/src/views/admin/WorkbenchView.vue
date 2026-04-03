<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl p-4 sm:p-6">
      <section
        class="mb-6 overflow-hidden rounded-[28px] border border-slate-200 bg-[radial-gradient(circle_at_top_left,_rgba(15,118,110,0.16),_transparent_38%),linear-gradient(135deg,_#f8fafc,_#eefbf7)] shadow-sm"
      >
        <div class="flex flex-col gap-4 p-5 sm:p-7 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <p class="text-xs font-semibold uppercase tracking-[0.28em] text-teal-700">Workbench</p>
            <h1 class="mt-2 text-2xl font-semibold text-slate-900 sm:text-3xl">
              {{ t('admin.workbench.title') }}
            </h1>
            <p class="mt-3 text-sm leading-6 text-slate-600 sm:text-base">
              {{ t('admin.workbench.description') }}
            </p>
          </div>
          <div class="flex flex-wrap gap-3">
            <button class="btn btn-secondary" @click="loadPresets">刷新预设</button>
            <button class="btn btn-primary" @click="openPresetManager">管理预设</button>
            <button class="btn btn-secondary" @click="openTemplateManager">管理模板</button>
          </div>
        </div>
      </section>

      <section class="mb-6 rounded-[28px] border border-slate-200 bg-white p-4 shadow-sm sm:p-6">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-slate-900 sm:text-xl">智能查询</h2>
            <p class="mt-1 text-sm leading-6 text-slate-500">
              支持整段消息、多行文本和混合内容，自动提取全部 API Key 或兑换码，并返回用户、用户生成的全部 API Key、生成时间、最后成功调用时间和成功调用次数。
            </p>
          </div>
          <button
            data-testid="copy-all-results"
            class="btn btn-secondary"
            :disabled="!lookupResults?.items.length"
            @click="copyAllLookupCards"
          >
            复制全部结果
          </button>
        </div>

        <div class="mt-4 rounded-3xl border border-slate-200 bg-slate-50 p-3 sm:p-4">
          <textarea
            data-testid="lookup-input"
            v-model="lookupInput"
            rows="7"
            class="min-h-[180px] w-full resize-y rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm leading-6 text-slate-800 outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-200"
            placeholder="把用户发来的整段消息粘贴到这里，系统会自动识别其中所有 API Key 或兑换码。"
          />
          <div class="mt-3 flex flex-wrap gap-3">
            <button
              data-testid="lookup-submit"
              class="btn btn-primary"
              :disabled="lookupLoading"
              @click="handlePasteLookup"
            >
              {{ lookupLoading ? '识别中...' : '粘贴识别' }}
            </button>
            <button class="btn btn-secondary" :disabled="lookupLoading" @click="clearLookup">清空内容</button>
          </div>
        </div>

        <div v-if="lookupResults" class="mt-4 flex flex-wrap gap-2">
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-700">
            识别到 {{ lookupResults.extracted_keys.length }} 个
          </span>
          <span class="rounded-full bg-emerald-100 px-3 py-1 text-xs font-medium text-emerald-700">
            已匹配 {{ lookupResults.matched_count }} 个
          </span>
          <span class="rounded-full bg-amber-100 px-3 py-1 text-xs font-medium text-amber-700">
            未匹配 {{ lookupResults.unmatched_count }} 个
          </span>
        </div>

        <div v-if="lookupResults?.items.length" class="mt-6 grid gap-4">
          <article
            v-for="item in lookupResults.items"
            :key="item.extracted_key"
            class="rounded-[26px] border border-slate-200 bg-white p-4 shadow-sm sm:p-5"
          >
            <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span
                    class="rounded-full px-3 py-1 text-xs font-semibold"
                    :class="
                      item.matched
                        ? 'bg-emerald-100 text-emerald-700'
                        : 'bg-slate-200 text-slate-700'
                    "
                  >
                    {{ item.matched ? '已匹配' : '未匹配' }}
                  </span>
                </div>
                <div class="mt-3 break-all rounded-2xl bg-slate-900 px-4 py-3 font-mono text-sm text-slate-100">
                  {{ item.api_key || item.extracted_key }}
                </div>
              </div>
              <div class="flex flex-wrap gap-2">
                <button class="btn btn-secondary" @click="copyText(item.api_key || item.extracted_key)">复制 Key</button>
                <button class="btn btn-secondary" @click="copyLookupCard(item)">复制卡片</button>
              </div>
            </div>

            <dl class="mt-4 grid gap-3 text-sm text-slate-600 sm:grid-cols-2 xl:grid-cols-5">
              <div class="rounded-2xl bg-slate-50 px-4 py-3">
                <dt class="text-xs font-medium uppercase tracking-wide text-slate-400">用户</dt>
                <dd class="mt-1 break-all text-slate-900">
                  {{ item.user_email || '-' }}
                  <span v-if="item.username" class="block text-xs text-slate-500">{{ item.username }}</span>
                </dd>
              </div>
              <div class="rounded-2xl bg-slate-50 px-4 py-3">
                <dt class="text-xs font-medium uppercase tracking-wide text-slate-400">用户状态</dt>
                <dd class="mt-2 flex flex-wrap items-center justify-between gap-3">
                  <span class="text-sm font-medium" :class="userStatusClass(item.user_status)">
                    {{ userStatusLabel(item.user_status) }}
                  </span>
                  <button
                    v-if="item.matched && item.user_id"
                    :data-testid="`toggle-user-status-${item.user_id}`"
                    class="btn btn-secondary"
                    :disabled="togglingUserId === item.user_id"
                    @click="toggleUserStatus(item)"
                  >
                    {{ togglingUserId === item.user_id ? '处理中...' : userToggleLabel(item.user_status) }}
                  </button>
                </dd>
              </div>
              <div class="rounded-2xl bg-slate-50 px-4 py-3">
                <dt class="text-xs font-medium uppercase tracking-wide text-slate-400">最近兑换时间</dt>
                <dd class="mt-1 text-slate-900">{{ formatDateTime(item.latest_redeem_at) }}</dd>
              </div>
              <div class="rounded-2xl bg-slate-50 px-4 py-3">
                <dt class="text-xs font-medium uppercase tracking-wide text-slate-400">最后成功调用时间</dt>
                <dd class="mt-1 text-slate-900">{{ formatDateTime(item.last_success_at) }}</dd>
              </div>
              <div class="rounded-2xl bg-slate-50 px-4 py-3">
                <dt class="text-xs font-medium uppercase tracking-wide text-slate-400">该用户 API Key 数量</dt>
                <dd class="mt-1 text-lg font-semibold text-slate-900">{{ item.api_keys.length }}</dd>
              </div>
              <div class="rounded-2xl bg-slate-50 px-4 py-3">
                <dt class="text-xs font-medium uppercase tracking-wide text-slate-400">总成功调用次数</dt>
                <dd class="mt-1 text-lg font-semibold text-slate-900">{{ item.success_call_count }}</dd>
              </div>
            </dl>

            <div class="mt-4 rounded-3xl border border-slate-200 bg-slate-50 p-4">
              <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h3 class="text-sm font-semibold text-slate-900">该用户生成的 API Key</h3>
                  <p class="text-xs leading-5 text-slate-500">
                    展示该用户名下全部 API Key 的生成时间、最后成功调用时间和成功调用次数。未使用的 Key 可直接删除。
                  </p>
                </div>
              </div>

              <div v-if="item.api_keys.length" class="mt-4 grid gap-3">
                <div
                  v-for="userApiKey in item.api_keys"
                  :key="userApiKey.id"
                  class="rounded-2xl border border-slate-200 bg-white px-4 py-3"
                >
                  <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div class="min-w-0">
                      <div class="break-all rounded-2xl bg-slate-900 px-4 py-3 font-mono text-sm text-slate-100">
                        {{ userApiKey.key }}
                      </div>
                      <div class="mt-3 grid gap-3 text-sm text-slate-600 md:grid-cols-4">
                        <div class="rounded-2xl bg-slate-50 px-3 py-2">
                          <p class="text-[11px] uppercase tracking-wide text-slate-400">生成时间</p>
                          <p class="mt-1 text-slate-900">{{ formatDateTime(userApiKey.created_at) }}</p>
                        </div>
                        <div class="rounded-2xl bg-slate-50 px-3 py-2">
                          <p class="text-[11px] uppercase tracking-wide text-slate-400">最后成功调用时间</p>
                          <p class="mt-1 text-slate-900">{{ formatDateTime(userApiKey.last_success_at) }}</p>
                        </div>
                        <div class="rounded-2xl bg-slate-50 px-3 py-2">
                          <p class="text-[11px] uppercase tracking-wide text-slate-400">成功调用次数</p>
                          <p class="mt-1 font-semibold text-slate-900">{{ userApiKey.success_call_count }}</p>
                        </div>
                        <div class="rounded-2xl bg-slate-50 px-3 py-2">
                          <p class="text-[11px] uppercase tracking-wide text-slate-400">状态</p>
                          <p class="mt-1 font-medium" :class="apiKeyStatusClass(userApiKey.status)">
                            {{ apiKeyStatusLabel(userApiKey.status) }}
                          </p>
                        </div>
                        <div class="rounded-2xl bg-slate-50 px-3 py-2 md:col-span-4">
                          <p class="text-[11px] uppercase tracking-wide text-slate-400">Tokens 用量</p>
                          <div class="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
                            <span>
                              <span class="text-slate-400">in:</span>
                              <span class="font-semibold text-blue-600">{{ formatNumber(userApiKey.total_input_tokens) }}</span>
                            </span>
                            <span>
                              <span class="text-slate-400">out:</span>
                              <span class="font-semibold text-emerald-600">{{ formatNumber(userApiKey.total_output_tokens) }}</span>
                            </span>
                            <span v-if="(userApiKey.total_cache_creation_tokens ?? 0) > 0">
                              <span class="text-slate-400">cache-w:</span>
                              <span class="font-semibold text-amber-600">{{ formatNumber(userApiKey.total_cache_creation_tokens) }}</span>
                            </span>
                            <span v-if="(userApiKey.total_cache_read_tokens ?? 0) > 0">
                              <span class="text-slate-400">cache-r:</span>
                              <span class="font-semibold text-purple-600">{{ formatNumber(userApiKey.total_cache_read_tokens) }}</span>
                            </span>
                            <span class="text-slate-400">
                              total: <span class="font-semibold text-slate-900">{{ formatNumber(userApiKey.total_tokens) }}</span>
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                    <div class="flex flex-wrap gap-2">
                      <button class="btn btn-secondary whitespace-nowrap" @click="copyText(userApiKey.key)">复制这个 Key</button>
                      <button
                        v-if="userApiKey.success_call_count === 0"
                        :data-testid="`delete-user-api-key-${userApiKey.id}`"
                        class="btn btn-secondary whitespace-nowrap border-red-200 text-red-600 hover:bg-red-50"
                        :disabled="deletingAPIKeyId === userApiKey.id"
                        @click="deleteUnusedAPIKey(userApiKey)"
                      >
                        {{ deletingAPIKeyId === userApiKey.id ? '删除中...' : '删除' }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
              <div
                v-else
                class="mt-4 rounded-2xl border border-dashed border-slate-200 bg-white px-4 py-6 text-sm text-slate-500"
              >
                该用户当前暂无 API Key。
              </div>
            </div>
          </article>
        </div>

        <div
          v-else-if="lookupResults"
          class="mt-6 rounded-3xl border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm text-slate-500"
        >
          暂未识别到可查询的 API Key 或兑换码。
        </div>
      </section>

      <section class="rounded-[28px] border border-slate-200 bg-white p-4 shadow-sm sm:p-6">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-slate-900 sm:text-xl">兑换码快捷生成</h2>
            <p class="mt-1 text-sm leading-6 text-slate-500">
              预设按钮全局共享，点击即可按规则生成 1 个兑换码和对应话术。
            </p>
          </div>
          <button class="btn btn-secondary" @click="openPresetManager">编辑预设</button>
        </div>

        <div
          v-if="enabledPresets.length"
          class="mt-4 flex flex-wrap items-stretch gap-3"
        >
          <button
            v-for="preset in enabledPresets"
            :key="preset.id"
            :data-testid="`preset-generate-${preset.id}`"
            class="min-w-[180px] flex-1 rounded-3xl border border-slate-200 bg-[linear-gradient(180deg,_#ffffff,_#f8fafc)] px-4 py-4 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-teal-300 hover:shadow"
            :disabled="generatingPresetId === preset.id"
            @click="generatePreset(preset)"
          >
            <div class="flex items-center justify-between gap-3">
              <span class="text-sm font-semibold text-slate-900">{{ preset.name }}</span>
              <span class="text-xs font-medium text-teal-700">
                {{ generatingPresetId === preset.id ? '生成中...' : '一键生成' }}
              </span>
            </div>
            <p class="mt-2 text-xs leading-5 text-slate-500">{{ presetSummary(preset) }}</p>
          </button>
        </div>
        <div
          v-else
          class="mt-4 rounded-3xl border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm text-slate-500"
        >
          暂无可用预设，先点击“编辑预设”创建。
        </div>

        <article
          v-if="generatedResult"
          class="mt-6 rounded-[26px] border border-slate-200 bg-[linear-gradient(180deg,_#f8fafc,_#ffffff)] p-4 shadow-sm sm:p-5"
        >
          <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.2em] text-teal-700">Latest Result</p>
              <h3 class="mt-2 text-lg font-semibold text-slate-900">
                {{ generatedResult.preset.name }}
              </h3>
            </div>
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-secondary" @click="copyGeneratedCode">复制兑换码</button>
              <button
                data-testid="copy-generated-message"
                class="btn btn-secondary"
                @click="copyGeneratedMessage"
              >
                复制话术
              </button>
              <button class="btn btn-primary" @click="copyGeneratedCard">复制完整内容</button>
            </div>
          </div>

          <div class="mt-4 grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
            <div class="rounded-3xl bg-slate-900 px-4 py-5">
              <p class="text-xs font-medium uppercase tracking-wide text-slate-400">兑换码</p>
              <p class="mt-3 break-all font-mono text-lg text-slate-100">{{ generatedResult.code }}</p>
            </div>
            <div class="rounded-3xl border border-slate-200 bg-slate-50 px-4 py-5">
              <p class="text-xs font-medium uppercase tracking-wide text-slate-400">话术</p>
              <pre class="mt-3 whitespace-pre-wrap break-words text-sm leading-6 text-slate-800">{{ generatedResult.rendered_message }}</pre>
            </div>
          </div>
        </article>
      </section>
    </div>

    <Teleport to="body">
      <div v-if="showPresetManager" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-slate-950/45" @click="closePresetManager"></div>
        <div class="relative z-10 flex max-h-[90vh] w-full max-w-6xl flex-col overflow-hidden rounded-[28px] bg-white shadow-2xl">
          <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4 sm:px-6">
            <div>
              <h3 class="text-lg font-semibold text-slate-900">预设管理</h3>
              <p class="mt-1 text-sm text-slate-500">预设全局共享，保存后所有管理员都能直接使用。</p>
            </div>
            <button class="btn btn-secondary" @click="closePresetManager">关闭</button>
          </div>

          <div class="grid min-h-0 flex-1 gap-0 lg:grid-cols-[280px_minmax(0,1fr)]">
            <aside class="border-b border-slate-200 bg-slate-50 p-4 lg:border-b-0 lg:border-r">
              <button class="btn btn-primary w-full" @click="addPresetDraft">新增预设</button>
              <div class="mt-4 space-y-2">
                <button
                  v-for="preset in presetDrafts"
                  :key="preset.id"
                  class="w-full rounded-2xl border px-3 py-3 text-left transition"
                  :class="
                    activePresetId === preset.id
                      ? 'border-teal-400 bg-white shadow-sm'
                      : 'border-slate-200 bg-white hover:border-slate-300'
                  "
                  @click="activePresetId = preset.id"
                >
                  <div class="flex items-center justify-between gap-3">
                    <span class="truncate text-sm font-medium text-slate-900">{{ preset.name || '未命名预设' }}</span>
                    <span
                      class="rounded-full px-2 py-0.5 text-[11px] font-medium"
                      :class="preset.enabled ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-200 text-slate-600'"
                    >
                      {{ preset.enabled ? '启用' : '停用' }}
                    </span>
                  </div>
                  <p class="mt-1 text-xs text-slate-500">{{ presetSummary(preset) }}</p>
                </button>
              </div>
            </aside>

            <div class="min-h-0 overflow-y-auto p-5 sm:p-6">
              <div v-if="activePreset" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="block">
                    <span class="mb-2 block text-sm font-medium text-slate-700">预设名称</span>
                    <input v-model="activePreset.name" class="input" type="text" placeholder="例如：50额度" />
                  </label>
                  <label class="block">
                    <span class="mb-2 block text-sm font-medium text-slate-700">预设 ID</span>
                    <input v-model="activePreset.id" class="input" type="text" placeholder="唯一标识" />
                  </label>
                </div>

                <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                  <label class="block">
                    <span class="mb-2 block text-sm font-medium text-slate-700">类型</span>
                    <select v-model="activePreset.type" class="input">
                      <option value="balance">余额</option>
                      <option value="concurrency">并发</option>
                      <option value="subscription">订阅</option>
                      <option value="invitation">邀请码</option>
                      <option value="group_request_quota">分组次数</option>
                    </select>
                  </label>
                  <label class="block" v-if="showPresetValue(activePreset.type)">
                    <span class="mb-2 block text-sm font-medium text-slate-700">数值</span>
                    <input v-model.number="activePreset.value" class="input" type="number" min="0" />
                  </label>
                  <label class="block" v-if="showPresetGroup(activePreset.type)">
                    <span class="mb-2 block text-sm font-medium text-slate-700">分组</span>
                    <select v-model.number="activePreset.group_id" class="input">
                      <option :value="undefined">请选择分组</option>
                      <option v-for="group in groups" :key="group.id" :value="group.id">
                        {{ group.name }}
                      </option>
                    </select>
                  </label>
                  <label class="block" v-if="showPresetValidity(activePreset.type)">
                    <span class="mb-2 block text-sm font-medium text-slate-700">有效天数</span>
                    <input v-model.number="activePreset.validity_days" class="input" type="number" min="1" />
                  </label>
                  <label class="block">
                    <span class="mb-2 block text-sm font-medium text-slate-700">排序</span>
                    <input v-model.number="activePreset.sort_order" class="input" type="number" />
                  </label>
                </div>

                <label class="inline-flex items-center gap-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700">
                  <input v-model="activePreset.enabled" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-teal-600 focus:ring-teal-500" />
                  <span>启用此预设</span>
                </label>

                <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                  <div class="rounded-3xl border border-slate-200 bg-slate-50 p-4">
                    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <div>
                        <h4 class="text-sm font-semibold text-slate-900">选择话术模板</h4>
                        <p class="mt-1 text-xs leading-5 text-slate-500">
                          话术模板独立管理，预设只选择要使用的模板。
                        </p>
                      </div>
                      <button class="btn btn-secondary" @click="openTemplateManager">管理模板</button>
                    </div>

                    <label class="mt-4 block">
                      <span class="mb-2 block text-sm font-medium text-slate-700">话术模板</span>
                      <select v-model="activePreset.template_id" class="input">
                        <option value="">请选择模板</option>
                        <option v-for="template in templates" :key="template.id" :value="template.id">
                          {{ template.name }}{{ template.enabled ? '' : '（停用）' }}
                        </option>
                      </select>
                    </label>
                  </div>

                  <div class="rounded-3xl border border-slate-200 bg-slate-50 p-4">
                    <div>
                      <h4 class="text-sm font-semibold text-slate-900">模板预览</h4>
                      <p class="mt-1 text-xs leading-5 text-slate-500">
                        当前只支持变量 <code class="rounded bg-slate-200 px-1.5 py-0.5 text-[11px]">{{ templateVariableLabel }}</code>
                      </p>
                    </div>
                    <pre class="mt-4 whitespace-pre-wrap break-words rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm leading-6 text-slate-800">{{ resolvePresetTemplatePreview(activePreset) }}</pre>
                  </div>
                </div>

                <div class="flex flex-wrap justify-between gap-3 border-t border-slate-200 pt-4">
                  <button class="btn btn-secondary" :disabled="presetDrafts.length <= 1" @click="removeActivePreset">
                    删除当前预设
                  </button>
                  <div class="flex flex-wrap gap-3">
                    <button class="btn btn-secondary" @click="resetPresetDrafts">重置</button>
                    <button class="btn btn-primary" :disabled="savingPresets" @click="savePresetDrafts">
                      {{ savingPresets ? '保存中...' : '保存预设' }}
                    </button>
                  </div>
                </div>
              </div>
              <div v-else class="rounded-3xl border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm text-slate-500">
                先新增一个预设开始配置。
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showTemplateManager" class="fixed inset-0 z-[60] flex items-center justify-center p-4">
        <div class="absolute inset-0 bg-slate-950/45" @click="closeTemplateManager"></div>
        <div class="relative z-10 flex max-h-[90vh] w-full max-w-6xl flex-col overflow-hidden rounded-[28px] bg-white shadow-2xl">
          <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4 sm:px-6">
            <div>
              <h3 class="text-lg font-semibold text-slate-900">模板管理</h3>
              <p class="mt-1 text-sm text-slate-500">支持多行文本，变量只支持 {{ templateVariableLabel }}。</p>
            </div>
            <button class="btn btn-secondary" @click="closeTemplateManager">关闭</button>
          </div>

          <div class="grid min-h-0 flex-1 gap-0 lg:grid-cols-[280px_minmax(0,1fr)]">
            <aside class="border-b border-slate-200 bg-slate-50 p-4 lg:border-b-0 lg:border-r">
              <button class="btn btn-primary w-full" @click="addTemplateDraft">新增模板</button>
              <div class="mt-4 space-y-2">
                <button
                  v-for="template in templateDrafts"
                  :key="template.id"
                  class="w-full rounded-2xl border px-3 py-3 text-left transition"
                  :class="
                    activeTemplateId === template.id
                      ? 'border-teal-400 bg-white shadow-sm'
                      : 'border-slate-200 bg-white hover:border-slate-300'
                  "
                  @click="activeTemplateId = template.id"
                >
                  <div class="flex items-center justify-between gap-3">
                    <span class="truncate text-sm font-medium text-slate-900">{{ template.name || '未命名模板' }}</span>
                    <span
                      class="rounded-full px-2 py-0.5 text-[11px] font-medium"
                      :class="template.enabled ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-200 text-slate-600'"
                    >
                      {{ template.enabled ? '启用' : '停用' }}
                    </span>
                  </div>
                  <p class="mt-1 text-xs text-slate-500">{{ templateSummary(template) }}</p>
                </button>
              </div>
            </aside>

            <div class="min-h-0 overflow-y-auto p-5 sm:p-6">
              <div v-if="activeTemplate" class="space-y-5">
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="block">
                    <span class="mb-2 block text-sm font-medium text-slate-700">模板名称</span>
                    <input v-model="activeTemplate.name" class="input" type="text" placeholder="例如：默认话术" />
                  </label>
                  <label class="block">
                    <span class="mb-2 block text-sm font-medium text-slate-700">模板 ID</span>
                    <input v-model="activeTemplate.id" class="input" type="text" placeholder="唯一标识" />
                  </label>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="block">
                    <span class="mb-2 block text-sm font-medium text-slate-700">排序</span>
                    <input v-model.number="activeTemplate.sort_order" class="input" type="number" />
                  </label>
                  <label class="inline-flex items-center gap-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700">
                    <input v-model="activeTemplate.enabled" type="checkbox" class="h-4 w-4 rounded border-slate-300 text-teal-600 focus:ring-teal-500" />
                    <span>启用此模板</span>
                  </label>
                </div>

                <div class="rounded-3xl border border-slate-200 bg-slate-50 p-4">
                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <h4 class="text-sm font-semibold text-slate-900">模板内容</h4>
                      <p class="mt-1 text-xs leading-5 text-slate-500">可点击插入变量到当前光标位置。</p>
                    </div>
                    <button class="btn btn-secondary" @click="insertTemplateVariable">插入 {{ templateVariableLabel }}</button>
                  </div>
                  <textarea
                    ref="templateTextareaRef"
                    v-model="activeTemplate.content"
                    class="mt-4 min-h-[260px] w-full rounded-3xl border border-slate-200 bg-white px-4 py-3 text-sm leading-6 text-slate-800 outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-200"
                  />
                </div>

                <div class="flex flex-wrap justify-between gap-3 border-t border-slate-200 pt-4">
                  <button class="btn btn-secondary" :disabled="templateDrafts.length <= 1" @click="removeActiveTemplate">
                    删除当前模板
                  </button>
                  <div class="flex flex-wrap gap-3">
                    <button class="btn btn-secondary" @click="resetTemplateDrafts">重置</button>
                    <button class="btn btn-primary" :disabled="savingTemplates" @click="saveTemplateDrafts">
                      {{ savingTemplates ? '保存中...' : '保存模板' }}
                    </button>
                  </div>
                </div>
              </div>
              <div v-else class="rounded-3xl border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm text-slate-500">
                先新增一个模板开始配置。
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores/app'
import type {
  AdminGroup,
  RedeemCodeType,
  WorkbenchLookupItem,
  WorkbenchLookupResponse,
  WorkbenchLookupUserApiKey,
  WorkbenchRedeemPreset,
  WorkbenchRedeemPresetGenerateResponse,
  WorkbenchRedeemTemplate,
} from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const templateVariableLabel = '{{code}}'

const lookupInput = ref('')
const lookupLoading = ref(false)
const lookupResults = ref<WorkbenchLookupResponse | null>(null)
const togglingUserId = ref<number | null>(null)
const deletingAPIKeyId = ref<number | null>(null)

const presets = ref<WorkbenchRedeemPreset[]>([])
const templates = ref<WorkbenchRedeemTemplate[]>([])
const generatingPresetId = ref<string | null>(null)
const generatedResult = ref<WorkbenchRedeemPresetGenerateResponse | null>(null)

const showPresetManager = ref(false)
const savingPresets = ref(false)
const presetDrafts = ref<WorkbenchRedeemPreset[]>([])
const activePresetId = ref<string | null>(null)

const showTemplateManager = ref(false)
const savingTemplates = ref(false)
const templateDrafts = ref<WorkbenchRedeemTemplate[]>([])
const activeTemplateId = ref<string | null>(null)
const templateTextareaRef = ref<HTMLTextAreaElement | null>(null)

const groups = ref<AdminGroup[]>([])
const groupsLoaded = ref(false)

const enabledPresets = computed(() =>
  [...presets.value]
    .filter((preset) => preset.enabled)
    .sort((left, right) => left.sort_order - right.sort_order || left.name.localeCompare(right.name)),
)

const activePreset = computed<WorkbenchRedeemPreset | undefined>(() =>
  presetDrafts.value.find((preset) => preset.id === activePresetId.value),
)

const activeTemplate = computed<WorkbenchRedeemTemplate | undefined>(() =>
  templateDrafts.value.find((template) => template.id === activeTemplateId.value),
)

onMounted(async () => {
  await Promise.all([loadPresets(), loadTemplates()])
})

async function loadPresets() {
  try {
    presets.value = sortPresets(await adminAPI.tools.getRedeemPresets())
  } catch (error) {
    appStore.showError(getErrorMessage(error, '加载预设失败'))
  }
}

async function loadTemplates() {
  try {
    templates.value = sortTemplates(await adminAPI.tools.getRedeemTemplates())
  } catch (error) {
    appStore.showError(getErrorMessage(error, '加载模板失败'))
  }
}

async function runLookup(rawText: string) {
  try {
    lookupResults.value = await adminAPI.tools.lookupAPIKeys(rawText)
  } catch (error) {
    appStore.showError(getErrorMessage(error, '查询失败'))
  }
}

async function handlePasteLookup() {
  if (lookupLoading.value) {
    return
  }

  lookupInput.value = ''
  lookupResults.value = null
  lookupLoading.value = true

  try {
    const clipboardText = (await navigator.clipboard.readText()).trim()
    if (!clipboardText) {
      throw new Error('剪贴板为空')
    }

    lookupInput.value = clipboardText
    await runLookup(clipboardText)
  } catch (error) {
    appStore.showError(getErrorMessage(error, '读取剪贴板失败'))
  } finally {
    lookupLoading.value = false
  }
}

function clearLookup() {
  lookupInput.value = ''
  lookupResults.value = null
}

async function toggleUserStatus(item: WorkbenchLookupItem) {
  if (!item.user_id) {
    return
  }

  const nextStatus = item.user_status === 'active' ? 'disabled' : 'active'
  togglingUserId.value = item.user_id
  try {
    const updated = await adminAPI.users.toggleStatus(item.user_id, nextStatus)
    item.user_status = updated.status
    appStore.showSuccess(`用户已${updated.status === 'active' ? '启用' : '禁用'}`)
  } catch (error) {
    appStore.showError(getErrorMessage(error, '切换用户状态失败'))
  } finally {
    togglingUserId.value = null
  }
}

async function generatePreset(preset: WorkbenchRedeemPreset) {
  generatingPresetId.value = preset.id
  try {
    generatedResult.value = await adminAPI.tools.generateRedeemPreset(preset.id)
    appStore.showSuccess('兑换码已生成')
  } catch (error) {
    appStore.showError(getErrorMessage(error, '生成兑换码失败'))
  } finally {
    generatingPresetId.value = null
  }
}

async function copyText(text: string) {
  if (!text) {
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    appStore.showSuccess('已复制')
  } catch (error) {
    appStore.showError(getErrorMessage(error, '复制失败'))
  }
}

async function deleteUnusedAPIKey(userApiKey: WorkbenchLookupUserApiKey) {
  deletingAPIKeyId.value = userApiKey.id
  try {
    await adminAPI.apiKeys.delete(userApiKey.id)
    removeLookupAPIKey(userApiKey.id)
    appStore.showSuccess('API Key 已删除')
  } catch (error) {
    appStore.showError(getErrorMessage(error, '删除 API Key 失败'))
  } finally {
    deletingAPIKeyId.value = null
  }
}

function copyLookupCard(item: WorkbenchLookupItem) {
  void copyText(formatLookupCard(item))
}

function copyAllLookupCards() {
  if (!lookupResults.value?.items.length) {
    return
  }
  void copyText(lookupResults.value.items.map(formatLookupCard).join('\n\n'))
}

function copyGeneratedCode() {
  if (generatedResult.value) {
    void copyText(generatedResult.value.code)
  }
}

function copyGeneratedMessage() {
  if (generatedResult.value) {
    void copyText(generatedResult.value.rendered_message)
  }
}

function copyGeneratedCard() {
  if (!generatedResult.value) {
    return
  }
  const lines = [
    `预设: ${generatedResult.value.preset.name}`,
    `兑换码: ${generatedResult.value.code}`,
    '话术:',
    generatedResult.value.rendered_message,
  ]
  void copyText(lines.join('\n'))
}

function removeLookupAPIKey(apiKeyID: number) {
  if (!lookupResults.value) {
    return
  }

  for (const item of lookupResults.value.items) {
    const hasTarget = item.api_keys.some((userApiKey) => userApiKey.id === apiKeyID)
    if (!hasTarget) {
      continue
    }

    item.api_keys = item.api_keys.filter((userApiKey) => userApiKey.id !== apiKeyID)
    item.success_call_count = item.api_keys.reduce(
      (total, userApiKey) => total + userApiKey.success_call_count,
      0,
    )
    item.last_success_at = getLatestSuccessAt(item.api_keys)
  }
}

function openPresetManager() {
  resetPresetDrafts()
  showPresetManager.value = true
  void ensureGroupsLoaded()
}

function closePresetManager() {
  showPresetManager.value = false
}

function resetPresetDrafts() {
  presetDrafts.value = sortPresets(presets.value).map(clonePreset)
  if (!presetDrafts.value.length) {
    presetDrafts.value = [createEmptyPreset()]
  }
  activePresetId.value = presetDrafts.value[0]?.id ?? null
}

function addPresetDraft() {
  const preset = createEmptyPreset()
  presetDrafts.value = sortPresets([...presetDrafts.value, preset])
  activePresetId.value = preset.id
}

function removeActivePreset() {
  if (!activePreset.value || presetDrafts.value.length <= 1) {
    return
  }
  presetDrafts.value = presetDrafts.value.filter((preset) => preset.id !== activePreset.value?.id)
  activePresetId.value = presetDrafts.value[0]?.id ?? null
}

async function savePresetDrafts() {
  savingPresets.value = true
  try {
    const saved = await adminAPI.tools.updateRedeemPresets(sortPresets(presetDrafts.value.map(clonePreset)))
    presets.value = sortPresets(saved)
    presetDrafts.value = sortPresets(saved).map(clonePreset)
    activePresetId.value = presetDrafts.value[0]?.id ?? null
    appStore.showSuccess('预设已保存')
  } catch (error) {
    appStore.showError(getErrorMessage(error, '保存预设失败'))
  } finally {
    savingPresets.value = false
  }
}

function openTemplateManager() {
  resetTemplateDrafts()
  showTemplateManager.value = true
  void nextTick(() => {
    templateTextareaRef.value?.focus()
  })
}

function closeTemplateManager() {
  showTemplateManager.value = false
}

function resetTemplateDrafts() {
  templateDrafts.value = sortTemplates(templates.value).map(cloneTemplate)
  if (!templateDrafts.value.length) {
    templateDrafts.value = [createEmptyTemplate()]
  }
  activeTemplateId.value = templateDrafts.value[0]?.id ?? null
}

function addTemplateDraft() {
  const template = createEmptyTemplate()
  templateDrafts.value = sortTemplates([...templateDrafts.value, template])
  activeTemplateId.value = template.id
}

function removeActiveTemplate() {
  if (!activeTemplate.value || templateDrafts.value.length <= 1) {
    return
  }
  templateDrafts.value = templateDrafts.value.filter((template) => template.id !== activeTemplate.value?.id)
  activeTemplateId.value = templateDrafts.value[0]?.id ?? null
}

async function saveTemplateDrafts() {
  savingTemplates.value = true
  try {
    const saved = await adminAPI.tools.updateRedeemTemplates(sortTemplates(templateDrafts.value.map(cloneTemplate)))
    templates.value = sortTemplates(saved)
    templateDrafts.value = sortTemplates(saved).map(cloneTemplate)
    activeTemplateId.value = templateDrafts.value[0]?.id ?? null
    appStore.showSuccess('模板已保存')
  } catch (error) {
    appStore.showError(getErrorMessage(error, '保存模板失败'))
  } finally {
    savingTemplates.value = false
  }
}

async function insertTemplateVariable() {
  if (!activeTemplate.value) {
    return
  }
  const textarea = templateTextareaRef.value
  const token = '{{code}}'

  if (!textarea) {
    activeTemplate.value.content = `${activeTemplate.value.content || ''}${token}`
    return
  }

  const start = textarea.selectionStart ?? activeTemplate.value.content.length
  const end = textarea.selectionEnd ?? start
  const current = activeTemplate.value.content || ''
  activeTemplate.value.content = `${current.slice(0, start)}${token}${current.slice(end)}`

  await nextTick()
  textarea.focus()
  const nextCursor = start + token.length
  textarea.setSelectionRange(nextCursor, nextCursor)
}

async function ensureGroupsLoaded() {
  if (groupsLoaded.value) {
    return
  }
  try {
    groups.value = await adminAPI.groups.getAll()
    groupsLoaded.value = true
  } catch (error) {
    appStore.showError(getErrorMessage(error, '加载分组失败'))
  }
}

function showPresetValue(type: RedeemCodeType) {
  return type === 'balance' || type === 'concurrency' || type === 'group_request_quota'
}

function showPresetGroup(type: RedeemCodeType) {
  return type === 'subscription' || type === 'group_request_quota'
}

function showPresetValidity(type: RedeemCodeType) {
  return type === 'subscription' || type === 'group_request_quota'
}

function presetSummary(preset: WorkbenchRedeemPreset) {
  const templateName = resolveTemplateName(preset.template_id)
  switch (preset.type) {
    case 'balance':
      return `余额 ${preset.value}${templateName ? ` · ${templateName}` : ''}`
    case 'concurrency':
      return `并发 ${preset.value}${templateName ? ` · ${templateName}` : ''}`
    case 'subscription':
      return `订阅 · ${preset.validity_days || 30} 天${templateName ? ` · ${templateName}` : ''}`
    case 'invitation':
      return templateName ? `邀请码 · ${templateName}` : '邀请码'
    case 'group_request_quota':
      return `分组次数 ${preset.value} · ${preset.validity_days || 30} 天${templateName ? ` · ${templateName}` : ''}`
    default:
      return preset.type
  }
}

function templateSummary(template: WorkbenchRedeemTemplate) {
  return template.content ? template.content.split('\n')[0] : templateVariableLabel
}

function resolveTemplateName(templateID?: string) {
  if (!templateID) {
    return ''
  }
  return templates.value.find((template) => template.id === templateID)?.name || ''
}

function resolvePresetTemplatePreview(preset: WorkbenchRedeemPreset) {
  if (preset.template_id) {
    const content = templates.value.find((template) => template.id === preset.template_id)?.content
    if (content) {
      return content
    }
  }
  return preset.template || '请选择一个话术模板'
}

function userStatusLabel(status?: WorkbenchLookupItem['user_status']) {
  switch (status) {
    case 'active':
      return '已启用'
    case 'disabled':
      return '已禁用'
    default:
      return '-'
  }
}

function userStatusClass(status?: WorkbenchLookupItem['user_status']) {
  switch (status) {
    case 'active':
      return 'text-emerald-700'
    case 'disabled':
      return 'text-slate-600'
    default:
      return 'text-slate-500'
  }
}

function userToggleLabel(status?: WorkbenchLookupItem['user_status']) {
  return status === 'active' ? '禁用' : '启用'
}

function apiKeyStatusLabel(status?: string) {
  switch (status) {
    case 'active':
      return '已启用'
    case 'inactive':
    case 'disabled':
      return '已禁用'
    case 'quota_exhausted':
      return '额度耗尽'
    case 'expired':
      return '已过期'
    default:
      return status || '-'
  }
}

function apiKeyStatusClass(status?: string) {
  switch (status) {
    case 'active':
      return 'text-emerald-700'
    case 'inactive':
    case 'disabled':
      return 'text-slate-600'
    case 'quota_exhausted':
    case 'expired':
      return 'text-amber-700'
    default:
      return 'text-slate-500'
  }
}

function formatNumber(value?: number | null) {
  if (value == null || value === 0) return '0'
  return value.toLocaleString('en-US')
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return '-'
  }
  return new Date(value).toLocaleString('zh-CN', {
    hour12: false,
  })
}

function getLatestSuccessAt(apiKeys: WorkbenchLookupUserApiKey[]) {
  let latestSuccessAt: string | null = null

  for (const apiKey of apiKeys) {
    if (!apiKey.last_success_at) {
      continue
    }
    if (!latestSuccessAt || new Date(apiKey.last_success_at).getTime() > new Date(latestSuccessAt).getTime()) {
      latestSuccessAt = apiKey.last_success_at
    }
  }

  return latestSuccessAt
}

function formatLookupCard(item: WorkbenchLookupItem) {
  const lines = [
    `识别值: ${item.api_key || item.extracted_key}`,
    `状态: ${item.matched ? '已匹配' : '未匹配'}`,
    `用户: ${item.user_email || '-'}`,
    `用户名: ${item.username || '-'}`,
    `用户状态: ${userStatusLabel(item.user_status)}`,
    `最近兑换时间: ${formatDateTime(item.latest_redeem_at)}`,
    `最后成功调用时间: ${formatDateTime(item.last_success_at)}`,
    `总成功调用次数: ${item.success_call_count}`,
  ]

  if (item.api_keys.length) {
    lines.push('该用户 API Key:')
    for (const userApiKey of item.api_keys) {
      lines.push(
        `- ${userApiKey.key} | 生成时间: ${formatDateTime(userApiKey.created_at)} | 最后成功调用时间: ${formatDateTime(userApiKey.last_success_at)} | 成功调用次数: ${userApiKey.success_call_count} | Tokens: in=${userApiKey.total_input_tokens ?? 0} out=${userApiKey.total_output_tokens ?? 0} total=${userApiKey.total_tokens ?? 0}`,
      )
    }
  } else {
    lines.push('该用户 API Key: 暂无')
  }

  return lines.join('\n')
}

function sortPresets(items: WorkbenchRedeemPreset[]) {
  return [...items].sort(
    (left, right) => left.sort_order - right.sort_order || left.name.localeCompare(right.name),
  )
}

function sortTemplates(items: WorkbenchRedeemTemplate[]) {
  return [...items].sort(
    (left, right) => left.sort_order - right.sort_order || left.name.localeCompare(right.name),
  )
}

function clonePreset(preset: WorkbenchRedeemPreset): WorkbenchRedeemPreset {
  return {
    ...preset,
    group_id: preset.group_id ?? undefined,
  }
}

function cloneTemplate(template: WorkbenchRedeemTemplate): WorkbenchRedeemTemplate {
  return {
    ...template,
  }
}

function createEmptyPreset(): WorkbenchRedeemPreset {
  return {
    id: `preset-${Date.now()}`,
    name: '新预设',
    enabled: true,
    sort_order: presetDrafts.value.length + 1,
    type: 'balance',
    value: 1,
    validity_days: 30,
  }
}

function createEmptyTemplate(): WorkbenchRedeemTemplate {
  return {
    id: `template-${Date.now()}`,
    name: '新模板',
    enabled: true,
    sort_order: templateDrafts.value.length + 1,
    content: '{{code}}',
  }
}

function getErrorMessage(error: unknown, fallback: string) {
  if (error && typeof error === 'object') {
    const maybeMessage = (error as { message?: string }).message
    if (maybeMessage) {
      return maybeMessage
    }
  }
  return fallback
}
</script>
