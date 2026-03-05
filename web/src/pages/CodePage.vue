<template>
  <div class="code-page">
    <div class="workbench" :style="workbenchStyle">
      <div
        ref="workspaceRef"
        class="workspace"
        :class="panelPosition === 'right' ? 'workspace-right' : 'workspace-bottom'"
      >
        <div class="editor-pane">
          <div ref="editorContainer" class="editor-surface"></div>
        </div>

        <div
          class="splitter"
          :class="[
            panelPosition === 'right' ? 'splitter-vertical' : 'splitter-horizontal',
            { 'is-active': isResizing }
          ]"
          @pointerdown="startResize"
        ></div>

        <div
          ref="ioPaneRef"
          class="io-pane"
          :class="panelPosition === 'bottom' ? 'io-pane-horizontal' : 'io-pane-vertical'"
          :style="ioPaneStyle"
        >
          <div class="io-section" :style="ioInputStyle">
            <div class="io-header">
              <span>输入</span>
              <div class="header-spacer"></div>
              <button class="action-btn" type="button" @click="handleRun(false)">
                <img :src="saveIcon" alt="save" />
              </button>
              <button class="action-btn" type="button" @click="handleRun(true)">
                <img :src="runIcon" alt="run" />
              </button>
            </div>
            <div class="io-content">
              <textarea
                id="stdin"
                name="stdin"
                class="input-area"
                placeholder="请输入测试样例"
                v-model="stdin"
              ></textarea>
            </div>
          </div>

          <div
            class="io-splitter"
            :class="[
              panelPosition === 'bottom' ? 'io-splitter-vertical' : 'io-splitter-horizontal',
              { 'is-active': isIoResizing }
            ]"
            @pointerdown="startIoResize"
          ></div>

          <div class="io-section" :style="ioOutputStyle">
            <div class="io-header">
              <span>输出</span>
            </div>
            <div class="io-content output-area">
              <div v-show="time !== 0" class="run-time">运行时间：{{ time }} ms</div>
              <pre class="output-text">{{ status !== 'completed' ? status : (stdout === '' ? 'Empty' : stdout) }}</pre>
              <pre v-show="stderr !== ''" class="output-text output-error">{{ stderr }}</pre>
              <pre v-show="log !== ''" class="output-text output-error">{{ log }}</pre>
            </div>
          </div>
        </div>
      </div>

      <aside class="right-toolbar">
        <button
          class="tool-btn"
          :class="{ active: panelPosition === 'right' }"
          type="button"
          @click="setPanelPosition('right')"
          title="输出区在右侧"
        >
          右
        </button>
        <button
          class="tool-btn"
          :class="{ active: panelPosition === 'bottom' }"
          type="button"
          @click="setPanelPosition('bottom')"
          title="输出区在下方"
        >
          下
        </button>

        <div class="toolbar-divider"></div>

        <button class="tool-btn" type="button" @click="adjustFontScale(ZOOM_STEP)" title="放大字体">
          A+
        </button>
        <button class="tool-btn" type="button" @click="adjustFontScale(-ZOOM_STEP)" title="缩小字体">
          A-
        </button>
        <button class="tool-btn" type="button" @click="resetFontScale" title="重置字体">
          100
        </button>
        <div class="zoom-label">{{ fontScale }}%</div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, defineProps, onUnmounted, computed, watch, nextTick } from 'vue'
import { EditorView } from '@codemirror/view'
import { Compartment, EditorState } from '@codemirror/state'
import { cpp } from '@codemirror/lang-cpp'
import { basicSetup } from 'codemirror'
import { languageServer } from 'codemirror-languageserver';
import { keymap } from '@codemirror/view'
import { indentMore, indentLess, insertTab } from "@codemirror/commands";
import { oneDark, oneDarkHighlightStyle } from "@codemirror/theme-one-dark";
import runIcon from '../assets/run.svg'
import saveIcon from '../assets/save.svg'
import { useRouter } from 'vue-router'
import { defaultHighlightStyle, indentUnit, syntaxHighlighting } from '@codemirror/language';

