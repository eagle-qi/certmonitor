import request from '../request'

// 消息通知 API
export const messageApi = {
  list: (params) => request.get('/messages', { params }),
  unreadCount: () => request.get('/messages/unread-count'),
  markRead: (id) => request.patch(`/messages/${id}/read`),
  markAllRead: () => request.patch('/messages/read-all'),
}
