<template>
  <div class="logo-container">
    <transition enter-active-class="animate__animated animate__fadeInLeft">
      <router-link v-if="collapse" class="wh-full flex-center" to="/">
        <img v-if="settingsStore.sidebarLogo" :src="logo" class="logo-image" />
        <button v-if="updateInfo?.update_available" class="update-badge collapsed" title="发现新版本" @click.prevent.stop="confirmUpdate">↑</button>
      </router-link>

      <router-link v-else class="wh-full flex-center" to="/">
        <img v-if="settingsStore.sidebarLogo" :src="logo" class="logo-image" />
        <span class="logo-title"> {{ defaultSettings.title }}</span>
        <button v-if="updateInfo?.update_available" class="update-badge" @click.prevent.stop="confirmUpdate">↑ 新版本</button>
      </router-link>
    </transition>
  </div>
</template>

<script lang="ts" setup>
import defaultSettings from "@/settings";
import { useSettingsStore } from "@/store";
import { checkSystemUpdate, startSystemUpdate, type SystemUpdateInfo } from "@/api/system/update";

const settingsStore = useSettingsStore();

defineProps({
  collapse: {
    type: Boolean,
    required: true,
  },
});

const logo = ref(new URL(`../../../../assets/logo.png`, import.meta.url).href);
const updateInfo = ref<SystemUpdateInfo>();
const updating = ref(false);

async function loadUpdateInfo() {
  try {
    const { data } = await checkSystemUpdate();
    updateInfo.value = data;
  } catch {
    // A version check must never prevent navigation from rendering.
  }
}

async function confirmUpdate() {
  if (!updateInfo.value?.update_available || updating.value) return;
  try {
    await ElMessageBox.confirm(
      `发现新版本 ${updateInfo.value.latest_version}，是否立即升级？服务会短暂重启。`,
      "新版本可用",
      { type: "warning", confirmButtonText: "立即升级", cancelButtonText: "暂不升级" }
    );
  } catch {
    return;
  }
  updating.value = true;
  try {
    await startSystemUpdate();
    ElMessage.success("已开始升级，服务正在重启");
  } catch {
    updating.value = false;
  }
}

onMounted(loadUpdateInfo);
</script>

<style lang="scss" scoped>
.logo-container {
	position: relative;
  width: 100%;
  height: $navbar-height;
  background-color: $sidebar-logo-background;
  border-bottom: 1px solid var(--el-border-color-lighter);

  .logo-image {
    width: 28px;
    height: 28px;
  }

  .logo-title {
    flex-shrink: 0;
    margin-left: 10px;
    font-size: 14px;
    font-weight: 650;
    color: var(--el-text-color-primary);
  }

  .update-badge {
    margin-left: auto;
    margin-right: 10px;
    padding: 3px 7px;
    border: 0;
    border-radius: 10px;
    color: #fff;
    background: var(--el-color-danger);
    font-size: 11px;
    font-weight: 650;
    cursor: pointer;

    &.collapsed { position: absolute; top: 4px; right: 2px; margin: 0; padding: 1px 5px; }
  }
}
</style>
