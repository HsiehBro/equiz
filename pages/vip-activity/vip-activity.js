const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    hasSetReminder: false,
    isLifetimeVIP: false,
    showRewardsModal: false,
    loadingRewards: false,
    totalRewardDays: 0,
    totalInviteCount: 0,
    referralList: []
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

      const userInfo = wx.getStorageSync('user_info') || {};
      const isLifetime = Boolean(wx.getStorageSync('user_is_lifetime_vip') || userInfo.isLifetimeVip || userInfo.role === 'admin');

      this.setData({
        statusBarHeight,
        navBarHeight,
        isLifetimeVIP: isLifetime
      });

      if (isLifetime) {
        wx.showModal({
          title: '温馨提示',
          content: '您已开通永久 VIP 会员，享有全站最高终身特权，不能参加任何会员活动！',
          showCancel: false,
          confirmText: '我知道了',
          confirmColor: '#0058bc'
        });
      } else {
        this.fetchReferralRewards();
      }
    } catch (e) {
      console.log('获取导航栏信息异常', e);
    }
  },

  onShow() {
    if (!this.data.isLifetimeVIP) {
      this.fetchReferralRewards();
    }
  },

  fetchReferralRewards() {
    this.setData({ loadingRewards: true });
    request({
      url: '/api/v1/activity/referral/rewards',
      method: 'GET'
    }).then((res) => {
      this.setData({
        totalRewardDays: (res && res.total_reward_days) || 0,
        totalInviteCount: (res && res.total_invite_count) || 0,
        referralList: (res && res.list) || [],
        loadingRewards: false
      });
    }).catch((err) => {
      console.warn('获取邀请奖励数据降级:', err);
      // 本地离线降级兜底
      const localList = wx.getStorageSync('local_referral_list') || [];
      const totalDays = localList.reduce((sum, item) => sum + (item.reward_days || 0), 0);
      this.setData({
        totalRewardDays: totalDays,
        totalInviteCount: localList.length,
        referralList: localList,
        loadingRewards: false
      });
    });
  },

  onNavBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack();
    } else {
      wx.redirectTo({
        url: '/pages/profile/profile'
      });
    }
  },

  checkLifetimeAlert() {
    if (this.data.isLifetimeVIP) {
      wx.showModal({
        title: '活动限制提示',
        content: '购买永久会员的用户不能参加任何活动！',
        showCancel: false,
        confirmText: '我知道了',
        confirmColor: '#0058bc'
      });
      return true;
    }
    return false;
  },

  onTapMyRewards() {
    if (this.checkLifetimeAlert()) return;
    this.fetchReferralRewards();
    this.setData({ showRewardsModal: true });
  },

  onCloseRewardsModal() {
    this.setData({ showRewardsModal: false });
  },

  preventProp() {},

  onShareAppMessage() {
    const userInfo = wx.getStorageSync('user_info') || {};
    const userId = userInfo.id || CONFIG.DEV_USER_ID || 1;
    return {
      title: '🌟 备考刷题神器！快来一起解锁全量真题与名师解析，双方同享会员时长！',
      path: `/pages/vip/vip?inviter_id=${userId}`,
      imageUrl: '/assets/icons/auto_awesome_primary.svg'
    };
  },

  onTapCheckInActivity() {
    if (this.checkLifetimeAlert()) return;

    wx.navigateTo({
      url: '/pages/planning/planning'
    });
  },

  onTapDiscountActivity() {
    if (this.checkLifetimeAlert()) return;

    const nextState = !this.data.hasSetReminder;
    this.setData({
      hasSetReminder: nextState
    });

    wx.showToast({
      title: nextState ? '已开启活动开售提醒' : '已取消提醒',
      icon: 'success'
    });
  }
});

