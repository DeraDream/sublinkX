<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { parseDocument, stringify } from "yaml";
import YamlEditor from "@/components/YamlEditor/index.vue";
import {
  createVisualTemplate,
  deleteVisualTemplate,
  getVisualTemplate,
  listVisualTemplates,
  updateVisualTemplate,
  type VisualTemplateItem,
} from "@/api/template-manager";

type ProxyGroup = Record<string, any> & { name: string; type: string; proxies?: string[] };
type TemplateRoot = Record<string, any>;

const templateList = ref<VisualTemplateItem[]>([]);
const selectedFile = ref("");
const filename = ref("");
const yamlText = ref("");
const templateRoot = ref<TemplateRoot>({});
const groups = ref<ProxyGroup[]>([]);
const mode = ref<"visual" | "yaml">("visual");
const loading = ref(false);
const saving = ref(false);
const parseError = ref("");
const visualReady = ref(false);
const dirty = ref(false);
const syncing = ref(false);

const groupTypes = [
  { value: "select", label: "手动选择" },
  { value: "url-test", label: "自动测速" },
  { value: "fallback", label: "故障转移" },
  { value: "load-balance", label: "负载均衡" },
  { value: "relay", label: "链式代理" },
];
const speedTestTypes = new Set(["url-test", "fallback", "load-balance"]);
const groupNames = computed(() => groups.value.map((group) => group.name).filter(Boolean));

function blankTemplate() {
  return {
    "mixed-port": 7890,
    "allow-lan": false,
    mode: "rule",
    "log-level": "info",
    "proxy-groups": [],
    rules: ["MATCH,DIRECT"],
  };
}

function resetEditor() {
  selectedFile.value = "";
  filename.value = "untitled.yaml";
  templateRoot.value = blankTemplate();
  groups.value = [];
  yamlText.value = stringify(templateRoot.value, { lineWidth: 0 });
  parseError.value = "";
  visualReady.value = true;
  mode.value = "visual";
  dirty.value = false;
}

function parseYaml(showMessage = true) {
  const doc = parseDocument(yamlText.value, { prettyErrors: false, strict: true });
  if (doc.errors.length) {
    parseError.value = String(doc.errors[0].message || "YAML 格式错误");
    visualReady.value = false;
    if (showMessage) ElMessage.error(`无法切换可视化编辑：${parseError.value}`);
    return false;
  }
  const parsed = doc.toJS();
  if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") {
    parseError.value = "模板根节点必须是 YAML 对象";
    visualReady.value = false;
    if (showMessage) ElMessage.error(parseError.value);
    return false;
  }
  const nextGroups = (parsed["proxy-groups"] || []) as unknown;
  if (!Array.isArray(nextGroups) || nextGroups.some((group) => !group || typeof group !== "object")) {
    parseError.value = "proxy-groups 必须是代理组数组";
    visualReady.value = false;
    if (showMessage) ElMessage.error(parseError.value);
    return false;
  }
  syncing.value = true;
  templateRoot.value = parsed as TemplateRoot;
  groups.value = nextGroups.map((group) => ({ proxies: [], ...(group as ProxyGroup) }));
  syncing.value = false;
  parseError.value = "";
  visualReady.value = true;
  return true;
}

function syncYamlFromVisual() {
  if (syncing.value || !visualReady.value) return;
  templateRoot.value["proxy-groups"] = groups.value.map((group) => {
    const next = { ...group };
    if (!next.proxies?.length) delete next.proxies;
    Object.keys(next).forEach((key) => {
      if (next[key] === "" || next[key] === false || next[key] === undefined) delete next[key];
    });
    return next;
  });
  yamlText.value = stringify(templateRoot.value, { lineWidth: 0 });
  dirty.value = true;
}

watch(groups, syncYamlFromVisual, { deep: true });

async function loadList() {
  const { data } = await listVisualTemplates();
  templateList.value = data || [];
}

async function selectTemplate(file: string) {
  if (dirty.value && !await confirmDiscard()) return;
  loading.value = true;
  try {
    const { data } = await getVisualTemplate(file);
    selectedFile.value = data.file;
    filename.value = data.file;
    yamlText.value = data.content;
    parseYaml(false);
    mode.value = visualReady.value ? "visual" : "yaml";
    dirty.value = false;
  } finally {
    loading.value = false;
  }
}

