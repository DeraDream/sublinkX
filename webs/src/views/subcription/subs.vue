<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from "vue";
import {
  getSubs,
  AddSub,
  DelSub,
  UpdateSub,
  ResetSubToken,
  SetSubRevoked,
} from "@/api/subcription/subs";
import { getTemp } from "@/api/subcription/temp";
import { getNodes } from "@/api/subcription/node";
import { beijingTimestamp, formatBeijingTime } from "@/utils/time";
import { useDraggableTableRows } from "@/utils/table-drag";
import QrcodeVue from "qrcode.vue";
import { VueDraggable } from "vue-draggable-plus";

interface Sub {
  ID: number;
  Name: string;
  CreateDate?: string;
  CreatedAt?: string;
  Config: Config | string;
  Nodes: Node[];
  SubLogs: SubLogs[];
  Token: string;
  Revoked: boolean;
  ExpireAt?: string;
  AccessLimit: number;
  AccessCount: number;
}

interface Node {
  ID: number;
  Name: string;
  Link: string;
  Disabled?: boolean;
  CreateDate?: string;
}

interface Config {
  clash: string;
  surge: string;
  udp: boolean | string;
  cert: boolean | string;
}

interface SubLogs {
  IP?: string;
  Count?: number;
  Addr?: string;
  Date?: string;
  date?: string;
  name?: string;
  count?: number;
  address?: string;
}

interface Temp {
  file: string;
  text: string;
  CreateDate?: string;
}

const tableData = ref<Sub[]>([]);
const table = ref();
const NodesList = ref<Node[]>([]);
const templist = ref<Temp[]>([]);
const multipleSelection = ref<Sub[]>([]);

const dialogVisible = ref(false);
const subMode = ref<"add" | "edit">("add");
const wizardStep = ref(0);
const SubTitle = computed(() =>
  subMode.value === "add" ? "添加订阅" : "编辑订阅"
);
const Subname = ref("");
const oldSubname = ref("");
const expireAt = ref("");
const accessLimit = ref<number | undefined>();

const Clash = ref("");
const Surge = ref("");
const clashTemplateMode = ref<"local" | "url">("local");
const surgeTemplateMode = ref<"local" | "url">("local");
const checkList = ref<string[]>([]);

const value1 = ref<number[]>([]);
const nodeKeyword = ref("");

const iplogsdialog = ref(false);
const IplogsList = ref<SubLogs[]>([]);

const qrcode = ref("");
const Qrdialog = ref(false);
const QrTitle = ref("");

const ClientDiaLog = ref(false);
const ClientList = ["v2ray", "clash", "surge"];
const ClientUrls = ref<Record<string, string>>({});
const ClientUrl = ref("");
const ClientSubName = ref("");

async function getsubs() {
  const { data } = await getSubs();
  tableData.value = data;
}

async function gettemps() {
  const { data } = await getTemp();
  templist.value = data;
}

async function getnodes() {
  const { data } = await getNodes();
  NodesList.value = data;
}

function formatCreatedAt(row: Sub) {
  return formatBeijingTime(row.CreatedAt || row.CreateDate);
}

onMounted(() => {
  getsubs();
  gettemps();
  getnodes();
});

const currentPage = ref(1);
const pageSize = ref(10);
const currentTableData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  return tableData.value.slice(start, start + pageSize.value);
});
useDraggableTableRows({
  tableRef: table,
  rows: tableData,
  startIndex: () => (currentPage.value - 1) * pageSize.value,
  storageKey: "sublink:subscriptions:order",
  rowKey: (row) => row.ID,
});

const availableNodes = computed(() => {
  const keyword = nodeKeyword.value.trim().toLowerCase();
  return NodesList.value.filter((node) => {
    if (node.Disabled) return false;
    if (value1.value.includes(node.ID)) return false;
    if (!keyword) return true;
    return node.Name.toLowerCase().includes(keyword);
  });
});

