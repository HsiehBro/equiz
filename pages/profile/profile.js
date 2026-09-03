const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    currentTab: 'profile',
    isDarkMode: false,
    userInfo: {
      avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuC6tZtmFH8sHcTBIJgE4CXv_uiZxRYuuUfO2XvYCKniiNOodibUyu7oV26hBTXqGRYGR7d_Dm0DjRcgI2DwaZb5VkZ2TEyLSXwKPzOsFN8rU_j48rtfj6CAFPx086ngO8lssh8-H8oFnt6obxKUU6QdVaANQRm-wl2cQWfsnumh2bYLfcV82PAJXC1JxZ6M0YN3smvep5qbnGCmh_D2UA3l2h_uSD_Dvn80lVoRNLchLImuDh1ffWZr',
      nickname: '学习者_8829',
      level: 'Lv.5',
      title: '备考达人',
      isVip: true
    },
    stats: {
      checkInDays: 42,
      totalQuestions: 1280,
      accuracyRate: 85
    },
    notesCount: 0,
    motivationTitle: '今日目标未完成',
    motivationDesc: '保持手感，坚持练习，考试必过！'
  },

  onLoad() {
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
        navBarHeight
      });
    } catch (e) {
      console.log('获取导航栏信息异常', e);
    }

    this.loadUserProfile();
    this.loadActivePlan();
    this.loadNotesCount();
  },

  onNavBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack();
    } else {
      wx.reLaunch({
        url: '/pages/index/index'
      });
    }
  },

  onToggleTheme() {
    const nextMode = !this.data.isDarkMode;
    this.setData({
      isDarkMode: nextMode
    });
    wx.showToast({
      title: nextMode ? '已切换至深色模式' : '已切换至浅色模式',
      icon: 'none'
    });
  },

  onEditProfile() {
    wx.showModal({
      title: '修改昵称',
      editable: true,
      placeholderText: '请输入新的昵称',
      content: this.data.userInfo.nickname,
      success: (res) => {
        if (res.confirm && res.content && res.content.trim()) {
          this.setData({
            'userInfo.nickname': res.content.trim()
          });
          wx.showToast({ title: '昵称已更新', icon: 'success' });
        }
      }
    });
  },

  onShow() {
    try {
      const isVip = wx.getStorageSync('user_is_vip');
      if (typeof isVip === 'boolean') {
        this.setData({
          'userInfo.isVip': isVip
        });
      }
    } catch (e) {
      console.log('读取个人中心缓存数据异常', e);
    }

    this.loadUserProfile();
    this.loadActivePlan();
    this.loadNotesCount();
  },

  loadUserProfile() {
    const cachedUser = wx.getStorageSync('user_info');
    if (cachedUser) {
      const updates = {};
      if (cachedUser.nickname) updates['userInfo.nickname'] = cachedUser.nickname;
      if (cachedUser.avatar_url || cachedUser.avatarUrl) {
        updates['userInfo.avatarUrl'] = cachedUser.avatar_url || cachedUser.avatarUrl;
      }
      if (cachedUser.level) updates['userInfo.level'] = cachedUser.level;
      if (cachedUser.title) updates['userInfo.title'] = cachedUser.title;
      if (cachedUser.role) updates['userInfo.role'] = cachedUser.role;
      if (typeof cachedUser.isVip === 'boolean') updates['userInfo.isVip'] = cachedUser.isVip;
      if (Object.keys(updates).length > 0) {
        this.setData(updates);
      }
    }
  },

  loadActivePlan() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/plans/active' })
        .then((plan) => {
          if (plan && plan.id) {
            const goal = plan.daily_goal || 30;
            const today = plan.today_count || 0;
            if (plan.is_today_goal_reached) {
              this.setData({
                motivationTitle: '今日目标已达成 🎉',
                motivationDesc: `今日已完成 ${today} 题，超额完成每日 ${goal} 题目标，保持手感！`
              });
            } else {
              const remaining = Math.max(0, goal - today);
              this.setData({
                motivationTitle: '今日目标未完成',
                motivationDesc: `今日已完成 ${today} 题，再做 ${remaining} 题即可达成目标，考试必过！`
              });
            }
            return;
          }
          this.loadLocalPlanMotivation();
        })
        .catch(() => {
          this.loadLocalPlanMotivation();
        });
      return;
    }

    this.loadLocalPlanMotivation();
  },

  loadLocalPlanMotivation() {
    try {
      const plan = wx.getStorageSync('user_study_plan');
      if (plan && plan.dailyGoal) {
        this.setData({
          motivationTitle: '今日目标进行中',
          motivationDesc: `今日目标 ${plan.dailyGoal} 题，预计 ${plan.estimatedDate || '近期'} 学完！`
        });
      }
    } catch (e) {}
  },

  loadNotesCount() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/notes' })
        .then((res) => {
          const count = res && typeof res.total === 'number' ? res.total : (res && res.list ? res.list.length : 0);
          this.setData({ notesCount: count });
        })
        .catch(() => {
          const notes = wx.getStorageSync('user_study_notes_list') || [];
          this.setData({ notesCount: notes.length });
        });
      return;
    }

    const notes = wx.getStorageSync('user_study_notes_list') || [];
    this.setData({ notesCount: notes.length });
  },

  onOpenVip() {
    wx.navigateTo({
      url: '/pages/vip/vip'
    });
  },

  onTapMenu(e) {
    const key = e.currentTarget.dataset.key;
    if (key === 'mockExam') {
      wx.navigateTo({
        url: '/pages/mock-exam/mock-exam'
      });
      return;
    }
    if (key === 'planning') {
      wx.navigateTo({
        url: '/pages/planning/planning'
      });
      return;
    }
    if (key === 'bookmark') {
      wx.navigateTo({
        url: '/pages/favorite/favorite'
      });
      return;
    }
    if (key === 'notes') {
      wx.navigateTo({
        url: '/pages/notes/notes'
      });
      return;
    }
    if (key === 'login') {
      wx.navigateTo({
        url: '/pages/login/login'
      });
      return;
    }

    const menuMap = {
      settings: '帮助与设置：常见问题、关于我们与版本号 v1.0.0'
    };
    
    wx.showToast({
      title: menuMap[key] || '正在前往...',
      icon: 'none',
      duration: 2000
    });
  },

  onGoQuiz() {
    wx.navigateTo({
      url: '/pages/quiz/quiz'
    });
  },

  onSwitchTab(e) {
    const tab = e.currentTarget.dataset.tab;
    if (tab === 'library') {
      wx.redirectTo({
        url: '/pages/index/index'
      });
    } else if (tab === 'errors') {
      wx.redirectTo({
        url: '/pages/errors/errors'
      });
    }
  }
});