async function confirmDiscard() {
  try {
    await ElMessageBox.confirm("当前模板有未保存的更改，确定放弃吗？", "未保存的更改", { type: "warning" });
    return true;
  } catch {
    return false;
  }
}

async function createTemplate() {
  if (dirty.value && !await confirmDiscard()) return;
  resetEditor();
}

function addGroup() {
  groups.value.push({
    name: `新代理组 ${groups.value.length + 1}`,
    type: "select",
    proxies: ["DIRECT"],
  });
}

function removeGroup(index: number) {
  groups.value.splice(index, 1);
}

function moveGroup(index: number, direction: number) {
  const target = index + direction;
  if (target < 0 || target >= groups.value.length) return;
  const [group] = groups.value.splice(index, 1);
  groups.value.splice(target, 0, group);
}

function addProxy(group: ProxyGroup) {
  group.proxies ||= [];
  group.proxies.push("");
}

function removeProxy(group: ProxyGroup, index: number) {
  group.proxies?.splice(index, 1);
}

function switchMode(next: "visual" | "yaml") {
  if (next === "visual" && !parseYaml()) return;
  mode.value = next;
}

function handleModeChange(value: string | number | boolean | undefined) {
  if (value === "visual" || value === "yaml") switchMode(value);
}

async function saveTemplate() {
  const safeName = filename.value.trim();
  if (!safeName) { ElMessage.warning("请输入模板文件名"); return; }
  if (mode.value === "yaml" && !parseYaml(false)) { ElMessage.error(`YAML 格式错误：${parseError.value}`); return; }
  if (!visualReady.value) { ElMessage.error("请先修复 YAML 格式错误"); return; }
  const names = groups.value.map((group) => group.name.trim()).filter(Boolean);
  if (names.length !== groups.value.length || new Set(names).size !== names.length) {
    ElMessage.error("每个代理组都需要唯一的名称");
    return;
  }
  saving.value = true;
  try {
    if (selectedFile.value) {
      await updateVisualTemplate({ filename: safeName, old_name: selectedFile.value, content: yamlText.value });
    } else {
      await createVisualTemplate({ filename: safeName, content: yamlText.value });
    }
    selectedFile.value = /\.ya?ml$/i.test(safeName) ? safeName : `${safeName}.yaml`;
    filename.value = selectedFile.value;
    dirty.value = false;
    await loadList();
    ElMessage.success("模板已保存");
  } finally {
    saving.value = false;
  }
}

async function deleteCurrentTemplate() {
  if (!selectedFile.value) return;
  try {
    await ElMessageBox.confirm(`确定删除模板「${selectedFile.value}」吗？`, "删除模板", { type: "warning" });
    await deleteVisualTemplate(selectedFile.value);
    await loadList();
    resetEditor();
    ElMessage.success("模板已删除");
  } catch (error: any) {
    if (error !== "cancel" && error !== "close") throw error;
  }
}

watch(yamlText, () => { if (!syncing.value && mode.value === "yaml") dirty.value = true; });
loadList().then(() => resetEditor());
</script>

