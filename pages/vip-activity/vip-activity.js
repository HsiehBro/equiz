Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    hasSetReminder: false
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
  },

  onNavBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack();
    } else {
      wx.reLaunch({
        url: '/pages/profile/profile'
      });
    }
  },

  onTapMyRewards() {
    wx.showModal({
      title: '🎁 我的活动奖励',
      content: '当前累计获得：会员时长奖励 14 天，全真模拟试卷 2 套。已自动生效至您的账号！',
      showCancel: false,
      confirmText: '太棒了',
      confirmColor: '#0058bc'
    });
  },

  onShareAppMessage() {
    return {
      title: '🌟 备考刷题神器！快来一起解锁全量真题与名师解析，双方同享会员时长！',
      path: '/pages/vip/vip',
      imageUrl: '/assets/icons/auto_awesome_primary.svg'
    };
  },

  onTapCheckInActivity() {
    wx.navigateTo({
      url: '/pages/planning/planning'
    });
  },

  onTapDiscountActivity() {
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
