<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { basicSetup } from "codemirror";
import { indentWithTab } from "@codemirror/commands";
import { yaml } from "@codemirror/lang-yaml";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { tags } from "@lezer/highlight";

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
        syntaxHighlighting(
          HighlightStyle.define([
            {
              tag: [tags.propertyName, tags.variableName, tags.typeName],
              color: "var(--editor-keyword)",
            },
            {
              tag: [tags.string, tags.special(tags.string)],
              color: "var(--editor-string)",
            },
            {
              tag: [tags.number, tags.bool, tags.null],
              color: "var(--editor-number)",
            },
            {
              tag: tags.comment,
              color: "var(--editor-comment)",
            },
          ])
        ),
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
            color: "var(--el-text-color-primary)",
            backgroundColor: "var(--el-bg-color)",
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
            color: "var(--el-text-color-placeholder)",
            backgroundColor: "var(--el-fill-color-light)",
            borderRight: "1px solid var(--el-border-color-lighter)",
          },
          ".cm-activeLine, .cm-activeLineGutter": {
            backgroundColor: "var(--editor-active-line)",
          },
          "&.cm-focused": {
            outline: "none",
          },
          ".cm-selectionBackground, &.cm-focused .cm-selectionBackground": {
            backgroundColor: "var(--editor-selection)",
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