<template>
  <div class="template-manager page-workspace">
    <header class="page-heading">
      <div>
        <h1>模板管理</h1>
        <p>独立管理 Clash / Mihomo 可视化模板，不影响现有模板列表或订阅配置。</p>
      </div>
      <div class="heading-actions">
        <el-button @click="createTemplate">新建模板</el-button>
        <el-button v-if="selectedFile" type="danger" plain @click="deleteCurrentTemplate">删除</el-button>
        <el-button type="primary" :loading="saving" @click="saveTemplate">保存模板</el-button>
      </div>
    </header>

    <section class="manager-shell" v-loading="loading">
      <aside class="template-library">
        <div class="library-title"><strong>可视化模板</strong><span>{{ templateList.length }}</span></div>
        <el-scrollbar>
          <button
            v-for="item in templateList"
            :key="item.file"
            class="template-item"
            :class="{ active: item.file === selectedFile }"
            @click="selectTemplate(item.file)"
          >
            <strong>{{ item.file }}</strong><small>{{ item.updated_at }}</small>
          </button>
          <div v-if="!templateList.length" class="empty-library">还没有可视化模板</div>
        </el-scrollbar>
      </aside>

      <main class="editor-workspace">
        <div class="editor-toolbar">
          <el-input v-model="filename" class="name-input" placeholder="模板文件名，例如 my-template.yaml">
            <template #prepend>文件名</template>
          </el-input>
          <el-radio-group :model-value="mode" size="default" @change="handleModeChange">
            <el-radio-button value="visual">可视化编辑</el-radio-button>
            <el-radio-button value="yaml">原始 YAML</el-radio-button>
          </el-radio-group>
        </div>

        <el-alert v-if="parseError" type="warning" :closable="false" show-icon class="parse-alert" :title="`可视化编辑不可用：${parseError}`" />

        <template v-if="mode === 'visual' && visualReady">
          <div class="visual-header">
            <div><h2>代理组配置</h2><p>更改会实时同步到右侧 YAML；未识别的顶级配置会原样保留。</p></div>
            <el-button type="primary" plain @click="addGroup">添加代理组</el-button>
          </div>
          <div class="visual-layout">
            <section class="group-list">
              <el-empty v-if="!groups.length" description="还没有代理组，点击上方按钮添加" :image-size="84" />
              <article v-for="(group, index) in groups" :key="`${index}-${group.name}`" class="group-card">
                <div class="group-card-head">
                  <el-input v-model="group.name" placeholder="代理组名称" />
                  <el-select v-model="group.type" class="type-select">
                    <el-option v-for="type in groupTypes" :key="type.value" :label="type.label" :value="type.value" />
                  </el-select>
                  <div class="group-actions">
                    <el-button circle size="small" :disabled="index === 0" @click="moveGroup(index, -1)">↑</el-button>
                    <el-button circle size="small" :disabled="index === groups.length - 1" @click="moveGroup(index, 1)">↓</el-button>
                    <el-button circle size="small" type="danger" plain @click="removeGroup(index)">×</el-button>
                  </div>
                </div>
                <div class="group-fields">
                  <el-form label-position="top" size="small">
                    <div class="field-grid">
                      <el-form-item label="节点筛选（正则）"><el-input v-model="group.filter" placeholder="香港|HK|Hong Kong" clearable /></el-form-item>
                      <el-form-item label="排除筛选（正则）"><el-input v-model="group['exclude-filter']" placeholder="倍率|剩余流量" clearable /></el-form-item>
                    </div>
                    <div class="switch-row">
                      <el-switch v-model="group['include-all']" active-text="包含全部代理" />
                      <el-switch v-model="group.hidden" active-text="在客户端隐藏" />
                    </div>
                    <div v-if="speedTestTypes.has(group.type)" class="field-grid speed-fields">
                      <el-form-item label="测试 URL"><el-input v-model="group.url" placeholder="https://www.gstatic.com/generate_204" /></el-form-item>
                      <el-form-item label="间隔（秒）"><el-input-number v-model="group.interval" :min="1" :max="86400" /></el-form-item>
                      <el-form-item label="容差（ms）"><el-input-number v-model="group.tolerance" :min="0" :max="1000" /></el-form-item>
                    </div>
                    <el-form-item label="代理顺序 / 固定成员">
                      <div class="proxy-list">
                        <div v-for="(_, proxyIndex) in group.proxies || []" :key="proxyIndex" class="proxy-line">
                          <el-select v-model="group.proxies![proxyIndex]" filterable allow-create default-first-option placeholder="选择组或输入 DIRECT / 节点名">
                            <el-option value="DIRECT" label="DIRECT" /><el-option value="REJECT" label="REJECT" />
                            <el-option v-for="name in groupNames.filter((name) => name !== group.name)" :key="name" :value="name" :label="name" />
                          </el-select>
                          <el-button type="danger" link @click="removeProxy(group, proxyIndex)">移除</el-button>
                        </div>
                        <el-button size="small" plain @click="addProxy(group)">添加成员</el-button>
                      </div>
                    </el-form-item>
                  </el-form>
                </div>
              </article>
            </section>
            <section class="yaml-panel"><div class="yaml-title">实时 YAML 预览</div><pre>{{ yamlText }}</pre></section>
          </div>
        </template>

        <section v-else class="raw-editor"><div class="yaml-title">原始 YAML（保存时会校验）</div><YamlEditor v-model="yamlText" /></section>
      </main>
    </section>
  </div>
