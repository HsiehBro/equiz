const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    searchKeyword: '',
    selectedCategory: '',
    currentTab: 'library',
    libraries: []
  },

  allLibraries: [],

  onLoad() {
    this.initNavBar();
    // 页面初始化即刻装载预设题库，确保搜索与过滤即开即用
    this.allLibraries = this.getDefaultLibrariesPreset();
    this.applyFilter();
  },

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

  getDefaultLibrariesPreset() {
    try {
      const userInfo = wx.getStorageSync('user_info') || {};
      const currentUserId = userInfo.id || userInfo.open_id || 'dev_user_001';
      const customLibs = wx.getStorageSync('custom_libraries') || [];

      // 私有题库隔离：仅展示公开题库或本人创建的私有题库
      const visibleCustom = customLibs.filter(lib => {
        if (lib.visibility === 'public' || lib.isOfficial) return true;
        return !lib.creatorId || String(lib.creatorId) === String(currentUserId);
      });

      const defaultLibs = [
        {
          id: 1,
          title: '项目管理基础考试',
          category: 'IT互联网',
          description: 'PMP/高项核心考点，包含单选与多选题',
          totalCount: 100,
          progress: 0,
          lastPractice: '未开始',
          visibility: 'public',
          isOfficial: true,
          isPrimary: true
        },
        {
          id: 2,
          title: '2023年护士执业资格考试',
          category: '医学类',
          description: '全国护士执业资格考试精选试题',
          totalCount: 50,
          progress: 12,
          lastPractice: '昨天',
          visibility: 'public',
          isOfficial: true,
          isPrimary: false
        },
        {
          id: 3,
          title: '初级会计实务 - 核心考点',
          category: '财经类',
          description: '初级会计实务真题演练与必背知识点',
          totalCount: 40,
          progress: 45,
          lastPractice: '3天前',
          visibility: 'public',
          isOfficial: true,
          isPrimary: false
        }
      ];
      if (visibleCustom.length > 0) {
        const customIds = new Set(visibleCustom.map(l => l.id));
        return [...visibleCustom, ...defaultLibs.filter(l => !customIds.has(l.id))];
      }
      return defaultLibs;
    } catch (e) {
      return [];
    }
  },

  onNavBack() {
    wx.showToast({ title: '已在首页', icon: 'none' });
  },

  onOpenNotifications() {
    wx.navigateTo({
      url: '/pages/messages/messages'
    });
  },

  /**
   * 搜索框实时输入 (实时模糊过滤)
   */
  onSearchInput(e) {
    const val = (e && e.detail && typeof e.detail.value === 'string') ? e.detail.value : (typeof e === 'string' ? e : '');
    this.applyFilter(val);
  },

  /**
   * 搜索框确认/回车 (移动键盘点击“搜索”或PC按Enter)
   */
  onSearchConfirm(e) {
    const val = (e && e.detail && typeof e.detail.value === 'string') ? e.detail.value : (this.data.searchKeyword || '');
    this.applyFilter(val);
  },

  /**
   * 搜索框失去焦点 (自动应用当前输入值)
   */
  onSearchBlur(e) {
    const val = (e && e.detail && typeof e.detail.value === 'string') ? e.detail.value : (this.data.searchKeyword || '');
    this.applyFilter(val);
  },

  /**
   * 清除搜索框内容
   */
  onClearSearch() {
    this.applyFilter('');
  },

  onInviteFriends() {
    wx.showModal({
      title: '邀请好友',
      content: '分享小程序给好友，双方各获赠 7 天题库会员！',
      showCancel: false
    });
  },

  onSelectCategory(e) {
    const category = e.currentTarget.dataset.category;
    if (category === '全部分类' || this.data.selectedCategory === category) {
      this.setData({ selectedCategory: '' }, () => {
        this.applyFilter();
        wx.showToast({ title: '已显示全部题库', icon: 'none', duration: 1200 });
      });
    } else {
      this.setData({ selectedCategory: category }, () => {
        this.applyFilter();
        wx.showToast({ title: `已按 [${category}] 筛选`, icon: 'none', duration: 1200 });
      });
    }
  },

  onShow() {
    this.loadLibraries();
  },

  loadLibraries() {
    if (!CONFIG.USE_MOCK) {
      // 真实后端模式：从 Go API + PostgreSQL 获取题库（自动附带 Token 进行私有题库隔离）
      request({ url: '/api/v1/banks', needAuth: true })
        .then((banks) => {
          if (Array.isArray(banks) && banks.length > 0) {
            const formatted = banks.map((b, idx) => ({
              id: b.id,
              title: b.title,
              category: b.category || '',
              description: b.description || '',
              totalCount: b.total_count,
              progress: b.progress || 0,
              lastPractice: b.last_practice || '未开始',
              visibility: b.visibility || (b.is_official ? 'public' : 'private'),
              isOfficial: Boolean(b.is_official),
              isPrimary: idx === 0
            }));
            this.allLibraries = formatted;
            this.applyFilter();
            return;
          }
          this.loadLocalLibrariesFallback();
        })
        .catch((err) => {
          console.log('[Index] 请求后端题库失败，降级读取本地缓存:', err);
          this.loadLocalLibrariesFallback();
        });
    } else {
      this.loadLocalLibrariesFallback();
    }
  },

  loadLocalLibrariesFallback() {
    try {
      this.allLibraries = this.getDefaultLibrariesPreset();
      this.applyFilter();
    } catch (e) {
      console.log('读取题库缓存异常', e);
    }
  },

  /**
   * 核心题库过滤逻辑 (仅过滤题库名称的关键词)
   */
  applyFilter(targetKeyword) {
    const rawKeyword = typeof targetKeyword === 'string' ? targetKeyword : (this.data.searchKeyword || '');
    const keyword = rawKeyword.trim().toLowerCase();
    const category = this.data.selectedCategory;

    let sourceList = Array.isArray(this.allLibraries) && this.allLibraries.length > 0 
      ? this.allLibraries 
      : this.getDefaultLibrariesPreset();

    let list = [...sourceList];

    // 1. 分类筛选
    if (category && category !== '全部分类') {
      list = list.filter(item => {
        const cat = (item.category || '').toLowerCase();
        const title = (item.title || '').toLowerCase();
        if (category === '医学类') {
          return cat.includes('医学') || title.includes('医') || title.includes('护士') || title.includes('药');
        }
        if (category === '财经类') {
          return cat.includes('财经') || cat.includes('会计') || title.includes('会计') || title.includes('cpa') || title.includes('财');
        }
        if (category === 'IT互联网') {
          return cat.includes('it') || cat.includes('工程') || cat.includes('互联网') || title.includes('项目管理') || title.includes('pmp') || title.includes('软考') || title.includes('技术');
        }
        return cat.includes(category.toLowerCase()) || title.includes(category.toLowerCase());
      });
    }

    // 2. 首页顶部搜索框仅过滤题库名称的关键词
    if (keyword) {
      list = list.filter(item => {
        const title = (item.title || item.name || '').toLowerCase();
        return title.includes(keyword);
      });
    }

    // 3. 动态维护首个高亮标识 isPrimary
    const formatted = list.map((item, idx) => ({
      ...item,
      isPrimary: idx === 0
    }));

    this.setData({
      searchKeyword: rawKeyword,
      libraries: formatted
    });
  },

  onImportLibrary() {
    wx.navigateTo({
      url: '/pages/import/import'
    });
  },

  onViewAllLibraries() {
    this.setData({
      searchKeyword: '',
      selectedCategory: ''
    }, () => {
      this.applyFilter('');
      wx.showToast({ title: '已显示全部题库', icon: 'none' });
    });
  },

  onContinuePractice(e) {
    const id = e.currentTarget.dataset.id;
    const currentLib = (this.allLibraries || []).find(l => l.id === id) || (this.data.libraries || []).find(l => l.id === id);
    const title = currentLib ? currentLib.title : '项目管理基础考试';

    wx.navigateTo({
      url: `/pages/quiz/quiz?mode=practice&bankId=${id}&title=${encodeURIComponent(title)}`
    });
  },

  onDeleteLibrary(e) {
    const id = e.currentTarget.dataset.id;
    const currentLib = (this.allLibraries || []).find(l => String(l.id) === String(id)) || (this.data.libraries || []).find(l => String(l.id) === String(id));
    if (currentLib && (currentLib.visibility === 'public' || currentLib.isOfficial)) {
      wx.showModal({
        title: '不可删除',
        content: '该题库为公开题库，已面向全员开放，不可删除。',
        showCancel: false,
        confirmColor: '#0058bc'
      });
      return;
    }

    wx.showModal({
      title: '确认删除题库',
      content: '删除后将清空此题库的所有答题进度与错题记录，且无法恢复。是否确认删除？',
      confirmColor: '#ba1a1a',
      confirmText: '确认删除',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          // 真实后端模式：调用服务端删除接口
          if (!CONFIG.USE_MOCK && typeof id === 'number') {
            request({
              url: `/api/v1/banks/${id}`,
              method: 'DELETE'
            }).then(() => {
              this.allLibraries = (this.allLibraries || []).filter(item => item.id !== id);
              this.applyFilter();
              wx.showToast({ title: '题库已删除', icon: 'success' });
            }).catch((err) => {
              wx.showToast({ title: err.message || '删除失败', icon: 'none' });
            });
            return;
          }

          // 本地/Mock 降级模式：从缓存中删除
          this.allLibraries = (this.allLibraries || []).filter(item => item.id !== id);
          this.applyFilter();
          try {
            const customLibs = wx.getStorageSync('custom_libraries') || [];
            wx.setStorageSync('custom_libraries', customLibs.filter(item => item.id !== id));
          } catch (e) {
            console.log('删除缓存题库异常', e);
          }
          wx.showToast({ title: '题库已删除', icon: 'success' });
        }
      }
    });
  },

  onSwitchTab(e) {
    const tab = e.currentTarget.dataset.tab;
    if (tab === this.data.currentTab) return;

    if (tab === 'errors') {
      wx.redirectTo({
        url: '/pages/errors/errors'
      });
    } else if (tab === 'profile') {
      wx.redirectTo({
        url: '/pages/profile/profile'
      });
    }
  }
});