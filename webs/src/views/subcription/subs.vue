<script setup lang='ts'>
import { ref,onMounted  } from 'vue'
import {getSubs,AddSub,DelSub,UpdateSub, ResetSubToken, SetSubRevoked} from "@/api/subcription/subs"
import {getTemp} from "@/api/subcription/temp"
import {getNodes} from "@/api/subcription/node"
import QrcodeVue from 'qrcode.vue'
import { VueDraggable } from 'vue-draggable-plus'

interface Sub {
  ID: number;
  Name: string;
  CreateDate: string;
  Config: Config;
  Nodes: Node[];
  SubLogs:SubLogs[];
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
  CreateDate: string;
}
interface Config {
  clash: string;
  surge:string;
  udp: string;
  cert: string;
}
interface SubLogs {
  date: string;
  name: string;
  count: number;
  address: string;
}
interface Temp {
  file: string;
  text: string;
  CreateDate: string;
}
const tableData = ref<Sub[]>([])
const Clash = ref('')
const Surge = ref('')
const SubTitle = ref('')
const Subname = ref('')
const oldSubname = ref('')
const expireAt = ref('')
const accessLimit = ref<number | undefined>()
const dialogVisible = ref(false)
const table = ref()
const NodesList = ref<Node[]>([])
const value1 = ref<string[]>([])
const checkList = ref<string[]>([]) // 配置列表
const iplogsdialog = ref(false)
const IplogsList = ref<SubLogs[]>([])
const qrcode = ref('')
const templist = ref<Temp[]>([])
async function getsubs() {
  const {data} = await getSubs();
    tableData.value = data
}
async function gettemps() {
    const {data} = await getTemp();
    templist.value = data
    console.log(templist.value);
}
onMounted(() => {
    getsubs()
    gettemps()
})
onMounted(async() => {
    const {data} = await getNodes();
    NodesList.value = data
})


const addSubs = async ()=>{
    const config = JSON.stringify({
    "clash": Clash.value.trim(),
    "surge": Surge.value.trim(),
    "udp": checkList.value.includes('udp') ? true :  false,
    "cert": checkList.value.includes('cert') ? true :  false

  })
  if (SubTitle.value === '添加订阅') {
    await AddSub({
      config: config,
      name: Subname.value.trim(),
      nodes: value1.value.join(','),
      expire_at: expireAt.value || '',
      access_limit: accessLimit.value || ''
    })
    getsubs()
    ElMessage.success("添加成功");
  }else{
    await UpdateSub({
      config: config,
      name: Subname.value.trim(),
      nodes: value1.value.join(','),
      oldname: oldSubname.value,
      expire_at: expireAt.value || '',
      access_limit: accessLimit.value || ''
    })
    getsubs()
    ElMessage.success("更新成功");
  }

    dialogVisible.value = false;
}

const multipleSelection = ref<Sub[]>([])
const handleSelectionChange = (val: Sub[]) => {
  multipleSelection.value = val
  
}
const selectAll = () => {
  tableData.value.forEach(row => {
            table.value.toggleRowSelection(row, true)
        })
}
// IP记录
const handleIplogs = (row: any) => {
  iplogsdialog.value = true
  nextTick(() => {
    tableData.value.forEach((item) => {
    if (item.ID === row.ID) {
      IplogsList.value = item.SubLogs
    }
  })
  
  })
}

const toggleSelection = () => {
  table.value.clearSelection()
}

const handleAddSub = ()=>{
  SubTitle.value = '添加订阅'
  Subname.value = ''
  oldSubname.value = ''
  expireAt.value = ''
  accessLimit.value = undefined
  checkList.value = []
  Clash.value = './template/clash.yaml'
  Surge.value = './template/surge.conf'
  dialogVisible.value = true
  value1.value = []
}

