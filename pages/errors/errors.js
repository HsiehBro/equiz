const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    currentTab: 'errors',
    searchKeyword: '',
    subjects: [],
    filteredSubjects: [],
    totalErrorCount: 0,
    isSearching: false,
    searchMatchTotal: 0,
    isLoading: true
  },

  onLoad() {
    this.initNavBar();
  },

  onShow() {
    this.loadRealBanksAndErrors();
  },

  /**
   * 初始化适配导航栏高度
   */
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
        navBarHeight: Math.max(navBarHeight, 44)
      });
    } catch (e) {
      console.log('获取导航栏信息异常', e);
    }
  },

  /**
   * 加载系统真实题库与用户未掌握错题统计
   */
  loadRealBanksAndErrors() {
    this.setData({ isLoading: true });

    if (!CONFIG.USE_MOCK) {
      Promise.all([
        request({ url: '/api/v1/banks' }).catch(() => null),
        request({ url: '/api/v1/errors' }).catch(() => null)
      ]).then(([banksRes, errorsRes]) => {
        let banks = Array.isArray(banksRes) ? banksRes : (banksRes && banksRes.list ? banksRes.list : []);
        let errors = Array.isArray(errorsRes) ? errorsRes : (errorsRes && errorsRes.list ? errorsRes.list : []);

        if (banks.length === 0) {
          banks = this.getLocalBanksFallback();
        }

        this.processBanksAndErrors(banks, errors);
      }).catch((err) => {
        console.log('[Errors] 加载后端题库与错题失败，使用本地缓存:', err);
        this.loadLocalFallback();
      });
      return;
    }

    this.loadLocalFallback();
  },

  /**
   * 聚合统计真实题库下的错题数及全部未掌握错题快照
   */
  processBanksAndErrors(banks, errors) {
    // 仅统计未掌握的错题
    const activeErrors = (errors || []).filter(item => !item.is_mastered);
    this.allActiveErrors = activeErrors;
    this.allBanks = banks;

    const countByBank = {};
    activeErrors.forEach(item => {
      const bid = String(item.bank_id);
      countByBank[bid] = (countByBank[bid] || 0) + 1;
    });

    let totalCount = 0;
    const subjects = banks.map(b => {
      const count = countByBank[String(b.id)] || 0;
      totalCount += count;

      let icon = '/assets/icons/assignment_primary.svg';
      const titleLower = (b.title || '').toLowerCase();
      const catLower = (b.category || '').toLowerCase();

      if (titleLower.includes('医') || titleLower.includes('护士') || catLower.includes('医')) {
        icon = '/assets/icons/medical_services_primary.svg';
      } else if (titleLower.includes('会') || titleLower.includes('财') || catLower.includes('财') || catLower.includes('会')) {
        icon = '/assets/icons/account_balance_primary.svg';
      }

      return {
        id: String(b.id),
        name: b.title,
        desc: b.description || `${b.title}核心错题集锦。`,
        count: count,
        totalCount: count,
        icon: icon,
        hasErrors: count > 0,
        matchingQuestions: []
      };
    });

    this.setData({
      subjects,
      totalErrorCount: totalCount,
      isLoading: false
    }, () => {
      this.filterSubjects();
    });
  },

  getLocalBanksFallback() {
    const customLibs = wx.getStorageSync('custom_libraries') || [];
    const defaultLibs = [
      { id: '1', title: '项目管理基础考试', category: 'IT互联网', description: '项目生命周期、风险管理与进度压缩错题。' },
      { id: '2', title: '2023年护士执业资格考试', category: '医学卫生', description: '全国护士执业资格考试核心专业实务与实践能力错题。' },
      { id: '3', title: '初级会计实务 - 核心考点', category: '财会经济', description: '会计基础理论、资产核算与财务报表错题汇总。' }
    ];
    return [...customLibs, ...defaultLibs.filter(d => !customLibs.some(c => String(c.id) === String(d.id)))];
  },

  loadLocalFallback() {
    const banks = this.getLocalBanksFallback();
    const errorsMap = wx.getStorageSync('user_errors_map') || {};
    const errorList = Object.values(errorsMap);
    this.processBanksAndErrors(banks, errorList);
  },

  onNavBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack();
    } else {
      wx.redirectTo({
        url: '/pages/index/index'
      });
    }
  },

  onMoreOptions() {
    wx.showActionSheet({
      itemList: ['清空已掌握错题', '导出错题集 (PDF)', '按错题数量排序'],
      success: (res) => {
        if (res.tapIndex === 0) {
          wx.showModal({
            title: '确认清空',
            content: '确定要清空所有已掌握的错题记录吗？',
            confirmColor: '#ba1a1a',
            success: (modalRes) => {
              if (modalRes.confirm) {
                wx.showToast({ title: '已清空已掌握错题', icon: 'success' });
              }
            }
          });
        } else if (res.tapIndex === 1) {
          wx.showLoading({ title: '正在生成 PDF...' });
          setTimeout(() => {
            wx.hideLoading();
            wx.showToast({ title: '导出成功，已保存至本地', icon: 'success' });
          }, 1000);
        } else if (res.tapIndex === 2) {
          const sorted = [...this.data.filteredSubjects].sort((a, b) => b.count - a.count);
          this.setData({ filteredSubjects: sorted });
          wx.showToast({ title: '已按错题数降序排列', icon: 'none' });
        }
      }
    });
  },

  onSearchInput(e) {
    const keyword = e.detail.value;
    this.setData({ searchKeyword: keyword }, () => {
      this.filterSubjects();
    });
  },

  onClearSearch() {
    this.setData({ searchKeyword: '' }, () => {
      this.filterSubjects();
    });
  },

  /**
   * 搜索各题库中错题的题干 (Stem Search)
   */
  filterSubjects() {
    const kw = (this.data.searchKeyword || '').trim().toLowerCase();
    if (!kw) {
      this.setData({
        filteredSubjects: this.data.subjects,
        isSearching: false,
        searchMatchTotal: 0
      });
      return;
    }

    const activeErrors = this.allActiveErrors || [];
    const matchingErrorsByBank = {};
    let totalMatchCount = 0;

    activeErrors.forEach(errItem => {
      const q = errItem.question || {};
      const title = (q.title || '').toLowerCase();
      const kp = (q.knowledge_point || '').toLowerCase();
      const section = (q.section || '').toLowerCase();

      // 核心比对：题目题干 (q.title) 以及所属考点与章节
      if (title.includes(kw) || kp.includes(kw) || section.includes(kw)) {
        const bid = String(errItem.bank_id);
        if (!matchingErrorsByBank[bid]) {
          matchingErrorsByBank[bid] = [];
        }
        matchingErrorsByBank[bid].push({
          id: q.id || errItem.question_id,
          title: q.title || '试题',
          knowledge_point: q.knowledge_point || '',
          section: q.section || ''
        });
        totalMatchCount++;
      }
    });

    const filtered = [];
    this.data.subjects.forEach(sub => {
      const matchingQList = matchingErrorsByBank[sub.id] || [];
      const subNameMatch = sub.name.toLowerCase().includes(kw);

      if (matchingQList.length > 0) {
        filtered.push({
          ...sub,
          count: matchingQList.length,
          hasErrors: true,
          matchingQuestions: matchingQList,
          searchHighlightDesc: `题干命中 ${matchingQList.length} 道错题`
        });
      } else if (subNameMatch && sub.hasErrors) {
        filtered.push({
          ...sub,
          matchingQuestions: [],
          searchHighlightDesc: `题库名称匹配`
        });
      }
    });

    this.setData({
      filteredSubjects: filtered,
      isSearching: true,
      searchMatchTotal: totalMatchCount
    });
  },

  /**
   * 点击立即复习：只复习该特定题库的未掌握错题
   */
  onReviewSubject(e) {
    const subjectId = e.currentTarget.dataset.id;
    const subjectName = e.currentTarget.dataset.name;

    wx.navigateTo({
      url: `/pages/quiz/quiz?mode=practice&type=error&bankId=${subjectId}&subject=${subjectId}&title=${encodeURIComponent(subjectName + ' - 错题复习')}`
    });
  },

  /**
   * 点击搜索命中的单道错题题干：直接跳转复习该题
   */
  onReviewSingleError(e) {
    const bankId = e.currentTarget.dataset.bankid;
    const questionId = e.currentTarget.dataset.questionid;
    const subjectName = e.currentTarget.dataset.name;

    wx.navigateTo({
      url: `/pages/quiz/quiz?mode=practice&type=error&bankId=${bankId}&questionId=${questionId}&subject=${bankId}&title=${encodeURIComponent(subjectName + ' - 错题复习')}`
    });
  },

  onNoErrorsTip(e) {
    const name = e.currentTarget.dataset.name || '该题库';
    wx.showToast({
      title: `${name}暂无错题，继续保持！`,
      icon: 'none'
    });
  },

  onExportPDF(e) {
    const subjectName = e.currentTarget.dataset.name;
    const count = e.currentTarget.dataset.count;

    wx.showLoading({ title: '正在生成 PDF...' });
    setTimeout(() => {
      wx.hideLoading();
      wx.showModal({
        title: '导出成功',
        content: `《${subjectName} - 错题集》(共 ${count} 题) 已成功导出为 PDF 文档，已保存至本地。`,
        confirmText: '去查看',
        cancelText: '完成',
        confirmColor: '#0058bc',
        success: (res) => {
          if (res.confirm) {
            wx.showToast({ title: '已加入下载任务', icon: 'success' });
          }
        }
      });
    }, 800);
  },

  onSwitchTab(e) {
    const tab = e.currentTarget.dataset.tab;
    if (tab === 'library') {
      wx.redirectTo({
        url: '/pages/index/index'
      });
    } else if (tab === 'profile') {
      wx.redirectTo({
        url: '/pages/profile/profile'
      });
    } else if (tab === 'errors') {
      this.setData({ currentTab: 'errors' });
    }
  }
});
