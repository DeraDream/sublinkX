<script setup lang="ts">
import { parseDocument, stringify } from "yaml";

type PolicyGroup = { name: string; type: string; policies: string[]; [key: string]: any };
type Rule = { type: string; match?: string; policy: string; disabled?: boolean; [key: string]: any };
const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ "update:modelValue": [value: string]; error: [value: string] }>();
const root = ref<Record<string, any>>({});
const groups = ref<PolicyGroup[]>([]);
const rules = ref<Rule[]>([]);
const syncing = ref(false);
const groupTypes = ["select", "auto_test", "smart", "fallback", "load_balance", "external", "conditional"];
const ruleTypes = ["domain", "domain_suffix", "domain_keyword", "domain_regex", "domain_wildcard", "geoip", "ip_cidr", "ip_cidr6", "asn", "url_regex", "user_agent", "dest_port", "protocol", "rule_set", "default"];

function normaliseRule(value: any): Rule {
  const entry = value && typeof value === "object" ? Object.entries(value)[0] : undefined;
  if (!entry) return { type: "domain_suffix", match: "", policy: "DIRECT" };
  const [type, fields] = entry;
  return { type, ...(fields && typeof fields === "object" ? fields : {}) } as Rule;
}
function normaliseGroup(value: any): PolicyGroup {
  const entry = value && typeof value === "object" ? Object.entries(value)[0] : undefined;
  if (!entry) return { name: "", type: "select", policies: [] };
  const [type, settings] = entry;
  return { name: "", type, policies: [], ...(settings && typeof settings === "object" ? settings : {}) } as PolicyGroup;
}
function load(value: string) {
  const doc = parseDocument(value, { prettyErrors: false, strict: true });
  if (doc.errors.length) { emit("error", String(doc.errors[0].message)); return; }
  const parsed = doc.toJS();
  if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") { emit("error", "Egern 根配置必须是 YAML 对象"); return; }
  if (parsed.policy_groups !== undefined && !Array.isArray(parsed.policy_groups)) { emit("error", "policy_groups 必须是数组"); return; }
  syncing.value = true;
  root.value = parsed;
  groups.value = (parsed.policy_groups || []).map(normaliseGroup);
  rules.value = (parsed.rules || []).map(normaliseRule);
  syncing.value = false;
}
function write() {
  if (syncing.value) return;
  root.value.policy_groups = groups.value.map(({ type, policies, ...settings }) => ({
    [type]: { ...settings, ...(policies?.length ? { policies } : {}) },
  }));
  root.value.rules = rules.value.map(({ type, ...fields }) => ({ [type]: fields }));
  emit("update:modelValue", stringify(root.value, { lineWidth: 0 }));
}
function addGroup() { groups.value.push({ name: `策略组 ${groups.value.length + 1}`, type: "select", policies: ["DIRECT"] }); }
function addRule() { rules.value.push({ type: "domain_suffix", match: "", policy: groups.value[0]?.name || "DIRECT" }); }
watch(() => props.modelValue, load, { immediate: true });
watch([groups, rules], write, { deep: true });
</script>

<template>
  <section class="egern-editor">
    <header><div><strong>Egern 策略组</strong><small>独立 Egern YAML 编辑器</small></div><el-button type="primary" size="small" @click="addGroup">添加策略组</el-button></header>
    <el-scrollbar class="body"><div class="list">
      <article v-for="(group, index) in groups" :key="index" class="card">
        <div class="row"><el-input v-model="group.name" placeholder="策略组名称"/><el-select v-model="group.type"><el-option v-for="type in groupTypes" :key="type" :label="type" :value="type"/></el-select><el-button link type="danger" @click="groups.splice(index, 1)">删除</el-button></div>
        <el-select v-model="group.policies" multiple filterable allow-create default-first-option placeholder="代理、策略组、DIRECT 或 REJECT" class="wide" />
      </article>
      <el-empty v-if="!groups.length" description="暂无策略组" :image-size="58" />
      <div class="rule-header"><div><strong>Egern 规则</strong><small>规则的 policy 可指向策略组、代理、DIRECT 或 REJECT</small></div><el-button size="small" @click="addRule">添加规则</el-button></div>
      <article v-for="(rule, index) in rules" :key="index" class="card rule"><el-switch v-model="rule.disabled" :active-value="false" :inactive-value="true"/><el-select v-model="rule.type"><el-option v-for="type in ruleTypes" :key="type" :label="type" :value="type"/></el-select><el-input v-if="rule.type !== 'default'" v-model="rule.match" placeholder="匹配内容"/><el-input v-model="rule.policy" placeholder="策略"/><el-button link type="danger" @click="rules.splice(index, 1)">删除</el-button></article>
    </div></el-scrollbar>
  </section>
</template>

<style scoped>
.egern-editor{display:flex;min-height:0;height:100%;flex-direction:column;border:1px solid var(--el-border-color);border-radius:4px}.egern-editor header,.rule-header{display:flex;align-items:center;justify-content:space-between;padding:14px;border-bottom:1px solid var(--el-border-color-lighter)}small{display:block;margin-top:4px;color:var(--el-text-color-secondary);font-size:12px}.body{min-height:0;flex:1}.list{display:grid;gap:10px;padding:14px}.card,.row,.rule{display:flex;align-items:center;gap:10px}.card{flex-wrap:wrap;padding:12px;border:1px solid var(--el-border-color-lighter);border-radius:4px}.row{width:100%}.row .el-input{flex:1}.row .el-select{width:150px}.wide{width:100%}.rule{display:grid;grid-template-columns:auto 160px minmax(140px,1fr) minmax(130px,1fr) auto;width:auto}.rule-header{margin-top:8px;border:1px solid var(--el-border-color-lighter);border-radius:4px}@media(max-width:800px){.rule{grid-template-columns:auto 1fr 1fr}.rule .el-button{grid-column:3}.row .el-select{width:120px}}
</style>
