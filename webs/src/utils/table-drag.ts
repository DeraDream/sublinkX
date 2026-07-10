import type Sortable from "sortablejs";
import type { Ref } from "vue";
import { nextTick, onBeforeUnmount, onMounted, watch } from "vue";

interface DraggableTableOptions<T> {
  tableRef: Ref<any>;
  rows: Ref<T[]>;
  enabled?: Ref<boolean>;
  startIndex?: () => number;
  storageKey?: string;
  rowKey?: (row: T) => string | number | undefined;
}

export function useDraggableTableRows<T>({
  tableRef,
  rows,
  enabled,
  startIndex = () => 0,
  storageKey,
  rowKey,
}: DraggableTableOptions<T>) {
  let sortable: Sortable | undefined;
  let tbody: HTMLElement | undefined;
  let applyingStoredOrder = false;

  const getRowKey = (row: T) => rowKey?.(row);

  const readStoredOrder = () => {
    if (!storageKey) return [];
    try {
      const value = localStorage.getItem(storageKey);
      return value ? (JSON.parse(value) as Array<string | number>) : [];
    } catch {
      return [];
    }
  };

  const persistOrder = () => {
    if (!storageKey || !rowKey) return;
    const order = rows.value
      .map((row) => getRowKey(row))
      .filter((key): key is string | number => key !== undefined);
    localStorage.setItem(storageKey, JSON.stringify(order));
  };

  const applyStoredOrder = () => {
    if (!storageKey || !rowKey || applyingStoredOrder) return;
    const order = readStoredOrder();
    if (!order.length) return;
    const position = new Map(order.map((key, index) => [String(key), index]));
    applyingStoredOrder = true;
    rows.value = [...rows.value].sort((left, right) => {
      const leftIndex =
        position.get(String(getRowKey(left))) ?? Number.MAX_SAFE_INTEGER;
      const rightIndex =
        position.get(String(getRowKey(right))) ?? Number.MAX_SAFE_INTEGER;
      return leftIndex - rightIndex;
    });
    applyingStoredOrder = false;
  };

  const destroy = () => {
    sortable?.destroy();
    sortable = undefined;
    tbody = undefined;
  };

  const bind = async () => {
    if (enabled && !enabled.value) {
      destroy();
      return;
    }
    await nextTick();
    const root = tableRef.value?.$el || tableRef.value;
    const nextTbody = root?.querySelector?.(".el-table__body-wrapper tbody") as
      | HTMLElement
      | undefined;
    if (!nextTbody || nextTbody === tbody) return;

    const { default: Sortable } = await import("sortablejs");
    destroy();
    tbody = nextTbody;
    sortable = Sortable.create(nextTbody, {
      animation: 150,
      draggable: "tr",
      handle: ".row-drag-handle",
      ghostClass: "table-row-ghost",
      onEnd: ({ oldIndex, newIndex }) => {
        if (
          oldIndex === undefined ||
          newIndex === undefined ||
          oldIndex === newIndex
        ) {
          return;
        }
        const from = startIndex() + oldIndex;
        const to = startIndex() + newIndex;
        const nextRows = [...rows.value];
        const moved = nextRows.splice(from, 1)[0];
        if (!moved) return;
        nextRows.splice(to, 0, moved);
        rows.value = nextRows;
        persistOrder();
      },
    });
  };

  onMounted(() => {
    if (!enabled || enabled.value) {
      bind();
    }
  });
  watch(
    () => [
      enabled?.value ?? true,
      rows.value.length,
      rows.value.map((row) => getRowKey(row)).join("|"),
      startIndex(),
    ],
    () => {
      if (enabled && !enabled.value) {
        destroy();
        return;
      }
      applyStoredOrder();
      bind();
    },
    { flush: "post" }
  );
  onBeforeUnmount(destroy);
}