const router = useRouter()

const props = defineProps({
  id: {
    type: String,
    default: null,
  }
});

const stdin = ref('')
const stdout = ref('')
const status = ref('completed')
const stderr = ref('')
const time = ref(0)
const log = ref('')
const isLoading = ref(false)
const editorView = ref<EditorView | null>(null)
const workspaceRef = ref<HTMLElement | null>(null)
const ioPaneRef = ref<HTMLElement | null>(null)
const isResizing = ref(false)
const isIoResizing = ref(false)

type PanelPosition = 'right' | 'bottom'
const panelPosition = ref<PanelPosition>('right')
const panelSize = ref(36)
const ioSplit = ref(50)
const fontScale = ref(100)

const ZOOM_STEP = 5
const ZOOM_MIN = 70
const ZOOM_MAX = 180

const PANEL_POSITION_KEY = 'code_panel_position'
const PANEL_SIZE_KEY = 'code_panel_size'
const IO_SPLIT_KEY = 'code_io_split_ratio'
const FONT_SCALE_KEY = 'code_font_scale'

const editorThemeCompartment = new Compartment()

const serverUri = window.CONFIG.LSP_SERVER !== '__LSP_SERVER_URL_PLACEHOLDER__' ? window.CONFIG.LSP_SERVER : import.meta.env.VITE_LSP_SERVER;
const backend = window.CONFIG.BACKEND !== '__BACKEND_URL_PLACEHOLDER__' ? window.CONFIG.BACKEND : import.meta.env.VITE_BACKEND;
const ls = languageServer({
  serverUri,
  rootUri: 'file:///main.cpp',
  workspaceFolders: [],
  documentUri: `file:///main.cpp`,
  languageId: 'cpp',
});
const editorContainer = ref<HTMLElement | null>(null)

const workbenchStyle = computed(() => ({
  '--ui-font-size': `${Math.max(11, Math.round((16 * fontScale.value) / 100))}px`,
} as Record<string, string>))

const ioPaneStyle = computed(() => ({
  flexBasis: `${panelSize.value}%`,
}))

const ioInputStyle = computed(() => ({
  flex: `0 0 ${ioSplit.value}%`,
}))

const ioOutputStyle = computed(() => ({
  flex: `0 0 ${100 - ioSplit.value}%`,
}))

function clampPanelSize(value: number, position: PanelPosition = panelPosition.value) {
  const min = position === 'right' ? 12 : 14
  const max = position === 'right' ? 56 : 70
  return Math.min(max, Math.max(min, value))
}

function clampIoSplit(value: number) {
  return Math.min(80, Math.max(20, value))
}

function clampFontScale(value: number) {
  return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, value))
}

function createEditorTheme(scale: number) {
  const fontPx = Math.max(12, Math.round((16 * scale) / 100))
  return EditorView.theme({
    "&": { height: "100%" },
    ".cm-scroller": { overflow: "auto" },
    ".cm-content, .cm-gutters": {
      fontSize: `${fontPx}px`,
      lineHeight: '1.6',
      fontFamily: '"JetBrains Mono", "Fira Code", "Cascadia Code", "Consolas", monospace'
    }
  })
}

function requestEditorMeasure() {
  if (!editorView.value) {
    return
  }
  editorView.value.requestMeasure()
}

function applyEditorTheme(scale: number = fontScale.value) {
  if (!editorView.value) {
    return
  }
  editorView.value.dispatch({
    effects: editorThemeCompartment.reconfigure(createEditorTheme(scale))
  })
}

function setPanelPosition(position: PanelPosition) {
  if (panelPosition.value === position) {
    return
  }
  panelPosition.value = position
}

function setFontScale(value: number) {
  const next = clampFontScale(value)
  if (next === fontScale.value) {
    return
  }
  fontScale.value = next
}

function adjustFontScale(delta: number) {
  setFontScale(fontScale.value + delta)
}

function resetFontScale() {
  setFontScale(100)
}

