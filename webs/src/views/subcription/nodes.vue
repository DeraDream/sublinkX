<script setup lang="ts">
import { ref, shallowRef, onMounted, nextTick, computed } from "vue";
import {
  getNodes,
  AddNodes,
  DelNode,
  UpdateNode,
  GetGroup,
  SetGroup,
  setNodeDisabled,
} from "@/api/subcription/node";
import { formatBeijingTime } from "@/utils/time";

interface GroupNode {
  ID: number;
  Name: string;
  Nodes: Node[];
}
interface Node {
  ID: number;
  Name: string;
  Link: string;
  Disabled?: boolean;
  CreateDate: string;
  GroupNodes?: GroupNode[]; // 分组信息
}
interface NodeInfo {
  ID?: number; // 编辑时需要传入ID
  Title?: string;
  Name?: string;
  Link: string;
  GroupName?: string[]; // 分组名称
}
onMounted(async () => {
  await refreshNodePageData();
});
const dialogMode = ref<"add" | "edit">("add");

// --- 表格选择与操作相关数据 ---
const multipleSelection = ref<Node[]>([]); // Stores selected table items
const multipleTable = ref<any>(null);

const tableRefs = ref<{ [key: string]: any }>({}); // Stores references to each el-table
// --- 表格选择与操作相关数据结束 ---
// const NodeNewLinkInput = ref("")
// const NodeNewNameInput = ref("")
const NodeGroupInput = ref("");
const tableData = shallowRef<Node[]>([]);
const currentPage = ref(1);
const pageSize = ref(20);
const totalNodes = ref(0);
const totalAllNodes = ref(0);
const tableLoading = ref(false);
let latestNodeRequest = 0;
// 分组列表临时存放数据
const activeName = ref("全部");
const Nodedialog = ref(false); // 弹窗是否可见
const Groupdialog = ref(false); // 弹窗是否可见
const NodeForm = ref<NodeInfo>({
  Title: "",
  Name: "",
  Link: "",
  GroupName: [],
});
const allGroupNames = ref<string[]>([]); // 所有分组名称
const nodelistShow = ref(false); // 节点列表
const SelectionNodeGroups = ref<string[]>([]); // 选中的分组
const SelectionNode = ref(""); // 选中的节点

// const SelectionNodes = ref([]); // 选中的节点
const RadioGroup = ref("1"); // 分组单选框
const parsedAddLinks = computed(() =>
  NodeForm.value.Link.trim()
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter((item) => item)
);
const inferredAddNames = computed(() =>
  parsedAddLinks.value.map((link) => extractNodeRemark(link))
);
const pagedTableData = computed(() => tableData.value);

function decodeBase64Text(text: string) {
  try {
    const normalized = text.replace(/-/g, "+").replace(/_/g, "/");
    const padding =
      normalized.length % 4 ? "=".repeat(4 - (normalized.length % 4)) : "";
    const binary = window.atob(normalized + padding);
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return "";
  }
}

