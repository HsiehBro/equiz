const { request, CONFIG } = require('../../utils/request.js');
const studyStats = require('../../utils/studyStats.js');

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
      checkInDays: 0,
      totalQuestions: 0,
      accuracyRate: 100
    },
    notesCount: 0,
    activePlan: {
      bankId: '1',
      bankTitle: '项目管理基础考试',
      isVip: true,
      dailyGoal: 30,
      todayCount: 0,
      daysNeeded: 4,
      estimatedDate: '2026-09-11',
      isTodayGoalReached: false
    },
    motivationTitle: '今日目标进行中',
    motivationDesc: '今日目标 30 题，预计 2026-09-11 学完 (需 4 天)',
    pendingBanks: [],
    adminCategories: [],
    isCategorySectionCollapsed: false,
    showCategoryModal: false,
    isEditingCategory: false,
    editingCatId: null,
    catFormName: '',
    catFormDesc: '',
    catFormSortOrder: 10,
    catFormIsActive: true,
    catFormIcon: '/assets/icons/menu_book_primary.svg',
    catFormBgClass: 'bg-blue-light',
    categoryIconPresets: [
      { id: 'book', name: '书籍', icon: '/assets/icons/menu_book_primary.svg', bgClass: 'bg-blue-light' },
      { id: 'gavel', name: '法槌', icon: '/assets/icons/gavel_primary.svg', bgClass: 'bg-gray-light' },
      { id: 'medical', name: '医疗', icon: '/assets/icons/medical_services_primary.svg', bgClass: 'bg-blue-light' },
      { id: 'finance', name: '财经', icon: '/assets/icons/account_balance_gray.svg', bgClass: 'bg-gray-light' },
      { id: 'terminal', name: 'IT', icon: '/assets/icons/terminal_gray.svg', bgClass: 'bg-gray-light' },
      { id: 'exam', name: '试卷', icon: '/assets/icons/assignment_primary.svg', bgClass: 'bg-blue-light' },
      { id: 'star', name: '精选', icon: '/assets/icons/workspace_premium_gold.svg', bgClass: 'bg-gold-light' },
      { id: 'grid', name: '综合', icon: '/assets/icons/grid_view_gray.svg', bgClass: 'bg-gray-light' }
    ]
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
    this.loadUserStats();
  },

  onNavBack() {
    const pages = getCurrentPages();
    const indexIdx = pages.findIndex(p => p.route && p.route.includes('index/index'));
    if (indexIdx !== -1) {
      wx.navigateBack({
        delta: pages.length - 1 - indexIdx
      });
    } else if (pages.length > 1) {
      wx.navigateBack();
    } else {
      wx.redirectTo({
        url: '/pages/index/index'
      });
    }
  },

  onOpenNotifications() {
    wx.navigateTo({
      url: '/pages/messages/messages'
    });
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
    const currentNickname = (this.data.userInfo && this.data.userInfo.nickname) || '';
    wx.showModal({
      title: '修改昵称',
      editable: true,
      placeholderText: '请输入新的昵称',
      content: currentNickname,
      confirmText: '确定',
      cancelText: '取消',
      success: (res) => {
        if (!res.confirm) return;

        const newNickname = res.content ? res.content.trim() : '';
        if (!newNickname) {
          wx.showToast({ title: '昵称不能为空', icon: 'none' });
          return;
        }

        if (newNickname === currentNickname) {
          wx.showToast({ title: '昵称未作更改', icon: 'none' });
          return;
        }

        if (newNickname.length > 20) {
          wx.showToast({ title: '昵称最多 20 个字符', icon: 'none' });
          return;
        }

        // 真实后端模式：提交至服务器接口进行全局唯一性校验并持久化
        if (!CONFIG.USE_MOCK) {
          wx.showLoading({ title: '正在校验昵称...' });
          request({
            url: '/api/v1/auth/profile',
            method: 'PUT',
            data: { nickname: newNickname },
            needAuth: true
          }).then(() => {
            wx.hideLoading();
            this.setData({
              'userInfo.nickname': newNickname
            });

            // 同步更新本地持久化缓存
            const cachedUser = wx.getStorageSync('user_info') || {};
            cachedUser.nickname = newNickname;
            wx.setStorageSync('user_info', cachedUser);

            // 同步级联更新本地缓存中的笔记作者及评论回复引用
            this.syncNicknameToLocalStorage(currentNickname, newNickname);

            wx.showToast({ title: '昵称修改成功', icon: 'success' });
          }).catch((err) => {
            wx.hideLoading();
            const errMsg = (err && (err.message || err.msg)) || '该昵称已被其他考友使用，请更换一个唯一的昵称';
            wx.showModal({
              title: '昵称已被占用',
              content: errMsg,
              showCancel: false,
              confirmText: '重新修改',
              success: () => {
                this.onEditProfile();
              }
            });
          });
          return;
        }

        // 本地离线/Mock模式：校验已知用户昵称全局唯一
        const existingMockUsers = [
          '备考先锋',
          'VIP尊享学员',
          '备考新手',
          '系统管理员'
        ];
        if (existingMockUsers.includes(newNickname) && newNickname !== currentNickname) {
          wx.showModal({
            title: '昵称已被占用',
            content: '该昵称已被其他考友使用，请重新输入一个唯一的昵称',
            showCancel: false,
            confirmText: '重新修改',
            success: () => {
              this.onEditProfile();
            }
          });
          return;
        }

        this.setData({
          'userInfo.nickname': newNickname
        });
        const cachedUser = wx.getStorageSync('user_info') || {};
        cachedUser.nickname = newNickname;
        wx.setStorageSync('user_info', cachedUser);

        // 同步级联更新本地缓存中的笔记作者及评论回复引用
        this.syncNicknameToLocalStorage(currentNickname, newNickname);

        wx.showToast({ title: '昵称修改成功', icon: 'success' });
      }
    });
  },

  syncNicknameToLocalStorage(oldNickname, newNickname) {
    if (!oldNickname || !newNickname || oldNickname === newNickname) return;
    try {
      const notes = wx.getStorageSync('user_study_notes_list') || [];
      let updatedNotes = false;
      notes.forEach(n => {
        if (n.author === oldNickname) {
          n.author = newNickname;
          updatedNotes = true;
        }
      });
      if (updatedNotes) wx.setStorageSync('user_study_notes_list', notes);

      const repliesStore = wx.getStorageSync('user_comment_replies') || {};
      let updatedReplies = false;
      Object.keys(repliesStore).forEach(k => {
        (repliesStore[k] || []).forEach(r => {
          if (r.author === oldNickname) {
            r.author = newNickname;
            updatedReplies = true;
          }
          if (r.replyToAuthor === oldNickname) {
            r.replyToAuthor = newNickname;
            updatedReplies = true;
          }
        });
      });
      if (updatedReplies) wx.setStorageSync('user_comment_replies', repliesStore);
    } catch (e) {
      console.log('同步更新本地存储昵称异常:', e);
    }
  },

  onShow() {
    // 自动重置页面滚动条至顶部，防止用户切换账号后页面停留在底部导致顶部头像卡片滑出视口
    try {
      if (wx.pageScrollTo) {
        wx.pageScrollTo({
          scrollTop: 0,
          duration: 0
        });
      }
    } catch (e) {}

    this.loadUserProfile();
    this.loadActivePlan();
    this.loadNotesCount();
    this.loadUserStats();
  },

  loadUserStats() {
    const local = studyStats.getStudyStats();
    this.setData({
      'stats.checkInDays': local.checkInDays || 0,
      'stats.totalQuestions': local.totalQuestions || 0,
      'stats.accuracyRate': local.totalQuestions > 0 ? local.accuracyRate : 100
    });

    if (!CONFIG.USE_MOCK) {
      studyStats.fetchStatsFromServer().then((serverStats) => {
        if (serverStats) {
          this.setData({
            'stats.checkInDays': serverStats.checkInDays || 0,
            'stats.totalQuestions': serverStats.totalQuestions || 0,
            'stats.accuracyRate': serverStats.totalQuestions > 0 ? serverStats.accuracyRate : 100
          });
        }
      });
    }
  },

  loadUserProfile() {
    const cachedUser = wx.getStorageSync('user_info') || {};
    const isVip = wx.getStorageSync('user_is_vip');
    const isLifetime = wx.getStorageSync('user_is_lifetime_vip');
    const vipExpire = wx.getStorageSync('vip_expire_date');

    const defaultAvatar = 'https://lh3.googleusercontent.com/aida-public/AB6AXuC6tZtmFH8sHcTBIJgE4CXv_uiZxRYuuUfO2XvYCKniiNOodibUyu7oV26hBTXqGRYGR7d_Dm0DjRcgI2DwaZb5VkZ2TEyLSXwKPzOsFN8rU_j48rtfj6CAFPx086ngO8lssh8-H8oFnt6obxKUU6QdVaANQRm-wl2cQWfsnumh2bYLfcV82PAJXC1JxZ6M0YN3smvep5qbnGCmh_D2UA3l2h_uSD_Dvn80lVoRNLchLImuDh1ffWZr';

    const userRole = cachedUser.role || 'user';
    const userIsVip = typeof isVip === 'boolean' ? isVip : (userRole === 'admin' || userRole === 'vip' || Boolean(cachedUser.isVip));
    const isLifetimeVip = Boolean(isLifetime || userRole === 'admin' || cachedUser.isLifetimeVip);
    const expireDate = vipExpire || cachedUser.vipExpire || (isLifetimeVip ? '永久' : '2027-12-31');

    // 统一构建 VIP 身份与截至日期合一的金标文案
    let goldBadgeText = '';
    if (userRole === 'admin') {
      goldBadgeText = 'ADMIN · 永久VIP';
    } else if (userIsVip) {
      if (isLifetimeVip || (expireDate && expireDate.includes('永久'))) {
        goldBadgeText = 'VIP · 永久';
      } else {
        goldBadgeText = `VIP · ${expireDate}`;
      }
    }

    const newUserInfo = {
      avatarUrl: cachedUser.avatar_url || cachedUser.avatarUrl || this.data.userInfo.avatarUrl || defaultAvatar,
      nickname: cachedUser.nickname || this.data.userInfo.nickname || '备考学员',
      level: cachedUser.level || (userRole === 'admin' ? 'Lv.99' : (userIsVip ? 'Lv.8' : 'Lv.1')),
      title: cachedUser.title || (userRole === 'admin' ? '系统主控' : (userIsVip ? '终身研习' : '初级备考')),
      role: userRole,
      isVip: userIsVip,
      isLifetimeVip: isLifetimeVip,
      vipExpireDate: expireDate,
      vipGoldBadgeText: goldBadgeText
    };

    this.setData({ userInfo: newUserInfo });

    if (userRole === 'admin') {
      this.loadPendingApprovals();
      this.loadAdminCategories();
      const savedCollapsed = wx.getStorageSync('admin_category_collapsed');
      if (typeof savedCollapsed === 'boolean') {
        this.setData({ isCategorySectionCollapsed: savedCollapsed });
      }
    } else {
      this.setData({ pendingBanks: [], adminCategories: [] });
    }
  },

  loadPendingApprovals() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/admin/approvals', needAuth: true })
        .then((res) => {
          const list = Array.isArray(res) ? res : (res && res.list ? res.list : []);
          this.setData({ pendingBanks: list });
        })
        .catch(() => {
          this.loadPendingApprovalsFallback();
        });
      return;
    }
    this.loadPendingApprovalsFallback();
  },

  loadPendingApprovalsFallback() {
    try {
      const customLibs = wx.getStorageSync('custom_libraries') || [];
      const pending = customLibs.filter(l => l.visibility === 'public' && l.reviewStatus === 'pending');
      this.setData({ pendingBanks: pending });
    } catch (e) {
      this.setData({ pendingBanks: [] });
    }
  },

  onApproveBank(e) {
    const { id, title, creatorid } = e.currentTarget.dataset;
    wx.showModal({
      title: '审批通过确认',
      content: `确认批准《${title}》面向全员公开？通过后所有学员均可在题库列表中刷题，且将向该上传者的个人消息中心推送通过提示。`,
      confirmText: '同意公开',
      confirmColor: '#0058bc',
      success: (res) => {
        if (res.confirm) {
          this.executeApprove(id, title, creatorid);
        }
      }
    });
  },

  executeApprove(id, title, creatorId) {
    wx.showLoading({ title: '正在审批...' });
    const numId = Number(id);
    if (!CONFIG.USE_MOCK && !isNaN(numId) && numId > 0) {
      request({
        url: `/api/v1/admin/approvals/${numId}/approve`,
        method: 'POST'
      }).then(() => {
        wx.hideLoading();
        this.setData({
          pendingBanks: (this.data.pendingBanks || []).filter(b => String(b.id) !== String(id))
        });
        wx.showToast({ title: '已审批通过并公开', icon: 'success' });
        this.loadPendingApprovals();
      }).catch((err) => {
        wx.hideLoading();
        wx.showToast({ title: err.message || '审批失败', icon: 'none' });
      });
      return;
    }

    // 本地/Mock 降级
    setTimeout(() => {
      wx.hideLoading();
      let actualCreatorId = creatorId;
      try {
        const customLibs = wx.getStorageSync('custom_libraries') || [];
        const updated = customLibs.map(l => {
          if (String(l.id) === String(id)) {
            if (!actualCreatorId && l.creatorId) actualCreatorId = l.creatorId;
            return { ...l, visibility: 'public', reviewStatus: 'approved' };
          }
          return l;
        });
        wx.setStorageSync('custom_libraries', updated);

        // 核心规则：谁申请那么通过审批的消息出现在谁的消息中心，而不是其他任何人的消息中心！
        this.pushSystemNotice(actualCreatorId, 'approval_pass', '【题库审核通过】公开申请已批准', `恭喜！您上传/申请公开的题库《${title}》已通过管理员审核，现已正式面向全员公开！`);
      } catch (e) {}

      wx.showToast({ title: '已审批通过并公开', icon: 'success' });
      this.loadPendingApprovals();
    }, 400);
  },

  onRejectBank(e) {
    const { id, title, creatorid } = e.currentTarget.dataset;
    wx.showModal({
      title: '驳回并删除确认',
      content: `确认驳回《${title}》的公开申请？\n根据防刷规则，公开申请被驳回后将直接彻底删除该题库（不保留为私有题库），防止借此绕过额度限制。`,
      confirmText: '确认删除',
      confirmColor: '#ba1a1a',
      success: (res) => {
        if (res.confirm) {
          this.executeReject(id, title, creatorid);
        }
      }
    });
  },

  executeReject(id, title, creatorId) {
    wx.showLoading({ title: '正在处理...' });
    const numId = Number(id);
    if (!CONFIG.USE_MOCK && !isNaN(numId) && numId > 0) {
      request({
        url: `/api/v1/admin/approvals/${numId}/reject`,
        method: 'POST'
      }).then(() => {
        wx.hideLoading();
        this.setData({
          pendingBanks: (this.data.pendingBanks || []).filter(b => String(b.id) !== String(id))
        });
        wx.showToast({ title: '已驳回并彻底删除', icon: 'none' });
        this.loadPendingApprovals();
      }).catch((err) => {
        wx.hideLoading();
        wx.showToast({ title: err.message || '操作失败', icon: 'none' });
      });
      return;
    }

    // 本地/Mock 降级
    setTimeout(() => {
      wx.hideLoading();
      let actualCreatorId = creatorId;
      try {
        const customLibs = wx.getStorageSync('custom_libraries') || [];
        const target = customLibs.find(l => String(l.id) === String(id));
        if (target && target.creatorId) actualCreatorId = target.creatorId;

        // 核心规则：被驳回就直接彻底删除题库，不要保留为私有题库，防止普通用户借此绕过额度限制
        const updated = customLibs.filter(l => String(l.id) !== String(id));
        wx.setStorageSync('custom_libraries', updated);

        // 同步清理可能关联的试题缓存
        wx.removeStorageSync(`custom_questions_${id}`);

        // 核心规则：谁申请那么驳回并删除消息只出现在该申请者的消息中心！
        this.pushSystemNotice(
          actualCreatorId,
          'approval_reject',
          '【题库审核未通过】公开申请已驳回并删除',
          `您申请公开的题库《${title}》未通过管理员审核。根据防刷配额规则，该题库已被直接删除（不保留为私有题库）。`
        );
      } catch (e) {}

      this.setData({
        pendingBanks: (this.data.pendingBanks || []).filter(b => String(b.id) !== String(id))
      });
      wx.showToast({ title: '已驳回并彻底删除', icon: 'none' });
      this.loadPendingApprovals();
    }, 400);
  },

  pushSystemNotice(targetUserId, type, title, content) {
    if (!targetUserId) targetUserId = 'dev_user_003';
    try {
      // 隔离存储至该用户专属通知池：user_system_notifications_${targetUserId}
      const userKey = `user_system_notifications_${targetUserId}`;
      const existing = wx.getStorageSync(userKey) || [];
      existing.unshift({
        id: 'notice_' + Date.now(),
        userId: targetUserId,
        type: type,
        title: title,
        content: content,
        time: '刚刚',
        isRead: false
      });
      wx.setStorageSync(userKey, existing);
    } catch (e) {}
  },

  loadAdminCategories() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/categories', needAuth: true })
        .then((cats) => {
          const list = Array.isArray(cats) ? cats : (cats && cats.list ? cats.list : []);
          this.setData({ adminCategories: list });
        })
        .catch(() => {
          this.loadAdminCategoriesFallback();
        });
      return;
    }
    this.loadAdminCategoriesFallback();
  },

  loadAdminCategoriesFallback() {
    try {
      let customCats = wx.getStorageSync('custom_categories');
      if (!customCats || !Array.isArray(customCats) || customCats.length === 0) {
        customCats = [
          { id: 1, name: '综合', description: '通用与综合测试题库', sort_order: 100, is_active: true, bank_count: 0 },
          { id: 2, name: '医学类', description: '执业医师 / 护理考点', sort_order: 10, is_active: true, bank_count: 0 },
          { id: 3, name: '财经类', description: 'CPA / 会计初级', sort_order: 20, is_active: true, bank_count: 0 },
          { id: 4, name: 'IT互联网', description: '软考 / PMP', sort_order: 30, is_active: true, bank_count: 0 }
        ];
      }
      const customLibs = wx.getStorageSync('custom_libraries') || [];
      const updated = customCats.map(c => {
        const count = customLibs.filter(b => b.category_id === c.id || b.categoryId === c.id || (!b.category_id && !b.categoryId && c.id === 1)).length;
        return {
          ...c,
          bank_count: (c.bank_count || 0) + count
        };
      });
      updated.sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0));
      this.setData({ adminCategories: updated });
    } catch (e) {
      this.setData({ adminCategories: [] });
    }
  },

  onToggleCategoryCollapse() {
    const next = !this.data.isCategorySectionCollapsed;
    this.setData({ isCategorySectionCollapsed: next });
    try {
      wx.setStorageSync('admin_category_collapsed', next);
    } catch (e) {}
  },

  onOpenAddCategoryModal() {
    this.setData({
      showCategoryModal: true,
      isEditingCategory: false,
      editingCatId: null,
      catFormName: '',
      catFormDesc: '',
      catFormSortOrder: 50,
      catFormIsActive: true,
      catFormIcon: '/assets/icons/menu_book_primary.svg',
      catFormBgClass: 'bg-blue-light',
      isCategorySectionCollapsed: false
    });
  },

  onOpenEditCategoryModal(e) {
    const item = e.currentTarget.dataset.item;
    if (!item) return;
    this.setData({
      showCategoryModal: true,
      isEditingCategory: true,
      editingCatId: item.id,
      catFormName: item.name || '',
      catFormDesc: item.description || '',
      catFormSortOrder: item.sort_order !== undefined ? item.sort_order : 10,
      catFormIsActive: item.is_active !== undefined ? item.is_active : true,
      catFormIcon: item.icon || '/assets/icons/menu_book_primary.svg',
      catFormBgClass: item.bg_class || item.bgClass || 'bg-blue-light'
    });
  },

  onSelectCatIcon(e) {
    const { icon, bg } = e.currentTarget.dataset;
    this.setData({
      catFormIcon: icon,
      catFormBgClass: bg || 'bg-blue-light'
    });
  },

  onCloseCategoryModal() {
    this.setData({ showCategoryModal: false });
  },

  onInputCatName(e) {
    this.setData({ catFormName: e.detail.value });
  },

  onInputCatDesc(e) {
    this.setData({ catFormDesc: e.detail.value });
  },

  onInputCatSortOrder(e) {
    this.setData({ catFormSortOrder: e.detail.value });
  },

  onToggleCatActive(e) {
    this.setData({ catFormIsActive: e.detail.value });
  },

  preventDumbTap() {
    // 阻止模态框内部点击事件冒泡到外层遮罩
  },

  onSubmitCategoryForm() {
    const name = (this.data.catFormName || '').trim();
    const desc = (this.data.catFormDesc || '').trim();
    let sortOrder = parseInt(this.data.catFormSortOrder, 10);
    if (isNaN(sortOrder)) sortOrder = 0;
    const isActive = Boolean(this.data.catFormIsActive);
    const icon = this.data.catFormIcon || '/assets/icons/menu_book_primary.svg';
    const bgClass = this.data.catFormBgClass || 'bg-blue-light';

    if (!name) {
      wx.showToast({ title: '请输入分类名称', icon: 'none' });
      return;
    }
    if (name.length > 20) {
      wx.showToast({ title: '分类名称最多20个字符', icon: 'none' });
      return;
    }

    const isEditing = this.data.isEditingCategory;
    const catId = this.data.editingCatId;

    // 重名校验（排除自身）
    const existing = (this.data.adminCategories || []).find(c => c.name.toLowerCase() === name.toLowerCase() && (!isEditing || c.id !== catId));
    if (existing) {
      wx.showToast({ title: '已存在同名分类', icon: 'none' });
      return;
    }

    // 默认综合分类保护：不可改名，不可下架
    if (isEditing && catId === 1) {
      if (name !== '综合') {
        wx.showToast({ title: '默认综合分类不可改名', icon: 'none' });
        return;
      }
      if (!isActive) {
        wx.showToast({ title: '默认综合分类不可下架', icon: 'none' });
        return;
      }
    }

    wx.showLoading({ title: isEditing ? '保存中...' : '创建中...' });

    if (!CONFIG.USE_MOCK) {
      const url = isEditing ? `/api/v1/categories/${catId}` : '/api/v1/categories';
      const method = isEditing ? 'PUT' : 'POST';
      const payload = {
        name,
        description: desc,
        icon,
        bg_class: bgClass,
        sort_order: sortOrder,
        ...(isEditing ? { is_active: isActive } : {})
      };

      request({
        url,
        method,
        data: payload,
        needAuth: true
      }).then(() => {
        wx.hideLoading();
        wx.showToast({ title: isEditing ? '分类修改成功' : '分类创建成功', icon: 'success' });
        this.setData({ showCategoryModal: false });
        this.loadAdminCategories();
      }).catch((err) => {
        wx.hideLoading();
        wx.showToast({ title: (err && (err.message || err.msg)) || '操作失败', icon: 'none' });
      });
      return;
    }

    // 本地/Mock 降级
    setTimeout(() => {
      wx.hideLoading();
      try {
        let customCats = wx.getStorageSync('custom_categories') || [];
        if (customCats.length === 0 && this.data.adminCategories.length > 0) {
          customCats = JSON.parse(JSON.stringify(this.data.adminCategories));
        }
        if (isEditing) {
          customCats = customCats.map(c => {
            if (c.id === catId) {
              return {
                ...c,
                name,
                description: desc,
                icon,
                bg_class: bgClass,
                sort_order: sortOrder,
                is_active: isActive
              };
            }
            return c;
          });
        } else {
          const maxId = customCats.reduce((max, c) => Math.max(max, c.id || 0), 10);
          customCats.push({
            id: maxId + 1,
            name,
            description: desc,
            icon,
            bg_class: bgClass,
            sort_order: sortOrder,
            is_active: true,
            bank_count: 0
          });
        }
        wx.setStorageSync('custom_categories', customCats);
        wx.showToast({ title: isEditing ? '分类修改成功' : '分类创建成功', icon: 'success' });
        this.setData({ showCategoryModal: false });
        this.loadAdminCategories();
      } catch (e) {
        wx.showToast({ title: '保存失败', icon: 'none' });
      }
    }, 300);
  },

  onDeleteCategory(e) {
    const { id, name } = e.currentTarget.dataset;
    const catId = Number(id);
    if (catId === 1) {
      wx.showModal({
        title: '系统保护提示',
        content: '「综合」是系统默认分类及其他分类删除时的兜底归宿，禁止删除。',
        showCancel: false,
        confirmText: '我知道了',
        confirmColor: '#0058bc'
      });
      return;
    }

    wx.showModal({
      title: '删除分类确认',
      content: `确定删除分类「${name}」？\n\n删除后，原属于该分类的题库将自动归入系统默认「综合」分类兜底，题库与做题记录不会丢失。`,
      confirmText: '确认删除',
      confirmColor: '#ba1a1a',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          this.executeDeleteCategory(catId, name);
        }
      }
    });
  },

  executeDeleteCategory(id, name) {
    wx.showLoading({ title: '正在删除...' });
    if (!CONFIG.USE_MOCK) {
      request({
        url: `/api/v1/categories/${id}`,
        method: 'DELETE',
        needAuth: true
      }).then(() => {
        wx.hideLoading();
        wx.showToast({ title: '已删除并归入综合', icon: 'success' });
        this.loadAdminCategories();
      }).catch((err) => {
        wx.hideLoading();
        wx.showToast({ title: (err && (err.message || err.msg)) || '删除失败', icon: 'none' });
      });
      return;
    }

    // 本地/Mock 降级
    setTimeout(() => {
      wx.hideLoading();
      try {
        let customCats = wx.getStorageSync('custom_categories') || [];
        customCats = customCats.filter(c => c.id !== id);
        wx.setStorageSync('custom_categories', customCats);

        // 将关联本地题库归入综合分类 (ID: 1)
        const customLibs = wx.getStorageSync('custom_libraries') || [];
        let modifiedLibs = false;
        const updatedLibs = customLibs.map(lib => {
          if (lib.category_id === id || lib.categoryId === id) {
            modifiedLibs = true;
            return {
              ...lib,
              category_id: 1,
              categoryId: 1,
              category: '综合'
            };
          }
          return lib;
        });
        if (modifiedLibs) {
          wx.setStorageSync('custom_libraries', updatedLibs);
        }

        wx.showToast({ title: '已删除并归入综合', icon: 'success' });
        this.loadAdminCategories();
      } catch (e) {
        wx.showToast({ title: '操作失败', icon: 'none' });
      }
    }, 300);
  },

  calculateTargetDateStr(days) {
    const d = new Date();
    d.setDate(d.getDate() + (parseInt(days, 10) || 0));
    const yyyy = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    const dd = String(d.getDate()).padStart(2, '0');
    return `${yyyy}-${mm}-${dd}`;
  },

  loadActivePlan() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/plans/active' })
        .then((plan) => {
          if (plan && plan.id) {
            const rawBankTitle = plan.bank_title || '项目管理基础考试';
            const bankTitle = rawBankTitle.replace(/\s*\(共\s*\d+\s*题\)/, '').trim() || '项目管理基础考试';
            const bankId = String(plan.bank_id || '1');
            const dailyGoal = plan.daily_goal || 30;
            const todayCount = plan.today_count || 0;
            const daysNeeded = plan.days_needed && plan.days_needed > 0 ? plan.days_needed : 4;
            const todayStr = this.calculateTargetDateStr(0);
            let estimatedDate = plan.estimated_finish_date;
            if (!estimatedDate || estimatedDate === todayStr) {
              estimatedDate = this.calculateTargetDateStr(daysNeeded);
            }
            const isReached = Boolean(plan.is_today_goal_reached || todayCount >= dailyGoal);

            let motivationTitle = '今日目标进行中';
            let motivationDesc = '';
            if (isReached) {
              motivationTitle = '今日目标已达成 🎉';
              motivationDesc = `今日已完成 ${todayCount}/${dailyGoal} 题，超额完成每日目标！预计 ${estimatedDate} 学完 (需 ${daysNeeded} 天)`;
            } else if (todayCount > 0) {
              motivationTitle = '今日目标进行中';
              motivationDesc = `今日已刷 ${todayCount}/${dailyGoal} 题，预计 ${estimatedDate} 学完 (需 ${daysNeeded} 天)`;
            } else {
              motivationTitle = '今日目标进行中';
              motivationDesc = `今日目标 ${dailyGoal} 题，预计 ${estimatedDate} 学完 (需 ${daysNeeded} 天)`;
            }

            const isVip = Boolean(
              plan.is_vip ||
              plan.isVip ||
              bankId === '1' ||
              bankId === '3' ||
              bankTitle.includes('项目管理') ||
              bankTitle.includes('会计')
            );

            const activePlan = {
              id: plan.id,
              bankId,
              bankTitle,
              isVip,
              dailyGoal,
              todayCount,
              daysNeeded,
              estimatedDate,
              isTodayGoalReached: isReached
            };

            const checkInDays = plan.check_in_days !== undefined ? plan.check_in_days : (this.data.stats.checkInDays || 0);
            this.setData({
              activePlan,
              motivationTitle,
              motivationDesc,
              'stats.checkInDays': checkInDays
            });

            // 保持本地存储与服务端主规划强一致
            try {
              wx.setStorageSync('user_study_plan', {
                bank: { id: bankId, title: bankTitle },
                bankId,
                bankTitle,
                dailyGoal,
                todayCount,
                daysNeeded,
                estimatedDate,
                updatedAt: new Date().toISOString()
              });
            } catch (e) {}
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
      const plan = wx.getStorageSync('user_study_plan') || {};
      const bank = plan.bank || {};
      let rawBankTitle = plan.bankTitle || (bank.rawTitle || (bank.title ? bank.title.replace(/\s*\(共\s*\d+\s*题\)/, '') : '')) || '项目管理基础考试';
      let bankTitle = rawBankTitle.trim();
      let bankId = String(plan.bankId || bank.id || '1');
      let dailyGoal = plan.dailyGoal || 30;
      let totalQuestions = bank.totalQuestions || plan.totalQuestions || 100;
      let todayCount = plan.todayCount || 0;

      let daysNeeded = plan.daysNeeded;
      if (!daysNeeded || daysNeeded <= 0) {
        daysNeeded = Math.max(1, Math.ceil(totalQuestions / dailyGoal));
      }

      const todayStr = this.calculateTargetDateStr(0);
      let estimatedDate = plan.estimatedDate;
      if (!estimatedDate || estimatedDate === todayStr) {
        estimatedDate = this.calculateTargetDateStr(daysNeeded);
      }

      const isReached = todayCount >= dailyGoal;
      let motivationTitle = isReached ? '今日目标已达成 🎉' : '今日目标进行中';
      let motivationDesc = '';
      if (isReached) {
        motivationDesc = `今日已完成 ${todayCount}/${dailyGoal} 题，超额完成每日目标！预计 ${estimatedDate} 学完 (需 ${daysNeeded} 天)`;
      } else if (todayCount > 0) {
        motivationDesc = `今日已刷 ${todayCount}/${dailyGoal} 题，预计 ${estimatedDate} 学完 (需 ${daysNeeded} 天)`;
      } else {
        motivationDesc = `今日目标 ${dailyGoal} 题，预计 ${estimatedDate} 学完 (需 ${daysNeeded} 天)`;
      }

      const isVip = Boolean(
        plan.isVip ||
        plan.is_vip ||
        bank.isVip ||
        bank.is_vip ||
        bankId === '1' ||
        bankId === '3' ||
        bankTitle.includes('项目管理') ||
        bankTitle.includes('会计')
      );

      const activePlan = {
        bankId,
        bankTitle,
        isVip,
        dailyGoal,
        todayCount,
        daysNeeded,
        estimatedDate,
        isTodayGoalReached: isReached
      };

      const checkInDays = plan.checkInDays !== undefined ? plan.checkInDays : (this.data.stats.checkInDays || 0);
      this.setData({
        activePlan,
        motivationTitle,
        motivationDesc,
        'stats.checkInDays': checkInDays
      });

      // 清洗并同步回写本地有效缓存
      wx.setStorageSync('user_study_plan', {
        ...plan,
        bank: { id: bankId, title: bankTitle, totalQuestions },
        bankId,
        bankTitle,
        dailyGoal,
        todayCount,
        daysNeeded,
        estimatedDate
      });
    } catch (e) {
      console.log('读取本地学习规划异常', e);
    }
  },

  onTapPlanCard() {
    wx.navigateTo({
      url: '/pages/planning/planning'
    });
  },

  onGoQuiz() {
    const plan = this.data.activePlan || {};
    const bankId = plan.bankId || '1';
    const bankTitle = plan.bankTitle || '项目管理基础考试';
    wx.navigateTo({
      url: `/pages/quiz/quiz?bankId=${bankId}&title=${encodeURIComponent(bankTitle)}`
    });
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

  onSwitchTab(e) {
    const tab = e.currentTarget.dataset.tab;
    if (tab === 'library') {
      const pages = getCurrentPages();
      const indexIdx = pages.findIndex(p => p.route && p.route.includes('index/index'));
      if (indexIdx !== -1) {
        wx.navigateBack({
          delta: pages.length - 1 - indexIdx
        });
      } else {
        wx.redirectTo({
          url: '/pages/index/index'
        });
      }
    } else if (tab === 'errors') {
      wx.redirectTo({
        url: '/pages/errors/errors'
      });
    }
  }
});