const selectedNodes = computed(() =>
  value1.value.map((id) => {
    return (
      NodesList.value.find((node) => node.ID === id) || {
        ID: id,
        Name: String(id),
        Link: "",
        Disabled: false,
      }
    );
  })
);

const nodeNameById = (id: number) =>
  NodesList.value.find((node) => node.ID === id)?.Name || String(id);

const selectedNodePreview = computed(() => {
  if (!value1.value.length) return "尚未选择节点";
  const head = value1.value.slice(0, 4).map(nodeNameById).join(" → ");
  return value1.value.length > 4 ? `${head} → ...` : head;
});

const defaultTemplate = (keyword: string, fallback: string) => {
  const hit = templist.value.find((item) =>
    item.file.toLowerCase().includes(keyword)
  );
  return hit ? `./template/${hit.file}` : fallback;
};

const parseConfig = (value: Config | string): Config => {
  if (typeof value === "string") {
    try {
      return JSON.parse(value) as Config;
    } catch {
      return { clash: "", surge: "", udp: false, cert: false };
    }
  }
  return value || { clash: "", surge: "", udp: false, cert: false };
};

const inferTemplateMode = (value: string) =>
  /^https?:\/\//i.test(value) ? "url" : "local";

const resetWizardForm = () => {
  wizardStep.value = 0;
  Subname.value = "";
  oldSubname.value = "";
  expireAt.value = "";
  accessLimit.value = undefined;
  checkList.value = [];
  Clash.value = defaultTemplate("clash", "./template/clash.yaml");
  Surge.value = defaultTemplate("surge", "./template/surge.conf");
  clashTemplateMode.value = inferTemplateMode(Clash.value);
  surgeTemplateMode.value = inferTemplateMode(Surge.value);
  value1.value = [];
  nodeKeyword.value = "";
};

const handleAddSub = () => {
  subMode.value = "add";
  resetWizardForm();
  dialogVisible.value = true;
};

const handleEdit = (row: any) => {
  const config = parseConfig(row.Config);
  subMode.value = "edit";
  wizardStep.value = 0;
  Subname.value = row.Name;
  oldSubname.value = row.Name;
  expireAt.value = row.ExpireAt
    ? new Date(row.ExpireAt).toISOString().slice(0, 19).replace("T", " ")
    : "";
  accessLimit.value = row.AccessLimit || undefined;
  checkList.value = [];
  if (config.udp === true || config.udp === "true") checkList.value.push("udp");
  if (config.cert === true || config.cert === "true")
    checkList.value.push("cert");
  Clash.value =
    config.clash || defaultTemplate("clash", "./template/clash.yaml");
  Surge.value =
    config.surge || defaultTemplate("surge", "./template/surge.conf");
  clashTemplateMode.value = inferTemplateMode(Clash.value);
  surgeTemplateMode.value = inferTemplateMode(Surge.value);
  value1.value = (row.Nodes || []).map((item: Node) => item.ID);
  nodeKeyword.value = "";
  dialogVisible.value = true;
};

const validateStep = (step: number) => {
  if (step === 0) {
    if (!Subname.value.trim()) {
      ElMessage.warning("请输入订阅名称");
      return false;
    }
  }
  if (step === 1) {
    if (!Clash.value.trim() || !Surge.value.trim()) {
      ElMessage.warning("请选择或填写 Clash / Surge 模板");
      return false;
    }
  }
  if (step === 2) {
    if (!value1.value.length) {
      ElMessage.warning("至少选择一个节点");
      return false;
    }
  }
  return true;
};

const nextStep = () => {
  if (!validateStep(wizardStep.value)) return;
  wizardStep.value = Math.min(wizardStep.value + 1, 3);
};

const prevStep = () => {
  wizardStep.value = Math.max(wizardStep.value - 1, 0);
};

const jumpStep = (step: number) => {
  if (step <= wizardStep.value) {
    wizardStep.value = step;
  }
};

