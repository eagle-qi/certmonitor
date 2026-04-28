import request from '../request'

/** 探测任务 API */
export const detectApi = {
  // 公网备案探测
  createCompanyTask: (data) => request.post('/detect/company', data),
  listCompanyTasks: (params) => request.get('/detect/company/tasks', { params }),
  companyTaskDetail: (id) => request.get(`/detect/company/tasks/${id}`),

  // 内网网段探测
  createIntranetTask: (data) => request.post('/detect/intranet', data),
  listIntranetTasks: (params) => request.get('/detect/intranet/tasks', { params }),
  intranetTaskDetail: (id) => request.get(`/detect/intranet/tasks/${id}`),
  intranetTaskDetails: (id, params) => request.get(`/detect/intranet/tasks/${id}/details`, { params }),
}
