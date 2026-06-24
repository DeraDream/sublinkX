<template>
  <div class="navbar-left">
    <hamburger
      :is-active="appStore.sidebar.opened"
      @toggle-click="toggleSideBar"
    />
    <span class="page-title">{{ pageTitle }}</span>
  </div>
</template>

<script setup lang="ts">
import { useAppStore } from "@/store";

const appStore = useAppStore();
const route = useRoute();
const { t } = useI18n();
const pageTitle = computed(() => {
  const title = route.meta.title as string | undefined;
  return title ? t(`route.${title}`) : "";
});

function toggleSideBar() {
  appStore.toggleSidebar();
}
</script>

<style lang="scss" scoped>
.navbar-left {
  display: flex;
  align-items: center;
  min-width: 0;
}

.page-title {
  overflow: hidden;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
