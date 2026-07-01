<script setup lang="ts">
import {
  createHomeAgent,
  deleteHomeAgent,
  listHomeAgents,
  setHomeAgentMode,
} from "@/api/speedtest";
import { formatBeijingTime } from "@/utils/time";

interface HomeAgent {
  id: number;
  name: string;
  online: boolean;
  state: "active" | "suspended";
  persistent_active: boolean;
  pending_tasks: number;
  last_seen?: string;
  agent_version?: string;
  platform?: string;
  update_available: boolean;
  upgrade_command?: string;
}

const agents = ref<HomeAgent[]>([]);
const loading = ref(false);
const createDialog = ref(false);
const commandDialog = ref(false);
const agentName = ref("");
const installCommand = ref("");
let refreshTimer: number | undefined;

async function loadAgents() {
  loading.value = true;
  try {
    const { data } = await listHomeAgents();
    agents.value = data || [];
  } finally {
    loading.value = false;
  }
}

async function handleCreate() {
  if (!agentName.value.trim()) {
    ElMessage.warning("请输入测速端名称");
    return;
  }
  const { data } = await createHomeAgent({ name: agentName.value.trim() });
  installCommand.value = data.install_command;
  createDialog.value = false;
  commandDialog.value = true;
  agentName.value = "";
  await loadAgents();
}

async function copyCommand() {
  await navigator.clipboard.writeText(installCommand.value);
  ElMessage.success("安装命令已复制");
}

async function copyUpgradeCommand(agent: any) {
  if (!agent.upgrade_command) return;
  await navigator.clipboard.writeText(agent.upgrade_command);
  ElMessage.success("更新命令已复制，请在家宽设备终端执行");
}

async function changeMode(agent: any) {
  await setHomeAgentMode({
    id: agent.id,
    active: !agent.persistent_active,
  });
  ElMessage.success(agent.persistent_active ? "测速端已挂起" : "测速端已激活");
  await loadAgents();
}

async function handleDelete(agent: any) {
  await ElMessageBox.confirm(
    `删除测速端“${agent.name}”及其历史任务？`,
    "删除测速端",
    { type: "warning", confirmButtonText: "删除", cancelButtonText: "取消" }
  );
  await deleteHomeAgent(agent.id);
  ElMessage.success("测速端已删除");
  await loadAgents();
}

function formatTime(value?: string) {
  return value ? formatBeijingTime(value) : "尚未连接";
}

onMounted(() => {
  loadAgents();
  refreshTimer = window.setInterval(loadAgents, 15000);
});

onBeforeUnmount(() => {
  if (refreshTimer) window.clearInterval(refreshTimer);
});
</script>

<template>
  <div class="page-workspace">
    <el-dialog
      v-model="createDialog"
      class="form-dialog"
      width="520px"
      title="创建家宽测速端"
    >
      <div class="dialog-form">
        <label class="field">
          <span class="field-label">测速端名称</span>
          <el-input
            v-model="agentName"
            placeholder="例如：家里飞牛 NAS"
            @keyup.enter="handleCreate"
          />
          <span class="field-help">创建后会生成一次性安装命令。</span>
        </label>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createDialog = false">取消</el-button>
          <el-button type="primary" @click="handleCreate"
            >生成安装命令</el-button
          >
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="commandDialog"
      class="form-dialog"
      width="680px"
      title="安装家宽测速端"
      :close-on-click-modal="true"
    >
      <div class="install-guide">
        <p>在飞牛、NAS 或 Linux 家庭设备终端执行以下命令：</p>
        <pre><code>{{ installCommand }}</code></pre>
        <el-alert
          title="令牌只在这里显示一次。安装成功后，本页状态会自动变为已连接。"
          type="info"
          :closable="false"
        />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="commandDialog = false">关闭</el-button>
          <el-button type="primary" @click="copyCommand"
            >复制安装命令</el-button
          >
        </div>
      </template>
    </el-dialog>

    <div class="page-heading">
      <div>
        <h1>家宽测速端</h1>
        <p>让家庭设备通过真实节点执行延迟和下载速度测试</p>
      </div>
      <el-button type="primary" @click="createDialog = true"
        >新增测速端</el-button
      >
    </div>

    <section class="work-surface">
      <el-table v-loading="loading" :data="agents">
        <el-table-column prop="name" label="名称" min-width="180">
          <template #default="{ row }">
            <div class="agent-name">
              <span class="status-dot" :class="{ online: row.online }" />
              <strong>{{ row.name }}</strong>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="连接状态" width="120">
          <template #default="{ row }">
            <span :class="row.online ? 'text-success' : 'muted-cell'">
              {{ row.online ? "已连接" : "离线" }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="运行模式" width="120">
          <template #default="{ row }">
            {{ row.state === "active" ? "激活" : "挂起" }}
          </template>
        </el-table-column>
        <el-table-column prop="platform" label="设备" min-width="170">
          <template #default="{ row }">
            {{ row.platform || "等待注册" }}
            <span v-if="row.agent_version" class="agent-version">
              · agent {{ row.agent_version }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="最后在线" min-width="180">
          <template #default="{ row }">{{
            formatTime(row.last_seen)
          }}</template>
        </el-table-column>
        <el-table-column label="待执行" width="90">
          <template #default="{ row }">{{ row.pending_tasks }}</template>
        </el-table-column>
        <el-table-column label="操作" width="250" align="right">
          <template #default="{ row }">
            <el-button
              v-if="row.update_available && row.upgrade_command"
              link
              @click="copyUpgradeCommand(row)"
            >
              更新命令
            </el-button>
            <el-button link type="primary" @click="changeMode(row)">
              {{ row.persistent_active ? "挂起" : "激活" }}
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)"
              >删除</el-button
            >
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!agents.length && !loading" class="empty-agents">
        <strong>还没有家宽测速端</strong>
        <span>创建测速端并在家庭设备执行安装命令。</span>
      </div>
    </section>
  </div>
</template>

<style scoped>
.install-guide {
  display: grid;
  gap: 14px;
}

.install-guide p {
  margin: 0;
  color: var(--el-text-color-regular);
}

.install-guide pre {
  max-height: 260px;
  padding: 16px;
  overflow: auto;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-all;
}

.agent-name {
  display: flex;
  gap: 9px;
  align-items: center;
}

.status-dot {
  width: 8px;
  height: 8px;
  background: var(--el-text-color-placeholder);
  border-radius: 50%;
}

.status-dot.online {
  background: #16a34a;
}

.text-success {
  color: #15803d;
}

.muted-cell {
  color: var(--el-text-color-secondary);
}

.agent-version {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.empty-agents {
  display: grid;
  gap: 5px;
  justify-items: center;
  padding: 52px 20px;
  color: var(--el-text-color-secondary);
}

.empty-agents strong {
  color: var(--el-text-color-primary);
  font-size: 15px;
}
</style>
