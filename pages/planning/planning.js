const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,

    // 题库列表
    banks: [
      {
        id: '1',
        title: '项目管理基础考试 (共 100 题)',
        totalQuestions: 100
      },
      {
        id: '2',
        title: '2023年护士执业资格考试 (共 50 题)',
        totalQuestions: 50
      },
      {
        id: '3',
        title: '初级会计实务 - 核心考点 (共 50 题)',
        totalQuestions: 50
      }
    ],
    selectedBankIndex: 0,

    // 每日目标与计算数据
    dailyGoal: 30,
    daysNeeded: 4,
    estimatedDateText: '',

    // 提醒设置
    appReminder: true,
    wechatReminder: false
  },

  onLoad() {
    this.initNavBar();
    this.loadBanks();
    this.loadSavedPlan();
  },

  // 初始化顶部自定义导航栏高度
  initNavBar() {
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
  },

  // 动态加载题库列表
  loadBanks() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/banks' })
        .then((res) => {
          const list = res && res.list ? res.list : (Array.isArray(res) ? res : []);
          if (list.length > 0) {
            const mapped = list.map(b => ({
              id: String(b.id),
              title: `${b.title} (共 ${b.total_count || 100} 题)`,
              totalQuestions: b.total_count || 100
            }));
            this.setData({ banks: mapped }, () => {
              this.loadSavedPlan();
            });
          }
        })
        .catch((err) => {
          console.log('[Planning] 获取后端题库失败，保留默认题库:', err);
        });
    }
  },

  // 读取已保存的规划配置（优先拉取后端活跃规划，并支持本地降级）
  loadSavedPlan() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/plans/active' })
        .then((activePlan) => {
          if (activePlan && activePlan.id) {
            const bankIdx = this.data.banks.findIndex(b => String(b.id) === String(activePlan.bank_id));
            const selectedIdx = bankIdx >= 0 ? bankIdx : 0;
            const goal = activePlan.daily_goal || 30;
            this.setData({
              selectedBankIndex: selectedIdx,
              dailyGoal: goal,
              appReminder: typeof activePlan.app_reminder === 'boolean' ? activePlan.app_reminder : true,
              wechatReminder: typeof activePlan.wechat_reminder === 'boolean' ? activePlan.wechat_reminder : false,
              daysNeeded: activePlan.days_needed || 4,
              estimatedDateText: activePlan.estimated_finish_date || ''
            }, () => {
              if (!this.data.estimatedDateText) {
                this.calculateEstimatedDate(goal);
              }
            });
            return;
          }
          this.loadLocalPlanFallback();
        })
        .catch((err) => {
          console.log('[Planning] 获取服务端主规划失败，回退本地缓存:', err);
          this.loadLocalPlanFallback();
        });
      return;
    }

    this.loadLocalPlanFallback();
  },

  loadLocalPlanFallback() {
    try {
      const plan = wx.getStorageSync('user_study_plan');
      if (plan) {
        const bankIdx = this.data.banks.findIndex(b => String(b.id) === String(plan.bank && plan.bank.id));
        const goal = plan.dailyGoal || 30;
        this.setData({
          selectedBankIndex: bankIdx >= 0 ? bankIdx : 0,
          dailyGoal: goal,
          appReminder: typeof plan.appReminder === 'boolean' ? plan.appReminder : true,
          wechatReminder: typeof plan.wechatReminder === 'boolean' ? plan.wechatReminder : false
        }, () => {
          this.calculateEstimatedDate(goal);
        });
      } else {
        this.calculateEstimatedDate(this.data.dailyGoal);
      }
    } catch (e) {
      this.calculateEstimatedDate(this.data.dailyGoal);
    }
  },

  // 计算预计完成日期
  calculateEstimatedDate(goal) {
    const validGoal = Math.max(1, parseInt(goal, 10) || 50);
    const selectedBank = this.data.banks[this.data.selectedBankIndex] || this.data.banks[0];
    const total = selectedBank.totalQuestions;

    const daysNeeded = Math.ceil(total / validGoal);
    const targetDate = new Date();
    targetDate.setDate(targetDate.getDate() + daysNeeded);

    const yyyy = targetDate.getFullYear();
    const mm = String(targetDate.getMonth() + 1).padStart(2, '0');
    const dd = String(targetDate.getDate()).padStart(2, '0');

    this.setData({
      daysNeeded,
      estimatedDateText: `${yyyy}-${mm}-${dd}`
    });
  },

  // 切换题库
  onBankChange(e) {
    const idx = parseInt(e.detail.value, 10);
    this.setData({
      selectedBankIndex: idx
    });
    this.calculateEstimatedDate(this.data.dailyGoal);
  },

  // 减少每日目标
  onDecreaseGoal() {
    let current = parseInt(this.data.dailyGoal, 10) || 50;
    if (current <= 1) return;

    let step = 5;
    if (current <= 5) {
      step = 1;
    }
    const newGoal = Math.max(1, current - step);
    this.setData({ dailyGoal: newGoal });
    this.calculateEstimatedDate(newGoal);
  },

  // 增加每日目标
  onIncreaseGoal() {
    let current = parseInt(this.data.dailyGoal, 10) || 50;
    if (current >= 1000) return;

    let step = 5;
    const newGoal = Math.min(1000, current + step);
    this.setData({ dailyGoal: newGoal });
    this.calculateEstimatedDate(newGoal);
  },

  // 输入框实时更新
  onGoalInput(e) {
    const val = e.detail.value;
    if (!val) {
      this.setData({ dailyGoal: '' });
      return;
    }
    const num = parseInt(val, 10);
    if (!isNaN(num)) {
      const clamped = Math.min(1000, Math.max(1, num));
      this.setData({ dailyGoal: clamped });
      this.calculateEstimatedDate(clamped);
    }
  },

  // 输入框失焦校验
  onGoalBlur(e) {
    let val = parseInt(e.detail.value, 10);
    if (isNaN(val) || val < 1) {
      val = 50;
    } else if (val > 1000) {
      val = 1000;
    }
    this.setData({ dailyGoal: val });
    this.calculateEstimatedDate(val);
  },

  // 切换应用内提示开关
  onToggleAppReminder() {
    const nextVal = !this.data.appReminder;
    this.setData({
      appReminder: nextVal
    });
    wx.showToast({
      title: nextVal ? '已开启应用内提示' : '已关闭应用内提示',
      icon: 'none',
      duration: 1500
    });
  },

  // 切换微信提示开关
  onToggleWechatReminder() {
    const nextVal = !this.data.wechatReminder;
    this.setData({
      wechatReminder: nextVal
    });
    if (nextVal) {
      wx.showModal({
        title: '微信推送提醒已开启',
        content: '每日 20:00 任务未达标时，系统将通过官方服务号为您发送学习进度提醒。',
        showCancel: false,
        confirmText: '我知道了',
        confirmColor: '#0058bc'
      });
    } else {
      wx.showToast({
        title: '已关闭微信消息提示',
        icon: 'none',
        duration: 1500
      });
    }
  },

  // 返回上一页
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

  // 导航栏更多操作
  onNavMore() {
    wx.showActionSheet({
      itemList: ['重置为默认规划 (50题/天)', '学习规划使用建议', '分享我的规划目标'],
      success: (res) => {
        if (res.tapIndex === 0) {
          this.setData({
            selectedBankIndex: 0,
            dailyGoal: 50,
            appReminder: true,
            wechatReminder: false
          });
          this.calculateEstimatedDate(50);
          wx.showToast({ title: '已恢复默认规划', icon: 'success' });
        } else if (res.tapIndex === 1) {
          wx.showModal({
            title: '科学刷题规划建议',
            content: '建议根据自身备考周期，每日保持 30-80 题的匀速练习，并配合错题复习，能大幅提升记忆与应试效果。',
            showCancel: false,
            confirmText: '好的',
            confirmColor: '#0058bc'
          });
        } else if (res.tapIndex === 2) {
          wx.showToast({ title: '已生成规划海报/链接', icon: 'none' });
        }
      }
    });
  },

  // 开启规划
  onStartPlan() {
    const selectedBank = this.data.banks[this.data.selectedBankIndex] || this.data.banks[0];
    const plan = {
      bank: selectedBank,
      dailyGoal: this.data.dailyGoal || 30,
      daysNeeded: this.data.daysNeeded,
      estimatedDate: this.data.estimatedDateText,
      appReminder: this.data.appReminder,
      wechatReminder: this.data.wechatReminder,
      updatedAt: new Date().toISOString()
    };

    // 本地缓存持久化
    try {
      wx.setStorageSync('user_study_plan', plan);
    } catch (e) {
      console.log('保存本地规划失败', e);
    }

    // 真实后端模式：同步持久化至 Go 服务端
    if (!CONFIG.USE_MOCK) {
      const bankIdNum = parseInt(selectedBank.id, 10) || 1;
      request({
        url: '/api/v1/plans',
        method: 'POST',
        data: {
          bank_id: bankIdNum,
          daily_goal: plan.dailyGoal,
          app_reminder: plan.appReminder,
          wechat_reminder: plan.wechatReminder,
          is_active: true
        }
      }).then((res) => {
        if (res && res.days_needed) {
          plan.daysNeeded = res.days_needed;
          plan.estimatedDate = res.estimated_finish_date;
        }
      }).catch((err) => {
        console.log('[Planning] 保存服务端规划异常:', err);
      });
    }

    wx.showModal({
      title: '学习规划已开启 🎉',
      content: `您已成功设置目标：每日刷题 ${plan.dailyGoal} 道，预计 ${plan.estimatedDate}（共 ${plan.daysNeeded} 天）冲刺完成！`,
      confirmText: '立即去刷题',
      cancelText: '完成',
      confirmColor: '#0058bc',
      success: (res) => {
        if (res.confirm) {
          wx.navigateTo({
            url: '/pages/quiz/quiz'
          });
        } else {
          this.onNavBack();
        }
      }
    });
  }
});