const handleEdit = (row:any) => {
  for (let i = 0; i < tableData.value.length; i++) {
    if (tableData.value[i].ID === row.ID) {
      function toConfig(value: string | Config): Config {
        if (typeof value === 'string') {
          return JSON.parse(value) as Config;
        } else {
          return value as Config;
        }
      }
      const config = toConfig(tableData.value[i].Config);
      SubTitle.value = '编辑订阅'
      Subname.value = tableData.value[i].Name
      oldSubname.value = Subname.value
      expireAt.value = tableData.value[i].ExpireAt
        ? new Date(tableData.value[i].ExpireAt as string).toISOString().slice(0, 19).replace('T', ' ')
        : ''
      accessLimit.value = tableData.value[i].AccessLimit || undefined
      if (config.udp)  {
        checkList.value.push('udp')
      }
      if (config.cert)  {
        checkList.value.push('cert')
      }
      Clash.value = config.clash
      Surge.value = config.surge
      dialogVisible.value = true
      value1.value = tableData.value[i].Nodes.map((item) => item.Name)
    }
  }
}
const handleDel = (row:any) => {
  ElMessageBox.confirm(
    `你是否要删除 ${row.Name} ?`,
    '提示',
    {
      confirmButtonText: 'OK',
      cancelButtonText: 'Cancel',
      type: 'warning',
    }
  ).then(async () => {
      await DelSub({
        id: row.ID
      })
      getsubs()
      ElMessage({
        type: 'success',
        message: '删除成功',
      })
      
    })
}

const handleResetToken = async (row: any) => {
  await ElMessageBox.confirm(
    `重置 ${row.Name} 的订阅 token？旧随机 token 会立即失效。`,
    '重置 token',
    {
      confirmButtonText: '重置',
      cancelButtonText: '取消',
      type: 'warning',
    }
  )
  await ResetSubToken({ id: row.ID })
  await getsubs()
  ElMessage.success('token 已重置')
}

const handleToggleRevoked = async (row: any) => {
  await SetSubRevoked({ id: row.ID, revoked: !row.Revoked })
  await getsubs()
  ElMessage.success(row.Revoked ? '订阅已恢复' : '订阅已手动失效')
}

const selectDel = () => {
  if (multipleSelection.value.length === 0) {
      return
  }
  ElMessageBox.confirm(
    `你是否要删除选中这些 ?`,
    '提示',
    {
      confirmButtonText: 'OK',
      cancelButtonText: 'Cancel',
      type: 'warning',
    }
  ).then( () => {
    for (let i = 0; i < multipleSelection.value.length; i++) {
       DelSub({
        id: multipleSelection.value[i].ID
      })
        tableData.value = tableData.value.filter((item) => item.ID !== multipleSelection.value[i].ID)
      }
      ElMessage({
        type: 'success',
        message: '删除成功',
      })
      
      
    })

}
// 分页显示
const currentPage = ref(1);
const pageSize = ref(10);
const handleSizeChange = (val: number) => {
  pageSize.value = val;
  // console.log(`每页 ${val} 条`);
}

const handleCurrentChange = (val: number) => {
  currentPage.value = val;
}
// 表格数据静态化
const currentTableData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return tableData.value.slice(start, end);
});

// 复制链接
const copyUrl = (url: string) => {
  const textarea = document.createElement('textarea');
  textarea.value = url;
  document.body.appendChild(textarea);
  textarea.select();
  try {
    const successful = document.execCommand('copy');
    const msg = successful ? 'success' : 'warning';
    const message = successful ? '复制成功！' : '复制失败！';
    ElMessage({
      type: msg,
      message,
    });
  } catch (err) {
    ElMessage({
      type: 'warning',
      message: '复制失败！',
    });
  } finally {
    document.body.removeChild(textarea);
  }
};

const copyInfo = (row: any) => {
  copyUrl(row.Link)
}
const handleBase64 = (text: string) => {
  return  window.btoa(unescape(encodeURIComponent(text)));
}
const ClientDiaLog = ref(false)
const ClientList = ['v2ray','clash','surge'] // 客户端列表
const ClientUrls = ref<Record<string, string>>({})
const ClientUrl = ref('')
const ClientSubName = ref('')
const handleClient = (row: any) => {
  let serverAddress = location.protocol + '//' + location.hostname + (location.port ? ':' + location.port : '');
  ClientDiaLog.value = true
  ClientSubName.value = row.Name
  ClientUrl.value = `${serverAddress}/c/?token=${row.Token}`
  ClientList.forEach((item:string) => {
    ClientUrls.value[item]=`${serverAddress}/c/?token=${row.Token}&client=${item}`
  })
}

const Qrdialog = ref(false)
const QrTitle = ref('')
const handleQrcode = (url:string,title:string)=>{
  Qrdialog.value = true
  qrcode.value = url 
  QrTitle.value = title
}
const OpenUrl = (url:string) => {
  window.open(url)
}
const clientradio = ref('1')

