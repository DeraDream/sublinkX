<script setup lang="ts">
import {
  getTelegramConfig,
  TelegramConfig,
  testTelegramBot,
  updateTelegramConfig,
} from "@/api/telegram";

defineOptions({
  name: "TelegramBot",
});

const loading = ref(false);
const saving = ref(false);
const testing = ref(false);

const form = reactive<TelegramConfig>({
  enabled: false,
  token: "",
  token_configured: false,
  admin_chat_ids: "",
  language: "zh-CN",
  api_base_url: "https://api.telegram.org",
  public_base_url: "https://sublink.yforward7.com",
});

const tokenPlaceholder = computed(() =>
  form.token_configured
    ? "Token 已配置，留空表示不修改"
    : "从 @BotFather 获取机器人 Token"
);

async function loadConfig() {
  loading.value = true;
  try {
    const { data } = await getTelegramConfig();
    Object.assign(form, data, { token: "" });
  } finally {
    loading.value = false;
  }
}

function validate(requireToken = false) {
  if (!form.admin_chat_ids.trim()) {
    ElMessage.warning("请输入管理员聊天 ID");
    return false;
  }
  if ((form.enabled || requireToken) && !form.token.trim() && !form.token_configured) {
    ElMessage.warning("请输入 Telegram Token");
    return false;
  }
  if (!/^https?:\/\//i.test(form.api_base_url.trim())) {
    ElMessage.warning("Telegram API 地址格式不正确");
    return false;
  }
  if (!/^https?:\/\//i.test(form.public_base_url.trim())) {
    ElMessage.warning("主控公网地址格式不正确");
    return false;
  }
  return true;
}

async function saveConfig() {
  if (form.enabled && !validate()) {
    return;
  }
  saving.value = true;
  try {
    await updateTelegramConfig(form);
    const tokenWasEntered = Boolean(form.token.trim());
    form.token = "";
    form.token_configured = form.token_configured || tokenWasEntered;
    ElMessage.success("Telegram 机器人配置已保存");
    await loadConfig();
  } finally {
    saving.value = false;
  }
}

async function sendTestMessage() {
  if (!validate(true)) {
    return;
  }
  testing.value = true;
  try {
    await testTelegramBot(form);
    ElMessage.success("测试消息已发送");
  } finally {
    testing.value = false;
  }
}

onMounted(loadConfig);
</script>

