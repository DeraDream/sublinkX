<template>
  <div class="dashboard">
    <header class="dashboard-header">
      <div>
        <p class="eyebrow">SUBLINKX</p>
        <h1>{{ greeting }}，{{ userStore.user.nickname }}</h1>
      </div>
      <span class="status">
        <i />
        服务运行中
      </span>
    </header>

    <section class="metrics">
      <article v-for="item in statisticData" :key="item.key" class="metric">
        <div class="metric-icon" :class="item.key">
          <svg-icon :icon-class="item.iconClass" size="22px" />
        </div>
        <div>
          <span>{{ item.title }}</span>
          <strong>{{ item.value }}</strong>
        </div>
      </article>
    </section>
  </div>
</template>

<script setup lang="ts">
defineOptions({
  name: "Dashboard",
  inheritAttrs: false,
});

import { useUserStore } from "@/store/modules/user";
import { getSubTotal, getNodeTotal } from "@/api/total";

const userStore = useUserStore();
const statisticData = ref([
  {
    value: 0,
    iconClass: "message",
    title: "订阅",
    key: "subscriptions",
  },
  {
    value: 0,
    iconClass: "link",
    title: "节点",
    key: "nodes",
  },
]);

const greeting = computed(() => {
  const hour = new Date().getHours();
  if (hour < 12) return "上午好";
  if (hour < 18) return "下午好";
  return "晚上好";
});

onMounted(async () => {
  const [subscriptions, nodes] = await Promise.all([
    getSubTotal(),
    getNodeTotal(),
  ]);
  statisticData.value[0].value = subscriptions.data;
  statisticData.value[1].value = nodes.data;
});
</script>

<style lang="scss" scoped>
.dashboard {
  padding: 32px;
}

.dashboard-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.eyebrow {
  margin: 0 0 8px;
  font-size: 11px;
  font-weight: 700;
  color: var(--el-color-primary);
  letter-spacing: 0.12em;
}

h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 650;
  letter-spacing: 0;
}

.status {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.status i {
  width: 8px;
  height: 8px;
  background: #16a34a;
  border-radius: 50%;
}

.metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 260px));
  gap: 16px;
  margin-top: 24px;
}

.metric {
  display: flex;
  gap: 16px;
  align-items: center;
  min-height: 112px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
}

.metric-icon {
  display: grid;
  width: 44px;
  height: 44px;
  color: #1d4ed8;
  background: #e8f0ff;
  place-items: center;
}

.metric-icon.nodes {
  color: #047857;
  background: #e5f7ef;
}

.metric span,
.metric strong {
  display: block;
}

.metric span {
  margin-bottom: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.metric strong {
  font-size: 28px;
  font-weight: 650;
}

@media (max-width: 640px) {
  .dashboard {
    padding: 20px 14px;
  }

  .dashboard-header {
    align-items: flex-start;
  }

  .status {
    display: none;
  }

  .metrics {
    grid-template-columns: 1fr;
  }
}
</style>
