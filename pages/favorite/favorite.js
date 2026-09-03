const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    currentTab: 'profile',
    selectedCategory: 'all',
    selectedBankTitle: '',
    removingId: null,
    // 动态题库分类页签（完全根据真实存在的题库动态生成，不写死任何静态预置科目）
    categories: [
      { id: 'all', name: '全部题库' }
    ],
    // 真实题库分组
    allGroups: [],
    filteredGroups: []
  },

  allKnownBanks: [],

  onLoad() {
    this.initNavbarInfo();
    this.loadBanksAndFavorites();
  },

  onShow() {
    this.loadBanksAndFavorites();
  },

  /**
   * 加载真实题库与用户收藏列表
   */
  loadBanksAndFavorites() {
    if (!CONFIG.USE_MOCK) {
      Promise.all([
        request({ url: '/api/v1/banks' }).catch(() => null),
        request({ url: '/api/v1/favorites' }).catch(() => null)
      ]).then(([banksRes, favList]) => {
        const banks = banksRes && banksRes.list ? banksRes.list : (Array.isArray(banksRes) ? banksRes : []);
        this.allKnownBanks = banks;
        const bankMap = {};
        banks.forEach(b => {
          bankMap[String(b.id)] = b.title;
        });

        const list = Array.isArray(favList) ? favList : (favList && favList.list ? favList.list : []);
        if (list.length > 0) {
          // 按真实题库动态分组
          const groupsDict = {};
          list.forEach(item => {
            const q = item.question || {};
            const bId = String(item.bank_id || q.bank_id || '1');
            const bTitle = (item.bank && item.bank.title) || bankMap[bId] || (q.bank && q.bank.title) || '项目管理基础考试';

            if (!groupsDict[bId]) {
              groupsDict[bId] = {
                subjectId: bId,
                subjectName: bTitle,
                questions: [],
                rawQuestions: []
              };
            }

            let dateText = '刚刚';
            if (item.created_at) {
              try {
                const d = new Date(item.created_at);
                dateText = `${String(d.getMonth() + 1).padStart(2, '0')}月${String(d.getDate()).padStart(2, '0')}日`;
              } catch (e) {}
            }

            // 构造传给刷题页的完整试题对象
            const fullQ = {
              id: q.id || item.question_id,
              bank_id: parseInt(bId, 10) || 1,
              type: q.type || '单选',
              section: q.section || bTitle,
              title: q.title || '试题',
              options: Array.isArray(q.options) && q.options.length > 0 ? q.options : [
                { key: 'A', text: '正确选项' },
                { key: 'B', text: '错误选项' }
              ],
              answer: Array.isArray(q.answer) ? q.answer : ['A'],
              difficulty: q.difficulty || '中等',
              knowledgePoint: q.knowledge_point || q.section || '核心考点',
              analysis: q.analysis || '详见题目考点解析。',
              knowledgeDetail: q.knowledge_detail || '',
              is_bookmarked: true
            };

            groupsDict[bId].rawQuestions.push(fullQ);
            groupsDict[bId].questions.push({
              id: item.question_id || q.id,
              tag: q.knowledge_point || q.section || '核心考点',
              date: dateText,
              subjectId: bId,
              subjectName: bTitle,
              title: q.title || '试题'
            });
          });

          const allGroups = Object.values(groupsDict);
          this.updateCategoriesAndGroups(allGroups, banks);
          try {
            wx.setStorageSync('user_favorites_groups', allGroups);
          } catch (e) {}
          return;
        }

        this.loadLocalFavoritesOrEmpty(banks);
      }).catch((err) => {
        console.log('[Favorites] 加载云端收藏异常:', err);
        this.loadLocalFavoritesOrEmpty();
      });
      return;
    }

    this.loadLocalFavoritesOrEmpty();
  },

  loadLocalFavoritesOrEmpty(banks = []) {
    try {
      const cached = wx.getStorageSync('user_favorites_groups');
      if (Array.isArray(cached) && cached.length > 0) {
        // 清洗掉旧版本遗留的非真实生物学/化学等静态数据
        const cleaned = cached.filter(g => g.subjectId !== 'biology' && g.subjectId !== 'chemistry' && g.subjectId !== 'physics');
        if (cleaned.length > 0) {
          this.updateCategoriesAndGroups(cleaned, banks);
          return;
        }
      }
    } catch (e) {}

    // 无真实收藏时根据已知真实题库生成空分类
    this.updateCategoriesAndGroups([], banks);
  },

  /**
   * 动态聚合题库分类页签并刷新当前分组
   */
  updateCategoriesAndGroups(allGroups, banks = this.allKnownBanks || []) {
    const categories = [{ id: 'all', name: '全部题库' }];
    const seenIds = new Set(['all']);

    // 1. 优先将有收藏题目的真实题库加入分类页签
    allGroups.forEach(g => {
      if (!seenIds.has(String(g.subjectId))) {
        seenIds.add(String(g.subjectId));
        categories.push({ id: String(g.subjectId), name: g.subjectName });
      }
    });

    // 2. 补充系统现存的其他真实题库（保持页签完整）
    if (Array.isArray(banks)) {
      banks.forEach(b => {
        const bid = String(b.id);
        if (!seenIds.has(bid)) {
          seenIds.add(bid);
          categories.push({ id: bid, name: b.title });
        }
      });
    }

    // 3. 校验选中分类有效性
    let selected = this.data.selectedCategory;
    if (!categories.some(c => c.id === selected)) {
      selected = 'all';
    }

    const matchedCategory = categories.find(c => String(c.id) === String(selected));
    const selectedBankTitle = (selected && selected !== 'all' && matchedCategory) ? matchedCategory.name : '';

    this.setData({
      categories,
      selectedCategory: selected,
      selectedBankTitle,
      allGroups
    }, () => {
      this.updateFilteredGroups();
    });
  },

  /**
   * 初始化适配自定义导航栏高度
   */
  initNavbarInfo() {
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

  /**
   * 根据当前选中的题库分类过滤列表
   */
  updateFilteredGroups() {
    const { selectedCategory, allGroups } = this.data;
    let filtered = [];

    if (selectedCategory === 'all') {
      filtered = allGroups.filter(g => g.questions && g.questions.length > 0);
    } else {
      filtered = allGroups.filter(g => (String(g.subjectId) === String(selectedCategory) || g.subjectName === selectedCategory) && g.questions && g.questions.length > 0);
    }

    this.setData({
      filteredGroups: filtered
    });
  },

  /**
   * 切换题库分类 Chip
   */
  onSelectCategory(e) {
    const id = e.currentTarget.dataset.id;
    if (this.data.selectedCategory === id) return;

    const matched = this.data.categories.find(c => String(c.id) === String(id));
    const selectedBankTitle = (id && id !== 'all' && matched) ? matched.name : '';

    this.setData({
      selectedCategory: id,
      selectedBankTitle
    }, () => {
      this.updateFilteredGroups();
    });
  },

  /**
   * 返回上一页
   */
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

  /**
   * 移除单道试题收藏 (带动画与确认交互)
   */
  onRemoveBookmark(e) {
    const questionId = e.currentTarget.dataset.id;
    
    wx.showModal({
      title: '取消收藏',
      content: '确定要将该题从“我的收藏”中移除吗？',
      confirmText: '确定移除',
      confirmColor: '#ba1a1a',
      cancelText: '保留',
      success: (res) => {
        if (res.confirm) {
          // 触发淡出缩放动画
          this.setData({ removingId: questionId });

          setTimeout(() => {
            if (!CONFIG.USE_MOCK && typeof questionId === 'number') {
              request({
                url: '/api/v1/favorites/toggle',
                method: 'POST',
                data: { question_id: questionId }
              }).catch((err) => {
                console.log('[Favorites] 取消收藏接口异常:', err);
              });
            }

            const newGroups = this.data.allGroups.map(group => {
              return {
                ...group,
                questions: group.questions.filter(q => q.id !== questionId)
              };
            });

            this.setData({
              allGroups: newGroups,
              removingId: null
            }, () => {
              this.updateFilteredGroups();
              wx.showToast({
                title: '已取消收藏',
                icon: 'success',
                duration: 1500
              });
            });
          }, 250);
        }
      }
    });
  },

  /**
   * 单题练习（从该题开始，但在所属题库的全部收藏题中刷题）
   */
  onPracticeQuestion(e) {
    const item = e.currentTarget.dataset.item;
    if (!item) return;

    const bankId = String(item.subjectId || '1');
    const group = this.data.allGroups.find(g => String(g.subjectId) === bankId || g.subjectName === item.subjectName);

    if (group && group.questions && group.questions.length > 0) {
      const targetIndex = group.questions.findIndex(q => q.id === item.id);
      const safeIndex = targetIndex >= 0 ? targetIndex : 0;
      const questionsToPractice = (group.rawQuestions && group.rawQuestions.length > 0) ? group.rawQuestions : group.questions.map(q => ({
        id: q.id,
        bank_id: parseInt(bankId, 10) || 1,
        type: '单选',
        section: group.subjectName,
        title: q.title,
        options: [
          { key: 'A', text: '选项 A' },
          { key: 'B', text: '选项 B' },
          { key: 'C', text: '选项 C' },
          { key: 'D', text: '选项 D' }
        ],
        answer: ['A'],
        difficulty: '中等',
        knowledgePoint: q.tag || '考点',
        analysis: '题目解析',
        is_bookmarked: true
      }));

      try {
        wx.setStorageSync('practice_custom_questions', questionsToPractice);
      } catch (err) {
        console.log('保存专属练习题目失败', err);
      }

      wx.navigateTo({
        url: `/pages/quiz/quiz?type=favorite&mode=practice&bankId=${bankId}&index=${safeIndex}&subject=${encodeURIComponent(group.subjectName)}&title=${encodeURIComponent(group.subjectName + ' - 收藏专练')}`
      });
      return;
    }

    // 兜底
    wx.navigateTo({
      url: `/pages/quiz/quiz?questionId=${item.id}&bankId=${bankId}&tag=${encodeURIComponent(item.tag || '')}`
    });
  },

  /**
   * 点击试题卡片
   */
  onTapCard(e) {
    this.onPracticeQuestion(e);
  },

  /**
   * 题库一键练习（专项练习该题库已收藏的全部题目）
   */
  onPracticeSubject(e) {
    const subject = e.currentTarget.dataset.subject;
    const group = this.data.allGroups.find(g => g.subjectName === subject || String(g.subjectId) === String(subject));
    if (!group || !group.questions || group.questions.length === 0) {
      wx.showToast({ title: '暂无收藏题目', icon: 'none' });
      return;
    }

    const bankId = group.subjectId || '1';
    const questionsToPractice = (group.rawQuestions && group.rawQuestions.length > 0) ? group.rawQuestions : group.questions.map(q => ({
      id: q.id,
      bank_id: parseInt(bankId, 10) || 1,
      type: '单选',
      section: group.subjectName,
      title: q.title,
      options: [
        { key: 'A', text: '选项 A' },
        { key: 'B', text: '选项 B' },
        { key: 'C', text: '选项 C' },
        { key: 'D', text: '选项 D' }
      ],
      answer: ['A'],
      difficulty: '中等',
      knowledgePoint: q.tag || '考点',
      analysis: '题目解析',
      is_bookmarked: true
    }));

    try {
      wx.setStorageSync('practice_custom_questions', questionsToPractice);
    } catch (err) {
      console.log('保存专属练习题目失败', err);
    }

    wx.showToast({
      title: `生成 ${group.subjectName} 专项练习 (${questionsToPractice.length}题)...`,
      icon: 'none',
      duration: 1200
    });

    setTimeout(() => {
      wx.navigateTo({
        url: `/pages/quiz/quiz?type=favorite&mode=practice&bankId=${bankId}&index=0&subject=${encodeURIComponent(group.subjectName)}&title=${encodeURIComponent(group.subjectName + ' - 收藏专练')}`
      });
    }, 400);
  },

  /**
   * 空状态引导去刷题 (跳转至当前选中的题库进行全真刷题)
   */
  onGoPractice() {
    const { selectedCategory, categories } = this.data;
    let targetBankId = '';
    let targetTitle = '';

    if (selectedCategory && selectedCategory !== 'all') {
      const matched = categories.find(c => String(c.id) === String(selectedCategory));
      if (matched) {
        targetBankId = matched.id;
        targetTitle = matched.name;
      }
    }

    // 若当前处于“全部题库”或未匹配到，则默认取第一个真实题库
    if (!targetBankId) {
      const firstBank = categories.find(c => c.id !== 'all');
      if (firstBank) {
        targetBankId = firstBank.id;
        targetTitle = firstBank.name;
      } else {
        targetBankId = '1';
        targetTitle = '项目管理基础考试';
      }
    }

    wx.navigateTo({
      url: `/pages/quiz/quiz?bankId=${targetBankId}&title=${encodeURIComponent(targetTitle)}`
    });
  },

  /**
   * 底部 Tab 切换
   */
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
    } else if (tab === 'profile') {
      wx.redirectTo({
        url: '/pages/profile/profile'
      });
    }
  }
});