<template>
  <div v-loading="loading" class="page-workspace telegram-page">
    <div class="page-heading">
      <div>
        <h1>Telegram 机器人</h1>
        <p>通过 Telegram 管理 SublinkX 节点与订阅</p>
      </div>
      <span class="bot-status" :class="{ active: form.enabled }">
        <i />
        {{ form.enabled ? "已启用" : "未启用" }}
      </span>
    </div>

    <section class="work-surface telegram-settings">
      <div class="section-heading">
        <svg-icon icon-class="setting" size="17px" />
        <span>常规设置</span>
      </div>

      <div class="setting-row">
        <div class="setting-copy">
          <strong>启用 Telegram 机器人</strong>
          <span>启用后由 SublinkX 后台持续接收机器人消息</span>
        </div>
        <el-switch v-model="form.enabled" />
      </div>

      <div class="setting-row">
        <div class="setting-copy">
          <strong>Telegram Token</strong>
          <span>从 @BotFather 获取的机器人令牌，保存后不会明文显示</span>
        </div>
        <el-input
          v-model="form.token"
          class="setting-control"
          type="password"
          show-password
          :placeholder="tokenPlaceholder"
          autocomplete="new-password"
        />
      </div>

      <div class="setting-row">
        <div class="setting-copy">
          <strong>管理员聊天 ID</strong>
          <span>只有这些 Chat ID 可以查看和修改数据，多个 ID 使用逗号分隔</span>
        </div>
        <el-input
          v-model="form.admin_chat_ids"
          class="setting-control"
          placeholder="例如：123456789, 987654321"
        />
      </div>

      <div class="setting-row">
        <div class="setting-copy">
          <strong>机器人语言</strong>
          <span>机器人菜单和回复消息使用的语言</span>
        </div>
        <el-select v-model="form.language" class="setting-control">
          <el-option label="简体中文" value="zh-CN" />
        </el-select>
      </div>

      <div class="setting-row">
        <div class="setting-copy">
          <strong>Telegram API 服务器</strong>
          <span>留空使用 Telegram 官方 API，也可以填写自建反向代理地址</span>
        </div>
        <el-input
          v-model="form.api_base_url"
          class="setting-control"
          placeholder="https://api.telegram.org"
        />
      </div>

      <div class="setting-row">
        <div class="setting-copy">
          <strong>主控公网地址</strong>
          <span>Telegram 生成订阅链接时使用，例如 https://sublink.yforward7.com</span>
        </div>
        <el-input
          v-model="form.public_base_url"
          class="setting-control"
          placeholder="https://sublink.yforward7.com"
        />
      </div>

      <div class="settings-actions">
        <el-button :loading="testing" @click="sendTestMessage">
          发送测试消息
        </el-button>
        <el-button type="primary" :loading="saving" @click="saveConfig">
          保存配置
        </el-button>
      </div>
    </section>

    <section class="work-surface bot-capabilities">
      <div class="section-heading">
        <svg-icon icon-class="message" size="17px" />
        <span>机器人功能</span>
      </div>
      <div class="capability-grid">
        <div class="capability">
          <svg-icon icon-class="publish" size="19px" />
          <div>
            <strong>节点列表</strong>
            <span>查看节点名称、协议和所属分组</span>
          </div>
        </div>
        <div class="capability">
          <svg-icon icon-class="link" size="19px" />
          <div>
            <strong>订阅列表</strong>
            <span>查看订阅名称及包含的节点数量</span>
          </div>
        </div>
        <div class="capability">
          <svg-icon icon-class="edit" size="19px" />
          <div>
            <strong>添加节点</strong>
            <span>发送节点链接，可同时指定一个或多个分组</span>
          </div>
        </div>
        <div class="capability danger">
          <svg-icon icon-class="close" size="19px" />
          <div>
            <strong>删除节点</strong>
            <span>从机器人菜单选择节点，并进行二次确认</span>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.telegram-page {
  max-width: 1440px;
  margin: 0 auto;
}

.bot-status {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.bot-status i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
}

.bot-status.active {
  color: #15803d;
}

.bot-status.active i {
  background: #16a34a;
}

.telegram-settings,
.bot-capabilities {
  padding: 0 24px;
}

.bot-capabilities {
  margin-top: 18px;
  padding-bottom: 24px;
}

.section-heading {
  display: flex;
  gap: 9px;
  align-items: center;
  min-height: 58px;
  color: var(--el-color-primary);
  font-size: 14px;
  font-weight: 650;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.setting-row {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) minmax(360px, 48%);
  gap: 32px;
  align-items: center;
  min-height: 82px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.setting-copy strong,
.setting-copy span,
.capability strong,
.capability span {
  display: block;
}

.setting-copy strong,
.capability strong {
  margin-bottom: 6px;
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.setting-copy span,
.capability span {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.55;
}

.setting-control {
  width: 100%;
}

.settings-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  padding: 18px 20px;
}

.settings-actions .el-button + .el-button {
  margin-left: 0;
}

.capability-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0;
  padding-top: 20px;
}

.capability {
  display: flex;
  gap: 13px;
  min-width: 0;
  padding: 16px 20px;
  color: var(--el-color-primary);
  border-right: 1px solid var(--el-border-color-lighter);
}

.capability:last-child {
  border-right: 0;
}

.capability.danger {
  color: var(--el-color-danger);
}

html.dark .bot-status.active {
  color: #86efac;
}

@media (max-width: 960px) {
  .setting-row {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .capability-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .capability:nth-child(2) {
    border-right: 0;
  }
}

@media (max-width: 640px) {
  .telegram-settings,
  .bot-capabilities {
    padding-right: 14px;
    padding-left: 14px;
  }

  .setting-row {
    padding-right: 0;
    padding-left: 0;
  }

  .capability-grid {
    grid-template-columns: 1fr;
  }

  .capability {
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .capability:last-child {
    border-bottom: 0;
  }
}
</style>
