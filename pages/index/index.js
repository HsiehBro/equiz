const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    searchKeyword: '',
    selectedCategory: '',
    currentTab: 'library',
    currentUserRole: 'user',
    isVip: false,
    categoryList: [
      {
        name: '医学类',
        desc: '执业医师 / 药师',
        icon: '/assets/icons/medical_services_primary.svg',
        bgClass: 'bg-blue-light',
        isVip: false
      },
      {
        name: '财经类',
        desc: 'CPA / 会计初级',
        icon: '/assets/icons/account_balance_gray.svg',
        bgClass: 'bg-gray-light',
        isVip: true
      },
      {
        name: 'IT互联网',
        desc: '软考 / PMP',
        icon: '/assets/icons/terminal_gray.svg',
        bgClass: 'bg-gray-light',
        isVip: true
      },
      {
        name: '全部分类',
        desc: '查看全部题库',
        icon: '/assets/icons/grid_view_gray.svg',
        bgClass: 'bg-gray-light',
        isVip: false
      }
    ],
    showCategoryDrawer: false,
    isDrawerExpanded: false,
    isDraggingDrawer: false,
    drawerHeightStyle: '',
    allCategories: [],
    libraries: []
  },

  allLibraries: [],

  onLoad() {
    this.initNavBar();
    this.initDrawerDimensions();
    // 页面初始化即刻装载预设题库，确保搜索与过滤即开即用
    this.allLibraries = this.getDefaultLibrariesPreset();
    this.loadCategories();
    this.syncCategoryVipStatus();
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
      const userRole = userInfo.role || 'user';
      const rawCustom = wx.getStorageSync('custom_libraries');
      const customLibs = Array.isArray(rawCustom) ? rawCustom : [];

      // 题库权限隔离：
      // 管理员可查看全部题库；
      // 普通/VIP用户仅能查看：官方题库、审核通过的公开题库、本人创建的题库（包括审核中与私有）
      const visibleCustom = customLibs.filter(lib => {
        if (userRole === 'admin') return true;
        const isOwner = !lib.creatorId || String(lib.creatorId) === String(currentUserId);
        if (isOwner) return true;
        if (lib.isOfficial) return true;
        return lib.visibility === 'public' && (lib.reviewStatus === 'approved' || !lib.reviewStatus);
      });

      const defaultLibs = [
        {
          id: 1,
          title: '项目管理基础考试',
          category: 'IT互联网',
          description: 'PMP/高项核心考点，包含单选与多选题 (VIP专属)',
          totalCount: 100,
          progress: 0,
          lastPractice: '未开始',
          visibility: 'public',
          reviewStatus: 'approved',
          isOfficial: true,
          isVip: true,
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
          reviewStatus: 'approved',
          isOfficial: true,
          isVip: false,
          isPrimary: false
        },
        {
          id: 3,
          title: '初级会计实务 - 核心考点',
          category: '财经类',
          description: '初级会计实务真题演练与必背知识点 (VIP专属)',
          totalCount: 40,
          progress: 45,
          lastPractice: '3天前',
          visibility: 'public',
          reviewStatus: 'approved',
          isOfficial: true,
          isVip: true,
          isPrimary: false
        }
      ];
      if (visibleCustom.length > 0) {
        const customIds = new Set(visibleCustom.map(l => l.id));
        const formattedCustom = visibleCustom.map(l => ({
          ...l,
          isOwner: !l.creatorId || String(l.creatorId) === String(currentUserId)
        }));
        return [...formattedCustom, ...defaultLibs.filter(l => !customIds.has(l.id))];
      }
      return defaultLibs;
    } catch (e) {
      return [];
    }
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
    const userInfo = wx.getStorageSync('user_info') || {};
    const isLifetime = wx.getStorageSync('user_is_lifetime_vip') || userInfo.isLifetimeVip || userInfo.role === 'admin';
    if (isLifetime) {
      wx.showModal({
        title: '温馨提示',
        content: '您已开通永久 VIP 会员，享有全站最高终身特权，不能参加任何会员活动！',
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

  initDrawerDimensions() {
    try {
      const windowInfo = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
      const windowHeight = windowInfo.windowHeight || 700;
      this.windowHeight = windowHeight;
      this.minDrawerHeight = Math.round(windowHeight * 0.58);
      this.maxDrawerHeight = Math.round(windowHeight * 0.90);
      this.currentDrawerHeight = this.minDrawerHeight;
    } catch (e) {
      this.windowHeight = 700;
      this.minDrawerHeight = 420;
      this.maxDrawerHeight = 630;
      this.currentDrawerHeight = 420;
    }
  },

  onSelectCategory(e) {
    const category = e.currentTarget.dataset.category;
    if (category === '全部分类') {
      // 点击「全部分类」卡片，呼出全部分类抽屉弹窗
      this.onOpenCategoryDrawer();
      return;
    }
    if (this.data.selectedCategory === category) {
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

  /**
   * 清除当前选中的分类筛选（仅重置筛选状态，绝不弹出全部分类抽屉）
   */
  onClearCategoryFilter() {
    this.setData({ selectedCategory: '' }, () => {
      this.applyFilter();
      wx.showToast({ title: '已清除分类筛选', icon: 'none', duration: 1200 });
    });
  },

  onOpenCategoryDrawer() {
    if (!this.windowHeight) {
      this.initDrawerDimensions();
    }
    this.currentDrawerHeight = this.minDrawerHeight;
    this.setData({
      showCategoryDrawer: true,
      isDrawerExpanded: false,
      isDraggingDrawer: false,
      drawerHeightStyle: `height: ${this.minDrawerHeight}px;`
    });
  },

  onCloseCategoryDrawer() {
    this.setData({
      showCategoryDrawer: false,
      isDrawerExpanded: false,
      isDraggingDrawer: false,
      drawerHeightStyle: ''
    });
  },

  /**
   * 顶部把手触摸开始：记录按压起点位置与当前高度
   */
  onDrawerTouchStart(e) {
    if (!e.touches || e.touches.length === 0) return;
    this.touchStartY = e.touches[0].clientY;
    this.touchStartHeight = this.currentDrawerHeight || this.minDrawerHeight;
    this.touchMoveDistance = 0;
    this.setData({ isDraggingDrawer: true });
  },

  /**
   * 顶部把手拖动中：实时向上或向下跟随手指拉升/压缩高度
   */
  onDrawerTouchMove(e) {
    if (!e.touches || e.touches.length === 0) return;
    const currentY = e.touches[0].clientY;
    const diffY = this.touchStartY - currentY; // 向上拖动为正，向下拖动为负
    this.touchMoveDistance = diffY;

    let targetHeight = this.touchStartHeight + diffY;
    const minLimit = Math.round((this.windowHeight || 700) * 0.40);
    const maxLimit = Math.round((this.windowHeight || 700) * 0.93);
    if (targetHeight < minLimit) targetHeight = minLimit;
    if (targetHeight > maxLimit) targetHeight = maxLimit;

    this.currentDrawerHeight = targetHeight;
    this.setData({
      drawerHeightStyle: `height: ${targetHeight}px;`
    });
  },

  /**
   * 顶部把手触摸松开：根据拖动距离与当前拉升高度自动磁吸吸附
   */
  onDrawerTouchEnd() {
    this.setData({ isDraggingDrawer: false });
    const diffY = this.touchMoveDistance || 0;

    // 1. 如果只是单纯轻点把手区域（位移很小），切换展开/折叠
    if (Math.abs(diffY) < 8) {
      this.toggleDrawerExpand();
      return;
    }

    // 2. 向上滑动超 35px 或当前高度超 70% 屏幕，向上吸附到 90% 满屏大抽屉
    const snapThreshold = Math.round((this.windowHeight || 700) * 0.70);
    if (diffY > 35 || this.currentDrawerHeight > snapThreshold) {
      this.currentDrawerHeight = this.maxDrawerHeight;
      this.setData({
        isDrawerExpanded: true,
        drawerHeightStyle: `height: ${this.maxDrawerHeight}px;`
      });
    } else if (diffY < -75 && this.touchStartHeight <= this.minDrawerHeight + 20) {
      // 3. 在折叠状态下较大幅度向下猛滑，直接关闭抽屉
      this.onCloseCategoryDrawer();
    } else {
      // 4. 其余情况磁吸复位至初始半屏默认高度
      this.currentDrawerHeight = this.minDrawerHeight;
      this.setData({
        isDrawerExpanded: false,
        drawerHeightStyle: `height: ${this.minDrawerHeight}px;`
      });
    }
  },

  /**
   * 轻点把手区域切换 60% 默认高度与 90% 展开高度
   */
  toggleDrawerExpand() {
    const nextExpanded = !this.data.isDrawerExpanded;
    this.currentDrawerHeight = nextExpanded ? this.maxDrawerHeight : this.minDrawerHeight;
    this.setData({
      isDrawerExpanded: nextExpanded,
      drawerHeightStyle: `height: ${this.currentDrawerHeight}px;`
    });
  },

  onSelectDrawerCategory(e) {
    const category = e.currentTarget.dataset.category;
    if (!category || category === '全部分类') {
      this.setData({
        selectedCategory: '',
        showCategoryDrawer: false
      }, () => {
        this.applyFilter();
        wx.showToast({ title: '已显示全部题库', icon: 'none', duration: 1200 });
      });
    } else {
      this.setData({
        selectedCategory: category,
        showCategoryDrawer: false
      }, () => {
        this.applyFilter();
        wx.showToast({ title: `已按 [${category}] 筛选`, icon: 'none', duration: 1200 });
      });
    }
  },

  onShow() {
    const userInfo = wx.getStorageSync('user_info') || {};
    const userRole = userInfo.role || 'user';
    const rawVip = wx.getStorageSync('user_is_vip');
    const isVip = userRole === 'admin' || userRole === 'vip' || (userRole !== 'user' && rawVip === true);
    this.setData({
      currentUserRole: userRole,
      isVip: Boolean(isVip)
    });
    this.loadCategories();
    this.loadLibraries();
  },

  loadCategories() {
    let fallbackList = [
      { id: 2, name: '医学类', desc: '执业医师 / 药师', icon: '/assets/icons/medical_services_primary.svg', bgClass: 'bg-blue-light', isVip: false },
      { id: 3, name: '财经类', desc: 'CPA / 会计初级', icon: '/assets/icons/account_balance_gray.svg', bgClass: 'bg-gray-light', isVip: true },
      { id: 4, name: 'IT互联网', desc: '软考 / PMP', icon: '/assets/icons/terminal_gray.svg', bgClass: 'bg-gray-light', isVip: true },
      { id: 1, name: '综合', desc: '通用与综合测试', icon: '/assets/icons/grid_view_gray.svg', bgClass: 'bg-gray-light', isVip: false }
    ];

    try {
      const customCats = wx.getStorageSync('custom_categories');
      if (Array.isArray(customCats) && customCats.length > 0) {
        fallbackList = customCats
          .filter(c => c.is_active !== false)
          .map(c => ({
            id: c.id,
            name: c.name,
            desc: c.description || c.desc || '专业题库',
            icon: c.icon || '/assets/icons/folder_primary.svg',
            bgClass: c.bg_class || c.bgClass || 'bg-gray-light',
            isVip: Boolean(c.is_vip || c.has_vip || c.isVip),
            bankCount: c.bank_count || 0
          }));
      }
    } catch (e) {}

    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/categories', needAuth: false })
        .then((cats) => {
          if (Array.isArray(cats) && cats.length > 0) {
            this.setCategoriesData(cats);
            return;
          }
          this.setCategoriesData(fallbackList);
        })
        .catch(() => {
          this.setCategoriesData(fallbackList);
        });
    } else {
      this.setCategoriesData(fallbackList);
    }
  },

  setCategoriesData(rawCats) {
    const formattedAll = rawCats.map(c => ({
      id: c.id,
      name: c.name,
      desc: c.description || c.desc || '专业题库',
      icon: c.icon || '/assets/icons/folder_primary.svg',
      bgClass: c.bg_class || c.bgClass || 'bg-gray-light',
      isVip: Boolean(c.is_vip || c.has_vip || c.isVip),
      bankCount: c.bank_count || 0
    }));

    // 筛选前 3 个优先分类卡片（优先非“综合”与非“全部分类”，若不足则顺延）
    const topThreeCandidates = formattedAll.filter(c => c.name !== '全部分类');
    topThreeCandidates.sort((a, b) => {
      if (a.name === '综合') return 1;
      if (b.name === '综合') return -1;
      return 0;
    });
    const topThree = topThreeCandidates.slice(0, 3);

    // 第 4 个固定为「全部分类」卡片
    const bentoCards = [
      ...topThree,
      {
        id: 0,
        name: '全部分类',
        desc: `查看全部 ${formattedAll.length} 个分类`,
        icon: '/assets/icons/grid_view_gray.svg',
        bgClass: 'bg-gray-light',
        isVip: false
      }
    ];

    this.setData({
      allCategories: formattedAll,
      categoryList: bentoCards
    }, () => {
      this.syncCategoryVipStatus();
    });
  },

  loadLibraries() {
    if (!CONFIG.USE_MOCK) {
      const userInfo = wx.getStorageSync('user_info') || {};
      const currentUserId = userInfo.id || userInfo.open_id || userInfo.dev_user_id || 'dev_user_001';

      // 真实后端模式：从 Go API + PostgreSQL 获取题库（自动附带 Token 进行私有题库与VIP隔离）
      request({ url: '/api/v1/banks', needAuth: false })
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
              reviewStatus: b.review_status || 'approved',
              creatorId: b.creator_id,
              isOwner: !b.creator_id || String(b.creator_id) === String(currentUserId),
              isVip: Boolean(b.is_vip),
              isOfficial: Boolean(b.is_official),
              isPrimary: idx === 0
            }));
            this.allLibraries = formatted;
            this.syncCategoryVipStatus();
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
      this.syncCategoryVipStatus();
      this.applyFilter();
    } catch (e) {
      console.log('读取题库缓存异常', e);
    }
  },

  /**
   * 同步分类题库的 VIP 金标状态与题库数统计：
   * 只要该分类下存在任意 VIP 题库，该分类即展示 VIP 金标；若无，则自动隐藏 VIP 金标。
   */
  syncCategoryVipStatus() {
    const allLibs = Array.isArray(this.allLibraries) && this.allLibraries.length > 0
      ? this.allLibraries
      : this.getDefaultLibrariesPreset();

    const updatedAll = (this.data.allCategories || []).map(cat => {
      let count = 0;
      let hasVip = false;
      const catName = (cat.name || '').toLowerCase();
      allLibs.forEach(lib => {
        const libCat = (lib.category || '').toLowerCase();
        let match = false;
        if (cat.name === '医学类') {
          match = libCat.includes('医学') || libCat.includes('医');
        } else if (cat.name === '财经类') {
          match = libCat.includes('财经') || libCat.includes('会计') || libCat.includes('财');
        } else if (cat.name === 'IT互联网') {
          match = libCat.includes('it') || libCat.includes('工程') || libCat.includes('互联网');
        } else {
          match = libCat === catName || libCat.includes(catName);
        }
        if (match) {
          count++;
          if (lib.isVip || lib.is_vip) {
            hasVip = true;
          }
        }
      });
      return {
        ...cat,
        bankCount: count,
        isVip: hasVip
      };
    });

    const updatedList = (this.data.categoryList || []).map(cat => {
      if (cat.name === '全部分类') {
        return {
          ...cat,
          desc: `查看全部 ${updatedAll.length} 个分类`
        };
      }
      const matched = updatedAll.find(c => c.name === cat.name);
      return {
        ...cat,
        isVip: matched ? matched.isVip : false,
        bankCount: matched ? matched.bankCount : 0
      };
    });

    this.setData({
      allCategories: updatedAll,
      categoryList: updatedList
    });
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

  /**
   * 点击「查看全部」：呼出全部分类抽屉弹窗供用户查看与选择
   */
  onViewAllLibraries() {
    this.onOpenCategoryDrawer();
  },

  /**
   * 搜索无结果空状态点击「查看全部题库」：重置搜索词与分类筛选
   */
  onResetEmptySearch() {
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
    const currentLib = (this.allLibraries || []).find(l => String(l.id) === String(id)) || (this.data.libraries || []).find(l => String(l.id) === String(id));
    const title = currentLib ? currentLib.title : '指定题库';

    // 严谨判断该题库是否为 VIP 专属题库
    const isVipBank = Boolean(currentLib && (currentLib.isVip === true || currentLib.is_vip === true));

    // 严谨判断当前用户是否为 VIP：管理员 admin 具有永久 VIP；角色为 vip 拥有 VIP；普通用户 user 绝非 VIP
    const userInfo = wx.getStorageSync('user_info') || {};
    const userRole = userInfo.role || this.data.currentUserRole || 'user';
    const rawVipStorage = wx.getStorageSync('user_is_vip');
    const isUserVip = userRole === 'admin' || userRole === 'vip' || (userRole !== 'user' && rawVipStorage === true);

    // 普通用户点击带 VIP 标识的题库时，直接跳转至会员解锁页面！
    if (isVipBank && !isUserVip) {
      wx.showToast({
        title: '该题库为VIP专属，正在前往会员解锁...',
        icon: 'none',
        duration: 1800
      });
      setTimeout(() => {
        wx.navigateTo({
          url: '/pages/vip/vip'
        });
      }, 200);
      return;
    }

    wx.navigateTo({
      url: `/pages/quiz/quiz?mode=practice&bankId=${id}&title=${encodeURIComponent(title)}`
    });
  },

  /**
   * 首页快捷申请公开私有题库
   */
  onApplyPublic(e) {
    const { id, title } = e.currentTarget.dataset;
    const currentLib = (this.allLibraries || []).find(l => String(l.id) === String(id));
    if (!currentLib) return;

    const userInfo = wx.getStorageSync('user_info') || {};
    const currentUserId = userInfo.id || userInfo.open_id || userInfo.dev_user_id || 'dev_user_003';
    const userName = userInfo.nickname || '备考学员';

    // 校验单次公开题库上传/申请限制：每次只能有1个待审题库，审核通过后才能再次提交
    const allLibs = Array.isArray(this.allLibraries) && this.allLibraries.length > 0 
      ? this.allLibraries 
      : (wx.getStorageSync('custom_libraries') || []);
    
    const hasPending = allLibs.some(l => {
      const isOwner = !l.creatorId || String(l.creatorId) === String(currentUserId);
      return isOwner && l.visibility === 'public' && l.reviewStatus === 'pending';
    });

    if (hasPending) {
      wx.showModal({
        title: '申请限制提示',
        content: '公开题库每次只能提交 1 个审核！\n您当前已有题库正在等待管理员审批，由 Admin 审批通过后方可再次提交公开申请（累计申请与上传次数不限）。',
        showCancel: false,
        confirmText: '我知道了',
        confirmColor: '#ba1a1a'
      });
      return;
    }

    wx.showModal({
      title: '提交公开申请',
      content: `确认申请将私有题库《${title}》面向全员公开？\n\n提交后将进入【待审批】状态（期间保持为您个人私有）。管理员审核通过后将正式公开，并向您的消息中心推送通过提示。`,
      confirmText: '确认申请',
      cancelText: '再想想',
      confirmColor: '#0058bc',
      success: (res) => {
        if (res.confirm) {
          this.executeApplyPublic(id, title, currentUserId, userName);
        }
      }
    });
  },

  executeApplyPublic(id, title, currentUserId, userName) {
    wx.showLoading({ title: '正在提交申请...' });

    // 真实后端模式
    if (!CONFIG.USE_MOCK && typeof id === 'number') {
      request({
        url: `/api/v1/banks/${id}/apply-public`,
        method: 'POST',
        data: { user_name: userName }
      }).then(() => {
        wx.hideLoading();
        wx.showModal({
          title: '申请已提交',
          content: `题库《${title}》已成功提交公开申请并进入【待审核】状态。\n管理员已收到微信通知提醒，审批通过后将自动面向全员公开！`,
          showCancel: false,
          confirmText: '我知道了',
          confirmColor: '#0058bc'
        });
        this.loadLibraries();
      }).catch((err) => {
        wx.hideLoading();
        wx.showToast({ title: err.message || '提交失败', icon: 'none' });
      });
      return;
    }

    // 本地/Mock 降级模式
    setTimeout(() => {
      wx.hideLoading();
      try {
        const customLibs = wx.getStorageSync('custom_libraries') || [];
        const updated = customLibs.map(l => {
          if (String(l.id) === String(id)) {
            return {
              ...l,
              visibility: 'public',
              reviewStatus: 'pending',
              creatorId: currentUserId,
              creatorName: userName
            };
          }
          return l;
        });
        wx.setStorageSync('custom_libraries', updated);

        // 向管理员微信通知池推入提醒
        const adminNotices = wx.getStorageSync('admin_system_notifications') || [];
        adminNotices.unshift({
          id: 'admin_notice_' + Date.now(),
          type: 'admin_pending',
          title: '【微信服务通知】有新的公开题库待审批',
          content: `用户「${userName}」申请将私有题库《${title}》公开，请前往个人中心手动审批栏进行审核。`,
          time: '刚刚',
          isRead: false
        });
        wx.setStorageSync('admin_system_notifications', adminNotices);
      } catch (e) {}

      wx.showModal({
        title: '申请已提交',
        content: `题库《${title}》已成功提交公开申请并进入【待审核】状态。\n管理员已收到微信通知提醒，审批通过后将自动面向全员公开！`,
        showCancel: false,
        confirmText: '我知道了',
        confirmColor: '#0058bc'
      });
      this.loadLibraries();
    }, 400);
  },

  onDeleteLibrary(e) {
    const id = e.currentTarget.dataset.id;
    const currentLib = (this.allLibraries || []).find(l => String(l.id) === String(id)) || (this.data.libraries || []).find(l => String(l.id) === String(id));
    const title = currentLib ? currentLib.title : '指定题库';
    const isAdmin = this.data.currentUserRole === 'admin';

    // 官方题库不可删除
    if (currentLib && currentLib.isOfficial) {
      wx.showModal({
        title: '不可删除',
        content: '系统预设官方题库不允许删除。',
        showCancel: false,
        confirmColor: '#0058bc'
      });
      return;
    }

    // 非管理员不能删除已公开题库
    if (!isAdmin && currentLib && (currentLib.visibility === 'public' || currentLib.isOfficial)) {
      wx.showModal({
        title: '不可删除',
        content: '该题库为公开题库，已面向全员开放，不可删除（仅管理员拥有删除公开题库权限）。',
        showCancel: false,
        confirmColor: '#0058bc'
      });
      return;
    }

    const modalContent = isAdmin && currentLib && (currentLib.visibility === 'public' || currentLib.isOfficial)
      ? `【管理员操作】确认删除公开题库《${title}》？\n删除后将面向全员下线并清空全部答题与错题数据，且无法恢复！`
      : `确认删除题库《${title}》？\n删除后将清空此题库的所有答题进度与错题记录，且无法恢复。是否确认删除？`;

    wx.showModal({
      title: isAdmin && currentLib && currentLib.visibility === 'public' ? '管理员删除公开题库' : '确认删除题库',
      content: modalContent,
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
              this.allLibraries = (this.allLibraries || []).filter(item => String(item.id) !== String(id));
              this.applyFilter();
              wx.showToast({ title: '题库已删除', icon: 'success' });
            }).catch((err) => {
              wx.showToast({ title: err.message || '删除失败', icon: 'none' });
            });
            return;
          }

          // 本地/Mock 降级模式：从缓存中删除
          this.allLibraries = (this.allLibraries || []).filter(item => String(item.id) !== String(id));
          this.applyFilter();
          try {
            const customLibs = wx.getStorageSync('custom_libraries') || [];
            wx.setStorageSync('custom_libraries', customLibs.filter(item => String(item.id) !== String(id)));
          } catch (e) {
            console.log('删除缓存题库异常', e);
          }
          wx.showToast({ title: '题库已删除', icon: 'success' });
        }
      }
    });
  },

  onToggleVipBank(e) {
    const { id, title, isvip } = e.currentTarget.dataset;
    const nextIsVip = !isvip;
    const actionText = nextIsVip ? '设为 VIP 专属题库' : '取消 VIP 专属标识';

    wx.showModal({
      title: 'VIP 题库设置',
      content: `【管理员操作】确认将《${title}》${actionText}？\n\n${nextIsVip ? '设置后所有用户均可见题库，但只有 VIP 用户与管理员可以刷题。' : '取消后所有用户均可自由作答刷题。'}`,
      confirmText: '确认设置',
      confirmColor: '#0058bc',
      success: (res) => {
        if (res.confirm) {
          this.executeToggleVip(id, nextIsVip);
        }
      }
    });
  },

  executeToggleVip(id, nextIsVip) {
    if (!CONFIG.USE_MOCK && typeof id === 'number') {
      request({
        url: `/api/v1/admin/banks/${id}/toggle-vip`,
        method: 'POST'
      }).then(() => {
        wx.showToast({ title: nextIsVip ? '已设为 VIP 题库' : '已取消 VIP 标识', icon: 'success' });
        this.loadLibraries();
      }).catch((err) => {
        wx.showToast({ title: err.message || '操作失败', icon: 'none' });
      });
      return;
    }

    // Mock / 本地更新
    try {
      const customLibs = wx.getStorageSync('custom_libraries') || [];
      const updated = customLibs.map(l => {
        if (String(l.id) === String(id)) {
          return { ...l, isVip: nextIsVip, is_vip: nextIsVip };
        }
        return l;
      });
      wx.setStorageSync('custom_libraries', updated);
    } catch (e) {}

    this.allLibraries = (this.allLibraries || []).map(l => {
      if (String(l.id) === String(id)) {
        return { ...l, isVip: nextIsVip, is_vip: nextIsVip };
      }
      return l;
    });
    this.syncCategoryVipStatus();
    this.applyFilter();
    wx.showToast({ title: nextIsVip ? '已设为 VIP 题库' : '已取消 VIP 标识', icon: 'success' });
  },

  onSwitchTab(e) {
    const tab = e.currentTarget.dataset.tab;
    if (tab === this.data.currentTab) return;

    if (tab === 'errors') {
      wx.navigateTo({
        url: '/pages/errors/errors'
      });
    } else if (tab === 'profile') {
      wx.navigateTo({
        url: '/pages/profile/profile'
      });
    }
  },

  preventDumbTap() {
    // 阻止底层蒙层点击穿透
  }
});