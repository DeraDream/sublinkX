<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { basicSetup } from "codemirror";
import { indentWithTab } from "@codemirror/commands";
import { yaml } from "@codemirror/lang-yaml";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";

const props = defineProps<{
  modelValue: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const editorElement = ref<HTMLElement>();
let editorView: EditorView | undefined;

onMounted(() => {
  editorView = new EditorView({
    parent: editorElement.value,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        basicSetup,
        yaml(),
        EditorState.tabSize.of(2),
        keymap.of([indentWithTab]),
        EditorView.lineWrapping,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            emit("update:modelValue", update.state.doc.toString());
          }
        }),
        EditorView.theme({
          "&": {
            height: "100%",
            color: "#1f2937",
            backgroundColor: "#ffffff",
            fontSize: "13px",
          },
          ".cm-scroller": {
            overflow: "auto",
            fontFamily:
              '"SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
          },
          ".cm-content": {
            padding: "14px 0",
          },
          ".cm-line": {
            padding: "0 16px",
          },
          ".cm-gutters": {
            color: "#94a3b8",
            backgroundColor: "#f8fafc",
            borderRight: "1px solid #e5e7eb",
          },
          ".cm-activeLine, .cm-activeLineGutter": {
            backgroundColor: "#eff6ff",
          },
          "&.cm-focused": {
            outline: "none",
          },
          ".cm-selectionBackground, &.cm-focused .cm-selectionBackground": {
            backgroundColor: "#bfdbfe",
          },
        }),
      ],
    }),
  });
});

watch(
  () => props.modelValue,
  (value) => {
    if (!editorView || value === editorView.state.doc.toString()) {
      return;
    }

    editorView.dispatch({
      changes: {
        from: 0,
        to: editorView.state.doc.length,
        insert: value,
      },
    });
  }
);

onBeforeUnmount(() => {
  editorView?.destroy();
});
</script>

<template>
  <div ref="editorElement" class="yaml-editor" />
</template>

<style scoped>
.yaml-editor {
  height: 100%;
  min-height: 0;
}
</style>