const addNodeToSelection = (id: number) => {
  if (!value1.value.includes(id)) {
    value1.value.push(id);
  }
};

const removeNodeFromSelection = (id: number) => {
  value1.value = value1.value.filter((item) => item !== id);
};

const addAllVisibleNodes = () => {
  availableNodes.value.forEach((node) => addNodeToSelection(node.ID));
};

const clearSelectedNodes = () => {
  value1.value = [];
};

const buildConfig = () =>
  JSON.stringify({
    clash: Clash.value.trim(),
    surge: Surge.value.trim(),
    udp: checkList.value.includes("udp"),
    cert: checkList.value.includes("cert"),
  });

const addSubs = async () => {
  if (!validateStep(0) || !validateStep(1) || !validateStep(2)) return;

  const payload = {
    config: buildConfig(),
    name: Subname.value.trim(),
    nodes: value1.value.join(","),
    expire_at: expireAt.value || "",
    access_limit: accessLimit.value || "",
  };

  if (subMode.value === "add") {
    await AddSub(payload);
    ElMessage.success("添加成功");
  } else {
    await UpdateSub({
      ...payload,
      oldname: oldSubname.value,
    });
    ElMessage.success("更新成功");
  }

  await getsubs();
  dialogVisible.value = false;
  const saved = tableData.value.find((item) => item.Name === payload.name);
  if (saved) {
    handleClient(saved);
  }
};

const handleSelectionChange = (val: Sub[]) => {
  multipleSelection.value = val;
};

const selectAll = () => {
  tableData.value.forEach((row) => {
    table.value.toggleRowSelection(row, true);
  });
};

const toggleSelection = () => {
  table.value.clearSelection();
};

const handleSizeChange = (val: number) => {
  pageSize.value = val;
};

const handleCurrentChange = (val: number) => {
  currentPage.value = val;
};

const handleIplogs = (row: any) => {
  iplogsdialog.value = true;
  nextTick(() => {
    IplogsList.value = row.SubLogs || [];
  });
};

