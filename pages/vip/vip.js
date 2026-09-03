Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    selectedPlanId: 'yearly',
    currentPlanPrice: 198,
    plans: {
      lifetime: { id: 'lifetime', name: '终身畅学卡', price: 398 },
      monthly: { id: 'monthly', name: '连续包月会员', price: 29 },
      quarterly: { id: 'quarterly', name: '连续包季会员', price: 68 },
      yearly: { id: 'yearly', name: '连续包年会员', price: 198 }
    },
    showHelpModal: false,
    showTermsModal: false
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
        navBarHeight,
        currentPlanPrice: this.data.plans[this.data.selectedPlanId].price
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

  onSelectPlan(e) {
    const id = e.currentTarget.dataset.id;
    if (id && id !== this.data.selectedPlanId) {
      const price = this.data.plans[id] ? this.data.plans[id].price : 198;
      this.setData({
        selectedPlanId: id,
        currentPlanPrice: price
      });
    }
  },

  onGoActivityCenter() {
    wx.navigateTo({
      url: '/pages/vip-activity/vip-activity'
    });
  },

  onOpenHelp() {
    this.setData({ showHelpModal: true });
  },

  onCloseHelp() {
    this.setData({ showHelpModal: false });
  },

  onOpenTerms() {
    this.setData({ showTermsModal: true });
  },

  onCloseTerms() {
    this.setData({ showTermsModal: false });
  },

  preventProp() {},

  onRestorePurchase() {
    wx.showLoading({ title: '正在检索购买记录...' });
    setTimeout(() => {
      wx.hideLoading();
      wx.showModal({
        title: '恢复购买成功',
        content: '已同步您的 VIP 会员特权至当前账号，有效期至 2027-12-31。',
        showCancel: false,
        confirmColor: '#0058bc'
      });
    }, 1000);
  },

  onUnlockWithWeChatPay() {
    const currentPlan = this.data.plans[this.data.selectedPlanId] || this.data.plans.yearly;
    
    wx.showModal({
      title: '确认开通 VIP',
      content: `您将开通【${currentPlan.name}】，支付金额：¥${currentPlan.price} 元。确认调起微信安全支付？`,
      confirmText: '立即支付',
      confirmColor: '#07C160',
      cancelText: '再想想',
      success: (res) => {
        if (res.confirm) {
          this.executePayment(currentPlan);
        }
      }
    });
  },

  executePayment(plan) {
    wx.showLoading({ title: '正在调起微信支付...' });
    
    setTimeout(() => {
      wx.hideLoading();
      
      // 更新全局与本地存储状态
      try {
        wx.setStorageSync('user_is_vip', true);
        wx.setStorageSync('vip_plan', plan.id);
        wx.setStorageSync('vip_expire_date', '2027-12-31');
      } catch (e) {
        console.error(e);
      }

      wx.showModal({
        title: '🎉 恭喜开通 VIP 会员！',
        content: `恭喜您已成功开通【${plan.name}】！全量题库、视频精讲与模拟考试现已全部解锁。`,
        showCancel: false,
        confirmText: '开启刷题',
        confirmColor: '#0058bc',
        success: () => {
          wx.navigateBack({
            fail: () => {
              wx.reLaunch({ url: '/pages/profile/profile' });
            }
          });
        }
      });
    }, 1200);
  }
});