function handleZoomKeydown(event: KeyboardEvent) {
  const hasModifier = event.ctrlKey || event.metaKey
  if (!hasModifier) {
    return
  }

  if (event.key === '+' || event.key === '=' || event.code === 'NumpadAdd') {
    event.preventDefault()
    adjustFontScale(ZOOM_STEP)
    return
  }

  if (event.key === '-' || event.code === 'NumpadSubtract') {
    event.preventDefault()
    adjustFontScale(-ZOOM_STEP)
    return
  }

  if (event.key === '0' || event.code === 'Digit0' || event.code === 'Numpad0') {
    event.preventDefault()
    resetFontScale()
  }
}

let lastWheelZoomTime = 0

function handleZoomWheel(event: WheelEvent) {
  if (!(event.ctrlKey || event.metaKey)) {
    return
  }

  event.preventDefault()
  const now = Date.now()
  if (now - lastWheelZoomTime < 35) {
    return
  }
  lastWheelZoomTime = now
  adjustFontScale(event.deltaY < 0 ? ZOOM_STEP : -ZOOM_STEP)
}

let resizeCleanup: (() => void) | null = null
let ioResizeCleanup: (() => void) | null = null

function stopResize() {
  if (!resizeCleanup) {
    return
  }
  resizeCleanup()
  resizeCleanup = null
}

function stopIoResize() {
  if (!ioResizeCleanup) {
    return
  }
  ioResizeCleanup()
  ioResizeCleanup = null
}

function startResize(event: PointerEvent) {
  if (!workspaceRef.value) {
    return
  }
  event.preventDefault()
  stopResize()
  isResizing.value = true
  document.body.classList.add('is-resizing-panel')
  document.body.style.cursor = panelPosition.value === 'right' ? 'col-resize' : 'row-resize'

  const handlePointerMove = (moveEvent: PointerEvent) => {
    if (!workspaceRef.value) {
      return
    }
    const rect = workspaceRef.value.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) {
      return
    }

    if (panelPosition.value === 'right') {
      const size = ((rect.right - moveEvent.clientX) / rect.width) * 100
      panelSize.value = clampPanelSize(size, 'right')
    } else {
      const size = ((rect.bottom - moveEvent.clientY) / rect.height) * 100
      panelSize.value = clampPanelSize(size, 'bottom')
    }
  }

  const handlePointerUp = () => {
    stopResize()
  }

  window.addEventListener('pointermove', handlePointerMove)
  window.addEventListener('pointerup', handlePointerUp, { once: true })

  resizeCleanup = () => {
    window.removeEventListener('pointermove', handlePointerMove)
    window.removeEventListener('pointerup', handlePointerUp)
    document.body.classList.remove('is-resizing-panel')
    document.body.style.removeProperty('cursor')
    isResizing.value = false
    requestEditorMeasure()
  }
}

function startIoResize(event: PointerEvent) {
  if (!ioPaneRef.value) {
    return
  }
  event.preventDefault()
  stopIoResize()
  isIoResizing.value = true
  document.body.classList.add('is-resizing-panel')
  document.body.style.cursor = panelPosition.value === 'bottom' ? 'col-resize' : 'row-resize'

  const handlePointerMove = (moveEvent: PointerEvent) => {
    if (!ioPaneRef.value) {
      return
    }
    const rect = ioPaneRef.value.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) {
      return
    }

    if (panelPosition.value === 'bottom') {
      const size = ((moveEvent.clientX - rect.left) / rect.width) * 100
      ioSplit.value = clampIoSplit(size)
    } else {
      const size = ((moveEvent.clientY - rect.top) / rect.height) * 100
      ioSplit.value = clampIoSplit(size)
    }
  }

  const handlePointerUp = () => {
    stopIoResize()
  }

  window.addEventListener('pointermove', handlePointerMove)
  window.addEventListener('pointerup', handlePointerUp, { once: true })

  ioResizeCleanup = () => {
    window.removeEventListener('pointermove', handlePointerMove)
    window.removeEventListener('pointerup', handlePointerUp)
    document.body.classList.remove('is-resizing-panel')
    document.body.style.removeProperty('cursor')
    isIoResizing.value = false
  }
}

