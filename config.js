// 全局网络请求与环境配置
const CONFIG = {
  // 后端服务基础 URL (开发环境支持本地 localhost:8080 或虚拟机局域网 IP)
  API_BASE_URL: 'http://127.0.0.1:8080',

  // 环境数据源开关:
  // false: 全面连接 Go 后端服务与 PostgreSQL 数据库
  // true:  使用纯前端内嵌 Mock 数据与本地 Storage (断网或后端未启动时可随时切回自测)
  USE_MOCK: false,

  // 开发调试用户标识 (用于快速免密换取 JWT Token)
  DEV_USER_ID: 'dev_user_001',

  // 开发调试多用户切换开关 (为 true 时在开发/体验环境下点击微信登录唤起角色选择)
  ENABLE_DEV_MOCK_LOGIN: true,

  // 超时时间 (毫秒)
  TIMEOUT: 15000
};

module.exports = CONFIG;
