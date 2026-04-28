import request from '../request'

// 系统管理 API
export const systemApi = {
  // 用户管理
  listUsers: (params) => request.get('/users', { params }),
  createUser: (data) => request.post('/users', data),
  updateUser: (id, data) => request.put(`/users/${id}`, data),
  deleteUser: (id) => request.delete(`/users/${id}`),

  // 角色管理
  listRoles: () => request.get('/roles'),
  createRole: (data) => request.post('/roles', data),
  updateRole: (id, data) => request.put(`/roles/${id}`, data),
  deleteRole: (id) => request.delete(`/roles/${id}`),

  // 系统配置
  listConfig: () => request.get('/system/config'),
  updateConfig: (key, value) => request.put(`/system/config/${key}`, { value }),

  // 操作日志
  listLogs: (params) => request.get('/logs', { params }),
  logDetail: (id) => request.get(`/logs/${id}`),
  exportLogs: (params) => request.get('/logs/export', { params, responseType: 'blob' }),

  // 告警规则
  listAlerts: () => request.get('/alerts/rules'),
  createAlert: (data) => request.post('/alerts/rules', data),
  updateAlert: (id, data) => request.put(`/alerts/rules/${id}`, data),
  deleteAlert: (id) => request.delete(`/alerts/rules/${id}`),
  toggleAlert: (id) => request.patch(`/alerts/rules/${id}/toggle`),
}
