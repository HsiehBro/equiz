// 学习笔记与题目评论页面逻辑
const { request, CONFIG } = require('../../utils/request.js');
const STORAGE_KEY = 'user_study_notes_list';

// 默认用户评论数据（按题库与题目关联）
const DEFAULT_COMMENTS = [
  {
    id: 'note_1',
    bankId: 'edu',
    bankTitle: '教育心理学',
    questionId: 10,
    questionIndex: 9,
    questionTitle: '教育心理学：核心理论与实践应用重点整理',
    content: '认知负荷理论认为工作记忆容量有限。教学设计需精简内生负荷、消除外生负荷、促进相关负荷，总结到位，备考核心！',
    visibility: 'public',
    date: '2026-09-02',
    time: '2026-09-02 16:15'
  },
  {
    id: 'note_2',
    bankId: 'math',
    bankTitle: '高等数学',
    questionId: 15,
    questionIndex: 14,
    questionTitle: '高等数学：微积分进阶解题技巧分析',
    content: '第三个知识点的公式推导中，第二步关键在于利用拉格朗日中值定理构造辅助函数 F(x) = f(x) - [f(b)-f(a)]/(b-a) * (x-a)，使两端函数值相等。',
    visibility: 'public',
    date: '2026-09-02',
    time: '2026-09-02 14:00'
  },
  {
    id: 'note_3',
    bankId: 'cet6',
    bankTitle: '英语六级',
    questionId: 20,
    questionIndex: 19,
    questionTitle: '英语六级：高频词汇分类记忆法',
    content: '词根词缀+近义词场景串记：spec-/spect- 表示看（inspect, retrospect, perspective），结合历年真题高频语境速记，考前必看！',
    visibility: 'public',
    date: '2026-09-01',
    time: '2026-09-01 18:30'
  },
  {
    id: 'cmt_1',
    bankId: '1',
    bankTitle: '项目管理基础考试',
    questionId: 1,
    questionIndex: 0,
    questionTitle: '关于项目生命周期中成本与人员投入水平',
    content: '关键路径法 (CPM) 关键在于总浮动时间为 0。如果出现任何任务延误，必须立即申请赶工或快速跟进！',
    visibility: 'public',
    date: '2026-09-02',
    time: '2026-09-02 14:20'
  },
  {
    id: 'cmt_2',
    bankId: '1',
    bankTitle: '项目管理基础考试',
    questionId: 2,
    questionIndex: 1,
    questionTitle: '干系人管理知识领域核心过程',
    content: '注意：快速跟进是通过并行增加风险来压缩工期；赶工是通过追加资源增加成本来压缩工期，两者常考对比题！',
    visibility: 'public',
    date: '2026-09-02',
    time: '2026-09-02 11:35'
  },
  {
    id: 'cmt_3',
    bankId: '1',
    bankTitle: '项目管理基础考试',
    questionId: 3,
    questionIndex: 2,
    questionTitle: '敏捷开发 Scrum 核心角色与事件',
    content: '【个人笔记】管理储备 (Management Reserve) 不包含在成本基准中，但属于项目总预算，这道题之前模拟做错了一次。',
    visibility: 'private',
    date: '2026-09-01',
    time: '2026-09-01 09:15'
  }
];

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    currentTab: 'profile',
    searchKeyword: '',
    
    // 题库分类列表（完全根据真实存在的题库动态生成，不写死任何静态预置科目）
    bankCategories: [
      { id: 'all', name: '全部题库' }
    ],
    selectedBank: 'all',

    commentsList: [],
    filteredComments: [],
    removingId: null,
    highlightNoteId: null
  },

  allKnownBanks: [],
  targetNoteId: null,

  onLoad(options) {
    if (options && (options.noteId || options.id)) {
      this.targetNoteId = String(options.noteId || options.id);
    }
    this.initNavBar();
    this.loadBankCategories();
    this.loadComments();
  },

  onShow() {
    // 页面再次展示时，重新拉取最新真实题库与笔记列表
    this.loadBankCategories();
    this.loadComments();
  },

  /**
   * 根据真实存在的题库（后端返回题库 + 笔记实际所属题库）动态汇聚分类页签
   */
  updateDynamicBankCategories(sourceList) {
    const list = Array.isArray(sourceList) ? sourceList : (this.data.commentsList || []);
    const categories = [{ id: 'all', name: '全部题库' }];
    const seenNames = new Set(['全部题库']);

    // 1. 优先从后端拉取的真实题库列表聚合
    if (this.allKnownBanks && Array.isArray(this.allKnownBanks)) {
      this.allKnownBanks.forEach(b => {
        const title = (b.title || b.name || '').trim();
        const id = String(b.id || title);
        if (title && !seenNames.has(title)) {
          seenNames.add(title);
          categories.push({ id, name: title });
        }
      });
    }

    // 2. 补充扫描当前笔记列表中实际出现的题库名称（避免离线或特殊题库漏掉）
    list.forEach(item => {
      const title = (item.bankTitle || '').trim();
      const id = String(item.bankId || title);
      if (title && !seenNames.has(title)) {
        seenNames.add(title);
        categories.push({ id, name: title });
      }
    });

    // 3. 检查当前选中的 selectedBank 是否仍然存在于真实题库列表中
    let currentSelected = this.data.selectedBank;
    const isSelectedValid = categories.some(c => c.id === currentSelected || c.name === currentSelected);
    if (!isSelectedValid) {
      currentSelected = 'all';
    }

    this.setData({
      bankCategories: categories,
      selectedBank: currentSelected
    }, () => {
      this.applyFilterAndSearch();
    });
  },

  /**
   * 从服务端加载真实题库列表
   */
  loadBankCategories() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/banks' })
        .then((res) => {
          const banks = res && res.list ? res.list : (Array.isArray(res) ? res : []);
          if (banks.length > 0) {
            this.allKnownBanks = banks;
            this.updateDynamicBankCategories();
          }
        })
        .catch(() => {
          this.updateDynamicBankCategories();
        });
      return;
    }
    this.updateDynamicBankCategories();
  },

  /**
   * 加载用户发表的评论与笔记列表
   */
  loadComments() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/notes' })
        .then((res) => {
          const list = res && res.list ? res.list : (Array.isArray(res) ? res : []);
          if (list.length > 0) {
            const mapped = list.map((item, idx) => {
              const d = item.created_at ? new Date(item.created_at) : new Date();
              const dateStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
              const timeStr = `${dateStr} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
              return {
                id: item.id,
                bankId: String(item.bank_id),
                bankTitle: item.bank_title || (item.question && item.question.bank_title) || '项目管理基础考试',
                questionId: item.question_id,
                questionIndex: item.question && item.question.sort_order ? item.question.sort_order - 1 : idx,
                questionTitle: item.question_title || (item.question && item.question.title) || '题目笔记',
                content: item.content || '',
                visibility: item.visibility || 'public',
                date: dateStr,
                time: timeStr,
                likeCount: item.like_count || 0,
                isLiked: item.is_liked || false
              };
            });
            this.setData({ commentsList: mapped }, () => {
              this.saveCommentsToStorage(mapped);
              this.updateDynamicBankCategories(mapped);
            });
            return;
          }
          this.loadFromStorageOrFallback();
        })
        .catch((err) => {
          console.log('[Notes] 获取后端笔记失败，使用本地缓存:', err);
          this.loadFromStorageOrFallback();
        });
      return;
    }

    this.loadFromStorageOrFallback();
  },

  loadFromStorageOrFallback() {
    try {
      let stored = wx.getStorageSync(STORAGE_KEY);
      if (stored && Array.isArray(stored) && stored.length > 0) {
        // 补充可能缺失的预置公开笔记（如教育心理学、高等数学、英语六级）
        const existingIds = new Set(stored.map(s => String(s.id)));
        DEFAULT_COMMENTS.forEach(def => {
          if (!existingIds.has(String(def.id))) {
            stored.push(def);
          }
        });

        // 规整本地历史遗留测试数据中的题库名称（将通用题库规整为真实科目名称）
        stored = stored.map((item, idx) => {
          let bTitle = (item.bankTitle || '').trim();
          let bId = String(item.bankId || '');
          if (!bTitle || bTitle === '通用题库') {
            if (bId === '1' || bId === 'pmp') bTitle = '项目管理基础考试';
            else if (bId === '2' || bId === 'nurse') bTitle = '2023年护士执业资格考试';
            else if (bId === '3' || bId === 'accounting') bTitle = '初级会计实务 - 核心考点';
            else if (idx === 0) bTitle = '项目管理基础考试';
            else if (idx === 1) bTitle = '项目管理基础考试';
            else bTitle = '项目管理基础考试';
          }
          return {
            ...item,
            bankId: bId || '1',
            bankTitle: bTitle
          };
        });
        this.setData({ commentsList: stored }, () => {
          this.updateDynamicBankCategories(stored);
          this.checkAndHighlightTargetNote();
        });
      } else {
        this.setData({ commentsList: DEFAULT_COMMENTS }, () => {
          this.saveCommentsToStorage(DEFAULT_COMMENTS);
          this.updateDynamicBankCategories(DEFAULT_COMMENTS);
          this.checkAndHighlightTargetNote();
        });
      }
    } catch (e) {
      this.setData({ commentsList: DEFAULT_COMMENTS }, () => {
        this.updateDynamicBankCategories(DEFAULT_COMMENTS);
        this.checkAndHighlightTargetNote();
      });
    }
  },

  /**
   * 检查并高亮展示来自消息中心指定跳转的目标笔记
   */
  checkAndHighlightTargetNote() {
    if (!this.targetNoteId) return;
    const list = this.data.commentsList || [];
    const target = list.find(c => String(c.id) === String(this.targetNoteId));
    if (target) {
      this.setData({
        highlightNoteId: String(this.targetNoteId)
      });
      // 确保目标笔记在当前分类过滤下可见
      if (this.data.selectedBank !== 'all' && this.data.selectedBank !== target.bankId && this.data.selectedBank !== target.bankTitle) {
        this.setData({ selectedBank: 'all' }, () => {
          this.applyFilterAndSearch();
        });
      }
      setTimeout(() => {
        wx.pageScrollTo({
          selector: '#note-' + this.targetNoteId,
          duration: 350
        });
      }, 300);
      wx.showToast({
        title: `已定位: ${target.questionTitle || target.bankTitle}`,
        icon: 'none',
        duration: 2500
      });
    }
  },

  /**
   * 持久化存储
   */
  saveCommentsToStorage(list) {
    try {
      wx.setStorageSync(STORAGE_KEY, list);
    } catch (e) {
      console.log('保存评论异常', e);
    }
  },

  /**
   * 执行题库分类过滤与关键词搜索
   */
  applyFilterAndSearch() {
    const { commentsList, selectedBank, searchKeyword } = this.data;
    const keyword = (searchKeyword || '').trim().toLowerCase();

    const filtered = commentsList.filter(item => {
      // 1. 题库分类过滤
      if (selectedBank !== 'all') {
        const bId = String(item.bankId || '');
        const bTitle = (item.bankTitle || '').trim();
        const sel = String(selectedBank);
        const isMatch = (bId && bId === sel) || (bTitle && (bTitle === sel || bTitle.includes(sel)));
        if (!isMatch) {
          return false;
        }
      }

      // 2. 关键词搜索 (匹配评论内容、题目标题、题库名称)
      if (keyword) {
        const matchContent = (item.content || '').toLowerCase().includes(keyword);
        const matchTitle = (item.questionTitle || item.title || '').toLowerCase().includes(keyword);
        const matchBank = (item.bankTitle || '').toLowerCase().includes(keyword);
        return matchContent || matchTitle || matchBank;
      }

      return true;
    });

    this.setData({
      filteredComments: filtered
    });
  },

  /**
   * 搜索输入
   */
  onSearchInput(e) {
    this.setData({
      searchKeyword: e.detail.value
    }, () => {
      this.applyFilterAndSearch();
    });
  },

  /**
   * 清空搜索输入
   */
  onClearSearch() {
    this.setData({
      searchKeyword: ''
    }, () => {
      this.applyFilterAndSearch();
    });
  },

  /**
   * 切换题库分类
   */
  onSelectBank(e) {
    const id = e.currentTarget.dataset.id;
    if (this.data.selectedBank === id) return;
    this.setData({
      selectedBank: id
    }, () => {
      this.applyFilterAndSearch();
    });
  },

  /**
   * 调整评论可见度 (Public <-> Private 快速切换)
   */
  onToggleItemVisibility(e) {
    const id = e.currentTarget.dataset.id;
    const item = this.data.commentsList.find(c => c.id === id);
    if (!item) return;

    const newVisibility = item.visibility === 'public' ? 'private' : 'public';
    const updatedList = this.data.commentsList.map(c => {
      if (c.id === id) {
        return {
          ...c,
          visibility: newVisibility
        };
      }
      return c;
    });

    this.setData({
      commentsList: updatedList
    }, () => {
      this.saveCommentsToStorage(updatedList);
      this.applyFilterAndSearch();

      // 真实后端模式：同步修改服务端可见性
      if (!CONFIG.USE_MOCK && typeof id === 'number') {
        request({
          url: `/api/v1/notes/${id}`,
          method: 'PUT',
          data: { visibility: newVisibility }
        }).catch(err => {
          console.log('[Notes] 修改可见性服务端异常:', err);
        });
      }

      wx.showToast({
        title: newVisibility === 'public' ? '已转为所有人可见' : '已转为仅自己可见',
        icon: 'none'
      });
    });
  },

  /**
   * 删除评论
   */
  onDeleteComment(e) {
    const id = e.currentTarget.dataset.id;
    const item = this.data.commentsList.find(c => c.id === id);
    if (!item) return;

    wx.showModal({
      title: '删除评论',
      content: '确定要删除这条题目评论吗？删除后将无法恢复。',
      confirmText: '删除',
      confirmColor: '#ba1a1a',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          const updatedList = this.data.commentsList.filter(c => c.id !== id);
          this.setData({
            commentsList: updatedList
          }, () => {
            this.saveCommentsToStorage(updatedList);
            this.applyFilterAndSearch();

            // 真实后端模式：同步删除服务端笔记
            if (!CONFIG.USE_MOCK && typeof id === 'number') {
              request({
                url: `/api/v1/notes/${id}`,
                method: 'DELETE'
              }).catch(err => {
                console.log('[Notes] 删除笔记服务端异常:', err);
              });
            }

            wx.showToast({ title: '评论已删除', icon: 'success' });
          });
        }
      }
    });
  },

  /**
   * 点击跳转到具体评论所在题目
   */
  onJumpToQuestion(e) {
    const item = e.currentTarget.dataset.item;
    if (!item) return;

    const qId = item.questionId || 1;
    const qIndex = typeof item.questionIndex === 'number' ? item.questionIndex : (qId > 0 ? qId - 1 : 0);
    const title = item.bankTitle || '项目管理基础考试';

    wx.navigateTo({
      url: `/pages/quiz/quiz?bankId=${item.bankId || '1'}&questionId=${qId}&index=${qIndex}&title=${encodeURIComponent(title)}&openComments=true`
    });
  },

  /**
   * 空状态去刷题
   */
  onGoPractice() {
    const { selectedBank, bankCategories } = this.data;
    let targetBankId = '';
    let targetTitle = '';
    if (selectedBank && selectedBank !== 'all') {
      const matched = bankCategories.find(c => String(c.id) === String(selectedBank));
      if (matched) {
        targetBankId = matched.id;
        targetTitle = matched.name;
      }
    }
    if (!targetBankId) {
      const firstBank = bankCategories.find(c => c.id !== 'all');
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

  onOpenMenu() {
    wx.showActionSheet({
      itemList: ['清空所有个人评论', '恢复默认评论样例', '关于题目评论说明'],
      success: (res) => {
        if (res.tapIndex === 0) {
          wx.showModal({
            title: '确认清空',
            content: '确定要清空全部个人评论吗？',
            confirmColor: '#ba1a1a',
            success: (modalRes) => {
              if (modalRes.confirm) {
                this.setData({ commentsList: [] }, () => {
                  this.saveCommentsToStorage([]);
                  this.applyFilterAndSearch();
                  wx.showToast({ title: '已清空评论', icon: 'success' });
                });
              }
            }
          });
        } else if (res.tapIndex === 1) {
          this.setData({ commentsList: DEFAULT_COMMENTS }, () => {
            this.saveCommentsToStorage(DEFAULT_COMMENTS);
            this.applyFilterAndSearch();
            wx.showToast({ title: '已恢复默认示例', icon: 'success' });
          });
        } else if (res.tapIndex === 2) {
          wx.showModal({
            title: '关于我的评论',
            content: '本页面汇集您在各题库做题时所发表的所有个人评论与解题心得。支持按题库分类浏览、随时调整公开/私密可见度，并支持一键直达原题。',
            showCancel: false
          });
        }
      }
    });
  },

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
