// Exam Prep - 登录页逻辑 (100% 还原 Stitch 原型)
const CONFIG = require('../../config.js');
const app = getApp();

// 开发调试环境预设的 3 个典型测试身份
const DEV_MOCK_USERS = [
  {
    name: '👨‍💼 系统管理员 (Admin)',
    role: 'admin',
    dev_user_id: 'dev_admin_001',
    nickname: '系统管理员',
    isVip: true,
    isLifetimeVip: true,
    vipExpire: '永久 VIP',
    level: 'Lv.99',
    title: '系统主控',
    avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuC6tZtmFH8sHcTBIJgE4CXv_uiZxRYuuUfO2XvYCKniiNOodibUyu7oV26hBTXqGRYGR7d_Dm0DjRcgI2DwaZb5VkZ2TEyLSXwKPzOsFN8rU_j48rtfj6CAFPx086ngO8lssh8-H8oFnt6obxKUU6QdVaANQRm-wl2cQWfsnumh2bYLfcV82PAJXC1JxZ6M0YN3smvep5qbnGCmh_D2UA3l2h_uSD_Dvn80lVoRNLchLImuDh1ffWZr',
    phone: '13800000001'
  },
  {
    name: '👑 VIP 会员 (VIP)',
    role: 'vip',
    dev_user_id: 'dev_vip_002',
    nickname: 'VIP尊享学员',
    isVip: true,
    isLifetimeVip: false,
    vipExpire: '2027-12-31',
    level: 'Lv.8',
    title: '终身研习',
    avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuC6tZtmFH8sHcTBIJgE4CXv_uiZxRYuuUfO2XvYCKniiNOodibUyu7oV26hBTXqGRYGR7d_Dm0DjRcgI2DwaZb5VkZ2TEyLSXwKPzOsFN8rU_j48rtfj6CAFPx086ngO8lssh8-H8oFnt6obxKUU6QdVaANQRm-wl2cQWfsnumh2bYLfcV82PAJXC1JxZ6M0YN3smvep5qbnGCmh_D2UA3l2h_uSD_Dvn80lVoRNLchLImuDh1ffWZr',
    phone: '13800000002'
  },
  {
    name: '👤 普通学员 (Non-VIP)',
    role: 'user',
    dev_user_id: 'dev_user_003',
    nickname: '备考新手',
    isVip: false,
    isLifetimeVip: false,
    vipExpire: '',
    level: 'Lv.1',
    title: '初级备考',
    avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuC6tZtmFH8sHcTBIJgE4CXv_uiZxRYuuUfO2XvYCKniiNOodibUyu7oV26hBTXqGRYGR7d_Dm0DjRcgI2DwaZb5VkZ2TEyLSXwKPzOsFN8rU_j48rtfj6CAFPx086ngO8lssh8-H8oFnt6obxKUU6QdVaANQRm-wl2cQWfsnumh2bYLfcV82PAJXC1JxZ6M0YN3smvep5qbnGCmh_D2UA3l2h_uSD_Dvn80lVoRNLchLImuDh1ffWZr',
    phone: '13800000003'
  }
];