function extractNodeRemark(link: string) {
  const trimmed = link.trim();
  if (!trimmed) return "";
  try {
    if (trimmed.startsWith("vmess://")) {
      const raw = decodeBase64Text(trimmed.replace("vmess://", ""));
      const data = JSON.parse(raw);
      return data.ps || "";
    }
    const parsed = new URL(trimmed);
    return decodeURIComponent(parsed.hash.replace(/^#/, ""));
  } catch {
    return "";
  }
}

// 将所有输入的值清空
function ClearInput() {
  SelectionNode.value = ""; // 清空选中的节点
  NodeForm.value = {
    // 清空节点链接输入框
    Title: "",
    Name: "",
    Link: "",
    GroupName: [],
  };
  NodeGroupInput.value = ""; // 清空创建分组输入框
  SelectionNodeGroups.value = []; // 清空选中的分组
  nodelistShow.value = false; // 隐藏节点列表
  Nodedialog.value = false; // 关闭节点添加弹窗
  Groupdialog.value = false; // 关闭分组绑定弹窗
}
async function getnodes() {
  const requestId = ++latestNodeRequest;
  tableLoading.value = true;
  try {
    const { data } = await getNodes({
      page: currentPage.value,
      page_size: pageSize.value,
      group: activeName.value === "全部" ? "" : activeName.value,
    });
    if (requestId !== latestNodeRequest) return;
    if (Array.isArray(data)) {
      tableData.value = data;
      totalNodes.value = data.length;
      return;
    }
    tableData.value = Array.isArray(data?.items) ? data.items : [];
    totalNodes.value = Number(data?.total || 0);
    if (activeName.value === "全部") {
      totalAllNodes.value = totalNodes.value;
    }
    clampCurrentPage();
  } finally {
    if (requestId === latestNodeRequest) {
      tableLoading.value = false;
    }
  }
}
async function GetGroups() {
  const { data } = await GetGroup();
  allGroupNames.value = Array.isArray(data) ? data : [];
  RadioGroup.value = allGroupNames.value.length > 0 ? "1" : "2"; // 自动选择单选框值
  // console.log("单选框",RadioGroup.value);
}

async function refreshNodePageData() {
  await Promise.all([getnodes(), GetGroups()]);
}

function clampCurrentPage() {
  const maxPage = Math.max(1, Math.ceil(totalNodes.value / pageSize.value));
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage;
  }
}

async function refreshFirstPage() {
  if (currentPage.value !== 1) {
    currentPage.value = 1;
    return;
  }
  await refreshNodePageData();
}

const handleAddNode = () => {
  dialogMode.value = "add";
  Nodedialog.value = true;
  NodeForm.value = {
    Title: "添加节点",
    Name: "",
    Link: "",
    GroupName: [],
  };
  SelectionNodeGroups.value = [];
  NodeGroupInput.value = "";
};

const handleEditNode = (row: any) => {
  // NodeNewNameInput.value = row.Name; // 编辑时使用原名称
  // NodeNewLinkInput.value = row.Link; // 编辑时使用原链接
  dialogMode.value = "edit";
  Nodedialog.value = true;
  NodeForm.value = {
    ID: row.ID,
    Title: "编辑节点",
    Name: row.Name,
    Link: row.Link,
    GroupName: (row.GroupNodes || []).map((g: GroupNode) => g.Name),
  };
  SelectionNodeGroups.value = NodeForm.value.GroupName || [];
  SelectionNode.value = row.Name;
};
const SubmitNodeForm = async (row: any) => {
  const isAdd = dialogMode.value === "add";
  const links = parsedAddLinks.value;
  if (isAdd && links.length === 0) {
    ElMessage.warning("节点链接不能为空");
    return;
  }
  if (isAdd && links.length > 1 && NodeForm.value.Name?.trim()) {
    ElMessage.warning("批量添加时请留空节点名称，每条链接会自动读取自己的注释");
    return;
  }

  try {
    if (isAdd) {
      for (const link of links) {
        await AddNodes({
          link,
          name: links.length === 1 ? NodeForm.value.Name?.trim() : "",
          group:
            RadioGroup.value === "1"
              ? SelectionNodeGroups.value.join(",")
              : NodeGroupInput.value,
        });
      }
      ElMessage.success("节点添加成功");
    } else {
      await UpdateNode({
        id: NodeForm.value.ID,
        name: NodeForm.value.Name?.trim(), // 新名称
        link: NodeForm.value.Link.trim(), // 新链接
        group:
          RadioGroup.value === "1"
            ? SelectionNodeGroups.value.join(",")
            : NodeGroupInput.value,
      });
      ElMessage.success("节点更新成功");
    }
  } catch (err) {
    ElMessage.error(`${isAdd ? "添加" : "更新"}失败`);
  }
  await refreshNodePageData();
  ClearInput();
};

// const AddNode = async() => {
//   // 多节点链接输入处理
//   let NodeLinkInputs = NodeNewLinkInput.value.trim().split(/[\n,]/); // 使用换行符或逗号分隔输入的节点链接
//   NodeLinkInputs = NodeLinkInputs.map((item) => item.trim()).filter((item) => item !== ''); // 去除空白和重复的链接
//   if (NodeNewLinkInput.value.trim() === '') {
//     ElMessage.warning('节点链接不能为空');
//     return;
//   }

//   try {
//     // 多节点同步循环添加节点
//     for(const link of NodeLinkInputs) {
//       if (link) {
//           const newNode = {
//           link: link.trim(), // 节点链接
//           group: SelectionNodeGroups.value.join(','), // 选中的分组
//           };
//           await AddNodes(newNode).then(() => {
//           ElMessage.success('节点添加成功');
//           Nodedialog.value = false; // 关闭弹窗
//           });
//       }
//     }
//     // getnodes(); // 刷新节点列表
//     // GetGroups(); // 刷新分组列表
//   } catch (error) {
//     console.error('添加节点失败:', error);
//     // ElMessage.error('添加节点失败，请稍后再试');
//   }
//   getnodes(); // 刷新节点列表
//   GetGroups(); // 刷新分组列表
//   ClearInput(); // 清空所有输入
// }
const AddGroup = async () => {
  try {
    if (RadioGroup.value === "1" && SelectionNodeGroups.value.length === 0) {
      ElMessage.warning("你还没有选择分组");
      return;
    }
    if (RadioGroup.value === "2" && NodeGroupInput.value.trim() === "") {
      ElMessage.warning("创建的分组名不能为空");
      return;
    }
    if (SelectionNode.value.length > 0) {
      // 如果没有选择节点
      const newNode = {
        name: SelectionNode.value, // 节点链接
        group:
          RadioGroup.value == "1"
            ? SelectionNodeGroups.value.join(",")
            : NodeGroupInput.value, // 条件选择已有节点|创建分组
      };
      await SetGroup(newNode).then(() => {
        ElMessage.success("分组绑定成功");
      });
    }
  } catch (error) {
    console.error("添加分组失败:", error);
    // ElMessage.error('添加分组失败');
  }
  await refreshNodePageData(); // 刷新节点和分组
  ClearInput(); // 清空所有输入
};
// 表格时间格式化
const Timeformatter = (row: any) => {
  return formatBeijingTime(row.CreatedAt);
};
// 选择已有节点显示所属分组
const handleShownodeGroupList = () => {
  // 显示这个节点关联的分组
  SelectionNodeGroups.value = [];
  tableData.value.forEach((item) => {
    if (item.Name === SelectionNode.value && (item.GroupNodes?.length ?? 0) > 0) {
      // console.log(`节点 ${nodeData} 的分组:`, item.GroupNodes);
      item.GroupNodes?.forEach((item) => {
        SelectionNodeGroups.value.push(item.Name); // 将分组名称添加到 SelectionNodeGroups 中
      });
    }
  });
};
// 表格所属分组格式化
const Groupformatter = (row: any, cellValue: any) => {
  const data = row.GroupNodes || [];
  if (!Array.isArray(data) || data.length === 0) {
    return "未分组"; // 如果没有分组，返回默认值
  }
  return data.map((group: any) => group.Name).join(", ");
};
// --- 复制链接 (保持不变) ---
const copyUrl = (url: string) => {
  if (navigator.clipboard) {
    navigator.clipboard
      .writeText(url)
      .then(() => {
        ElMessage.success("链接已复制到剪贴板！");
      })
      .catch((err) => {
        console.error("复制失败:", err);
        ElMessage.error("复制失败！请手动复制。");
      });
  } else {
    const textarea = document.createElement("textarea");
    textarea.value = url;
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand("copy");
      ElMessage.success("链接已复制到剪贴板！");
    } catch (err) {
      ElMessage.warning("复制失败！");
    } finally {
      document.body.removeChild(textarea);
    }
  }
};
// 复制表格节点信息
const copyInfo = (row: any) => {
  copyUrl(row.Link);
};
const handleDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(`你是否要删除 ${row.Name} ?`, "提示", {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning",
    });
    await DelNode({ id: row.ID });
    ElMessage.success("删除成功");
  } catch (error) {
    if (error !== "cancel") {
      console.error("删除失败:", error);
      ElMessage.error("删除失败！");
    }
  }
  // 刷新节点列表
  await refreshNodePageData(); // 刷新节点和分组
  ClearInput(); // 清空所有输入
};
const selectDel = async () => {
  if (multipleSelection.value.length === 0) {
    ElMessage.warning("请选择要删除的节点！");
    return;
  }
  try {
    await ElMessageBox.confirm(
      `你是否要删除选中的 ${multipleSelection.value.length} 条节点 ?`,
      "提示",
      {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      }
    );

    for (const item of multipleSelection.value) {
      await DelNode({ id: item.ID });
    }
    ElMessage.success("批量删除成功");
  } catch (error) {
    if (error !== "cancel") {
      console.error("批量删除失败:", error);
      ElMessage.error("批量删除失败！");
    }
  }
  // 刷新节点列表
  await refreshNodePageData();
};
// 全选
const selectAll = () => {
  nextTick(() => {
      const table = multipleTable.value;
    if (table) {
      // 否则全选
      pagedTableData.value.forEach((row) => {
        table.toggleRowSelection(row, true);
      });
    }
  });
};
// 取消全选
const selectClear = () => {
  nextTick(() => {
    const table = multipleTable.value;
    if (table) {
      table.clearSelection();
    }
  });
};
// --- 表格选择操作 (保持不变) ---
const setTableRef = (el: any, name: string) => {
  if (el) {
    tableRefs.value[name] = el;
  } else {
    delete tableRefs.value[name];
  }
};
//批量复制
const selectCopy = async () => {
  if (multipleSelection.value.length === 0) {
    ElMessage.warning("请选择要复制的节点！");
    return;
  }
  try {
    copyUrl(multipleSelection.value.map((item) => item.Link).join("\n"));
  } catch (error) {
    if (error !== "cancel") {
      console.error("批量复制失败:", error);
      ElMessage.error("批量复制失败");
    }
  }
};