function getStatus(id: string) {
  fetch(backend + `/api/pastes/${id}`)
    .then(res => res.json())
    .then(res => {
      stdin.value = res.stdin
      status.value = res.status
      stdout.value = res.stdout
      stderr.value = res.stderr
      time.value = res.execution_time_ms
      log.value = res.compile_log
    })
    .catch(err => {
      console.log(err)
    })
}

watch(panelPosition, (position) => {
  panelSize.value = clampPanelSize(panelSize.value, position)
  localStorage.setItem(PANEL_POSITION_KEY, position)
  nextTick(() => requestEditorMeasure())
})

watch(panelSize, (size) => {
  localStorage.setItem(PANEL_SIZE_KEY, String(size))
  requestEditorMeasure()
})

watch(ioSplit, (size) => {
  localStorage.setItem(IO_SPLIT_KEY, String(size))
})

watch(fontScale, (size) => {
  localStorage.setItem(FONT_SCALE_KEY, String(size))
  applyEditorTheme(size)
  requestEditorMeasure()
})

onMounted(() => {
  const savedPosition = localStorage.getItem(PANEL_POSITION_KEY)
  if (savedPosition === 'right' || savedPosition === 'bottom') {
    panelPosition.value = savedPosition
  }

  const savedSize = Number(localStorage.getItem(PANEL_SIZE_KEY))
  if (!Number.isNaN(savedSize)) {
    panelSize.value = clampPanelSize(savedSize, panelPosition.value)
  }

  const savedIoSplit = Number(localStorage.getItem(IO_SPLIT_KEY))
  if (!Number.isNaN(savedIoSplit)) {
    ioSplit.value = clampIoSplit(savedIoSplit)
  }

  const savedScale = Number(localStorage.getItem(FONT_SCALE_KEY))
  if (!Number.isNaN(savedScale)) {
    fontScale.value = clampFontScale(savedScale)
  }

  window.addEventListener('keydown', handleZoomKeydown)
  window.addEventListener('wheel', handleZoomWheel, { passive: false })

  const oldCode = localStorage.getItem('code')
  stdin.value = localStorage.getItem('stdin') || ''
  const state = EditorState.create({
    doc: oldCode || `#include <bits/stdc++.h>\n\nint main() {\n\n}`,
    extensions: [
      basicSetup,
      cpp(),
      ls,
      indentUnit.of("    "),
      editorThemeCompartment.of(createEditorTheme(fontScale.value)),
      EditorView.updateListener.of((update) => {
        if (update.docChanged && update.selectionSet) {
          const cursorParams = update.state.selection.main.head;

          update.view.dispatch({
            effects: EditorView.scrollIntoView(cursorParams, { y: "nearest" })
          });
        }
      }),
      keymap.of([
        { key: "Tab", run: indentMore },
        { key: "Shift-Tab", run: indentLess },
        // 如果需要在行中插入真实 Tab：
        { key: "Mod-Tab", run: insertTab },
      ]),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      syntaxHighlighting(oneDarkHighlightStyle),
      oneDark,
    ]
  })
  editorView.value = new EditorView({
    state,
    parent: editorContainer.value as HTMLElement
  })

  nextTick(() => requestEditorMeasure())

  console.log(props.id)
  if (props.id !== null && props.id !== undefined && props.id !== '') {
    isLoading.value = true
    fetch(backend + `/api/pastes/${props.id}`)
      .then(res => res.json())
      .then(res => {
        if (!editorView.value) {
          return;
        }
        editorView.value.dispatch({
          changes: {
            from: 0,
            to: editorView.value.state.doc.length,
            insert: res.code
          }
        })
        stdin.value = res.stdin
        status.value = res.status
        stdout.value = res.stdout
        stderr.value = res.stderr
        time.value = res.execution_time_ms
        log.value = res.compile_log
        if (status.value === 'pending' || status.value === 'running') {
          const timer = setInterval(() => {
            getStatus(props.id)
            if (status.value !== 'pending' && status.value !== 'running') {
              clearInterval(timer)
              isLoading.value = false
            }
          }, 1000)
        } else {
          isLoading.value = false
        }
      })
      .catch(err => {
        console.log(err)
      })

  }
})

