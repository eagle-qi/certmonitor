import request from '../request'

/** 用户认证相关 API */
export const authApi = {
  // 登录
  login: (data) => request.post('/auth/login', data),

  // 邮箱注册
  register: (data) => request.post('/auth/register', data),

  // 发送验证码
  sendCaptcha: (data) => request.post('/auth/send-captcha', data),

  // SSO 登录入口
  ssoLogin: () => request.get('/auth/sso/login'),

  // SSO 回调
  ssoCallback: (params) => request.get('/auth/sso/callback', { params }),
}
