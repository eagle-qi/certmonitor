import request from '../request'

/** URL 资产管理 API */
export const assetApi = {
  // 资产列表(分页+筛选)
  list: (params) => request.get('/assets', { params }),

  // 新增资产
  create: (data) => request.post('/assets', data),

  // 资产详情
  detail: (id) => request.get(`/assets/${id}`),

  // 编辑资产
  update: (id, data) => request.put(`/assets/${id}`, data),

  // 删除资产
  delete: (id) => request.delete(`/assets/${id}`),

  // 审核通过
  confirm: (id) => request.patch(`/assets/${id}/confirm`),

  // 审核驳回
  reject: (id, reason) => request.patch(`/assets/${id}/reject`, { reason }),

  // 批量导入(Excel上传)
  batchImport: (file, onProgress) => {
    const formData = new FormData()
    formData.append('file', file)
    return request.post('/assets/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: onProgress,
    })
  },

  // 下载导入模板
  downloadTemplate: () => request.get('/assets/template/download', { responseType: 'blob' }),

  // 导入错误日志
  importLog: (taskId) => request.get(`/assets/import/logs/${taskId}`),

  // 导出 Excel
  export: (params) => request.get('/assets/export', { params, responseType: 'blob' }),
}
