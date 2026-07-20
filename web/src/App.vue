<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import Highlight from './components/Highlight.vue'

const query = ref('')
const field = ref('')
const size = ref(20)
const from = ref(0)

const hits = ref([])
const total = ref(0)
const took = ref(0)
const cursor = ref(0)
const loading = ref(false)
const error = ref('')
const mobileDetail = ref(false)

const searchInput = ref(null)
const listEl = ref(null)

const fields = [
  { value: '', label: 'Todos los campos' },
  { value: 'subject', label: 'Asunto' },
  { value: 'from', label: 'Remitente' },
  { value: 'to', label: 'Destinatario' },
  { value: 'content', label: 'Contenido' },
]

const selected = computed(() => hits.value[cursor.value] ?? null)
const page = computed(() => Math.floor(from.value / size.value) + 1)
const pages = computed(() => Math.max(1, Math.ceil(Math.min(total.value, 10000) / size.value)))
const hasPrev = computed(() => from.value > 0)
const hasNext = computed(() => from.value + size.value < Math.min(total.value, 10000))

let timer = null
let controller = null

async function search() {
  loading.value = true
  error.value = ''

  controller?.abort()
  controller = new AbortController()

  const params = new URLSearchParams({
    q: query.value,
    from: String(from.value),
    size: String(size.value),
  })
  if (field.value) params.set('field', field.value)

  try {
    const res = await fetch(`/api/search?${params}`, { signal: controller.signal })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)

    const data = await res.json()
    hits.value = data.hits ?? []
    total.value = data.total ?? 0
    took.value = data.took ?? 0
    cursor.value = 0
    listEl.value?.scrollTo({ top: 0 })
  } catch (e) {
    if (e.name === 'AbortError') return
    error.value = 'No se pudo completar la búsqueda. Verifica que el índice esté disponible.'
    hits.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

watch([query, field], () => {
  from.value = 0
  clearTimeout(timer)
  timer = setTimeout(search, 300)
})

watch(from, search)

function move(delta) {
  if (!hits.value.length) return
  cursor.value = Math.min(Math.max(cursor.value + delta, 0), hits.value.length - 1)
  nextTick(() => {
    listEl.value
      ?.querySelector(`[data-row="${cursor.value}"]`)
      ?.scrollIntoView({ block: 'nearest' })
  })
}

function onKeydown(e) {
  const typing = document.activeElement === searchInput.value

  if ((e.key === '/' && !typing) || (e.key === 'k' && (e.metaKey || e.ctrlKey))) {
    e.preventDefault()
    searchInput.value?.focus()
    searchInput.value?.select()
    return
  }

  if (e.key === 'Escape') {
    if (typing) searchInput.value?.blur()
    else if (mobileDetail.value) mobileDetail.value = false
    return
  }

  if (e.key === 'ArrowDown' || (e.key === 'j' && !typing)) {
    e.preventDefault()
    move(1)
  } else if (e.key === 'ArrowUp' || (e.key === 'k' && !typing && !e.metaKey && !e.ctrlKey)) {
    e.preventDefault()
    move(-1)
  }
}

function select(i) {
  cursor.value = i
  mobileDetail.value = true
}

function prev() {
  if (hasPrev.value) from.value = Math.max(0, from.value - size.value)
}

function next() {
  if (hasNext.value) from.value = from.value + size.value
}

function shortDate(raw) {
  if (!raw) return ''
  const d = new Date(raw)
  if (isNaN(d)) return raw.slice(0, 16)
  return d.toLocaleDateString('es-PE', { year: 'numeric', month: 'short', day: '2-digit' })
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  search()
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  clearTimeout(timer)
  controller?.abort()
})
</script>

