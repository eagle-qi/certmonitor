import request from '../request'

/** SSL 证书管理 API */
export const certApi = {
  list: (params) => request.get('/certificates', { params }),
  detail: (id) => request.get(`/certificates/${id}`),
  createManual: (data) => request.post('/certificates', data),
  update: (id, data) => request.put(`/certificates/${id}`, data),
  delete: (id) => request.delete(`/certificates/${id}`),
  collectByAsset: (assetId) => request.post(`/certificates/collect/${assetId}`),
  overviewStats: () => request.get('/certificates/stats/overview'),
  riskStats: () => request.get('/certificates/stats/risk'),
  download: (id) => request.get(`/certificates/${id}/download`, { responseType: 'blob' }),
}

/** 证书自助申请 API */
export const certApplyApi = {
  submit: (data) => request.post('/cert-apply', data),
  listTasks: (params) => request.get('/cert-apply/tasks', { params }),
  taskDetail: (id) => request.get(`/cert-apply/tasks/${id}`),
  retry: (id) => request.post(`/cert-apply/tasks/${id}/retry`),
  downloadPackage: (id) => request.get(`/cert-apply/tasks/${id}/download`, { responseType: 'blob' }),
}
