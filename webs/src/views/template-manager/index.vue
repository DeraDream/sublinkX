<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { parseDocument, stringify } from "yaml";
import YamlEditor from "@/components/YamlEditor/index.vue";
import { getTemp } from "@/api/subcription/temp";
import { createVisualTemplate, deleteVisualTemplate, fetchVisualTemplateSource, getVisualTemplate, listVisualTemplates, updateVisualTemplate, type VisualTemplateItem } from "@/api/template-manager";

type Group = Record<string, any> & { name: string; type: string; proxies?: string[] };
type Legacy = { file: string; text: string };
const templates = ref<VisualTemplateItem[]>([]);
const legacy = ref<Legacy[]>([]);
const createVisible = ref(false), editorVisible = ref(false), loading = ref(false), saving = ref(false);
const createMode = ref<"blank" | "import" | "paste" | "subscription">("blank"), importedFile = ref(""), pastedYaml = ref(""), subscriptionURL = ref(""), fetchingSubscription = ref(false), createFilename = ref("untitled.yaml"), creating = ref(false);
const selectedFile = ref(""), filename = ref(""), yamlText = ref("");
const root = ref<Record<string, any>>({}), groups = ref<Group[]>([]);
const mode = ref<"visual" | "yaml">("visual"), parseError = ref(""), visualReady = ref(false), dirty = ref(false), syncing = ref(false);
const expandedGroups = ref<number[]>([]);
const types = [{ value: "select", label: "手动选择" }, { value: "url-test", label: "自动测速" }, { value: "fallback", label: "故障转移" }, { value: "load-balance", label: "负载均衡" }, { value: "relay", label: "链式代理" }];
const speedTypes = new Set(["url-test", "fallback", "load-balance"]);
const names = computed(() => groups.value.map((group) => group.name).filter(Boolean));
const previewYaml = computed(() => {
  const doc = parseDocument(yamlText.value, { prettyErrors: false, strict: true });
  if (doc.errors.length) return "# YAML 格式错误，修复后会显示代理组预览";
  const value = doc.toJS();
  if (!value || Array.isArray(value) || typeof value !== "object") return "# 暂无代理组配置";
  return stringify({ "proxy-groups": value["proxy-groups"] || [] }, { lineWidth: 0 });
});
const blank = () => ({ "mixed-port": 7890, "allow-lan": false, mode: "rule", "log-level": "info", "proxy-groups": [], rules: ["MATCH,DIRECT"] });

