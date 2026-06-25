<script setup lang="ts">
import { ref, onMounted, nextTick } from "vue";
import {
  getNodes,
  AddNodes,
  DelNode,
  UpdateNode,
  GetGroup,
  SetGroup,
} from "@/api/subcription/node";
import {
  createSpeedTask,
  listHomeAgents,
  listSpeedTasks,
} from "@/api/speedtest";

interface GroupNode {
  ID: number;
  Name: string;
  Nodes: Node[];
}
interface Node {
  ID: number;
  Name: string;
  Link: string;
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
  // 页面开始执行函数
  getnodes();
  GetGroups();
  loadSpeedAgents();
  loadSpeedResults();
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
const tableData = ref<Node[]>([]);
// 分组列表临时存放数据
const tableDataTemp = ref<Node[]>([]);
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
const allNodes = ref<string[]>([]); // 所有节点
const nodelistShow = ref(false); // 节点列表
const SelectionNodeGroups = ref<string[]>([]); // 选中的分组
const SelectionNode = ref(""); // 选中的节点
const speedLoading = ref(false);
const speedAgents = ref<any[]>([]);
const selectedAgentId = ref<number>();
const speedResultMap = ref<Record<number, { latency?: any; speed?: any }>>({});
let speedRunCancelled = false;

// const SelectionNodes = ref([]); // 选中的节点
const RadioGroup = ref("1"); // 分组单选框
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
  const { data } = await getNodes();
  if (data.length > 0) tableDataTemp.value = tableData.value = data;
  allNodes.value = []; // 清空 allNodes 数组
  data.forEach((item: any) => {
    allNodes.value.push(item.Name); // 将所有节点添加到 allNodes 中
  });
}
async function GetGroups() {
  const { data } = await GetGroup();
  if (Array.isArray(data) && data.length > 0) {
    allGroupNames.value = data; // 将所有分组名称添加到 allGroupNames 中
  }
  RadioGroup.value = allGroupNames.value.length > 0 ? "1" : "2"; // 自动选择单选框值
  // console.log("单选框",RadioGroup.value);
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
  let links = NodeForm.value.Link.trim()
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter((item) => item);
  if (isAdd && links.length === 0) {
    ElMessage.warning("节点链接不能为空");
    return;
  }

  try {
    if (isAdd) {
      for (const link of links) {
        await AddNodes({
          link,
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
        name: NodeForm.value.Name, // 新名称
        link: NodeForm.value.Link, // 新链接
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
  getnodes();
  GetGroups();
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
  console.log(SelectionNode.value);

  try {
    // 检查是否选择了已有分组或输入了新分组名
    console.log(
      RadioGroup.value,
      SelectionNodeGroups.value,
      NodeGroupInput.value
    );

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
  getnodes(); // 刷新节点列表
  GetGroups(); // 刷新分组列表
  ClearInput(); // 清空所有输入
};
// 表格时间格式化
const Timeformatter = (row: any) => {
  row.CreatedAt = new Date(row.CreatedAt).toLocaleString(); // 转换为本地时间字符串
  return row.CreatedAt;
};
// 选择已有节点显示所属分组
const handleShownodeGroupList = () => {
  // 显示这个节点关联的分组
  const nodeData = allNodes.value.find((node) => node === SelectionNode.value);
  SelectionNodeGroups.value = [];
  tableData.value.forEach((item) => {
    if (item.Name === nodeData && (item.GroupNodes?.length ?? 0) > 0) {
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
  await GetGroups(); // 刷新分组列表
  await getnodes(); // 刷新节点列表
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

    const IDs: number[] = [];

    for (const item of multipleSelection.value) {
      await DelNode({ id: item.ID });
      IDs.push(item.ID); // 收集所有已删除的节点ID
    }
    ElMessage.success("批量删除成功");
    // 从 tableData 中删除已删除的节点
    tableData.value = tableData.value.filter((item) => !IDs.includes(item.ID));
  } catch (error) {
    if (error !== "cancel") {
      console.error("批量删除失败:", error);
      ElMessage.error("批量删除失败！");
    }
  }
  // 刷新节点列表
  await GetGroups(); // 刷新分组列表
  await getnodes();
};
// 全选
const selectAll = () => {
  nextTick(() => {
    const table = multipleTable.value;
    if (table) {
      // 否则全选
      tableData.value.forEach((row) => {
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
async function loadSpeedAgents() {
  const { data } = await listHomeAgents();
  speedAgents.value = (data || []).filter((item: any) => item.online);
  if (!selectedAgentId.value && speedAgents.value.length > 0) {
    selectedAgentId.value = speedAgents.value[0].id;
  }
}

async function loadSpeedResults() {
  const { data } = await listSpeedTasks();
  const latest: Record<number, { latency?: any; speed?: any }> = {};
  for (const item of data || []) {
    if (!latest[item.node_id]) latest[item.node_id] = {};
    if (!latest[item.node_id][item.test_type as "latency" | "speed"]) {
      latest[item.node_id][item.test_type as "latency" | "speed"] = item;
    }
  }
  speedResultMap.value = latest;
}

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

async function waitSpeedTask(
  taskId: number,
  nodeId: number,
  type: "latency" | "speed"
) {
  const deadline = Date.now() + (type === "latency" ? 90000 : 120000);
  while (!speedRunCancelled && Date.now() < deadline) {
    const { data } = await listSpeedTasks({ id: taskId });
    const task = data?.[0];
    if (task) {
      if (!speedResultMap.value[nodeId]) speedResultMap.value[nodeId] = {};
      speedResultMap.value[nodeId][type] = task;
      if (task.status === "success" || task.status === "failed") {
        return task;
      }
    }
    await delay(1000);
  }
  throw new Error("等待家宽测速端返回结果超时");
}

const runSpeedTest = async (type: "latency" | "speed") => {
  const targets =
    multipleSelection.value.length > 0
      ? multipleSelection.value
      : tableData.value;
  if (targets.length === 0) {
    ElMessage.warning("没有可测速的节点");
    return;
  }
  if (!selectedAgentId.value) {
    ElMessage.warning("请选择已连接的家宽测速端");
    return;
  }
  speedLoading.value = true;
  speedRunCancelled = false;
  let successCount = 0;
  let failedCount = 0;
  try {
    for (let index = 0; index < targets.length; index++) {
      const item = targets[index];
      ElMessage.info(`正在测试 ${index + 1}/${targets.length}：${item.Name}`);
      try {
        const { data: task } = await createSpeedTask({
          node_id: item.ID,
          agent_id: selectedAgentId.value,
          type,
        });
        if (!speedResultMap.value[item.ID]) speedResultMap.value[item.ID] = {};
        speedResultMap.value[item.ID][type] = task;
        const result = await waitSpeedTask(task.id, item.ID, type);
        if (result.status === "success") {
          successCount++;
        } else {
          failedCount++;
        }
      } catch (error: any) {
        failedCount++;
        if (!speedResultMap.value[item.ID]) speedResultMap.value[item.ID] = {};
        speedResultMap.value[item.ID][type] = {
          status: "failed",
          error_message:
            error?.response?.data?.msg || error?.message || "测速失败",
        };
      }
    }
    ElMessage.success(`测速完成：成功 ${successCount}，失败 ${failedCount}`);
  } catch (error) {
    console.error("测速失败:", error);
    ElMessage.error("测速失败");
  } finally {
    speedLoading.value = false;
  }
};
const handleSelectionChange = (val: Node[]) => {
  multipleSelection.value = val;
};

watch(activeName, (newVal) => {
  if (newVal === "全部") {
    tableData.value = tableDataTemp.value;
  } else {
    tableData.value = tableDataTemp.value.filter((item) => {
      return item.GroupNodes?.some((group) => group.Name === newVal);
    });
  }
});

onBeforeUnmount(() => {
  speedRunCancelled = true;
});
</script>

<template>
  <div class="page-workspace">
    <el-dialog
      v-model="Nodedialog"
      class="form-dialog node-dialog"
      width="680px"
      :close-on-click-modal="false"
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
        <label v-if="dialogMode === 'edit'" class="field">
          <span class="field-label">节点名称</span>
          <el-input
            v-model="NodeForm.Name"
            placeholder="输入便于识别的节点名称"
          />
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
        <p>管理代理节点、分组以及连通性测试</p>
      </div>
      <el-button type="primary" @click="handleAddNode">添加节点</el-button>
    </div>

    <section class="work-surface">
      <div class="node-filters">
        <el-tabs v-model="activeName">
          <el-tab-pane :label="`全部 ${allNodes.length}`" name="全部" />
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
        :data="tableData"
        tooltip-effect="dark"
        row-key="ID"
        :tree-props="{ children: 'Nodes' }"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="48" />
        <el-table-column type="index" width="56" label="#" />
        <el-table-column prop="Name" label="节点名称" min-width="130" sortable>
          <template #default="{ row }">
            <span class="primary-cell">{{ row.Name }}</span>
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
        <el-table-column label="延迟" width="85">
          <template #default="{ row }">
            <span
              v-if="speedResultMap[row.ID]?.latency?.status === 'success'"
              class="latency"
              :class="'is-success'"
            >
              {{ speedResultMap[row.ID].latency.latency_ms }} ms
            </span>
            <span
              v-else-if="
                speedResultMap[row.ID]?.latency?.status === 'running' ||
                speedResultMap[row.ID]?.latency?.status === 'pending'
              "
              class="muted-cell"
              >测试中</span
            >
            <span
              v-else-if="speedResultMap[row.ID]?.latency?.status === 'failed'"
              class="latency is-failed"
            >
              <el-tooltip
                :content="
                  speedResultMap[row.ID].latency.error_message || '测速失败'
                "
                placement="top"
              >
                <span>失败</span>
              </el-tooltip>
            </span>
            <span v-else class="muted-cell">未测试</span>
          </template>
        </el-table-column>
        <el-table-column label="下载速度" width="105">
          <template #default="{ row }">
            <span
              v-if="
                speedResultMap[row.ID]?.speed?.status === 'success' &&
                speedResultMap[row.ID].speed.download_mbps > 0
              "
              class="latency is-success"
            >
              {{ speedResultMap[row.ID].speed.download_mbps.toFixed(1) }} Mbps
            </span>
            <span
              v-else-if="
                speedResultMap[row.ID]?.speed?.status === 'running' ||
                speedResultMap[row.ID]?.speed?.status === 'pending'
              "
              class="muted-cell"
              >测试中</span
            >
            <span
              v-else-if="speedResultMap[row.ID]?.speed?.status === 'failed'"
              class="latency is-failed"
            >
              <el-tooltip
                :content="
                  speedResultMap[row.ID].speed.error_message || '测速失败'
                "
                placement="top"
              >
                <span>失败</span>
              </el-tooltip>
            </span>
            <span v-else class="muted-cell">--</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="145" align="right" fixed="right">
          <template #default="scope">
            <el-button link type="primary" @click="handleEditNode(scope.row)"
              >编辑</el-button
            >
            <el-button link @click="copyInfo(scope.row)">复制</el-button>
            <el-button link type="danger" @click="handleDel(scope.row)"
              >删除</el-button
            >
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <div class="batch-actions">
          <el-button @click="selectAll">全选</el-button>
          <el-button @click="selectClear">取消选择</el-button>
          <el-button type="primary" plain @click="selectCopy"
            >复制选中</el-button
          >
          <el-button type="danger" plain @click="selectDel">删除选中</el-button>
          <el-select
            v-model="selectedAgentId"
            placeholder="选择家宽测速端"
            style="width: 180px"
            @visible-change="(visible: boolean) => visible && loadSpeedAgents()"
          >
            <el-option
              v-for="agent in speedAgents"
              :key="agent.id"
              :label="agent.name"
              :value="agent.id"
            />
          </el-select>
          <el-button :loading="speedLoading" @click="runSpeedTest('latency')"
            >测试延迟</el-button
          >
          <el-button
            :loading="speedLoading"
            type="primary"
            @click="runSpeedTest('speed')"
            >测试下载速度</el-button
          >
        </div>
        <span class="record-count">共 {{ tableData.length }} 个节点</span>
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

.node-link {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: transparent;
}

.record-count,
.muted-cell {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.latency {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.latency.is-success {
  color: #15803d;
}

.latency.is-failed {
  color: var(--el-color-danger);
}
</style>
