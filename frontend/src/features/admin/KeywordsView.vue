<script setup lang="ts">
import { useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import { api, ApiError, errorText, unwrap } from '@/lib/api/client'
import { RULE_SHORT } from '@/lib/forum'
import { dm, fold } from '@/lib/format'
import { keys, useBlocklist } from '@/lib/queries'
import { toast } from '@/lib/toast'
import type { BlockEntry } from '@/lib/types'
import AdminPage from './AdminPage.vue'
import { useAccounts } from './queries'

// Từ khóa chặn (FR-3.3): thêm, xóa, thử một câu. Vẫn chạy khi bộ lọc AI lỗi.
const qc = useQueryClient()
const list = useBlocklist()
const accounts = useAccounts()
const byName = (id?: string) => accounts.data.value?.find((a) => a.id === id)?.name ?? '—'

const term = ref('')
const rule = ref(6)
const err = ref('')
const busy = ref(false)
async function add() {
  err.value = term.value.trim() ? '' : 'Hãy nhập từ hoặc cụm từ.'
  if (err.value) return document.getElementById('kw-t')?.focus()
  busy.value = true
  try {
    await unwrap(api.screening.POST('/screening/blocklist', { body: { term: term.value.trim(), rule: rule.value } }))
    toast(`Đã thêm từ khóa “${term.value.trim()}”.`)
    term.value = ''
    qc.invalidateQueries({ queryKey: keys.blocklist })
  } catch (e) {
    err.value = e instanceof ApiError && e.status === 409 ? 'Từ khóa này đã có trong danh sách.' : errorText(e)
  } finally {
    busy.value = false
  }
}
async function remove(b: BlockEntry) {
  try {
    await unwrap(api.screening.DELETE('/screening/blocklist/{term_id}', { params: { path: { term_id: b.id } } }))
    toast(`Đã xóa từ khóa “${b.term}”.`)
    qc.invalidateQueries({ queryKey: keys.blocklist })
  } catch (e) {
    toast(errorText(e))
  }
}

// Thử một câu: khớp cả từ, không phân biệt hoa thường và dấu, như backend
const sample = ref('')
const words = (s: string) => ' ' + fold(s).replace(/[^\p{L}\p{N}]+/gu, ' ').trim() + ' '
const hit = computed(() => {
  if (!sample.value.trim()) return null
  const w = words(sample.value)
  return (list.data.value ?? []).find((b) => w.includes(' ' + words(b.term).trim() + ' ')) ?? false
})

const q = ref('')
const shown = computed(() => {
  const f = fold(q.value.trim())
  return (list.data.value ?? []).filter((b) => !f || fold(b.term).includes(f))
})
</script>

<template>
  <AdminPage title="Từ khóa chặn" lead="Danh sách này vẫn chạy khi bộ lọc AI lỗi.">
    <section class="card" aria-labelledby="kwa-h">
      <h2 id="kwa-h">Thêm từ khóa</h2>
      <form class="kwf" novalidate @submit.prevent="add">
        <div class="kwf-t">
          <label class="lb" for="kw-t">Từ hoặc cụm từ</label>
          <input id="kw-t" v-model="term" class="field" aria-required="true" :aria-invalid="err ? 'true' : undefined" aria-describedby="kw-e" @input="err = ''" />
        </div>
        <div>
          <label class="lb" for="kw-r">Quy tắc</label>
          <select id="kw-r" v-model="rule" class="field">
            <option v-for="(t, n) in RULE_SHORT" :key="n" :value="Number(n)">Quy tắc {{ n }}. {{ t }}</option>
          </select>
        </div>
        <button class="btn pri" type="submit" :aria-busy="busy">Thêm</button>
      </form>
      <div id="kw-e" aria-live="polite"><p v-if="err" class="err">{{ err }}</p></div>
      <p class="hint">Khớp cả từ, không phân biệt hoa thường.</p>
    </section>

    <section class="card" aria-labelledby="kwt-h">
      <h2 id="kwt-h">Thử một câu</h2>
      <label class="sr" for="kt-i">Câu để thử</label>
      <input id="kt-i" v-model="sample" class="field" placeholder="Nhập một câu để xem có khớp không" aria-describedby="kt-o" />
      <p id="kt-o" class="kto" aria-live="polite">
        <template v-if="hit === null">Kết quả hiện ở đây.</template>
        <template v-else-if="hit">
          <Icon name="alert" /><span
            >Khớp <b>“{{ hit.term }}”</b> · Quy tắc {{ hit.rule }}: {{ RULE_SHORT[hit.rule] }}.
            {{ hit.rule === 5 ? 'Người viết được nhắc trước khi gửi; nội dung không bị giữ lại.' : 'Nội dung sẽ bị giữ lại.' }}</span
          >
        </template>
        <template v-else><Icon name="check" /><span>Không khớp từ khóa nào.</span></template>
      </p>
    </section>

    <div class="atb">
      <label class="af" for="kq-i"><span class="sr">Tìm từ khóa</span><Icon name="search" /><input id="kq-i" v-model="q" class="field" type="search" placeholder="Tìm từ khóa" /></label>
      <p class="tcount">{{ shown.length }} trong {{ list.data.value?.length ?? 0 }} từ khóa</p>
    </div>
    <LoadError v-if="list.isError.value" title="Không tải được từ khóa" @retry="list.refetch()" />
    <section v-else class="card tc">
      <table v-if="shown.length" class="tbl">
        <caption class="sr">Từ khóa chặn</caption>
        <thead>
          <tr><th scope="col">Từ khóa</th><th scope="col">Quy tắc</th><th scope="col">Thêm bởi</th><th scope="col" class="tac"><span class="sr">Thao tác</span></th></tr>
        </thead>
        <tbody>
          <tr v-for="b in shown" :key="b.id">
            <td><b class="kwt">{{ b.term }}</b></td>
            <td data-l="Quy tắc">Quy tắc {{ b.rule }}<small>{{ RULE_SHORT[b.rule] }}</small></td>
            <td data-l="Thêm bởi">{{ byName(b.created_by) }}<small>{{ dm(b.created_at) }}</small></td>
            <td class="tac"><button class="btn ghost" :aria-label="`Xóa từ khóa ${b.term}`" @click="remove(b)">Xóa</button></td>
          </tr>
        </tbody>
      </table>
      <p v-else-if="!list.isPending.value" class="tbl-e">Không có từ khóa nào khớp.</p>
    </section>
  </AdminPage>
</template>