function reset() {
  selectedFile.value = ""; filename.value = "untitled.yaml"; root.value = blank(); groups.value = [];
  yamlText.value = stringify(root.value, { lineWidth: 0 }); parseError.value = ""; visualReady.value = true; mode.value = "visual"; dirty.value = false;
}
function parse(showError = true) {
  const doc = parseDocument(yamlText.value, { prettyErrors: false, strict: true });
  if (doc.errors.length) { parseError.value = String(doc.errors[0].message || "YAML 格式错误"); visualReady.value = false; if (showError) ElMessage.error(`无法切换可视化编辑：${parseError.value}`); return false; }
  const value = doc.toJS();
  if (!value || Array.isArray(value) || typeof value !== "object") { parseError.value = "模板根节点必须是 YAML 对象"; visualReady.value = false; if (showError) ElMessage.error(parseError.value); return false; }
  const parsedGroups = value["proxy-groups"] || [];
  if (!Array.isArray(parsedGroups) || parsedGroups.some((item) => !item || typeof item !== "object")) { parseError.value = "proxy-groups 必须是代理组数组"; visualReady.value = false; if (showError) ElMessage.error(parseError.value); return false; }
  syncing.value = true; root.value = value; groups.value = parsedGroups.map((item) => ({ proxies: [], ...item })); expandedGroups.value = []; syncing.value = false;
  parseError.value = ""; visualReady.value = true; return true;
}
function sync() {
  if (syncing.value || !visualReady.value) return;
  root.value["proxy-groups"] = groups.value.map((group) => { const item = { ...group }; if (!item.proxies?.length) delete item.proxies; Object.keys(item).forEach((key) => { if (item[key] === "" || item[key] === false || item[key] === undefined) delete item[key]; }); return item; });
  yamlText.value = stringify(root.value, { lineWidth: 0 }); dirty.value = true;
}
watch(groups, sync, { deep: true });
watch(yamlText, () => { if (!syncing.value && mode.value === "yaml") dirty.value = true; });
async function loadList() { const { data } = await listVisualTemplates(); templates.value = data || []; }
async function openCreate() {
  createMode.value = "blank"; importedFile.value = ""; pastedYaml.value = ""; subscriptionURL.value = ""; createFilename.value = "untitled.yaml"; createVisible.value = true;
  const { data } = await getTemp(); legacy.value = (data || []).filter((item: Legacy) => /.ya?ml$/i.test(item.file));
}
function selectLegacy(file: string) { const item = legacy.value.find((candidate) => candidate.file === file); if (item) { createFilename.value = item.file; pastedYaml.value = item.text; } }
async function fetchSubscription() { if (!subscriptionURL.value.trim()) { ElMessage.warning("请输入订阅地址"); return; } fetchingSubscription.value = true; try { const { data } = await fetchVisualTemplateSource(subscriptionURL.value); pastedYaml.value = data.content; ElMessage.success("订阅内容已拉取，请确认后继续"); } finally { fetchingSubscription.value = false; } }
async function create() {
  const name = createFilename.value.trim();
  if (!name) { ElMessage.warning("请输入模板文件名"); return; }
  if (createMode.value === "import" && !importedFile.value) { ElMessage.warning("请选择要导入的现有模板"); return; }
  const content = createMode.value === "blank" ? stringify(blank(), { lineWidth: 0 }) : pastedYaml.value;
  if (!content.trim()) { ElMessage.warning(createMode.value === "paste" ? "请粘贴 YAML 内容" : "请先拉取订阅内容"); return; }
  const doc = parseDocument(content, { prettyErrors: false, strict: true });
  if (doc.errors.length) { ElMessage.error(`YAML 格式错误：${doc.errors[0].message}`); return; }
  creating.value = true;
  try { await createVisualTemplate({ filename: name, content }); await loadList(); createVisible.value = false; ElMessage.success("模板已创建，请从列表点击编辑"); } finally { creating.value = false; }
}
async function edit(file: string) {
  loading.value = true;
  try { const { data } = await getVisualTemplate(file); selectedFile.value = data.file; filename.value = data.file; yamlText.value = data.content; parse(false); mode.value = visualReady.value ? "visual" : "yaml"; dirty.value = false; editorVisible.value = true; } finally { loading.value = false; }
}
function addGroup() { groups.value.push({ name: `新代理组 ${groups.value.length + 1}`, type: "select", proxies: ["DIRECT"] }); }
function move(index: number, delta: number) { const target = index + delta; if (target < 0 || target >= groups.value.length) return; const [item] = groups.value.splice(index, 1); groups.value.splice(target, 0, item); }
function isExpanded(index: number) { return expandedGroups.value.includes(index); }
function toggleGroup(index: number) { expandedGroups.value = isExpanded(index) ? expandedGroups.value.filter((item) => item !== index) : [...expandedGroups.value, index]; }
function switchMode(value: string | number | boolean | undefined) { if (value !== "visual" && value !== "yaml") return; if (value === "visual" && !parse()) return; mode.value = value; }
async function copyPreview() { try { await navigator.clipboard.writeText(previewYaml.value); ElMessage.success("代理组配置已复制"); } catch { ElMessage.error("复制失败，请检查浏览器剪贴板权限"); } }
function refreshPreview() { if (parse(false)) ElMessage.success("预览已刷新"); else ElMessage.error(`YAML 格式错误：${parseError.value}`); }
async function save() {
  const name = filename.value.trim(); if (!name) { ElMessage.warning("请输入模板文件名"); return; }
  if (mode.value === "yaml" && !parse(false)) { ElMessage.error(`YAML 格式错误：${parseError.value}`); return; }
  const groupNames = groups.value.map((item) => item.name.trim()).filter(Boolean);
  if (groupNames.length !== groups.value.length || new Set(groupNames).size !== groupNames.length) { ElMessage.error("每个代理组都需要唯一的名称"); return; }
  saving.value = true;
  try { if (selectedFile.value) await updateVisualTemplate({ filename: name, old_name: selectedFile.value, content: yamlText.value }); else await createVisualTemplate({ filename: name, content: yamlText.value }); selectedFile.value = /.ya?ml$/i.test(name) ? name : `${name}.yaml`; filename.value = selectedFile.value; dirty.value = false; await loadList(); ElMessage.success("模板已保存"); } finally { saving.value = false; }
}
async function close() { if (!dirty.value) { editorVisible.value = false; return; } try { await ElMessageBox.confirm("当前模板有未保存的更改，确定关闭吗？", "未保存的更改", { type: "warning" }); editorVisible.value = false; } catch {} }
async function remove(file: string) { try { await ElMessageBox.confirm(`确定删除模板「${file}」吗？`, "删除模板", { type: "warning" }); await deleteVisualTemplate(file); await loadList(); ElMessage.success("模板已删除"); } catch (error: any) { if (error !== "cancel" && error !== "close") throw error; } }
loadList();
</script>