// 注册拖拽函数
const toggleSelect = (name: string) => {
  const index = value1.value.indexOf(name)
  if (index === -1) {
    value1.value.push(name)
  } else {
    value1.value.splice(index, 1)
  }
}

</script>

<template>
  <div class="page-workspace">
    <el-dialog v-model="Qrdialog" class="form-dialog qr-dialog" width="400px" :title="QrTitle">
      <div class="qr-content">
        <div class="qr-frame">
          <qrcode-vue :value="qrcode" :size="196" level="H" />
        </div>
        <el-input v-model="qrcode" readonly />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="copyUrl(qrcode)">复制地址</el-button>
          <el-button type="primary" @click="OpenUrl(qrcode)">打开链接</el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="ClientDiaLog" class="form-dialog client-dialog" width="760px" title="订阅链接">
      <div class="client-dialog-head">
        <div>
          <p class="dialog-intro">直接复制完整订阅地址，二维码作为备用入口。</p>
          <strong>{{ ClientSubName }}</strong>
        </div>
        <el-button type="primary" plain @click="copyUrl(ClientUrl)">复制自动识别</el-button>
      </div>
      <div class="client-card-list">
        <article class="client-card">
          <div class="client-card-title">
            <span>自动识别</span>
            <el-tag size="small" effect="plain">推荐</el-tag>
          </div>
          <p>客户端根据请求头自动判断订阅格式。</p>
          <el-input :model-value="ClientUrl" readonly class="client-url-input" />
          <div class="client-actions">
            <el-button @click="copyUrl(ClientUrl)">复制链接</el-button>
            <el-button @click="handleQrcode(ClientUrl,'自动识别客户端')">二维码</el-button>
            <el-button link type="primary" @click="OpenUrl(ClientUrl)">打开</el-button>
          </div>
        </article>

        <article v-for="(item,index) in ClientUrls" :key="index" class="client-card">
          <div class="client-card-title">
            <span>{{ index }}</span>
            <el-tag size="small" effect="plain">{{ index }}</el-tag>
          </div>
          <p>使用 {{ index }} 专用订阅地址。</p>
          <el-input :model-value="item" readonly class="client-url-input" />
          <div class="client-actions">
            <el-button @click="copyUrl(item)">复制链接</el-button>
            <el-button @click="handleQrcode(item, String(index))">二维码</el-button>
            <el-button link type="primary" @click="OpenUrl(item)">打开</el-button>
          </div>
        </article>
      </div>
    </el-dialog>
    
    <el-dialog v-model="iplogsdialog" class="data-dialog" title="访问记录" width="min(880px, calc(100vw - 32px))">
      <el-table :data="IplogsList" style="width: 100%">
        <el-table-column prop="IP" label="Ip" />
        <el-table-column prop="Count" label="总访问次数" />
        <el-table-column prop="Addr" label="来源" />
        <el-table-column prop="Date" label="最近时间" />
      </el-table>
    </el-dialog>
    <el-dialog
      v-model="dialogVisible"
      class="form-dialog subscription-dialog"
      width="760px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <template #header>
        <div class="dialog-heading">
          <h2>{{ SubTitle }}</h2>
          <p>配置订阅名称、输出模板和节点顺序。</p>
        </div>
      </template>

      <div class="dialog-form">
        <label class="field">
          <span class="field-label">订阅名称</span>
          <el-input v-model="Subname" placeholder="输入便于识别的订阅名称" />
        </label>

        <div class="form-grid">
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

        <section class="form-section">
          <div class="section-heading">
            <strong>输出模板</strong>
            <span>为不同客户端选择本地模板或 URL</span>
          </div>

          <div class="template-field">
            <div class="template-meta">
              <strong>Clash</strong>
              <el-radio-group v-model="clientradio" class="flat-segmented" size="small">
                <el-radio-button value="1">本地模板</el-radio-button>
                <el-radio-button value="2">URL</el-radio-button>
              </el-radio-group>
            </div>
            <el-select v-if="clientradio === '1'" v-model="Clash" placeholder="选择 Clash 模板">
              <el-option v-for="template in templist" :key="template.file" :label="template.file" :value="'./template/'+template.file" />
            </el-select>
            <el-input v-else v-model="Clash" placeholder="输入 Clash 模板 URL" />
          </div>

          <div class="template-field">
            <div class="template-meta">
              <strong>Surge</strong>
              <el-radio-group v-model="clientradio" class="flat-segmented" size="small">
                <el-radio-button value="1">本地模板</el-radio-button>
                <el-radio-button value="2">URL</el-radio-button>
              </el-radio-group>
            </div>
            <el-select v-if="clientradio === '1'" v-model="Surge" placeholder="选择 Surge 模板">
              <el-option v-for="template in templist" :key="template.file" :label="template.file" :value="'./template/'+template.file" />
            </el-select>
            <el-input v-else v-model="Surge" placeholder="输入 Surge 模板 URL" />
          </div>
        </section>

        <div class="field">
          <span class="field-label">连接选项</span>
          <el-checkbox-group v-model="checkList" class="option-list">
            <el-checkbox value="udp" border>启用 UDP</el-checkbox>
            <el-checkbox value="cert" border>跳过证书验证</el-checkbox>
          </el-checkbox-group>
        </div>

        <div class="field">
          <span class="field-label">订阅节点</span>
          <el-select v-model="value1" multiple filterable placeholder="搜索并选择节点" style="width: 100%">
            <el-option
              v-for="item in NodesList"
              :key="item.Name"
              :label="item.Disabled ? `${item.Name}（已禁用）` : item.Name"
              :value="item.Name"
            />
          </el-select>
          <span class="field-help">已选择 {{ value1.length }} 个节点，可在下方拖拽调整顺序。</span>
          <VueDraggable v-if="value1.length" v-model="value1" :animation="150" ghost-class="ghost" class="selected-nodes">
            <div v-for="(nodeName, index) in value1" :key="nodeName" class="draggable-item">
              <span class="drag-handle">⋮⋮</span>
              <span class="row-number">{{ index + 1 }}</span>
              <span>{{ nodeName }}</span>
            </div>
          </VueDraggable>
          <div v-else class="empty-selection">尚未选择节点</div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="addSubs">保存订阅</el-button>
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
              <el-tag v-if="row.Revoked" type="danger" effect="plain">已失效</el-tag>
              <el-tag
                v-else-if="row.ExpireAt && new Date(row.ExpireAt).getTime() < Date.now()"
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
        <el-table-column prop="CreatedAt" label="创建时间" min-width="180" sortable />
        <el-table-column label="操作" width="280" align="right">
          <template #default="scope">
            <template v-if="scope.row.Nodes">
              <el-button link @click="handleIplogs(scope.row)">记录</el-button>
              <el-button link @click="handleResetToken(scope.row)">重置 token</el-button>
              <el-button link @click="handleToggleRevoked(scope.row)">
                {{ scope.row.Revoked ? "恢复" : "失效" }}
              </el-button>
              <el-button link type="primary" @click="handleEdit(scope.row)">编辑</el-button>
              <el-button link type="danger" @click="handleDel(scope.row)">删除</el-button>
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
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.primary-cell {
  font-weight: 550;
  color: var(--el-text-color-primary);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
/**拖拽样式 */
.draggable-item {
  display: grid;
  grid-template-columns: 18px 24px minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  min-height: 40px;
  padding: 0 12px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  cursor: grab;
}

.draggable-item:last-child {
  border-bottom: 0;
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
  font-variant-numeric: tabular-nums;
  font-size: 12px;
}

.drag-handle {
  color: var(--el-text-color-placeholder);
  letter-spacing: -3px;
}

.selected-nodes {
  max-height: 200px;
  margin-top: 10px;
  overflow: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
}

.empty-selection {
  margin-top: 10px;
  padding: 18px;
  color: var(--el-text-color-placeholder);
  background: var(--el-fill-color-lighter);
  border: 1px dashed var(--el-border-color);
  border-radius: 4px;
  font-size: 13px;
  text-align: center;
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
  background: #fff;
  border: 1px solid var(--el-border-color-lighter);
}

.dialog-intro {
  margin: 0 0 14px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
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
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
}

.client-card-title,
.client-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.client-card-title span {
  font-weight: 650;
  color: var(--el-text-color-primary);
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
  justify-content: flex-start;
  flex-wrap: wrap;
}

@media (max-width: 640px) {
  .client-dialog-head,
  .client-card-title {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