onUnmounted(() => {
  stopResize()
  stopIoResize()
  window.removeEventListener('keydown', handleZoomKeydown)
  window.removeEventListener('wheel', handleZoomWheel)

  if (editorView.value) {
    console.log("Destroying CodeMirror EditorView and closing LSP connection...");
    editorView.value.destroy();
    editorView.value = null;
    console.log("EditorView destroyed.");
  }
});

const handleRun = (isRun: boolean) => {
  if (isLoading.value) {
    console.log("别急")
    return
  }
  if (!editorView.value) {
    return;
  }
  isLoading.value = true
  console.log('run')
  fetch(backend + '/api/pastes', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      code: editorView.value.state.doc.toString() ?? '',
      stdin: isRun ? stdin.value : '',
      language: 'c++20',
      run: isRun
    })
  })
    .then(res => res.json())
    .then(res => {
      if (res.message === 'Created') {
        const pasteid = res.paste_id;
        router.push({
          name: 'code',
          params: {
            id: pasteid
          }
        })
      }
    })
    .catch(err => {
      console.log(err)
    })
}

setInterval(() => {
  if (editorView.value) {
    localStorage.setItem('code', editorView.value.state.doc.toString())
    localStorage.setItem('stdin', stdin.value)
  }
}, 1000)

</script>

<style scoped>
.code-page {
  flex: 1;
  min-height: 0;
  width: 100%;
  background:
    radial-gradient(circle at 10% 6%, rgba(59, 67, 102, 0.32), transparent 36%),
    radial-gradient(circle at 88% 88%, rgba(0, 122, 204, 0.15), transparent 33%),
    #1e1e1e;
  color: #d4d4d4;
  overflow: hidden;
}

.workbench {
  width: 100%;
  height: 100%;
  min-height: 0;
  min-width: 0;
  display: flex;
  background: #1e1e1e;
  font-family: "JetBrains Mono", "Fira Code", "Cascadia Code", "Consolas", monospace;
}

.workspace {
  flex: 1;
  min-height: 0;
  min-width: 0;
  display: flex;
}

.workspace-right {
  flex-direction: row;
}

.workspace-bottom {
  flex-direction: column;
}

.editor-pane {
  flex: 1;
  min-width: 0;
  min-height: 0;
  background: #1e1e1e;
}

.editor-surface {
  width: 100%;
  height: 100%;
}

.splitter {
  flex-shrink: 0;
  background: #2d2d30;
  position: relative;
  transition: background-color 120ms ease;
}

.splitter:hover,
.splitter.is-active {
  background: #007acc;
}

.splitter-vertical {
  width: 6px;
  cursor: col-resize;
}

.splitter-vertical::after {
  content: "";
  position: absolute;
  left: 2px;
  right: 2px;
  top: 50%;
  height: 52px;
  transform: translateY(-50%);
  border-radius: 999px;
  background: repeating-linear-gradient(to bottom, transparent, transparent 6px, rgba(255, 255, 255, 0.24) 6px, rgba(255, 255, 255, 0.24) 8px);
}

.splitter-horizontal {
  height: 6px;
  cursor: row-resize;
}

.splitter-horizontal::after {
  content: "";
  position: absolute;
  top: 2px;
  bottom: 2px;
  left: 50%;
  width: 52px;
  transform: translateX(-50%);
  border-radius: 999px;
  background: repeating-linear-gradient(to right, transparent, transparent 6px, rgba(255, 255, 255, 0.24) 6px, rgba(255, 255, 255, 0.24) 8px);
}

.io-pane {
  min-height: 110px;
  min-width: 130px;
  display: flex;
  background: #252526;
  border-left: 1px solid #2d2d30;
}

.io-pane-vertical {
  flex-direction: column;
}

.io-pane-horizontal {
  flex-direction: row;
}

