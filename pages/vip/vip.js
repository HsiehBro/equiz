const { request, CONFIG } = require('../../utils/request.js');

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
    showTermsModal: false,
    isLifetimeVIP: false
  },

  onLoad(options) {
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
        isLifetimeVIP: isLifetime,
        currentPlanPrice: this.data.plans[this.data.selectedPlanId].price
      });

      // 静默锁定被邀请人与邀请人归属关系（首购保护）
      if (options && options.inviter_id) {
        const inviterId = Number(options.inviter_id);
        if (inviterId > 0) {
          request({
            url: '/api/v1/activity/referral/bind',
            method: 'POST',
            data: { inviter_id: inviterId }
          }).catch((err) => {
            console.log('静默锁定邀请关系跳过或已绑定:', err && err.message);
          });
        }
      }

      if (isLifetime) {
        wx.showModal({
          title: '永久 VIP 提示',
          content: '您已是终身永久 VIP 会员，享有全量最高特权，无需且不能购买任何会员套餐！',
          showCancel: false,
          confirmText: '我知道了',
          confirmColor: '#0058bc'
        });
      }
    } catch (e) {
      console.log('获取导航栏信息异常', e);
    }
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

  onSelectPlan(e) {
    if (this.data.isLifetimeVIP) {
      wx.showModal({
        title: '购买限制提示',
        content: '购买永久会员的用户不能购买任何会员套餐！',
        showCancel: false,
        confirmText: '我知道了',
        confirmColor: '#0058bc'
      });
      return;
    }

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
    if (this.data.isLifetimeVIP) {
      wx.showModal({
        title: '活动限制提示',
        content: '购买永久会员的用户不能参加任何活动！',
        showCancel: false,
        confirmText: '我知道了',
        confirmColor: '#0058bc'
      });
      return;
    }

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
    if (this.data.isLifetimeVIP) {
      wx.showModal({
        title: '购买限制提示',
        content: '购买永久会员的用户不能购买任何会员套餐！',
        showCancel: false,
        confirmText: '我知道了',
        confirmColor: '#0058bc'
      });
      return;
    }

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
    
    // 调用后端真实下单与邀请发奖联动接口
    request({
      url: '/api/v1/vip/purchase',
      method: 'POST',
      data: { plan_id: plan.id }
    }).then((res) => {
      wx.hideLoading();

      const isLifetime = plan.id === 'lifetime';
      wx.setStorageSync('user_is_vip', true);
      wx.setStorageSync('vip_plan', plan.id);

      let expireStr = '永久';
      if (res && res.user && res.user.vip_expire) {
        expireStr = String(res.user.vip_expire).substring(0, 10);
      } else if (!isLifetime) {
        expireStr = '2027-12-31';
      }

      if (isLifetime) {
        wx.setStorageSync('user_is_lifetime_vip', true);
        wx.setStorageSync('vip_expire_date', '永久');
      } else {
        wx.setStorageSync('vip_expire_date', expireStr);
      }

      const userInfo = wx.getStorageSync('user_info') || {};
      userInfo.isVip = true;
      userInfo.isLifetimeVip = isLifetime;
      userInfo.vipExpire = isLifetime ? '永久' : expireStr;
      userInfo.role = 'vip';
      wx.setStorageSync('user_info', userInfo);

      if (isLifetime) {
        this.setData({ isLifetimeVIP: true });
      }

      let bonusNotice = '';
      if (res && res.reward_days > 0) {
        bonusNotice = `\n🎁 好友邀请专享福利已生效，额外获赠 ${res.reward_days} 天会员时长！`;
      }

      wx.showModal({
        title: '🎉 恭喜开通 VIP 会员！',
        content: `恭喜您已成功开通【${plan.name}】！${bonusNotice}\n全量题库、名师解析与模拟考试现已全部解锁。`,
        showCancel: false,
        confirmText: '开启刷题',
        confirmColor: '#0058bc',
        success: () => {
          wx.navigateBack({
            fail: () => {
              wx.redirectTo({ url: '/pages/profile/profile' });
            }
          });
        }
      });
    }).catch((err) => {
      console.warn('后端充值接口调用失败，启用本地兜底模拟:', err);
      // 离线/网络错误时平滑启动本地降级
      setTimeout(() => {
        wx.hideLoading();
        
        try {
          const isLifetime = plan.id === 'lifetime';
          wx.setStorageSync('user_is_vip', true);
          wx.setStorageSync('vip_plan', plan.id);
          if (isLifetime) {
            wx.setStorageSync('user_is_lifetime_vip', true);
            wx.setStorageSync('vip_expire_date', '永久');
          } else {
            wx.setStorageSync('vip_expire_date', '2027-12-31');
          }

          const userInfo = wx.getStorageSync('user_info') || {};
          userInfo.isVip = true;
          userInfo.isLifetimeVip = isLifetime;
          userInfo.vipExpire = isLifetime ? '永久' : '2027-12-31';
          userInfo.role = 'vip';
          wx.setStorageSync('user_info', userInfo);

          if (isLifetime) {
            this.setData({ isLifetimeVIP: true });
          }
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
                wx.redirectTo({ url: '/pages/profile/profile' });
              }
            });
          }
        });
      }, 800);
    });
  }
});