const handleDel = (row: any) => {
  ElMessageBox.confirm(`你是否要删除 ${row.Name} ?`, "提示", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(async () => {
    await DelSub({ id: row.ID });
    await getsubs();
    ElMessage.success("删除成功");
  });
};

const handleResetToken = async (row: any) => {
  await ElMessageBox.confirm(
    `重置 ${row.Name} 的订阅 token？旧 token 会立刻失效。`,
    "重置 token",
    {
      confirmButtonText: "重置",
      cancelButtonText: "取消",
      type: "warning",
    }
  );
  await ResetSubToken({ id: row.ID });
  await getsubs();
  ElMessage.success("token 已重置");
};

const handleToggleRevoked = async (row: any) => {
  await SetSubRevoked({ id: row.ID, revoked: !row.Revoked });
  await getsubs();
  ElMessage.success(row.Revoked ? "订阅已恢复" : "订阅已手动失效");
};

const selectDel = () => {
  if (multipleSelection.value.length === 0) return;
  ElMessageBox.confirm("你是否要删除选中的订阅？", "提示", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(async () => {
    await Promise.all(
      multipleSelection.value.map((item) => DelSub({ id: item.ID }))
    );
    await getsubs();
    ElMessage.success("删除成功");
  });
};

const copyUrl = (url: string) => {
  const textarea = document.createElement("textarea");
  textarea.value = url;
  document.body.appendChild(textarea);
  textarea.select();
  try {
    const successful = document.execCommand("copy");
    ElMessage({
      type: successful ? "success" : "warning",
      message: successful ? "复制成功" : "复制失败",
    });
  } catch {
    ElMessage.warning("复制失败");
  } finally {
    document.body.removeChild(textarea);
  }
};

const copyInfo = (row: any) => {
  copyUrl(row.Link);
};

const handleClient = (row: any) => {
  const serverAddress =
    location.protocol +
    "//" +
    location.hostname +
    (location.port ? ":" + location.port : "");
  ClientDiaLog.value = true;
  ClientSubName.value = row.Name;
  ClientUrls.value = {};
  ClientUrl.value = `${serverAddress}/c/?token=${row.Token}`;
  ClientList.forEach((item: string) => {
    ClientUrls.value[item] =
      `${serverAddress}/c/?token=${row.Token}&client=${item}`;
  });
};

const handleQrcode = (url: string, title: string) => {
  Qrdialog.value = true;
  qrcode.value = url;
  QrTitle.value = title;
};

const OpenUrl = (url: string) => {
  window.open(url);
};
</script>

<template>
  <div class="page-workspace">
    <el-dialog
      v-model="Qrdialog"
      class="form-dialog qr-dialog"
      width="400px"
      :title="QrTitle"
    >
      <div class="qr-content">
        <div class="qr-frame">
          <qrcode-vue :value="qrcode" :size="196" level="H" />
        </div>
        <el-input v-model="qrcode" readonly />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="copyUrl(qrcode)">复制地址</el-button>
          <el-button type="primary" @click="OpenUrl(qrcode)"
            >打开链接</el-button
          >
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="ClientDiaLog"
      class="form-dialog client-dialog"
      width="760px"
      title="订阅链接"
    >
      <div class="client-dialog-head">
        <div>
          <p class="dialog-intro">直接复制完整订阅地址，二维码作为备用入口。</p>
          <strong>{{ ClientSubName }}</strong>
        </div>
        <el-button type="primary" plain @click="copyUrl(ClientUrl)"
          >复制自动识别</el-button
        >
      </div>
      <div class="client-card-list">
        <article class="client-card">
          <div class="client-card-title">
            <span>自动识别</span>
            <el-tag size="small" effect="plain">推荐</el-tag>
          </div>
          <p>客户端根据请求头自动判断订阅格式。</p>
          <el-input
            :model-value="ClientUrl"
            readonly
            class="client-url-input"
          />
          <div class="client-actions">
            <el-button @click="copyUrl(ClientUrl)">复制链接</el-button>
            <el-button @click="handleQrcode(ClientUrl, '自动识别客户端')"
              >二维码</el-button
            >
            <el-button link type="primary" @click="OpenUrl(ClientUrl)"
              >打开</el-button
            >
          </div>
        </article>

        <article
          v-for="(item, index) in ClientUrls"
          :key="index"
          class="client-card"
        >
          <div class="client-card-title">
            <span>{{ index }}</span>
            <el-tag size="small" effect="plain">{{ index }}</el-tag>
          </div>
          <p>使用 {{ index }} 专用订阅地址。</p>
          <el-input :model-value="item" readonly class="client-url-input" />
          <div class="client-actions">
            <el-button @click="copyUrl(item)">复制链接</el-button>
            <el-button @click="handleQrcode(item, String(index))"
              >二维码</el-button
            >
            <el-button link type="primary" @click="OpenUrl(item)"
              >打开</el-button
            >
          </div>
        </article>
      </div>
    </el-dialog>

    <el-dialog
      v-model="iplogsdialog"
      class="data-dialog"
      title="访问记录"
      width="min(880px, calc(100vw - 32px))"
    >
      <el-table :data="IplogsList" style="width: 100%">
        <el-table-column prop="IP" label="IP" />
        <el-table-column prop="Count" label="总访问次数" />
        <el-table-column prop="Addr" label="来源" />
        <el-table-column prop="Date" label="最近时间" />
      </el-table>
    </el-dialog>

    <el-dialog
      v-model="dialogVisible"
      class="form-dialog subscription-wizard-dialog"
      width="min(1040px, calc(100vw - 32px))"
      :close-on-click-modal="true"
      destroy-on-close
    >
      <template #header>
        <div class="dialog-heading">
          <h2>{{ SubTitle }}</h2>
          <p>按步骤配置订阅信息、输出模板、节点顺序，最后确认保存。</p>
        </div>
      </template>

      <el-steps
        :active="wizardStep"
        finish-status="success"
        class="wizard-steps"
        simple
      >
        <el-step title="基本信息" @click="jumpStep(0)" />
        <el-step title="输出模板" @click="jumpStep(1)" />
        <el-step title="选择节点" @click="jumpStep(2)" />
        <el-step title="确认保存" @click="jumpStep(3)" />
      </el-steps>

      <section v-show="wizardStep === 0" class="wizard-panel">
        <div class="panel-copy">
          <h3>基本信息</h3>
          <p>先定义订阅名称、到期时间和访问限制。</p>
        </div>
        <div class="form-grid">
          <label class="field field-full">
            <span class="field-label">订阅名称</span>
            <el-input v-model="Subname" placeholder="例如 myclash" />
          </label>
          <label class="field">
            <span class="field-label">到期时间</span>
            <el-date-picker
              v-model="expireAt"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="不填表示永不过期"
              style="width: 100%"
            />
          </label>
          <label class="field">
            <span class="field-label">访问次数限制</span>
            <el-input-number
              v-model="accessLimit"
              :min="0"
              controls-position="right"
              placeholder="0 为不限"
              style="width: 100%"
            />
          </label>
        </div>
      </section>

      <section v-show="wizardStep === 1" class="wizard-panel">
        <div class="panel-copy">
          <h3>输出模板</h3>
          <p>分别为 Clash 和 Surge 选择本地模板或填写远程 URL。</p>
        </div>
        <div class="template-grid">
          <article class="template-card">
            <div class="template-card-head">
              <strong>Clash</strong>
              <el-radio-group
                v-model="clashTemplateMode"
                class="flat-segmented"
                size="small"
              >
                <el-radio-button value="local">本地模板</el-radio-button>
                <el-radio-button value="url">URL</el-radio-button>
              </el-radio-group>
            </div>
            <el-select
              v-if="clashTemplateMode === 'local'"
              v-model="Clash"
              placeholder="选择 Clash 模板"
            >
              <el-option
                v-for="template in templist"
                :key="template.file"
                :label="template.file"
                :value="'./template/' + template.file"
              />
            </el-select>
            <el-input
              v-else
              v-model="Clash"
              placeholder="输入 Clash 模板 URL"
            />
          </article>

          <article class="template-card">
            <div class="template-card-head">
              <strong>Surge</strong>
              <el-radio-group
                v-model="surgeTemplateMode"
                class="flat-segmented"
                size="small"
              >
                <el-radio-button value="local">本地模板</el-radio-button>
                <el-radio-button value="url">URL</el-radio-button>
              </el-radio-group>
            </div>
            <el-select
              v-if="surgeTemplateMode === 'local'"
              v-model="Surge"
              placeholder="选择 Surge 模板"
            >
              <el-option
                v-for="template in templist"
                :key="template.file"
                :label="template.file"
                :value="'./template/' + template.file"
              />
            </el-select>
            <el-input
              v-else
              v-model="Surge"
              placeholder="输入 Surge 模板 URL"
            />
          </article>
        </div>
        <div class="field">
          <span class="field-label">连接选项</span>
          <el-checkbox-group v-model="checkList" class="option-list">
            <el-checkbox value="udp" border>启用 UDP</el-checkbox>
            <el-checkbox value="cert" border>跳过证书验证</el-checkbox>
          </el-checkbox-group>
        </div>
      </section>

      <section v-show="wizardStep === 2" class="wizard-panel">
        <div class="panel-copy">
          <h3>选择节点</h3>
          <p>左侧选择节点，右侧拖拽排序。订阅输出会按右侧顺序生成。</p>
        </div>
        <div class="node-picker">
          <section class="node-column">
            <div class="node-column-head">
              <strong>可选节点</strong>
              <el-button link type="primary" @click="addAllVisibleNodes"
                >加入当前列表</el-button
              >
            </div>
            <el-input
              v-model="nodeKeyword"
              clearable
              placeholder="搜索节点"
              class="node-search"
            />
            <div class="node-list">
              <button
                v-for="node in availableNodes"
                :key="node.ID || node.Name"
                class="node-option"
                type="button"
                @click="addNodeToSelection(node.ID)"
              >
                <span>{{ node.Name }}</span>
                <span>加入</span>
              </button>
              <div v-if="!availableNodes.length" class="node-empty">
                没有可加入的节点
              </div>
            </div>
          </section>

          <section class="node-column selected-column">
            <div class="node-column-head">
              <strong>已选节点 {{ value1.length }} 个</strong>
              <el-button link type="danger" @click="clearSelectedNodes"
                >清空</el-button
              >
            </div>
            <VueDraggable
              v-if="value1.length"
              v-model="value1"
              :animation="150"
              ghost-class="ghost"
              class="selected-nodes"
            >
              <div
                v-for="(nodeId, index) in value1"
                :key="nodeId"
                class="draggable-item"
              >
                <span class="drag-handle">⋮⋮</span>
                <span class="row-number">{{ index + 1 }}</span>
                <span class="node-name">{{ nodeNameById(nodeId) }}</span>
                <el-button
                  link
                  type="danger"
                  @click.stop="removeNodeFromSelection(nodeId)"
                  >移除</el-button
                >
              </div>
            </VueDraggable>
            <div v-else class="empty-selection">尚未选择节点</div>
          </section>
        </div>
      </section>

      <section v-show="wizardStep === 3" class="wizard-panel">
        <div class="panel-copy">
          <h3>确认保存</h3>
          <p>确认无误后保存。保存成功后会直接打开订阅链接弹窗。</p>
        </div>
        <div class="summary-grid">
          <article class="summary-card">
            <span>订阅名称</span>
            <strong>{{ Subname || "未填写" }}</strong>
          </article>
          <article class="summary-card">
            <span>有效期</span>
            <strong>{{ expireAt || "永不过期" }}</strong>
          </article>
          <article class="summary-card">
            <span>访问限制</span>
            <strong
              >{{ accessLimit || 0 }}
              {{ accessLimit ? "次" : "为不限" }}</strong
            >
          </article>
          <article class="summary-card">
            <span>节点数量</span>
            <strong>{{ value1.length }} 个</strong>
          </article>
        </div>
        <div class="summary-block">
          <strong>模板</strong>
          <p>Clash：{{ Clash || "未配置" }}</p>
          <p>Surge：{{ Surge || "未配置" }}</p>
        </div>
        <div class="summary-block">
          <strong>连接选项</strong>
          <p>
            {{ checkList.includes("udp") ? "启用 UDP" : "未启用 UDP" }}，{{
              checkList.includes("cert") ? "跳过证书验证" : "不跳过证书验证"
            }}
          </p>
        </div>
        <div class="summary-block">
          <strong>节点顺序</strong>
          <p>{{ selectedNodePreview }}</p>
        </div>
      </section>

      <template #footer>
        <div class="wizard-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <div>
            <el-button v-if="wizardStep > 0" @click="prevStep"
              >上一步</el-button
            >
            <el-button v-if="wizardStep < 3" type="primary" @click="nextStep"
              >下一步</el-button
            >
            <el-button v-else type="primary" @click="addSubs"
              >保存订阅</el-button
            >
          </div>
        </div>
      </template>
    </el-dialog>

    <div class="page-heading">
      <div>
        <h1>订阅列表</h1>
        <p>管理订阅地址、客户端入口和访问记录</p>
      </div>
      <el-button type="primary" @click="handleAddSub">添加订阅</el-button>
    </div>

    <section class="work-surface">
      <div class="table-toolbar">
        <span class="record-count">共 {{ tableData.length }} 条订阅</span>
      </div>

      <el-table
        ref="table"
        :data="currentTableData"
        row-key="ID"
        :tree-props="{ children: 'Nodes' }"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" fixed width="48" />
        <el-table-column width="42" label="">
          <template #default>
            <span class="row-drag-handle" title="拖动排序">☰</span>
          </template>
        </el-table-column>
        <el-table-column prop="Name" label="订阅名称 / 节点" min-width="220">
          <template #default="{ row }">
            <span class="primary-cell">{{ row.Name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="Link" label="客户端" min-width="160">
          <template #default="{ row }">
            <el-button
              v-if="row.Nodes"
              link
              type="primary"
              @click="handleClient(row)"
            >
              查看客户端
            </el-button>
            <span v-else class="muted-cell">节点</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <template v-if="row.Nodes">
              <el-tag v-if="row.Revoked" type="danger" effect="plain"
                >已失效</el-tag
              >
              <el-tag
                v-else-if="
                  row.ExpireAt && beijingTimestamp(row.ExpireAt) < Date.now()
                "
                type="warning"
                effect="plain"
              >
                已过期
              </el-tag>
              <el-tag v-else type="success" effect="plain">有效</el-tag>
            </template>
            <span v-else class="muted-cell">节点</span>
          </template>
        </el-table-column>
        <el-table-column label="访问" width="110">
          <template #default="{ row }">
            <span v-if="row.Nodes">
              {{ row.AccessCount || 0 }}/{{ row.AccessLimit || "不限" }}
            </span>
            <span v-else class="muted-cell">--</span>
          </template>
        </el-table-column>
        <el-table-column
          prop="CreatedAt"
          label="创建时间"
          min-width="180"
          sortable
          :formatter="formatCreatedAt"
        />
        <el-table-column label="操作" width="280" align="right">
          <template #default="scope">
            <template v-if="scope.row.Nodes">
              <el-button link @click="handleIplogs(scope.row)">记录</el-button>
              <el-button link @click="handleResetToken(scope.row)"
                >重置 token</el-button
              >
              <el-button link @click="handleToggleRevoked(scope.row)">
                {{ scope.row.Revoked ? "恢复" : "失效" }}
              </el-button>
              <el-button link type="primary" @click="handleEdit(scope.row)"
                >编辑</el-button
              >
              <el-button link type="danger" @click="handleDel(scope.row)"
                >删除</el-button
              >
            </template>
            <el-button v-else link type="primary" @click="copyInfo(scope.row)">
              复制
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <div class="batch-actions">
          <el-button @click="selectAll()">全选</el-button>
          <el-button @click="toggleSelection()">取消选择</el-button>
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
.record-count,
.muted-cell {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.primary-cell {
  color: var(--el-text-color-primary);
  font-weight: 550;
}

.dialog-heading h2 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 20px;
}

.dialog-heading p,
.dialog-intro,
.panel-copy p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.wizard-steps {
  margin-bottom: 18px;
  border-radius: 10px;
}

.wizard-panel {
  min-height: 470px;
  padding: 18px 2px 4px;
}

.panel-copy {
  margin-bottom: 18px;
}

.panel-copy h3 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 17px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.field {
  display: grid;
  gap: 8px;
}

.field-full {
  grid-column: 1 / -1;
}

.field-label {
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 650;
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}

.template-card,
.summary-card,
.summary-block {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-fill-color-extra-light);
}

.template-card {
  display: grid;
  gap: 14px;
  padding: 16px;
}

.template-card-head,
.node-column-head,
.client-card-title,
.client-actions,
.wizard-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.option-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.node-picker {
  display: grid;
  grid-template-columns: minmax(0, 0.95fr) minmax(0, 1.05fr);
  gap: 14px;
}

.node-column {
  min-height: 390px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
}

.node-column-head {
  min-height: 48px;
  padding: 0 14px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-extra-light);
}

.node-search {
  padding: 12px;
}

.node-list,
.selected-nodes {
  max-height: 300px;
  overflow: auto;
}

.node-option {
  display: flex;
  width: 100%;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  border: 0;
  border-top: 1px solid var(--el-border-color-lighter);
  background: transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.node-option:hover {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.node-empty,
.empty-selection {
  margin: 14px;
  padding: 22px;
  border: 1px dashed var(--el-border-color);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-placeholder);
  font-size: 13px;
  text-align: center;
}

.draggable-item {
  display: grid;
  grid-template-columns: 18px 30px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  min-height: 44px;
  padding: 0 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  cursor: grab;
}

.draggable-item:hover {
  background: var(--el-fill-color-light);
}

.ghost {
  opacity: 0.45;
  background: var(--el-color-primary-light-9);
}

.row-number {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.drag-handle {
  color: var(--el-text-color-placeholder);
  letter-spacing: -3px;
}

.node-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.summary-card {
  display: grid;
  gap: 8px;
  padding: 14px;
}

.summary-card span,
.summary-block p {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.summary-card strong,
.summary-block strong {
  color: var(--el-text-color-primary);
}

.summary-block {
  padding: 14px;
  margin-top: 12px;
}

.summary-block p {
  margin: 8px 0 0;
  word-break: break-all;
}

.qr-content {
  display: grid;
  gap: 16px;
}

.qr-frame {
  display: grid;
  width: 228px;
  height: 228px;
  margin: 0 auto;
  place-items: center;
  border: 1px solid var(--el-border-color-lighter);
  background: #fff;
}

.client-dialog-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.client-dialog-head .dialog-intro {
  margin-bottom: 4px;
}

.client-card-list {
  display: grid;
  gap: 12px;
  margin-top: 16px;
}

.client-card {
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-extra-light);
}

.client-card-title span {
  color: var(--el-text-color-primary);
  font-weight: 650;
}

.client-card p {
  margin: 7px 0 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.client-url-input {
  margin-bottom: 10px;
}

.client-actions {
  flex-wrap: wrap;
  justify-content: flex-start;
}

@media (max-width: 860px) {
  .form-grid,
  .template-grid,
  .node-picker,
  .summary-grid {
    grid-template-columns: 1fr;
  }

  .wizard-panel {
    min-height: auto;
  }

  .node-column {
    min-height: 300px;
  }

  .client-dialog-head,
  .client-card-title {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 640px) {
  :deep(.subscription-wizard-dialog) {
    width: calc(100vw - 12px) !important;
    margin-top: 6px !important;
  }

  :deep(.subscription-wizard-dialog .el-dialog__body) {
    padding: 14px 12px;
  }

  :deep(.subscription-wizard-dialog .el-dialog__footer) {
    padding: 10px 12px;
  }

  .wizard-steps {
    overflow-x: auto;
    padding: 6px;
    -webkit-overflow-scrolling: touch;
  }

  .wizard-steps :deep(.el-step) {
    min-width: 96px;
  }

  .wizard-panel {
    padding-top: 12px;
  }

  .panel-copy {
    margin-bottom: 12px;
  }

  .template-card,
  .summary-card,
  .summary-block {
    border-radius: 10px;
  }

  .template-card-head,
  .node-column-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .node-column {
    min-height: auto;
  }

  .node-list,
  .selected-nodes {
    max-height: 220px;
  }

  .draggable-item {
    grid-template-columns: 18px 24px minmax(0, 1fr);
    padding: 8px 10px;
  }

  .draggable-item .el-button {
    grid-column: 3;
    justify-self: start;
    padding: 0;
  }

  .client-dialog-head {
    gap: 10px;
  }

  .client-card {
    padding: 12px;
  }
}
</style>