// 环境判断：仅在开发版 (develop) 或体验版 (trial) 且未显式关闭开关时开启调试选择器
function isDevEnvironment() {
  if (CONFIG.ENABLE_DEV_MOCK_LOGIN === false) {
    return false;
  }
  try {
    const accountInfo = wx.getAccountInfoSync ? wx.getAccountInfoSync() : null;
    const envVersion = accountInfo && accountInfo.miniProgram && accountInfo.miniProgram.envVersion;
    // release 为正式生产版，绝对走真实微信登录流程，严格禁止弹出角色选择
    if (envVersion === 'release') {
      return false;
    }
    return true;
  } catch (e) {
    return true;
  }
}

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    isAgreed: false,          // 是否勾选用户服务协议与隐私政策
    isShaking: false,         // 未勾选协议时的抖动提示动效
    
    // 手机号登录弹窗状态
    showPhoneModal: false,
    phoneNumber: '',
    smsCode: '',
    countdown: 0,
    timer: null,

    // 协议内容展示弹窗状态
    showAgreementModal: false,
    agreementTitle: '',
    agreementContent: '',
    agreementPoints: [],

    redirectUrl: ''
  },

  onLoad(options) {
    // 适配各机型自定义导航栏高度
    try {
      const windowInfo = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
      const statusBarHeight = windowInfo.statusBarHeight || 20;
      let navBarHeight = 44;
      if (wx.getMenuButtonBoundingClientRect) {
        const menu = wx.getMenuButtonBoundingClientRect();
        if (menu && menu.top) {
          navBarHeight = (menu.top - statusBarHeight) * 2 + menu.height;
        }
      }
      this.setData({
        statusBarHeight,
        navBarHeight,
        redirectUrl: options && options.redirect ? decodeURIComponent(options.redirect) : ''
      });
    } catch (e) {
      console.log('获取系统窗口信息异常', e);
    }
  },

  onUnload() {
    if (this.data.timer) {
      clearInterval(this.data.timer);
    }
  },

  // 返回上一页或返回首页
  onNavBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack();
    } else {
      wx.redirectTo({
        url: '/pages/index/index'
      });
    }
  },

  // 切换协议勾选状态
  onToggleTerms() {
    this.setData({
      isAgreed: !this.data.isAgreed
    });
  },

  // 未勾选协议时的警告与抖动反馈
  triggerAgreementAlert() {
    this.setData({ isShaking: true });
    setTimeout(() => {
      this.setData({ isShaking: false });
    }, 600);

    wx.showToast({
      title: '请先阅读并同意用户协议及隐私政策',
      icon: 'none',
      duration: 2500
    });
  },

  // 微信一键登录
  onWeChatLogin() {
    if (!this.data.isAgreed) {
      this.triggerAgreementAlert();
      return;
    }

    // 开发/测试调试环境：唤起 3 个预设用户选择菜单
    if (isDevEnvironment()) {
      wx.showActionSheet({
        itemList: DEV_MOCK_USERS.map(u => u.name),
        success: (res) => {
          const selectedUser = DEV_MOCK_USERS[res.tapIndex];
          if (selectedUser) {
            this.executeDevMockLogin(selectedUser);
          }
        },
        fail: (err) => {
          console.log('用户取消选择测试账号', err);
        }
      });
      return;
    }

    // 生产正式环境下：强制执行真实微信登录
    this.executeProductionWeChatLogin();
  },

  // 开发环境 Mock 登录流程 (优先打通 Go 后端并落库，若未启动则降级纯离线)
  executeDevMockLogin(mockProfile) {
    wx.showLoading({ title: `正在以 ${mockProfile.nickname} 登录...` });

    if (!CONFIG.USE_MOCK) {
      wx.request({
        url: `${CONFIG.API_BASE_URL}/api/v1/auth/mock-login`,
        method: 'POST',
        data: {
          dev_user_id: mockProfile.dev_user_id,
          role: mockProfile.role,
          nickname: mockProfile.nickname
        },
        header: { 'Content-Type': 'application/json' },
        success: (res) => {
          wx.hideLoading();
          if (res.statusCode === 200 && res.data && res.data.code === 0 && res.data.data.token) {
            const backendUser = res.data.data.user || {};
            const finalUser = {
              ...mockProfile,
              ...backendUser,
              isVip: Boolean(mockProfile.isVip),
              level: mockProfile.level,
              title: mockProfile.title
            };
            this.handleLoginSuccess(res.data.data.token, finalUser);
          } else {
            this.pureLocalLogin(null, mockProfile);
          }
        },
        fail: () => {
          wx.hideLoading();
          this.pureLocalLogin(null, mockProfile);
        }
      });
      return;
    }

    wx.hideLoading();
    this.pureLocalLogin(null, mockProfile);
  },

  // 生产环境真实微信授权流程
  executeProductionWeChatLogin() {
    wx.showLoading({ title: '正在授权微信登录...' });

    wx.login({
      success: (loginRes) => {
        if (loginRes.code) {
          this.requestBackendLogin(loginRes.code);
        } else {
          wx.hideLoading();
          wx.showToast({
            title: '获取微信登录凭证失败',
            icon: 'none'
          });
        }
      },
      fail: () => {
        wx.hideLoading();
        wx.showToast({
          title: '微信登录调起失败',
          icon: 'none'
        });
      }
    });
  },

  // 向 Go 后端发起微信 code2session 真实授权验证
  requestBackendLogin(code) {
    wx.request({
      url: `${CONFIG.API_BASE_URL}/api/v1/auth/wechat-login`,
      method: 'POST',
      data: { code },
      header: { 'Content-Type': 'application/json' },
      success: (res) => {
        wx.hideLoading();
        if (res.statusCode === 200 && res.data && res.data.code === 0 && res.data.data.token) {
          const user = res.data.data.user || {};
          const isVip = user.role === 'vip' || user.role === 'admin';
          const userProfile = {
            ...user,
            isVip: isVip,
            level: user.role === 'admin' ? 'Lv.99' : (isVip ? 'Lv.5' : 'Lv.1'),
            title: user.role === 'admin' ? '系统管理员' : (isVip ? 'VIP研习官' : '备考学员')
          };
          this.handleLoginSuccess(res.data.data.token, userProfile);
        } else {
          wx.showToast({
            title: res.data && res.data.message ? res.data.message : '登录授权失败',
            icon: 'none'
          });
        }
      },
      fail: () => {
        wx.hideLoading();
        wx.showToast({
          title: '网络连接异常，请重试',
          icon: 'none'
        });
      }
    });
  },

  // 纯离线/本地环境登录数据组装
  pureLocalLogin(customPhone, customMockProfile) {
    const fakeToken = 'mock_token_' + Date.now();
    const mock = customMockProfile || DEV_MOCK_USERS[1]; // 默认 VIP 用户
    const randomSuffix = Math.random().toString(36).substring(2, 10);
    const defaultRandomNickname = `用户_${randomSuffix}`;
    const fakeUser = {
      id: mock.dev_user_id || 1,
      openid: `mock_${mock.dev_user_id || Date.now()}`,
      nickname: customPhone ? `用户_${customPhone.slice(-4)}` : (mock.nickname || defaultRandomNickname),
      avatarUrl: mock.avatarUrl,
      phone: customPhone || mock.phone,
      level: mock.level,
      title: mock.title,
      role: mock.role,
      isVip: Boolean(mock.isVip)
    };
    this.handleLoginSuccess(fakeToken, fakeUser);
  },

  // 统一登录成功后置处理
  handleLoginSuccess(token, user) {
    const isLifetime = Boolean(user.isLifetimeVip || user.role === 'admin');
    wx.setStorageSync('auth_token', token);
    wx.setStorageSync('user_info', {
      ...user,
      isLifetimeVip: isLifetime,
      vipExpire: isLifetime ? '永久' : (user.vipExpire || '2027-12-31')
    });
    wx.setStorageSync('user_is_vip', Boolean(user.isVip || user.role === 'admin'));
    wx.setStorageSync('user_is_lifetime_vip', isLifetime);
    if (isLifetime) {
      wx.setStorageSync('vip_expire_date', '永久');
    } else if (user.isVip) {
      wx.setStorageSync('vip_expire_date', user.vipExpire || '2027-12-31');
    } else {
      wx.removeStorageSync('vip_expire_date');
    }
    if (app && app.globalData) {
      app.globalData.userInfo = user;
    }

    wx.showToast({
      title: `${user.nickname} 登录成功`,
      icon: 'success',
      duration: 1500
    });

    setTimeout(() => {
      if (this.data.redirectUrl) {
        wx.redirectTo({
          url: this.data.redirectUrl,
          fail: () => {
            wx.redirectTo({ url: '/pages/index/index' });
          }
        });
      } else {
        const pages = getCurrentPages();
        const prevPage = pages.length > 1 ? pages[pages.length - 2] : null;
        if (prevPage && prevPage.route && prevPage.route.includes('profile')) {
          wx.navigateBack();
        } else if (pages.length > 1) {
          wx.navigateBack();
        } else {
          wx.redirectTo({ url: '/pages/index/index' });
        }
      }
    }, 1500);
  },

  // 打开手机号登录弹窗
  onOpenPhoneModal() {
    if (!this.data.isAgreed) {
      this.triggerAgreementAlert();
      return;
    }
    this.setData({
      showPhoneModal: true
    });
  },

  onClosePhoneModal() {
    this.setData({
      showPhoneModal: false
    });
  },

  onPhoneInput(e) {
    this.setData({
      phoneNumber: e.detail.value.trim()
    });
  },

  onCodeInput(e) {
    this.setData({
      smsCode: e.detail.value.trim()
    });
  },

  // 发送短信验证码
  onSendSmsCode() {
    const { phoneNumber, countdown } = this.data;
    if (countdown > 0) return;

    if (!phoneNumber || !/^1\d{10}$/.test(phoneNumber)) {
      wx.showToast({
        title: '请输入有效的11位手机号',
        icon: 'none'
      });
      return;
    }

    // 开始 60 秒倒计时
    this.setData({ countdown: 60 });
    const timer = setInterval(() => {
      if (this.data.countdown <= 1) {
        clearInterval(timer);
        this.setData({ countdown: 0, timer: null });
      } else {
        this.setData({ countdown: this.data.countdown - 1 });
      }
    }, 1000);
    this.setData({ timer });

    wx.showToast({
      title: '验证码已发送 (测试码: 123456)',
      icon: 'none',
      duration: 3000
    });
  },

  // 提交手机号登录
  onSubmitPhoneLogin() {
    const { phoneNumber, smsCode } = this.data;
    if (!phoneNumber || !/^1\d{10}$/.test(phoneNumber)) {
      wx.showToast({
        title: '请输入正确的11位手机号',
        icon: 'none'
      });
      return;
    }

    if (!smsCode || smsCode.length < 4) {
      wx.showToast({
        title: '请输入验证码 (测试码: 123456)',
        icon: 'none'
      });
      return;
    }

    wx.showLoading({ title: '登录验证中...' });
    setTimeout(() => {
      wx.hideLoading();
      this.setData({ showPhoneModal: false });
      this.pureLocalLogin(phoneNumber);
    }, 800);
  },

  // 打开服务协议与隐私政策详情模态窗
  onOpenAgreementModal(e) {
    const type = e.currentTarget.dataset.type;
    if (type === 'service') {
      this.setData({
        showAgreementModal: true,
        agreementTitle: '用户服务协议',
        agreementContent: '欢迎使用 Exam Prep 智学刷题助手！在您使用本小程序提供的题库浏览、智能刷题、错题本汇总、模拟考试等备考服务前，请仔细阅读本协议条款。',
        agreementPoints: [
          '服务范围：Exam Prep 为用户提供题库检索、智能练习与分析服务。',
          '账号安全：用户需妥善保管绑定的微信号与手机号信息，因个人原因泄露导致的安全后果由用户自行承担。',
          '知识产权：本应用所展示之题库解析、模拟考试算法及界面设计均归开发者合法所有，未经许可不得私自抓取或商业营利。',
          '服务变更与中止：我们保留因系统升级优化对部分功能进行调整或下线的权利。'
        ]
      });
    } else {
      this.setData({
        showAgreementModal: true,
        agreementTitle: '隐私保护政策',
        agreementContent: 'Exam Prep 非常重视您的个人隐私与数据安全。本隐私政策向您阐述我们如何收集、使用、存储及保护您的个人信息：',
        agreementPoints: [
          '信息收集：在获得您的明示授权后，我们会获取您的微信公开头像、昵称、手机号以建立学习档案。',
          '使用目的：所收集的数据仅用于为您生成题目练习记录、智能错题统计、模拟考试成绩报告及定制专属学习计划。',
          '数据保密：我们采取符合行业标准的加密传输与安全防护技术，绝不会将您的个人隐私信息出售或共享给任何第三方。',
          '权限控制：您随时可以在个人中心修改资料或向我们申请注销学习档案与数据清除。'
        ]
      });
    }
  },

  onCloseAgreementModal() {
    this.setData({
      showAgreementModal: false
    });
  },

  onAgreeInModal() {
    this.setData({
      isAgreed: true,
      showAgreementModal: false
    });
    wx.showToast({
      title: '已同意相关协议',
      icon: 'none'
    });
  },

  stopPropagation() {
    // 阻止遮罩层事件冒泡
  }
});
