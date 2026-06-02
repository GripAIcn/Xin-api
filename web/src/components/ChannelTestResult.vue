<script setup lang="ts">
import type { ChannelTestResult } from '@/types/api'
import { CircleCheck, CircleClose, Warning } from '@element-plus/icons-vue'

interface Props {
  result: ChannelTestResult
}

const props = defineProps<Props>()

const formatTime = (ms: number) => {
  if (ms < 1000) {
    return `${ms}ms`
  }
  return `${(ms / 1000).toFixed(2)}s`
}

const successCount = props.result.results.filter(r => r.success).length
const totalCount = props.result.results.length
</script>

<template>
  <div class="py-2">
    <!-- 渠道信息 -->
    <div class="mb-4 p-3 rounded-lg" style="background: #f5f7fa; border-left: 3px solid #409eff;">
      <div class="text-base font-semibold text-gray-800 mb-1">{{ result.channel_name }}</div>
      <div class="text-sm text-gray-500 break-all">{{ result.base_url }}</div>
    </div>

    <!-- 汇总信息 -->
    <div class="mb-4">
      <el-tag v-if="successCount === totalCount" type="success" size="small" class="inline-flex items-center gap-1">
        <el-icon><CircleCheck /></el-icon>
        全部通过 ({{ successCount }}/{{ totalCount }})
      </el-tag>
      <el-tag v-else-if="successCount === 0" type="danger" size="small" class="inline-flex items-center gap-1">
        <el-icon><CircleClose /></el-icon>
        全部失败 (0/{{ totalCount }})
      </el-tag>
      <el-tag v-else type="warning" size="small" class="inline-flex items-center gap-1">
        <el-icon><Warning /></el-icon>
        部分通过 ({{ successCount }}/{{ totalCount }})
      </el-tag>
    </div>

    <!-- 模型测试结果表格 -->
    <el-table :data="result.results" stripe size="small" class="mt-2">
      <el-table-column prop="model" label="模型" min-width="120" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.success" type="success" size="small" class="inline-flex items-center gap-1">
            <el-icon><CircleCheck /></el-icon>
            成功
          </el-tag>
          <el-tag v-else type="danger" size="small" class="inline-flex items-center gap-1">
            <el-icon><CircleClose /></el-icon>
            失败
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="响应时间" width="100">
        <template #default="{ row }">
          <span :class="row.success ? 'text-success' : 'text-danger'">
            {{ formatTime(row.response_time_ms) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="error_msg" label="错误信息" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.error_msg" class="text-danger text-sm">{{ row.error_msg }}</span>
          <span v-else class="text-gray-400">-</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.text-success {
  color: #67c23a;
}

.text-danger {
  color: #f56c6c;
}
</style>
