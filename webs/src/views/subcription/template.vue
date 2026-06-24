<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from "vue";
import YamlEditor from "@/components/YamlEditor/index.vue";
import { AddTemp, DelTemp, getTemp, UpdateTemp } from "@/api/subcription/temp";

interface Temp {
  file: string;
  text: string;
  create_date: string;
}

const tableData = ref<Temp[]>([]);
const Tempoldname = ref("");
const Tempname = ref("");
const TempText = ref("");
const dialogVisible = ref(false);
const table = ref();
const TempTitle = ref("");

async function gettemps() {
  const {data} = await getTemp();
  tableData.value = data;
}

onMounted(gettemps);

const handleAddTemp = () => {
  TempTitle.value = "添加模板";
  Tempname.value = "";
  TempText.value = "";
  dialogVisible.value = true;
};

const addtemp = async () => {
  const filename = Tempname.value.trim();
  if (!filename) {
    ElMessage.warning("请输入模板文件名");
    return;
  }

  if (TempTitle.value === "添加模板") {
    await AddTemp({
      filename,
      text: TempText.value,
    });
    ElMessage.success("添加成功");
  } else {
    await UpdateTemp({
      filename,
      oldname: Tempoldname.value.trim(),
      text: TempText.value,
    });
    ElMessage.success("更新成功");
  }

  await gettemps();
  Tempname.value = "";
  TempText.value = "";
  dialogVisible.value = false;
};

const multipleSelection = ref<Temp[]>([]);
const handleSelectionChange = (val: Temp[]) => {
  multipleSelection.value = val;
};

const selectAll = () => {
  nextTick(() => {
    tableData.value.forEach((row) => {
      table.value.toggleRowSelection(row, true);
    });
  });
};

const handleExport = (row: any) => {
  const blob = new Blob([row.text], {
    type: "application/yaml;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const downloadLink = document.createElement("a");
  downloadLink.href = url;
  downloadLink.download = row.file;
  document.body.appendChild(downloadLink);
  downloadLink.click();
  downloadLink.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
};

const handleEdit = (row: any) => {
  TempTitle.value = "编辑模板";
  Tempname.value = row.file;
  Tempoldname.value = row.file;
  TempText.value = row.text;
  dialogVisible.value = true;
};

const toggleSelection = () => {
  table.value.clearSelection();
};

const handleDel = (row: any) => {
  ElMessageBox.confirm(
    `你是否要删除 ${row.file} ?`,
    "提示",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning",
    }
  ).then(async () => {
    await DelTemp({
      filename: row.file,
    });
    ElMessage({
      type: "success",
      message: "删除成功",
    });
    gettemps();
  });
};

const selectDel = () => {
  if (multipleSelection.value.length === 0) {
    return;
  }
  ElMessageBox.confirm(
    "你是否要删除选中的模板？",
    "提示",
    {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning",
    }
  ).then(async () => {
    await Promise.all(
      multipleSelection.value.map((item) =>
        DelTemp({
          filename: item.file,
        })
      )
    );
    await gettemps();
    ElMessage({
      type: "success",
      message: "删除成功",
    });
  });
};

const currentPage = ref(1);
const pageSize = ref(10);
const handleSizeChange = (val: number) => {
  pageSize.value = val;
};

const handleCurrentChange = (val: number) => {
  currentPage.value = val;
};

const currentTableData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  return tableData.value.slice(start, start + pageSize.value);
});
</script>

<template>
  <div class="page-workspace">
    <el-dialog
      v-model="dialogVisible"
      class="template-editor-dialog"
      width="calc(100vw - 64px)"
      top="32px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <template #header>
        <div class="editor-dialog-heading">
          <div>
            <h2>{{ TempTitle }}</h2>
            <p>编辑 YAML 配置，右侧内容会实时同步</p>
          </div>
          <el-input
            v-model="Tempname"
            class="filename-input"
            placeholder="模板文件名，例如 config.yaml"
          >
            <template #prepend>文件名</template>
          </el-input>
        </div>
      </template>

      <div class="editor-layout">
        <section class="editor-panel">
          <div class="panel-heading">
            <span>YAML 编辑</span>
            <span class="panel-hint">Tab 缩进 2 个空格</span>
          </div>
          <div class="editor-content">
            <YamlEditor v-model="TempText" />
          </div>
        </section>

        <section class="editor-panel">
          <div class="panel-heading">
            <span>实时预览</span>
            <span class="panel-hint">同步显示当前内容</span>
          </div>
          <pre class="yaml-preview"><code>{{ TempText || "# 暂无内容" }}</code></pre>
        </section>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="addtemp">保存模板</el-button>
        </div>
      </template>
    </el-dialog>

    <div class="page-heading">
      <div>
        <h1>模板列表</h1>
        <p>维护 Clash、Surge 等订阅输出模板</p>
      </div>
      <el-button type="primary" @click="handleAddTemp">添加模板</el-button>
    </div>

    <section class="work-surface">
      <div class="table-toolbar">
        <span class="record-count">共 {{ tableData.length }} 个模板</span>
      </div>

      <el-table
        ref="table"
        :data="currentTableData"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" fixed width="48" />
        <el-table-column prop="file" label="模板文件名" min-width="320">
          <template #default="scope">
            <span class="primary-cell">{{ scope.row.file }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="create_date" label="创建时间" min-width="200" sortable />
        <el-table-column fixed="right" label="操作" width="190" align="right">
          <template #default="scope">
            <el-button link type="primary" @click="handleExport(scope.row)">导出</el-button>
            <el-button link type="primary" @click="handleEdit(scope.row)">编辑</el-button>
            <el-button link type="danger" @click="handleDel(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <div class="batch-actions">
          <el-button @click="selectAll">全选</el-button>
          <el-button @click="toggleSelection">取消选择</el-button>
          <el-button type="danger" plain @click="selectDel">批量删除</el-button>
        </div>
        <el-pagination
          class="table-pagination"
          :current-page="currentPage"
          :page-size="pageSize"
          layout="total, sizes, prev, pager, next, jumper"
          :page-sizes="[10, 20, 30, 40]"
          :total="tableData.length"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.record-count {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.primary-cell {
  font-weight: 550;
}

.editor-dialog-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 32px;
  padding-right: 32px;
}

.editor-dialog-heading h2 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 18px;
  letter-spacing: 0;
}

.editor-dialog-heading p {
  margin: 5px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.filename-input {
  width: min(440px, 45vw);
}

.editor-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
  height: calc(100vh - 220px);
  min-height: 480px;
}

.editor-panel {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-bg-color);
}

.panel-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 44px;
  padding: 0 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}

.panel-hint {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  font-weight: 400;
}

.editor-content {
  min-height: 0;
  flex: 1;
}

.yaml-preview {
  min-height: 0;
  flex: 1;
  overflow: auto;
  box-sizing: border-box;
  margin: 0;
  padding: 14px 18px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  font-size: 13px;
  line-height: 1.55;
  tab-size: 2;
  white-space: pre;
}

@media (max-width: 900px) {
  .editor-dialog-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }

  .filename-input {
    width: 100%;
  }

  .editor-layout {
    grid-template-columns: 1fr;
    height: auto;
  }

  .editor-panel {
    height: 420px;
  }
}
</style>