<template>
  <div class="template-manager page-workspace">
    <header class="page-heading"><div><h1>模板管理</h1><p>创建或导入 Clash / Mihomo 模板，再进入可视化编辑。</p></div><el-button type="primary" @click="openCreate">新建模板</el-button></header>
    <section class="work-surface manager-list" v-loading="loading"><el-table :data="templates" empty-text="暂无模板，点击右上角新建模板"><el-table-column prop="file" label="模板名称" min-width="360"><template #default="{ row }"><strong>{{ row.file }}</strong></template></el-table-column><el-table-column prop="updated_at" label="更新时间" min-width="220" /><el-table-column label="操作" width="180" align="right"><template #default="{ row }"><el-button link type="primary" @click="edit(row.file)">编辑</el-button><el-button link type="danger" @click="remove(row.file)">删除</el-button></template></el-table-column></el-table></section>

    <el-dialog v-model="createVisible" title="创建模板" width="min(680px, 94vw)" destroy-on-close>
      <p class="intro">创建空白 Clash/Mihomo YAML，或从当前项目的“模板列表”导入一个 YAML 模板。</p>
      <el-form label-position="top"><el-form-item label="模板文件名"><el-input v-model="createFilename" placeholder="例如 my-template.yaml" /></el-form-item></el-form>
      <el-radio-group v-model="createMode" class="create-tabs"><el-radio-button value="blank">创建空白模板</el-radio-button><el-radio-button value="import">从模板列表导入</el-radio-button><el-radio-button value="paste">粘贴代码</el-radio-button><el-radio-button value="subscription">从订阅导入</el-radio-button></el-radio-group>
      <div v-if="createMode === 'blank'" class="create-option"><el-alert type="info" :closable="false" show-icon title="会创建基础 Clash/Mihomo YAML；之后可在可视化编辑器添加代理组。" /></div>
      <div v-else-if="createMode === 'import'" class="create-option"><el-form label-position="top"><el-form-item label="选择现有 YAML 模板"><el-select v-model="importedFile" filterable placeholder="选择要导入的模板" class="wide" @change="selectLegacy"><el-option v-for="item in legacy" :key="item.file" :label="item.file" :value="item.file" /></el-select></el-form-item></el-form><el-alert type="warning" :closable="false" show-icon title="仅显示 YAML 模板；导入后另存为可视化模板，不会修改原模板。" /></div>
      <div v-else-if="createMode === 'paste'" class="create-option"><el-form label-position="top"><el-form-item label="粘贴 YAML 内容"><el-input v-model="pastedYaml" type="textarea" :rows="7" placeholder="粘贴 Clash / Mihomo YAML 配置" /></el-form-item></el-form></div>
      <div v-else class="create-option"><el-form label-position="top"><el-form-item label="订阅地址"><div class="subscription-input"><el-input v-model="subscriptionURL" placeholder="https://example.com/config.yaml" /><el-button :loading="fetchingSubscription" @click="fetchSubscription">拉取</el-button></div></el-form-item><el-form-item label="拉取结果"><el-input v-model="pastedYaml" type="textarea" :rows="5" placeholder="拉取成功后会显示在这里，也可直接修改" /></el-form-item></el-form><el-alert type="warning" :closable="false" show-icon title="仅支持可直接返回 Clash/Mihomo YAML 的 HTTP(S) 地址；本地和私有网络地址会被拒绝。" /></div>
      <template #footer><el-button @click="createVisible = false">取消</el-button><el-button type="primary" :loading="creating" @click="create">创建</el-button></template>
    </el-dialog>

    <el-dialog v-model="editorVisible" class="editor-dialog" width="calc(100vw - 48px)" top="24px" :close-on-click-modal="false" destroy-on-close @close="close">
      <template #header><div class="editor-title"><div><h2>{{ selectedFile || filename || "新模板" }}</h2><p>编辑模板配置</p></div><div class="actions"><el-button type="primary" :loading="saving" @click="save">保存</el-button><el-button @click="close">关闭</el-button></div></div></template>
      <div class="toolbar"><el-input v-model="filename" class="filename" placeholder="模板文件名，例如 my-template.yaml"><template #prepend>文件名</template></el-input></div>
      <el-alert v-if="parseError" type="warning" :closable="false" show-icon class="parse-error" :title="`可视化编辑不可用：${parseError}`" />
      <div class="editor-layout">
        <section class="panel editor-panel"><div class="editor-tabs"><el-radio-group :model-value="mode" @change="switchMode"><el-radio-button value="visual">可视化编辑</el-radio-button><el-radio-button value="yaml">YAML 代码</el-radio-button></el-radio-group></div><template v-if="mode === 'visual' && visualReady"><div class="panel-title"><span>代理组配置 <em>{{ groups.length }}</em></span><el-button size="small" type="primary" @click="addGroup">添加代理组</el-button></div><el-scrollbar class="scroll"><div class="groups"><el-empty v-if="!groups.length" description="暂无代理组" :image-size="72" /><article v-for="(group, index) in groups" :key="index" class="group" :class="{ expanded: isExpanded(index) }"><div class="group-head"><button class="group-toggle" :aria-label="isExpanded(index) ? '收起代理组' : '展开代理组'" @click="toggleGroup(index)"><span class="drag">⠿</span><strong>{{ group.name || `未命名代理组 ${index + 1}` }}</strong><small>{{ types.find((item) => item.value === group.type)?.label || group.type }}</small><span class="chevron">⌄</span></button><div class="group-actions"><el-button circle size="small" :disabled="index === 0" @click="move(index, -1)">↑</el-button><el-button circle size="small" :disabled="index === groups.length - 1" @click="move(index, 1)">↓</el-button><el-button circle size="small" type="danger" plain @click="groups.splice(index, 1)">×</el-button></div></div><div v-show="isExpanded(index)" class="fields"><div class="field-grid"><label>组名称<el-input v-model="group.name" placeholder="例如：🚀 手动选择" /></label><label>组类型<el-select v-model="group.type"><el-option v-for="item in types" :key="item.value" :label="item.label" :value="item.value" /></el-select></label></div><div class="switches"><el-switch v-model="group['include-all']" active-text="包含全部代理" /><el-switch v-model="group.hidden" active-text="在客户端隐藏" /></div><div v-if="speedTypes.has(group.type)" class="speed"><label>测试 URL<el-input v-model="group.url" placeholder="https://www.gstatic.com/generate_204" /></label><label>测速间隔<el-input-number v-model="group.interval" :min="1" placeholder="300" /></label><label>容差<el-input-number v-model="group.tolerance" :min="0" placeholder="50" /></label></div><div class="field-grid"><label>筛选关键词（filter）<el-input v-model="group.filter" placeholder="例如：香港|HK" clearable /></label><label>排除关键词（exclude-filter）<el-input v-model="group['exclude-filter']" placeholder="例如：游戏|IPLC" clearable /></label></div><div class="proxies"><label>代理组成员</label><div v-for="(_, proxyIndex) in group.proxies || []" :key="proxyIndex" class="proxy"><el-select v-model="group.proxies![proxyIndex]" filterable allow-create default-first-option placeholder="选择组或输入成员"><el-option value="DIRECT" label="DIRECT" /><el-option value="REJECT" label="REJECT" /><el-option v-for="name in names.filter((name) => name !== group.name)" :key="name" :value="name" :label="name" /></el-select><el-button type="danger" link @click="group.proxies?.splice(proxyIndex, 1)">移除</el-button></div><el-button size="small" plain @click="(group.proxies ||= []).push('')">添加成员</el-button></div></div></article></div></el-scrollbar></template><template v-else><div class="panel-title">YAML 代码</div><YamlEditor v-model="yamlText" /></template></section>
        <section class="panel preview"><div class="panel-title"><span>代理组配置预览</span><div class="preview-actions"><el-button link title="复制预览" @click="copyPreview">复制</el-button><el-button link title="刷新预览" @click="refreshPreview">刷新</el-button></div></div><pre>{{ previewYaml }}</pre></section>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.manager-list{min-height:360px}.intro{margin:0 0 16px;color:var(--el-text-color-secondary)}.create-tabs{margin-bottom:18px}.create-option{min-height:120px}.wide{width:100%}.subscription-input{display:flex;width:100%;gap:10px}.subscription-input .el-input{flex:1}.editor-title,.actions,.toolbar,.group-head,.switches,.proxy,.preview-actions{display:flex;align-items:center;gap:10px}.editor-title{justify-content:space-between;padding-right:32px}.editor-title h2{margin:0;font-size:19px}.editor-title p{margin:4px 0 0;color:var(--el-text-color-secondary);font-size:13px}.toolbar{padding-bottom:14px;border-bottom:1px solid var(--el-border-color-lighter)}.filename{max-width:440px}.parse-error{margin:14px 0}.editor-layout{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:18px;min-height:0;flex:1}.panel{display:flex;min-width:0;min-height:0;flex-direction:column;overflow:hidden;border:1px solid var(--el-border-color);border-radius:4px;background:var(--el-bg-color)}.editor-tabs{padding:0;background:var(--el-fill-color-light)}.editor-tabs :deep(.el-radio-group){display:grid;width:100%;grid-template-columns:1fr 1fr}.editor-tabs :deep(.el-radio-button__inner){width:100%;height:44px;border:0!important;border-radius:0!important;background:transparent;box-shadow:none!important;font-size:15px;font-weight:700;line-height:42px}.editor-tabs :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner){color:var(--el-color-primary);background:var(--el-bg-color);box-shadow:inset 0 -3px 0 var(--el-color-primary)!important}.panel-title{display:flex;min-height:52px;align-items:center;justify-content:space-between;padding:0 18px;border-bottom:1px solid var(--el-border-color-lighter);background:var(--el-bg-color);font-size:15px;font-weight:700}.panel-title em{display:inline-flex;align-items:center;justify-content:center;min-width:20px;height:20px;margin-left:5px;border-radius:10px;color:var(--el-color-primary);background:var(--el-color-primary-light-9);font-size:12px;font-style:normal}.scroll{min-height:0;flex:1}.groups{display:grid;gap:12px;padding:14px}.group{overflow:hidden;border:1px solid var(--el-border-color);border-radius:4px;transition:border-color .18s}.group.expanded{border-color:var(--el-color-primary-light-5)}.group-head{justify-content:space-between;min-height:54px;padding:0 11px;background:var(--el-fill-color-light)}.group-toggle{display:flex;min-width:0;flex:1;align-items:center;gap:9px;padding:0;border:0;color:var(--el-text-color-primary);background:transparent;text-align:left;cursor:pointer}.group-toggle strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:15px}.group-toggle small{flex:none;padding:3px 6px;border-radius:2px;color:var(--el-text-color-secondary);background:var(--el-fill-color);font-size:11px}.drag{color:var(--el-text-color-placeholder);font-size:19px;letter-spacing:-3px}.chevron{margin-left:auto;color:var(--el-text-color-secondary);font-size:19px;line-height:1;transition:transform .18s}.expanded .chevron{transform:rotate(180deg)}.group-actions{display:flex;gap:5px;flex:none}.fields{display:grid;gap:16px;padding:16px;background:var(--el-bg-color)}.fields label,.proxies>label{display:grid;gap:7px;color:var(--el-text-color-regular);font-size:13px;font-weight:650}.field-grid{display:grid;grid-template-columns:1fr 1fr;gap:14px}.field-grid :deep(.el-select),.speed :deep(.el-input-number){width:100%}.switches{gap:20px;padding:2px 0}.switches :deep(.el-switch){--el-switch-on-color:var(--el-color-primary);height:26px}.switches :deep(.el-switch__core){min-width:48px;height:26px;border-radius:14px}.switches :deep(.el-switch__action){width:20px;height:20px}.speed{display:grid;grid-template-columns:minmax(0,1fr) 120px 120px;gap:14px}.proxies{display:grid;gap:8px;padding-top:2px}.proxy .el-select{min-width:0;flex:1}.fields :deep(.el-input__wrapper),.fields :deep(.el-select__wrapper),.fields :deep(.el-input-number .el-input__wrapper){min-height:40px}.preview pre{min-height:0;flex:1;overflow:auto;box-sizing:border-box;margin:0;padding:16px;color:var(--el-text-color-primary);font:12px/1.6 Consolas,monospace;white-space:pre}.editor-panel :deep(.yaml-editor){min-height:0;flex:1}@media(min-width:901px){:deep(.editor-dialog){display:flex;height:calc(100dvh - 48px);flex-direction:column;margin-bottom:0}:deep(.editor-dialog .el-dialog__body){display:flex;min-height:0;flex:1;flex-direction:column;overflow:hidden}}@media(max-width:900px){.editor-title,.toolbar{align-items:stretch;flex-direction:column}.actions{justify-content:flex-end}.filename{max-width:none}.editor-layout{grid-template-columns:1fr;min-height:0;flex:none}.panel{min-height:420px}.field-grid,.speed{grid-template-columns:1fr}.subscription-input{flex-direction:column}:deep(.editor-dialog){width:min(96vw,720px)!important}.editor-title{padding-right:28px}}
</style>