</template>

<style scoped>
.heading-actions,.editor-toolbar,.group-card-head,.switch-row,.proxy-line { display:flex; align-items:center; gap:10px; }
.manager-shell { display:grid; grid-template-columns:260px minmax(0,1fr); min-height:calc(100vh - 185px); overflow:hidden; border:1px solid var(--el-border-color-lighter); border-radius:12px; background:var(--el-bg-color); }
.template-library { min-width:0; border-right:1px solid var(--el-border-color-lighter); background:var(--el-fill-color-light); }
.library-title { display:flex; justify-content:space-between; padding:18px 16px 12px; color:var(--el-text-color-primary); }.library-title span { color:var(--el-text-color-secondary); font-size:12px; }
.template-item { display:grid; width:100%; gap:5px; padding:12px 16px; border:0; border-left:3px solid transparent; color:var(--el-text-color-primary); background:transparent; text-align:left; cursor:pointer; }.template-item:hover { background:var(--el-fill-color); }.template-item.active { border-left-color:var(--el-color-primary); background:var(--el-color-primary-light-9); }.template-item strong { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:13px; }.template-item small,.empty-library { color:var(--el-text-color-secondary); font-size:12px; }.empty-library { padding:24px 16px; text-align:center; }
.editor-workspace { min-width:0; padding:18px; }.editor-toolbar { justify-content:space-between; padding-bottom:16px; border-bottom:1px solid var(--el-border-color-lighter); }.name-input { max-width:440px; }.parse-alert { margin:14px 0; }.visual-header { display:flex; align-items:center; justify-content:space-between; gap:18px; margin:18px 0 12px; }.visual-header h2 { margin:0; font-size:16px; }.visual-header p { margin:5px 0 0; color:var(--el-text-color-secondary); font-size:13px; }
.visual-layout { display:grid; grid-template-columns:minmax(430px,1.15fr) minmax(300px,.85fr); gap:16px; }.group-list { display:grid; align-content:start; gap:12px; }.group-card,.yaml-panel,.raw-editor { overflow:hidden; border:1px solid var(--el-border-color-lighter); border-radius:8px; }.group-card-head { padding:12px; background:var(--el-fill-color-light); }.type-select { width:130px; }.group-actions { display:flex; margin-left:auto; gap:5px; }.group-fields { padding:12px 14px 4px; }.field-grid { display:grid; grid-template-columns:1fr 1fr; gap:12px; }.speed-fields { grid-template-columns:minmax(0,1fr) 110px 110px; }.switch-row { margin:0 0 12px; flex-wrap:wrap; }.proxy-list { display:grid; gap:7px; }.proxy-line { width:100%; }.proxy-line .el-select { flex:1; }.yaml-panel,.raw-editor { min-height:520px; display:flex; flex-direction:column; }.yaml-title { padding:12px 14px; border-bottom:1px solid var(--el-border-color-lighter); background:var(--el-fill-color-light); font-size:13px; font-weight:600; }.yaml-panel pre { flex:1; overflow:auto; margin:0; padding:14px; color:var(--el-text-color-primary); font:12px/1.55 Consolas,monospace; white-space:pre; }.raw-editor :deep(.yaml-editor) { flex:1; min-height:520px; }
@media (max-width:900px) { .manager-shell { grid-template-columns:1fr; }.template-library { max-height:185px; border-right:0; border-bottom:1px solid var(--el-border-color-lighter); }.visual-layout { grid-template-columns:1fr; }.yaml-panel { min-height:300px; }.field-grid,.speed-fields { grid-template-columns:1fr; }.editor-toolbar,.visual-header { align-items:stretch; flex-direction:column; }.name-input { max-width:none; }.heading-actions { flex-wrap:wrap; } }
</style>