const toggleNodeDisabled = async (row: any) => {
  await setNodeDisabled({
    id: row.ID,
    disabled: !row.Disabled,
  });
  ElMessage.success(row.Disabled ? "节点已恢复" : "节点已禁用");
  await refreshNodePageData();
};
const handleSelectionChange = (val: Node[]) => {
  multipleSelection.value = val;
};

watch(activeName, () => {
  refreshFirstPage();
});

watch(pageSize, () => {
  refreshFirstPage();
});

watch(currentPage, () => {
  getnodes();
});
</script>

<template>
  <div class="page-workspace">
    <el-dialog
      v-model="Nodedialog"
      class="form-dialog node-dialog"
      width="680px"
      :close-on-click-modal="true"
      destroy-on-close
    >
      <template #header>
        <div class="dialog-heading">
          <h2>{{ NodeForm.Title }}</h2>
          <p>
            {{
              dialogMode === "add"
                ? "粘贴一个或多个节点链接，并为它们设置分组。"
                : "修改节点名称、链接与所属分组。"
            }}
          </p>
        </div>
      </template>

      <div class="dialog-form">
        <label class="field">
          <span class="field-label">节点名称</span>
          <el-input
            v-model="NodeForm.Name"
            :placeholder="
              dialogMode === 'add'
                ? '可选；留空则自动读取链接末尾注释'
                : '输入便于识别的节点名称'
            "
          />
          <span v-if="dialogMode === 'add'" class="field-help">
            单条链接可手动填写，例如
            po0-HKT；批量添加请留空，每条链接会使用自己的注释名。
          </span>
        </label>

        <label class="field">
          <span class="field-label">节点链接</span>
          <el-input
            v-model="NodeForm.Link"
            placeholder="支持换行或逗号分隔多个链接"
            type="textarea"
            :autosize="{ minRows: dialogMode === 'add' ? 5 : 3, maxRows: 10 }"
          />
          <span v-if="dialogMode === 'add'" class="field-help"
            >每个链接会被分别创建为一个节点。</span
          >
        </label>

        <div
          v-if="dialogMode === 'add' && parsedAddLinks.length > 0"
          class="node-import-preview"
        >
          <div class="preview-head">
            <strong>将创建 {{ parsedAddLinks.length }} 个节点</strong>
            <span>{{
              NodeForm.Name && parsedAddLinks.length === 1
                ? "使用手动节点名"
                : "留空名称时使用链接注释"
            }}</span>
          </div>
          <div class="preview-list">
            <span
              v-for="(link, index) in parsedAddLinks.slice(0, 4)"
              :key="`${link}-${index}`"
              class="preview-chip"
            >
              {{
                parsedAddLinks.length === 1 && NodeForm.Name
                  ? NodeForm.Name
                  : inferredAddNames[index] || "未识别名称"
              }}
            </span>
            <span v-if="parsedAddLinks.length > 4" class="preview-more">
              +{{ parsedAddLinks.length - 4 }}
            </span>
          </div>
        </div>

        <div class="field">
          <span class="field-label">所属分组</span>
          <el-radio-group v-model="RadioGroup" class="flat-segmented">
            <el-radio-button v-if="allGroupNames.length > 0" label="1"
              >选择已有分组</el-radio-button
            >
            <el-radio-button label="2">创建新分组</el-radio-button>
          </el-radio-group>
        </div>

        <label
          v-if="RadioGroup === '1' && allGroupNames.length > 0"
          class="field"
        >
          <span class="field-label">选择分组</span>
          <el-select
            v-model="SelectionNodeGroups"
            multiple
            placeholder="可选择多个分组"
            class="field-control"
          >
            <el-option
              v-for="item in allGroupNames"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </label>

        <label v-if="RadioGroup === '2'" class="field">
          <span class="field-label">新分组名称</span>
          <el-input v-model="NodeGroupInput" placeholder="例如：香港节点" />
        </label>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="Nodedialog = false">取消</el-button>
          <el-button type="primary" @click="SubmitNodeForm">
            {{ dialogMode === "add" ? "添加节点" : "保存修改" }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <div class="page-heading">
      <div>
        <h1>节点列表</h1>
        <p>管理代理节点和分组</p>
      </div>
      <el-button type="primary" @click="handleAddNode">添加节点</el-button>
    </div>

    <section class="work-surface">
      <div class="node-filters">
        <el-tabs v-model="activeName">
          <el-tab-pane :label="`全部 ${totalAllNodes}`" name="全部" />
          <el-tab-pane
            v-for="item in allGroupNames"
            :key="item"
            :label="item"
            :name="item"
          />
        </el-tabs>
      </div>

      <el-table
        ref="multipleTable"
        :data="pagedTableData"
        :empty-text="tableLoading ? '加载中...' : '暂无数据'"
        tooltip-effect="dark"
        row-key="ID"
        :tree-props="{ children: 'Nodes' }"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="48" />
        <el-table-column type="index" width="56" label="#" />
        <el-table-column prop="Name" label="节点名称" min-width="130" sortable>
          <template #default="{ row }">
            <span
              class="primary-cell"
              :class="{ 'is-disabled-node': row.Disabled }"
            >
              {{ row.Name }}
            </span>
            <el-tag v-if="row.Disabled" size="small" type="info" effect="plain">
              已禁用
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          prop="Link"
          label="节点链接"
          min-width="180"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <code class="node-link">{{ row.Link }}</code>
          </template>
        </el-table-column>
        <el-table-column
          prop="CreatedAt"
          label="创建时间"
          min-width="150"
          :formatter="Timeformatter"
          sortable
          show-overflow-tooltip
        />
        <el-table-column
          label="所属分组"
          min-width="95"
          :formatter="Groupformatter"
          show-overflow-tooltip
        />
        <el-table-column label="操作" width="190" align="right" fixed="right">
          <template #default="scope">
            <el-button link type="primary" @click="handleEditNode(scope.row)"
              >编辑</el-button
            >
            <el-button link @click="copyInfo(scope.row)">复制</el-button>
            <el-button link @click="toggleNodeDisabled(scope.row)">
              {{ scope.row.Disabled ? "恢复" : "禁用" }}
            </el-button>
            <el-button link type="danger" @click="handleDel(scope.row)"
              >删除</el-button
            >
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <div class="batch-actions">
          <el-button @click="selectAll">全选当前页</el-button>
          <el-button @click="selectClear">取消选择</el-button>
          <el-button type="primary" plain @click="selectCopy"
            >复制选中</el-button
          >
          <el-button type="danger" plain @click="selectDel">删除选中</el-button>
        </div>
        <div class="table-pagination">
          <span class="record-count">共 {{ totalNodes }} 个节点</span>
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            layout="sizes, prev, pager, next"
            :page-sizes="[10, 20, 50, 100]"
            :total="totalNodes"
            background
            small
          />
        </div>
      </div>
    </section>
  </div>
</template>
<style scoped>
.field-control {
  width: 100%;
}

.node-filters {
  margin: -8px 0 12px;
}

.node-filters :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.primary-cell {
  font-weight: 550;
}

.is-disabled-node {
  color: var(--el-text-color-placeholder);
  text-decoration: line-through;
}

.node-link {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: transparent;
}

.node-import-preview {
  display: grid;
  gap: 10px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.preview-head strong {
  color: var(--el-text-color-primary);
  font-size: 13px;
}

.preview-head span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.preview-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.preview-chip,
.preview-more {
  max-width: 180px;
  padding: 5px 9px;
  overflow: hidden;
  color: var(--el-color-primary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 28%, transparent);
  border-radius: 999px;
}

.record-count,
.muted-cell {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.table-pagination {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-left: auto;
}
</style>