<template>
  <div class="flex h-full flex-col bg-slate-100 text-slate-900">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-4 sm:px-6 sm:py-5">
        <div class="flex items-baseline gap-3">
          <h1 class="text-xl font-semibold tracking-tight sm:text-2xl">
            Yana<span class="text-indigo-600">killa</span>
          </h1>
          <span class="hidden text-sm text-slate-500 sm:inline">buscador de correos</span>
        </div>

        <div class="flex flex-col gap-3 sm:flex-row">
          <div class="relative flex-1">
            <svg
              class="pointer-events-none absolute left-4 top-1/2 size-5 -translate-y-1/2 text-slate-400"
              fill="none" viewBox="0 0 24 24" stroke-width="1.8" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round"
                d="m21 21-5.2-5.2m2.2-5.3a7.5 7.5 0 1 1-15 0 7.5 7.5 0 0 1 15 0Z" />
            </svg>
            <input
              ref="searchInput"
              v-model="query"
              type="search"
              placeholder="Buscar en los correos..."
              class="w-full rounded-lg border border-slate-300 bg-white py-3 pl-12 pr-16 text-base outline-none transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200" />
            <kbd
              class="pointer-events-none absolute right-4 top-1/2 hidden -translate-y-1/2 rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 font-sans text-xs text-slate-400 sm:block">
              /
            </kbd>
          </div>

          <select
            v-model="field"
            class="rounded-lg border border-slate-300 bg-white px-4 py-3 text-sm outline-none transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200">
            <option v-for="f in fields" :key="f.value" :value="f.value">{{ f.label }}</option>
          </select>
        </div>

        <div class="flex min-h-5 items-center gap-3 text-sm text-slate-500">
          <span v-if="loading">Buscando...</span>
          <span v-else-if="error" class="text-red-600">{{ error }}</span>
          <span v-else-if="total">
            {{ total.toLocaleString('es-PE') }} resultados en {{ took }} ms
          </span>
          <span v-else>Sin resultados</span>
        </div>
      </div>
    </header>

    <main class="mx-auto flex w-full max-w-7xl flex-1 gap-4 overflow-hidden px-4 py-4 sm:px-6 sm:py-5">
      <section
        :class="[
          'w-full flex-col overflow-hidden rounded-xl border border-slate-200 bg-white md:flex md:w-2/5 md:min-w-80',
          mobileDetail ? 'hidden' : 'flex',
        ]">
        <div class="grid grid-cols-[1fr_auto] gap-2 border-b border-slate-200 bg-slate-50 px-4 py-2.5 text-xs font-semibold uppercase tracking-wide text-slate-500">
          <span>Asunto / Remitente</span>
          <span>Fecha</span>
        </div>

        <ul ref="listEl" class="flex-1 divide-y divide-slate-100 overflow-y-auto">
          <template v-if="loading && !hits.length">
            <li v-for="n in 8" :key="n" class="px-4 py-3.5">
              <div class="h-3.5 w-3/4 animate-pulse rounded bg-slate-200"></div>
              <div class="mt-2 h-3 w-1/2 animate-pulse rounded bg-slate-100"></div>
            </li>
          </template>

          <li v-for="(hit, i) in hits" :key="hit.id" :data-row="i">
            <button
              type="button"
              @click="select(i)"
              :class="[
                'grid w-full grid-cols-[1fr_auto] gap-2 px-4 py-3 text-left transition',
                cursor === i ? 'bg-indigo-50' : 'hover:bg-slate-50',
              ]">
              <span class="min-w-0">
                <span class="block truncate text-sm font-medium text-slate-900">
                  <Highlight :text="hit.subject || '(sin asunto)'" :term="query" />
                </span>
                <span class="mt-0.5 block truncate text-xs text-slate-500">
                  <Highlight :text="hit.from" :term="query" />
                </span>
              </span>
              <span class="shrink-0 text-xs text-slate-400">{{ shortDate(hit.date) }}</span>
            </button>
          </li>

          <li v-if="!hits.length && !loading" class="px-4 py-12 text-center">
            <p class="text-sm text-slate-500">No hay correos que coincidan</p>
            <p class="mt-1 text-xs text-slate-400">Prueba con otro término o cambia el campo de búsqueda</p>
          </li>
        </ul>

        <div class="flex items-center justify-between border-t border-slate-200 px-4 py-2.5 text-sm">
          <button
            type="button" @click="prev" :disabled="!hasPrev"
            class="rounded-md px-3 py-1.5 text-slate-600 transition hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40">
            Anterior
          </button>
          <span class="text-xs text-slate-500">Página {{ page }} de {{ pages }}</span>
          <button
            type="button" @click="next" :disabled="!hasNext"
            class="rounded-md px-3 py-1.5 text-slate-600 transition hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40">
            Siguiente
          </button>
        </div>
      </section>

      <section
        :class="[
          'w-full flex-1 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white md:flex',
          mobileDetail ? 'flex' : 'hidden',
        ]">
        <template v-if="selected">
          <div class="border-b border-slate-200 px-4 py-4 sm:px-6">
            <button
              type="button" @click="mobileDetail = false"
              class="mb-3 -ml-1 flex items-center gap-1 rounded-md px-1 py-0.5 text-sm text-indigo-600 transition hover:bg-indigo-50 md:hidden">
              <svg class="size-4" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
              </svg>
              Resultados
            </button>

            <h2 class="text-base font-semibold text-slate-900 sm:text-lg">
              {{ selected.subject || '(sin asunto)' }}
            </h2>

            <dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
              <dt class="text-slate-400">De</dt>
              <dd class="truncate text-slate-700">{{ selected.from }}</dd>
              <dt class="text-slate-400">Para</dt>
              <dd class="truncate text-slate-700">{{ selected.to || '—' }}</dd>
              <template v-if="selected.cc">
                <dt class="text-slate-400">CC</dt>
                <dd class="truncate text-slate-700">{{ selected.cc }}</dd>
              </template>
              <dt class="text-slate-400">Fecha</dt>
              <dd class="text-slate-700">{{ selected.date || '—' }}</dd>
              <dt class="text-slate-400">Carpeta</dt>
              <dd class="truncate text-slate-500">{{ selected.folder || '—' }}</dd>
            </dl>
          </div>

          <div class="flex-1 overflow-y-auto px-4 py-5 sm:px-6">
            <pre class="whitespace-pre-wrap break-words font-sans text-sm leading-relaxed text-slate-700"><Highlight :text="selected.content" :term="query" /></pre>
          </div>
        </template>

        <div v-else class="flex flex-1 flex-col items-center justify-center gap-1 px-6 text-center">
          <p class="text-sm text-slate-400">Selecciona un correo para leerlo</p>
          <p class="hidden text-xs text-slate-300 md:block">
            Usa las flechas para navegar y / para buscar
          </p>
        </div>
      </section>
    </main>
  </div>
</template>
