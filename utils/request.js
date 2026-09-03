const CONFIG = require('../config.js');

/**
 * 确保获取有效的登录 Token
 */
function ensureAuthToken() {
  return new Promise((resolve, reject) => {
    const existingToken = wx.getStorageSync('auth_token');
    if (existingToken) {
      resolve(existingToken);
      return;
    }

    // 优先尝试 Mock 登录快速换发 Token
    wx.request({
      url: `${CONFIG.API_BASE_URL}/api/v1/auth/mock-login`,
      method: 'POST',
      data: {
        dev_user_id: CONFIG.DEV_USER_ID || 'dev_tester'
      },
      header: { 'Content-Type': 'application/json' },
      success: (res) => {
        if (res.statusCode === 200 && res.data && res.data.code === 0 && res.data.data.token) {
          const token = res.data.data.token;
          wx.setStorageSync('auth_token', token);
          wx.setStorageSync('user_info', res.data.data.user);
          resolve(token);
        } else {
          // 若 Mock 登录失败，尝试微信静默授权登录
          fallbackWeChatLogin(resolve, reject);
        }
      },
      fail: () => {
        fallbackWeChatLogin(resolve, reject);
      }
    });
  });
}

function fallbackWeChatLogin(resolve, reject) {
  wx.login({
    success: (loginRes) => {
      if (loginRes.code) {
        wx.request({
          url: `${CONFIG.API_BASE_URL}/api/v1/auth/wechat-login`,
          method: 'POST',
          data: { code: loginRes.code },
          success: (res) => {
            if (res.statusCode === 200 && res.data && res.data.code === 0) {
              const token = res.data.data.token;
              wx.setStorageSync('auth_token', token);
              wx.setStorageSync('user_info', res.data.data.user);
              resolve(token);
            } else {
              reject(new Error(res.data ? res.data.message : '登录授权失败'));
            }
          },
          fail: reject
        });
      } else {
        reject(new Error('微信登录获取 code 失败'));
      }
    },
    fail: reject
  });
}

/**
 * 统一网络请求封装
 */
function request(options = {}) {
  const {
    url,
    method = 'GET',
    data = {},
    header = {},
    needAuth = true
  } = options;

  const fullUrl = url.startsWith('http') ? url : `${CONFIG.API_BASE_URL}${url}`;

  return new Promise((resolve, reject) => {
    const doRequest = (token = '') => {
      const headers = {
        'Content-Type': 'application/json',
        ...header
      };

      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }

      wx.request({
        url: fullUrl,
        method: method.toUpperCase(),
        data,
        header: headers,
        timeout: CONFIG.TIMEOUT || 15000,
        success: (res) => {
          if (res.statusCode === 200) {
            if (res.data && res.data.code === 0) {
              resolve(res.data.data);
            } else {
              const errMsg = (res.data && res.data.message) || '业务请求失败';
              reject(new Error(errMsg));
            }
          } else if (res.statusCode === 401) {
            // Token 过期或失效，清空后重试一次
            wx.removeStorageSync('auth_token');
            if (needAuth) {
              ensureAuthToken().then((newToken) => {
                doRequest(newToken);
              }).catch(reject);
            } else {
              reject(new Error('登录状态已失效，请重新操作'));
            }
          } else {
            const errMsg = (res.data && res.data.message) || `服务响应异常 (${res.statusCode})`;
            reject(new Error(errMsg));
          }
        },
        fail: (err) => {
          reject(new Error(err.errMsg || '网络连接异常，请检查后端服务是否启动'));
        }
      });
    };

    if (needAuth) {
      ensureAuthToken().then(doRequest).catch(() => {
        // 允许未认证降级发出请求
        doRequest('');
      });
    } else {
      doRequest(wx.getStorageSync('auth_token') || '');
    }
  });
}

/**
 * 上传文件封装 (针对 Excel/题库导入)
 */
function uploadFile(options = {}) {
  const {
    url,
    filePath,
    name = 'file',
    formData = {}
  } = options;

  const fullUrl = url.startsWith('http') ? url : `${CONFIG.API_BASE_URL}${url}`;

  return new Promise((resolve, reject) => {
    const token = wx.getStorageSync('auth_token') || '';
    const header = {};
    if (token) {
      header['Authorization'] = `Bearer ${token}`;
    }

    wx.uploadFile({
      url: fullUrl,
      filePath,
      name,
      formData,
      header,
      success: (res) => {
        if (res.statusCode === 200) {
          try {
            const parsed = JSON.parse(res.data);
            if (parsed.code === 0) {
              resolve(parsed.data);
            } else {
              reject(new Error(parsed.message || '文件解析失败'));
            }
          } catch (e) {
            reject(new Error('解析服务器响应失败'));
          }
        } else {
          reject(new Error(`上传失败 (${res.statusCode})`));
        }
      },
      fail: (err) => {
        reject(new Error(err.errMsg || '文件上传连接失败'));
      }
    });
  });
}

module.exports = {
  request,
  uploadFile,
  ensureAuthToken,
  CONFIG
};
