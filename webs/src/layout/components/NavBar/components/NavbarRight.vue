<template>
  <div class="flex">
    <template v-if="!isMobile">
      <!--全屏 -->
      <div class="setting-item" @click="toggle">
        <svg-icon
          :icon-class="isFullscreen ? 'fullscreen-exit' : 'fullscreen'"
        />
      </div>

      <!-- 布局大小 -->
      <el-tooltip
        :content="$t('sizeSelect.tooltip')"
        effect="dark"
        placement="bottom"
      >
        <size-select class="setting-item" />
      </el-tooltip>

      <!-- 语言选择 -->
      <lang-select class="setting-item" />
    </template>

    <!-- 用户头像 -->
    <el-dropdown class="setting-item" trigger="click">
      <div class="flex-center h100% p10px">
        <img
          :src="userStore.user.avatar + '?imageView2/1/w/80/h/80'"
          class="rounded-full mr-10px w24px w24px"
        />
        <span>{{ userStore.user.username }}</span>
      </div>
      <template #dropdown>
          <el-dropdown-menu>
          <el-dropdown-item @click="logout">
            {{ $t("navbar.logout") }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

  </div>
</template>
<script setup lang="ts">
import {
  useAppStore,
  useUserStore,
} from "@/store";
import { DeviceEnum } from "@/enums/DeviceEnum";

const appStore = useAppStore();
const userStore = useUserStore();

const route = useRoute();
const router = useRouter();

const isMobile = computed(() => appStore.device === DeviceEnum.MOBILE);

const { isFullscreen, toggle } = useFullscreen();

/**
 * 注销
 */
function logout() {
  ElMessageBox.confirm("确定注销并退出系统吗？", "提示", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
    lockScroll: false,
  }).then(() => {
    userStore
      .logout()
      .then(() => {
        router.push(`/login?redirect=${route.fullPath}`);
      });
  });
}
</script>
<style lang="scss" scoped>
.setting-item {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 40px;
  height: $navbar-height;
  color: var(--el-text-color);
  cursor: pointer;

  &:hover {
    background: var(--el-fill-color-light);
  }
}

.dark .setting-item:hover {
  background: rgb(255 255 255 / 20%);
}
</style>