.workspace-bottom .io-pane {
  border-left: 0;
  border-top: 1px solid #2d2d30;
}

.io-section {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.io-splitter {
  flex-shrink: 0;
  background: #2d2d30;
  position: relative;
  transition: background-color 120ms ease;
}

.io-splitter:hover,
.io-splitter.is-active {
  background: #007acc;
}

.io-splitter-horizontal {
  height: 6px;
  cursor: row-resize;
}

.io-splitter-horizontal::after {
  content: "";
  position: absolute;
  top: 2px;
  bottom: 2px;
  left: 50%;
  width: 40px;
  transform: translateX(-50%);
  border-radius: 999px;
  background: repeating-linear-gradient(to right, transparent, transparent 5px, rgba(255, 255, 255, 0.24) 5px, rgba(255, 255, 255, 0.24) 7px);
}

.io-splitter-vertical {
  width: 6px;
  cursor: col-resize;
}

.io-splitter-vertical::after {
  content: "";
  position: absolute;
  left: 2px;
  right: 2px;
  top: 50%;
  height: 40px;
  transform: translateY(-50%);
  border-radius: 999px;
  background: repeating-linear-gradient(to bottom, transparent, transparent 5px, rgba(255, 255, 255, 0.24) 5px, rgba(255, 255, 255, 0.24) 7px);
}

.io-pane-horizontal .io-section + .io-section .io-header {
  border-left: 1px solid #2d2d30;
}

.io-header {
  height: 32px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  font-size: 16px;
  font-weight: 600;
  color: #c5c8ce;
  background: #2d2d30;
  border-bottom: 1px solid #222;
}

.header-spacer {
  flex: 1;
  min-width: 0;
}

.action-btn {
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  cursor: pointer;
}

.action-btn:hover {
  background: #3c3c40;
}

.action-btn img {
  width: 18px;
  height: 18px;
}

.io-content {
  flex: 1;
  min-height: 0;
}

.input-area {
  width: 100%;
  height: 100%;
  border: 0;
  outline: 0;
  resize: none;
  padding: 10px;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: inherit;
  font-size: var(--ui-font-size);
  line-height: 1.6;
  white-space: pre-wrap;
}

.output-area {
  overflow: auto;
  padding: 10px;
  background: #1e1e1e;
}

.run-time {
  color: #4fc1ff;
  margin-bottom: 8px;
  font-size: var(--ui-font-size);
}

.output-text {
  margin: 0;
  font-size: var(--ui-font-size);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.output-error {
  color: #f48771;
}

.right-toolbar {
  width: 60px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 10px 6px;
  border-left: 1px solid #2d2d30;
  background: linear-gradient(180deg, #2a2a2d 0%, #252526 30%, #252526 100%);
}

.tool-btn {
  width: 100%;
  min-height: 30px;
  border: 1px solid #3a3a3d;
  border-radius: 6px;
  background: #2f2f32;
  color: #c9ccd3;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.02em;
  cursor: pointer;
  transition: background-color 120ms ease, border-color 120ms ease, color 120ms ease;
}

.tool-btn:hover {
  background: #38383c;
  border-color: #4b4b4f;
}

.tool-btn.active {
  background: #0e639c;
  border-color: #1177bb;
  color: #ffffff;
}

.toolbar-divider {
  width: 100%;
  height: 1px;
  background: #3b3b3d;
  margin: 4px 0;
}

.zoom-label {
  width: 100%;
  text-align: center;
  padding: 4px 0;
  border-radius: 6px;
  background: #1f1f21;
  color: #9fa4ad;
  font-size: 11px;
  border: 1px solid #353538;
}

:deep(.cm-editor) {
  height: 100%;
}

:global(body.is-resizing-panel) {
  user-select: none;
}

@media (max-width: 900px) {
  .right-toolbar {
    width: 54px;
    padding: 8px 4px;
    gap: 6px;
  }

  .io-pane {
    min-width: 110px;
    min-height: 100px;
  }

  .io-splitter-vertical {
    width: 5px;
  }
}
</style>
