<script setup lang='ts'>
import { ref,onMounted,nextTick  } from 'vue'
import {getTemp,AddTemp,UpdateTemp,DelTemp} from "@/api/subcription/temp"
interface Temp {
  file: string;
  text: string;
  CreateDate: string;
}
const tableData = ref<Temp[]>([])
const Tempoldname = ref('')
const Tempname = ref('')
const TempText = ref('')
const dialogVisible = ref(false)
const table = ref()
const TempTitle = ref('')
const radio1 = ref('1')

async function gettemps() {
  const {data} = await getTemp();
    tableData.value = data
}
onMounted(async() => {
  gettemps()
})
const handleAddTemp = ()=>{
  TempTitle.value= '添加模版'
  Tempname.value = ''
  TempText.value = ''
  radio1.value = '1'
  dialogVisible.value = true
}
const addtemp = async ()=>{
   if (TempTitle.value== '添加模版'){
        await AddTemp({
        filename: Tempname.value.trim(),
        text: TempText.value.trim(),
      })
      ElMessage.success("添加成功");
   }else{
    await UpdateTemp({
        filename: Tempname.value.trim(),
        oldname: Tempoldname.value.trim(),
        text: TempText.value.trim(),
      })
    ElMessage.success("更新成功");
   }
    gettemps()
    Tempname.value = ''
    TempText.value = ''
    dialogVisible.value = false;
}

const multipleSelection = ref<Temp[]>([])
const handleSelectionChange = (val: Temp[]) => {
  multipleSelection.value = val
  
}
const selectAll = () => {
    nextTick(() => {
        tableData.value.forEach(row => {
            table.value.toggleRowSelection(row, true)
        })
    })
}
const handleEdit = (row:any) => {
  for (let i = 0; i < tableData.value.length; i++) {
    if (tableData.value[i].file === row.file) {
      TempTitle.value= '编辑模版'
      Tempname.value = tableData.value[i].file
      Tempoldname.value = Tempname.value
      TempText.value = tableData.value[i].text
      dialogVisible.value = true
      // value1.value = tableData.value[i].Nodes.map((item) => item.Name)
    }
  }
}
const toggleSelection = () => {
  table.value.clearSelection()
}

const handleDel = (row:any) => {
  ElMessageBox.confirm(
    `你是否要删除 ${row.file} ?`,
    '提示',
    {
      confirmButtonText: 'OK',
      cancelButtonText: 'Cancel',
      type: 'warning',
    }
  ).then(async () => {
      await DelTemp({
        filename: row.file,
        type: row.type
      })
      ElMessage({
        type: 'success',
        message: '删除成功',
      })
      gettemps()      
    })
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
       DelTemp({
        filename: multipleSelection.value[i].file,
      })
        tableData.value = tableData.value.filter((item) => item.file !== multipleSelection.value[i].file)
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
  // console.log(`当前页: ${val}`);
}
// 表格数据静态化
const currentTableData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return tableData.value.slice(start, end);
});

</script>

<template>
  <div class="page-workspace">
    <el-dialog
    v-model="dialogVisible"
    :title="TempTitle"
    width="80%"
  >
  <el-input 
  v-model="TempText" 
  placeholder="模版内容" 
  :rows="10" 
  type="textarea" 
  style="margin-bottom: 10px"
  />
  <el-input v-model="Tempname" placeholder="模版文件名"/>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="dialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="addtemp">确定</el-button>
      </div>
    </template>
  </el-dialog>
    <div class="page-heading">
      <div>
        <h1>模板列表</h1>
        <p>维护 Clash、Surge 等订阅输出模板</p>
      </div>
      <el-button type="primary" @click="handleAddTemp">添加模板</el-button>
    </div>

    <section class="work-surface">
      <div class="table-toolbar">
        <span class="record-count">共 {{ tableData.length }} 个模板</span>
      </div>

      <el-table
        ref="table"
        :data="currentTableData"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" fixed width="48" />
        <el-table-column prop="file" label="模板文件名" min-width="320">
          <template #default="scope">
            <span class="primary-cell">{{ scope.row.file }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="create_date" label="创建时间" min-width="200" sortable />
        <el-table-column fixed="right" label="操作" width="140" align="right">
          <template #default="scope">
            <el-button link type="primary" @click="handleEdit(scope.row)">编辑</el-button>
            <el-button link type="danger" @click="handleDel(scope.row)">删除</el-button>
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
.el-input{
  margin-bottom: 10px;
}

.record-count {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.primary-cell {
  font-weight: 550;
}
</style>@/api/subcription/temp
