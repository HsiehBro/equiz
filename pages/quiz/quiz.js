const { request, CONFIG } = require('../../utils/request.js');
const studyStats = require('../../utils/studyStats.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    bankId: '',
    examTitle: '项目管理基础考试',
    isVipBank: true,
    durationSeconds: 0, // 从 00:00 开始计时
    timerText: '00:00',
    timerInterval: null,
    totalCount: 100,

    // 学习规划打卡完成弹窗状态 (3秒自动消失进度条)
    showCheckInModal: false,
    checkInDays: 0,
    checkInDailyGoal: 30,
    checkInBarActive: false,
    
    // 模式: 'practice' (答题模式) | 'recite' (背题模式)
    mode: 'practice',
    
    // 是否为错题模式
    isErrorMode: false,
    subjectId: '',
    
    // 抽屉弹窗状态
    showQuestionDrawer: false,
    showAnalysisDrawer: false,
    showCommentsDrawer: false,
    
    // 当前题目索引 (0-based, 默认从第1题开始)
    currentIndex: 0,
    
    // 用户作答记录: 默认为空，由用户主动选择
    userAnswers: {},
    
    // 确认判题记录: { [id]: boolean } (在答题模式下点击确定后显示对错反馈)
    confirmedMap: {},
    
    // 收藏状态: { [id]: boolean }
    bookmarkMap: {},
    
    // 当前题目的计算状态 (默认未选)
    currentSelectedMap: {},
    currentCorrectMap: {},
    currentAnswerFormatted: '',
    currentUserAnswerFormatted: '未作答',
    
    // 题库数据列表 (100道题目矩阵，包含单选与多选题)
    questions: [],
    
    // 题目评论列表（每道题独享独立评论列表）
    commentList: [],
    newCommentText: '',
    newCommentVisibility: 'public',
    replyTarget: null,
    isInputFocused: false,

    // 评论抽屉高度拖拽与展开状态
    commentsDrawerHeight: 0,
    isDraggingDrawer: false,
    isDrawerExpanded: false
  },

  onLoad(options) {
    this.initNavBar();
    
    let targetMode = 'practice';
    if (options && options.mode && options.mode !== 'favorite') {
      targetMode = options.mode;
    }
    const isFavoriteMode = Boolean(options && (options.type === 'favorite' || options.mode === 'favorite'));
    const isError = Boolean(options && (options.type === 'error' || options.mode === 'error' || (options.title && decodeURIComponent(options.title).includes('错题'))));
    const openComments = Boolean(options && (options.openComments === 'true' || options.openComments === '1' || options.replyToCommentId));
    const replyToId = options && options.replyToCommentId;
    const replyToAuthor = options && options.replyToAuthor ? decodeURIComponent(options.replyToAuthor) : '';
    let targetTitle = this.data.examTitle;
    if (options && options.title) {
      targetTitle = decodeURIComponent(options.title);
    }
    const targetBankId = (options && (options.bankId || options.id)) || '';

    // 防卫校准：若传入的 targetTitle 明显是题干或通用占位符，自动纠正为规范题库名称
    if (String(targetBankId) === '2') {
      if (!targetTitle || targetTitle.length > 25 || targetTitle.includes('心肺复苏') || targetTitle.includes('项目管理')) {
        targetTitle = '2023年护士执业资格考试';
      }
    } else if (String(targetBankId) === '1') {
      if (!targetTitle || targetTitle.length > 25 || targetTitle.includes('生命周期')) {
        targetTitle = '项目管理基础考试';
      }
    } else if (String(targetBankId) === '3') {
      if (!targetTitle || targetTitle.length > 25) {
        targetTitle = '初级会计实务 - 核心考点';
      }
    }

    this.targetQuestionId = (options && options.questionId) ? parseInt(options.questionId, 10) : null;

    // VIP 专属题库权限校验拦截：普通用户若访问 VIP 专属题库，拦截并直接跳转至会员解锁页
    const isVipBank = this.checkIsVipBank(targetBankId, targetTitle);
    this.setData({ isVipBank: Boolean(isVipBank) });
    const userInfo = wx.getStorageSync('user_info') || {};
    const userRole = userInfo.role || 'user';
    const rawVip = wx.getStorageSync('user_is_vip');
    const isUserVip = userRole === 'admin' || userRole === 'vip' || (userRole !== 'user' && rawVip === true);

    if (isVipBank && !isUserVip) {
      wx.showToast({
        title: '该题库为VIP专属，请先开通会员',
        icon: 'none',
        duration: 2000
      });
      setTimeout(() => {
        wx.redirectTo({
          url: '/pages/vip/vip'
        });
      }, 300);
      return;
    }

    // 专属「我的收藏」强化刷题模式：仅加载该题库已收藏题目，严禁拉取全量题库
    if (isFavoriteMode) {
      let customQuestions = [];
      try {
        customQuestions = wx.getStorageSync('practice_custom_questions') || [];
      } catch (e) {}

      if (Array.isArray(customQuestions) && customQuestions.length > 0) {
        let startIndex = 0;
        if (options && options.index !== undefined) {
          startIndex = parseInt(options.index, 10) || 0;
        } else if (options && options.questionId) {
          const qId = parseInt(options.questionId, 10);
          const foundIdx = customQuestions.findIndex(q => q.id === qId);
          if (foundIdx !== -1) startIndex = foundIdx;
        }
        if (startIndex >= customQuestions.length || startIndex < 0) {
          startIndex = 0;
        }

        const bookmarkMap = {};
        const userAnswers = {};
        customQuestions.forEach(q => {
          bookmarkMap[q.id] = true;
          if (q.user_answer && q.user_answer.length > 0) {
            userAnswers[q.id] = q.user_answer;
          }
        });

        this.setData({
          bankId: targetBankId,
          questions: customQuestions,
          totalCount: customQuestions.length,
          currentIndex: startIndex,
          bookmarkMap: bookmarkMap,
          userAnswers: userAnswers,
          mode: targetMode, // 确保 mode 值为 'practice'，让「答题模式」正确激活蓝色胶囊态
          isFavoriteMode: true,
          examTitle: targetTitle,
          isErrorMode: false,
          subjectId: (options && options.subject) || '',
          showCommentsDrawer: openComments
        });

        this.startTimer();
        this.updateCurrentQuestionState(startIndex);

        if (openComments) {
          this.loadCommentsForCurrentQuestion(startIndex);
        }
        return;
      }
    }

    // 专属「错题集复习」模式：只加载被点击题库的真实未掌握错题，严禁加载全量题库
    if (isError) {
      let startIndex = 0;
      if (options && options.index !== undefined) {
        startIndex = parseInt(options.index, 10) || 0;
      }
      this.setData({
        bankId: targetBankId,
        mode: targetMode,
        examTitle: targetTitle,
        isErrorMode: true,
        subjectId: (options && options.subject) || '',
        showCommentsDrawer: openComments
      });
      const targetQuestionId = (options && options.questionId) || null;
      this.loadErrorQuestions(targetBankId, targetTitle, startIndex, targetQuestionId);
      return;
    }

    // 常规模式
    this.initQuestions();
    
    let startIndex = 0;
    if (options && options.index) {
      startIndex = parseInt(options.index, 10) || 0;
    } else if (options && options.questionId) {
      const qId = parseInt(options.questionId, 10);
      const foundIdx = this.data.questions.findIndex(q => q.id === qId);
      if (foundIdx !== -1) {
        startIndex = foundIdx;
      }
    }

    this.setData({
      bankId: targetBankId,
      currentIndex: startIndex,
      mode: targetMode,
      examTitle: targetTitle,
      isErrorMode: isError,
      subjectId: (options && options.subject) || '',
      showCommentsDrawer: openComments
    });

    this.startTimer();
    this.updateCurrentQuestionState(startIndex);

    if (openComments) {
      this.loadCommentsForCurrentQuestion(startIndex);
    }

    if (replyToId) {
      this.setData({
        replyTarget: { id: replyToId, author: replyToAuthor || '学员' },
        newCommentVisibility: 'public',
        isInputFocused: true
      });
    }

    if (!CONFIG.USE_MOCK && targetBankId && !isNaN(Number(targetBankId))) {
      this.loadQuestionsFromBackend(targetBankId, startIndex);
    }
  },

  /**
   * 错题复习专有加载器：只拉取指定题库的未掌握错题
   */
  loadErrorQuestions(bankId, examTitle, startIndex = 0, targetQuestionId = null) {
    wx.showLoading({ title: '加载错题中...' });

    const finishLoad = (errorQuestions) => {
      wx.hideLoading();
      if (!Array.isArray(errorQuestions) || errorQuestions.length === 0) {
        wx.showModal({
          title: '暂无错题',
          content: `题库《${examTitle.replace(' - 错题复习', '')}》暂无未掌握错题！`,
          showCancel: false,
          confirmText: '返回',
          confirmColor: '#0058bc',
          success: () => {
            wx.navigateBack();
          }
        });
        return;
      }

      const bookmarkMap = {};
      const userAnswers = {};
      errorQuestions.forEach(q => {
        if (q.is_bookmarked) {
          bookmarkMap[q.id] = true;
        }
      });

      let safeIndex = startIndex < errorQuestions.length ? startIndex : 0;
      if (targetQuestionId) {
        const found = errorQuestions.findIndex(q => String(q.id) === String(targetQuestionId));
        if (found !== -1) {
          safeIndex = found;
        }
      }
      this.setData({
        questions: errorQuestions,
        totalCount: errorQuestions.length,
        currentIndex: safeIndex,
        userAnswers,
        bookmarkMap,
        confirmedMap: {},
        isErrorMode: true
      });
      this.startTimer();
      this.updateCurrentQuestionState(safeIndex);
    };

    if (!CONFIG.USE_MOCK && bankId) {
      request({ url: `/api/v1/errors?bank_id=${bankId}` })
        .then((res) => {
          const list = Array.isArray(res) ? res : (res && res.list ? res.list : []);
          const activeErrors = list.filter(item => !item.is_mastered);
          if (activeErrors.length > 0) {
            const formatted = activeErrors.map(item => {
              const q = item.question || {};
              return {
                id: q.id || item.question_id,
                bank_id: parseInt(bankId, 10) || 1,
                type: q.type || '单选',
                section: q.section || examTitle,
                title: q.title || '试题',
                options: Array.isArray(q.options) && q.options.length > 0 ? q.options : [
                  { key: 'A', text: '选项 A' },
                  { key: 'B', text: '选项 B' }
                ],
                answer: Array.isArray(q.answer) ? q.answer : ['A'],
                difficulty: q.difficulty || '中等',
                knowledgePoint: q.knowledge_point || '易错考点',
                analysis: q.analysis || '详见解析',
                knowledgeDetail: q.knowledge_detail || '',
                is_error: true,
                wrong_count: item.wrong_count || 1
              };
            });
            finishLoad(formatted);
            return;
          }
          this.loadLocalErrorsFallback(bankId, finishLoad);
        })
        .catch((err) => {
          console.log('[Quiz] 加载云端错题失败，尝试本地缓存:', err);
          this.loadLocalErrorsFallback(bankId, finishLoad);
        });
      return;
    }

    this.loadLocalErrorsFallback(bankId, finishLoad);
  },

  loadLocalErrorsFallback(bankId, callback) {
    try {
      const errorsMap = wx.getStorageSync('user_errors_map') || {};
      const list = Object.values(errorsMap).filter(item => 
        String(item.bank_id) === String(bankId) && !item.is_mastered
      );
      if (list.length > 0) {
        const formatted = list.map(item => ({
          ...(item.question || {}),
          id: item.question_id || (item.question && item.question.id),
          bank_id: parseInt(bankId, 10) || 1,
          is_error: true
        }));
        callback(formatted);
        return;
      }
    } catch (e) {
      console.log('读取本地错题异常', e);
    }
    callback([]);
  },

  loadQuestionsFromBackend(bankId, startIndex) {
    request({ url: `/api/v1/banks/${bankId}/questions` })
      .then((questions) => {
        if (Array.isArray(questions) && questions.length > 0) {
          const userAnswers = {};
          const bookmarkMap = {};
          questions.forEach(q => {
            if (q.user_answer && q.user_answer.length > 0) {
              userAnswers[q.id] = q.user_answer;
            }
            if (q.is_bookmarked) {
              bookmarkMap[q.id] = true;
            }
          });
          let safeIndex = startIndex < questions.length ? startIndex : 0;
          if (this.targetQuestionId) {
            const foundIdx = questions.findIndex(q => q.id === this.targetQuestionId);
            if (foundIdx !== -1) {
              safeIndex = foundIdx;
            }
          }
          this.setData({
            questions: questions,
            totalCount: questions.length,
            currentIndex: safeIndex,
            userAnswers: userAnswers,
            bookmarkMap: bookmarkMap
          });
          this.updateCurrentQuestionState(safeIndex);
          if (this.data.showCommentsDrawer) {
            this.loadCommentsForCurrentQuestion(safeIndex);
          }
        }
      })
      .catch((err) => {
        console.log('[Quiz] 加载后端试题失败，保留本地题库数据:', err);
      });
  },

  onUnload() {
    this.clearTimer();
    if (this.checkInTimeout) {
      clearTimeout(this.checkInTimeout);
      this.checkInTimeout = null;
    }
    try {
      wx.removeStorageSync('practice_custom_questions');
    } catch (e) {}
  },

  initNavBar() {
    try {
      const windowInfo = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
      const statusBarHeight = windowInfo.statusBarHeight || 20;
      const windowHeight = windowInfo.windowHeight || 667;
      let navBarHeight = 44;
      if (wx.getMenuButtonBoundingClientRect) {
        const menu = wx.getMenuButtonBoundingClientRect();
        if (menu && menu.top) {
          navBarHeight = (menu.top - statusBarHeight) * 2 + menu.height;
        }
      }

      this.windowHeight = windowHeight;
      this.statusBarHeight = statusBarHeight;
      this.standardDrawerHeight = Math.round(windowHeight * 0.62);
      this.expandedDrawerHeight = Math.min(Math.round(windowHeight * 0.90), windowHeight - statusBarHeight - 8);

      this.setData({
        statusBarHeight,
        navBarHeight: Math.max(navBarHeight, 44),
        commentsDrawerHeight: this.data.commentsDrawerHeight || this.standardDrawerHeight
      });
    } catch (e) {
      console.log('获取导航栏信息异常', e);
    }
  },

  // 生成 100 题完整题库列表 (包含精选单选题与多选题)
  initQuestions() {
    const coreQuestions = [
      {
        id: 1,
        type: '单选',
        section: 'Section 1: 基础理论 (1-20)',
        title: '项目生命周期中，成本和人员投入水平最高的阶段是（ ）。',
        options: [
          { key: 'A', text: '执行阶段' },
          { key: 'B', text: '启动阶段' },
          { key: 'C', text: '规划阶段' },
          { key: 'D', text: '收尾阶段' }
        ],
        answer: ['A'],
        difficulty: '容易',
        knowledgePoint: '项目生命周期',
        analysis: '项目生命周期通常分为启动、规划、执行和收尾四个阶段。在执行阶段，团队执行项目管理计划中规定的工作，因此资源消耗、人员投入和成本支出达到最高峰。',
        knowledgeDetail: '启动阶段成本最低，执行阶段成本和人员投入最高，收尾阶段迅速下降。'
      },
      {
        id: 2,
        type: '单选',
        section: 'Section 1: 基础理论 (1-20)',
        title: '下列哪一项不属于项目干系人管理的主要过程？（ ）',
        options: [
          { key: 'A', text: '识别干系人' },
          { key: 'B', text: '规划干系人参与' },
          { key: 'C', text: '估算活动资源' },
          { key: 'D', text: '监督干系人参与' }
        ],
        answer: ['C'],
        difficulty: '中等',
        knowledgePoint: '干系人管理',
        analysis: '项目干系人管理过程包括：识别干系人、规划干系人参与、管理干系人参与和监督干系人参与。估算活动资源属于项目资源管理知识领域。',
        knowledgeDetail: '项目干系人包括所有对项目有利益关系或受到项目决策/结果影响的个人或组织。'
      },
      {
        id: 3,
        type: '单选',
        section: 'Section 1: 基础理论 (1-20)',
        title: '在敏捷开发中，Scrum 框架中的三大角色不包括（ ）。',
        options: [
          { key: 'A', text: '产品负责人 (Product Owner)' },
          { key: 'B', text: 'Scrum 主管 (Scrum Master)' },
          { key: 'C', text: '开发团队 (Development Team)' },
          { key: 'D', text: '项目经理 (Project Manager)' }
        ],
        answer: ['D'],
        difficulty: '中等',
        knowledgePoint: '敏捷项目管理',
        analysis: 'Scrum 团队由产品负责人 (Product Owner)、Scrum 主管 (Scrum Master) 和开发团队 (Development Team) 组成。Scrum 体系中没有传统意义上的“项目经理”角色。',
        knowledgeDetail: '产品负责人负责最大化产品价值，Scrum Master 负责推行和支持 Scrum，开发团队负责交付增量。'
      },
      {
        id: 4,
        type: '单选',
        section: 'Section 1: 基础理论 (1-20)',
        title: '制定项目章程的主要输入是（ ）。',
        options: [
          { key: 'A', text: '立项管理文件与商业论证' },
          { key: 'B', text: '范围基准' },
          { key: 'C', text: '工作分解结构 (WBS)' },
          { key: 'D', text: '质量测量指标' }
        ],
        answer: ['A'],
        difficulty: '中等',
        knowledgePoint: '项目整合管理',
        analysis: '制定项目章程是编写一份正式批准项目并授权项目经理在项目活动中使用组织资源的文件过程。主要输入包括商业论证、协议、事业环境因素和组织过程资产。',
        knowledgeDetail: '项目章程一旦批准，即标志着项目的正式启动。'
      },
      {
        id: 5,
        type: '单选',
        section: 'Section 1: 基础理论 (1-20)',
        title: '在项目管理中，关键路径是指网络图中（ ）。',
        options: [
          { key: 'A', text: '最早开始时间相连的路径' },
          { key: 'B', text: '持续时间最长的路径' },
          { key: 'C', text: '包含最多活动的路径' },
          { key: 'D', text: '资源消耗最多的路径' }
        ],
        answer: ['B'],
        difficulty: '中等',
        knowledgePoint: '项目进度管理',
        analysis: '关键路径法 (CPM) 用于在进度模型中估算项目最短工期，确定逻辑网络路径的进度灵活性。关键路径是项目中时间最长的活动顺序，决定了项目最短可能完成时间。',
        knowledgeDetail: '关键路径的总浮动时间（总时差）通常为零或负数。网络图中可能存在多条关键路径。'
      },
      {
        id: 6,
        type: '多选',
        section: 'Section 1: 基础理论 (1-20)',
        title: '在敏捷项目管理中，Scrum 框架包含的核心事件（仪式）有（ ）。',
        options: [
          { key: 'A', text: '冲刺规划会 (Sprint Planning)' },
          { key: 'B', text: '每日站会 (Daily Scrum)' },
          { key: 'C', text: '冲刺评审会 (Sprint Review)' },
          { key: 'D', text: '冲刺回顾会 (Sprint Retrospective)' }
        ],
        answer: ['A', 'B', 'C', 'D'],
        difficulty: '中等',
        knowledgePoint: '敏捷仪式',
        analysis: 'Scrum 包含五大事件：Sprint 本身、Sprint 规划会、每日站会、Sprint 评审会和 Sprint 回顾会。四个会议均为 Scrum 保证透明与检视的核心仪式。',
        knowledgeDetail: '回顾会专注于团队工作流程的持续改进，评审会专注于向干系人展示可工作的产品增量。'
      },
      {
        id: 7,
        type: '多选',
        section: 'Section 1: 基础理论 (1-20)',
        title: '项目进度管理中，常用于缩短进度工期的压缩技术包括（ ）。',
        options: [
          { key: 'A', text: '赶工 (Crashing)' },
          { key: 'B', text: '快速跟进 (Fast Tracking)' },
          { key: 'C', text: '资源平衡 (Resource Leveling)' },
          { key: 'D', text: '蒙特卡洛模拟 (Monte Carlo Simulation)' }
        ],
        answer: ['A', 'B'],
        difficulty: '较难',
        knowledgePoint: '进度压缩技术',
        analysis: '进度压缩技术指在不缩减项目范围的前提下缩短工期的技术，包括：赶工（增加资源以最小成本缩短工期）和快速跟进（按顺序进行的活动改为并行）。资源平衡通常导致工期延长。',
        knowledgeDetail: '赶工可能增加成本，快速跟进可能增加返工风险。'
      },
      {
        id: 8,
        type: '多选',
        section: 'Section 1: 基础理论 (1-20)',
        title: '根据 PMBOK 规范，下列属于针对威胁（负面风险）的应对策略有（ ）。',
        options: [
          { key: 'A', text: '规避 (Avoid)' },
          { key: 'B', text: '转移 (Transfer)' },
          { key: 'C', text: '开拓 (Exploit)' },
          { key: 'D', text: '减轻 (Mitigate)' }
        ],
        answer: ['A', 'B', 'D'],
        difficulty: '中等',
        knowledgePoint: '风险应对策略',
        analysis: '针对负面风险（威胁）的策略有：规避、转移、减轻、接受；针对正面风险（机会）的策略有：开拓、提高、分享、接受。',
        knowledgeDetail: '开拓属于机会应对策略，消除不确定性确保机会出现。'
      }
    ];

    // 填充至 100 题完整矩阵 (穿插单选与多选)
    const fullList = [...coreQuestions];
    for (let i = 9; i <= 100; i++) {
      let secName = 'Section 1: 基础理论 (1-20)';
      if (i > 20 && i <= 40) secName = 'Section 2: 范围与进度 (21-40)';
      else if (i > 40 && i <= 60) secName = 'Section 3: 成本与质量 (41-60)';
      else if (i > 60 && i <= 80) secName = 'Section 4: 风险与采购 (61-80)';
      else if (i > 80) secName = 'Section 5: 综合实务 (81-100)';

      const isMulti = (i % 4 === 0 || i % 7 === 0);
      const multiAns = (i % 2 === 0) ? ['A', 'B', 'C'] : ['A', 'C', 'D'];
      const singleAns = [['A'], ['B'], ['C'], ['D']][(i % 4)];

      fullList.push({
        id: i,
        type: isMulti ? '多选' : '单选',
        section: secName,
        title: isMulti 
          ? `【多选题】关于项目管理知识领域中第 ${i} 题综合实务考点，下列正确的选项有（ ）。`
          : `关于项目管理知识领域中第 ${i} 题标准规程与考点，下列正确的选项是（ ）。`,
        options: [
          { key: 'A', text: `在规划阶段必须明确定义的关键基准与测量指标 (${i}-A)` },
          { key: 'B', text: `通过定性与定量风险分析确定应对策略与储备金 (${i}-B)` },
          { key: 'C', text: `采用挣值分析技术持续监控成本绩效与进度偏差 (${i}-C)` },
          { key: 'D', text: `在项目收尾阶段组织经验教训总结并归档过程资产 (${i}-D)` }
        ],
        answer: isMulti ? multiAns : singleAns,
        difficulty: i % 3 === 0 ? '较难' : (i % 2 === 0 ? '中等' : '容易'),
        knowledgePoint: `知识域模块 ${Math.ceil(i / 10)}`,
        analysis: `本题考查项目管理实务要点。正确答案是 ${(isMulti ? multiAns : singleAns).join('、')}。在实际项目运作中，必须严格遵循标准化流程与规范输出。`,
        knowledgeDetail: `相关核心定义与考纲重点：强化过程输入、输出与工具技术的掌握。`
      });
    }

    this.setData({ questions: fullList });
  },

  // 刷新当前题目的答题与判定辅助状态
  updateCurrentQuestionState(index) {
    const qList = this.data.questions;
    if (!qList || !qList[index]) return;
    
    const currentQ = qList[index];
    const userChoices = this.data.userAnswers[currentQ.id] || [];
    const correctChoices = currentQ.answer || [];

    const currentSelectedMap = {};
    userChoices.forEach(k => { currentSelectedMap[k] = true; });

    const currentCorrectMap = {};
    correctChoices.forEach(k => { currentCorrectMap[k] = true; });

    this.setData({
      currentIndex: index,
      currentSelectedMap,
      currentCorrectMap,
      currentAnswerFormatted: correctChoices.join('、'),
      currentUserAnswerFormatted: userChoices.length > 0 ? userChoices.join('、') : '未作答'
    });

    // 题目切换时立即加载当前题目的专属独立评论（保证底部评论徽标数字与抽屉内容精准对应）
    this.loadCommentsForCurrentQuestion(index);
  },

  // 计时器管理
  startTimer() {
    this.clearTimer();
    this.data.timerInterval = setInterval(() => {
      let dur = this.data.durationSeconds + 1;
      const minutes = Math.floor(dur / 60);
      const seconds = dur % 60;
      const timerText = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
      this.setData({
        durationSeconds: dur,
        timerText: timerText
      });
    }, 1000);
  },

  clearTimer() {
    if (this.data.timerInterval) {
      clearInterval(this.data.timerInterval);
      this.data.timerInterval = null;
    }
  },

  // 返回上一页
  onNavBack() {
    if (getCurrentPages().length > 1) {
      wx.navigateBack();
    } else {
      wx.redirectTo({ url: '/pages/index/index' });
    }
  },

  // 切换模式: 答题模式 vs 背题模式
  onSwitchMode(e) {
    const targetMode = e.currentTarget.dataset.mode;
    if (targetMode === this.data.mode) return;
    
    this.setData({ mode: targetMode });
    if (targetMode === 'recite') {
      wx.showToast({ title: '已切换至背题模式', icon: 'none' });
    } else {
      wx.showToast({ title: '已切换至答题模式', icon: 'none' });
    }
  },

  // 选择选项 (支持单选与多选切换，无震动)
  onSelectOption(e) {
    const key = e.currentTarget.dataset.key;
    const currentQ = this.data.questions[this.data.currentIndex];
    const userAnswers = { ...this.data.userAnswers };
    let currentChoices = userAnswers[currentQ.id] ? [...userAnswers[currentQ.id]] : [];

    if (currentQ.type === '多选') {
      // 多选逻辑：点击已选中的则取消，未选中的则添加
      if (currentChoices.includes(key)) {
        currentChoices = currentChoices.filter(k => k !== key);
      } else {
        currentChoices.push(key);
        currentChoices.sort();
      }
    } else {
      // 单选逻辑：直接选中该项
      currentChoices = [key];
    }

    if (currentChoices.length > 0) {
      userAnswers[currentQ.id] = currentChoices;
    } else {
      delete userAnswers[currentQ.id];
    }

    this.setData({ userAnswers });
    this.updateCurrentQuestionState(this.data.currentIndex);
  },

  // 点击确定按钮 (提交/确认当前题目答案并判定对错)
  onConfirmAnswer() {
    const currentQ = this.data.questions[this.data.currentIndex];
    if (!currentQ) return;

    // 防止用户重复点击已确认的题目导致累计答题重复计算
    if (this.data.confirmedMap && this.data.confirmedMap[currentQ.id]) {
      return;
    }

    const userChoices = this.data.userAnswers[currentQ.id] || [];
    
    if (userChoices.length === 0) {
      wx.showToast({ 
        title: currentQ.type === '多选' ? '请至少选择一个选项' : '请先选择答案', 
        icon: 'none' 
      });
      return;
    }

    const correctChoices = currentQ.answer || [];
    const userSorted = userChoices.slice().sort().join('');
    const correctSorted = correctChoices.slice().sort().join('');
    const isCorrect = userSorted === correctSorted;
    
    const isPureMissed = currentQ.type === '多选' && 
      userChoices.every(k => correctChoices.includes(k)) && 
      userChoices.length < correctChoices.length;

    const confirmedMap = { ...this.data.confirmedMap };
    confirmedMap[currentQ.id] = true;
    this.setData({ confirmedMap });

    // 核心数据看板统计：无论刷哪个题库，累计答题+1；答对+1；更新正确率；达标触发打卡
    const statsResult = studyStats.recordAnswer(this.data.bankId, isCorrect);
    if (statsResult && statsResult.justCheckedIn) {
      this.triggerCheckInModal(statsResult.checkInDays, statsResult.dailyGoal);
    }

    if (isCorrect) {
      wx.showToast({ title: '回答正确！', icon: 'success' });
    } else {
      wx.showToast({ 
        title: isPureMissed ? `漏选，正确答案是 ${correctChoices.join('、')}` : `回答错误，正确答案是 ${correctChoices.join('、')}`, 
        icon: 'none' 
      });

      // 答错试题自动同步至本地错题集缓存
      try {
        const errorsMap = wx.getStorageSync('user_errors_map') || {};
        errorsMap[currentQ.id] = {
          question_id: currentQ.id,
          bank_id: Number(this.data.bankId) || 1,
          question: currentQ,
          wrong_count: (errorsMap[currentQ.id] ? errorsMap[currentQ.id].wrong_count : 0) + 1,
          is_mastered: false,
          last_wrong_at: new Date().toISOString()
        };
        wx.setStorageSync('user_errors_map', errorsMap);
      } catch (e) {}
    }

    // 真实后端模式：异步上报作答结果与错题记录
    if (!CONFIG.USE_MOCK && this.data.bankId && !isNaN(Number(this.data.bankId))) {
      request({
        url: '/api/v1/practice/submit-single',
        method: 'POST',
        data: {
          bank_id: Number(this.data.bankId),
          question_id: currentQ.id,
          user_answer: userChoices,
          duration_seconds: this.data.durationSeconds
        }
      }).catch((err) => {
        console.log('[Quiz] 作答记录上报异常:', err);
      });
    }
  },

  // 触发学习规划达成打卡弹窗（带逐渐消失的进度条，3秒后自动关闭）
  triggerCheckInModal(checkInDays, dailyGoal) {
    if (this.checkInTimeout) {
      clearTimeout(this.checkInTimeout);
      this.checkInTimeout = null;
    }

    this.setData({
      showCheckInModal: true,
      checkInDays: checkInDays || 1,
      checkInDailyGoal: dailyGoal || 30,
      checkInBarActive: false
    });

    // 开启进度条逐渐消失动画
    setTimeout(() => {
      this.setData({
        checkInBarActive: true
      });
    }, 50);

    // 3秒后自动消失
    this.checkInTimeout = setTimeout(() => {
      this.closeCheckInModal();
    }, 3000);
  },

  // 关闭打卡弹窗
  closeCheckInModal() {
    if (this.checkInTimeout) {
      clearTimeout(this.checkInTimeout);
      this.checkInTimeout = null;
    }
    this.setData({
      showCheckInModal: false,
      checkInBarActive: false
    });
  },

  // 上一题
  onPrevQuestion() {
    if (this.data.currentIndex > 0) {
      const nextIndex = this.data.currentIndex - 1;
      this.setData({ currentIndex: nextIndex });
      this.updateCurrentQuestionState(nextIndex);
    } else {
      wx.showToast({ title: '已经是第一题了', icon: 'none' });
    }
  },

  // 下一题
  onNextQuestion() {
    if (this.data.currentIndex < this.data.questions.length - 1) {
      const nextIndex = this.data.currentIndex + 1;
      this.setData({ currentIndex: nextIndex });
      this.updateCurrentQuestionState(nextIndex);
    } else {
      wx.showToast({ title: '已经是最后一题了', icon: 'none' });
    }
  },

  // 错题模式：将当前题目移出错题集 (标记已掌握)
  onRemoveFromErrors() {
    const currentQ = this.data.questions[this.data.currentIndex];
    if (!currentQ) return;

    wx.showModal({
      title: '移出错题集',
      content: '确定要将本题从错题集中移除吗？（标记为已掌握）',
      confirmText: '移除',
      confirmColor: '#ba1a1a',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          const currentList = [...this.data.questions];
          const removedIndex = this.data.currentIndex;
          currentList.splice(removedIndex, 1);

          // 真实后端模式：同步标记掌握
          if (!CONFIG.USE_MOCK && typeof currentQ.id === 'number') {
            request({
              url: `/api/v1/errors/${currentQ.id}/master`,
              method: 'POST'
            }).catch((err) => {
              console.log('[Quiz] 移出错题接口异常:', err);
            });
          }

          // 本地缓存同步标记掌握
          try {
            const errorsMap = wx.getStorageSync('user_errors_map') || {};
            if (errorsMap[currentQ.id]) {
              errorsMap[currentQ.id].is_mastered = true;
              wx.setStorageSync('user_errors_map', errorsMap);
            }
          } catch (e) {}

          if (currentList.length === 0) {
            wx.showToast({
              title: '错题已全部清空！',
              icon: 'success',
              duration: 1500
            });
            setTimeout(() => {
              wx.navigateBack();
            }, 1500);
            return;
          }

          const nextIndex = removedIndex >= currentList.length ? currentList.length - 1 : removedIndex;
          this.setData({
            questions: currentList,
            currentIndex: nextIndex
          });
          this.updateCurrentQuestionState(nextIndex);
          wx.showToast({
            title: '已移出错题集',
            icon: 'success'
          });
        }
      }
    });
  },

  // 收藏 / 取消收藏
  onToggleBookmark() {
    const currentQ = this.data.questions[this.data.currentIndex];
    const bookmarkMap = { ...this.data.bookmarkMap };
    const isBookmarked = !bookmarkMap[currentQ.id];
    bookmarkMap[currentQ.id] = isBookmarked;
    
    this.setData({ bookmarkMap });

    // 真实后端模式：同步收藏状态
    if (!CONFIG.USE_MOCK && typeof currentQ.id === 'number') {
      request({
        url: '/api/v1/favorites/toggle',
        method: 'POST',
        data: { question_id: currentQ.id }
      }).catch((err) => {
        console.log('[Quiz] 收藏接口异常:', err);
      });
    }
    
    wx.showToast({
      title: isBookmarked ? '已添加收藏' : '已取消收藏',
      icon: 'success'
    });
  },

  // 展开/收起解析抽屉
  onToggleAnalysis() {
    this.setData({
      showAnalysisDrawer: !this.data.showAnalysisDrawer
    });
  },

  // 打开/关闭题库导航抽屉
  onToggleQuestionDrawer() {
    this.setData({
      showQuestionDrawer: !this.data.showQuestionDrawer
    });
  },

  // 题库导航跳转题目
  onJumpToQuestion(e) {
    const index = Number(e.currentTarget.dataset.index);
    this.setData({
      currentIndex: index,
      showQuestionDrawer: false
    });
    this.updateCurrentQuestionState(index);
  },

  // 打开/关闭评论抽屉
  onToggleCommentsDrawer() {
    const show = !this.data.showCommentsDrawer;
    const standardHeight = this.standardDrawerHeight || Math.round((this.windowHeight || 667) * 0.62);
    this.setData({
      showCommentsDrawer: show,
      commentsDrawerHeight: this.data.isDrawerExpanded ? (this.expandedDrawerHeight || Math.round((this.windowHeight || 667) * 0.90)) : (this.data.commentsDrawerHeight || standardHeight),
      replyTarget: null,
      isInputFocused: false,
      isDraggingDrawer: false
    });
    if (show) {
      this.loadCommentsForCurrentQuestion();
    }
  },

  // 拖拽顶部手柄扩大/缩小评论区 - Touch Start
  onDrawerTouchStart(e) {
    if (!e.touches || e.touches.length === 0) return;
    this.drawerTouchStartY = e.touches[0].clientY;
    this.drawerTouchStartX = e.touches[0].clientX;
    const standardHeight = this.standardDrawerHeight || Math.round((this.windowHeight || 667) * 0.62);
    this.drawerStartHeight = this.data.commentsDrawerHeight || standardHeight;
    this.drawerStartTime = Date.now();
    this.setData({ isDraggingDrawer: true });
  },

  // 拖拽顶部手柄扩大/缩小评论区 - Touch Move
  onDrawerTouchMove(e) {
    if (!e.touches || e.touches.length === 0 || typeof this.drawerTouchStartY !== 'number') return;
    const touch = e.touches[0];
    const deltaY = touch.clientY - this.drawerTouchStartY;
    
    // 向上滑动 (deltaY < 0) 增加高度，向下滑动 (deltaY > 0) 减小高度
    const prospectiveHeight = this.drawerStartHeight - deltaY;

    const winH = this.windowHeight || 667;
    const minH = Math.round(winH * 0.38);
    const maxH = Math.min(Math.round(winH * 0.94), winH - (this.statusBarHeight || 20) - 8);

    let clampedHeight = prospectiveHeight;
    if (clampedHeight > maxH) {
      clampedHeight = maxH + (clampedHeight - maxH) * 0.2;
    } else if (clampedHeight < minH) {
      clampedHeight = minH - (minH - clampedHeight) * 0.2;
    }

    this.setData({
      commentsDrawerHeight: Math.round(clampedHeight)
    });
  },

  // 拖拽顶部手柄扩大/缩小评论区 - Touch End
  onDrawerTouchEnd(e) {
    this.setData({ isDraggingDrawer: false });
    const winH = this.windowHeight || 667;
    const standardH = this.standardDrawerHeight || Math.round(winH * 0.62);
    const expandedH = this.expandedDrawerHeight || Math.min(Math.round(winH * 0.90), winH - (this.statusBarHeight || 20) - 8);
    const currentH = this.data.commentsDrawerHeight || standardH;

    const timeDiff = Date.now() - (this.drawerStartTime || Date.now());
    const totalDeltaY = (e.changedTouches && e.changedTouches[0]) ? (e.changedTouches[0].clientY - this.drawerTouchStartY) : 0;

    // 快速轻扫手势快捷判定
    if (timeDiff < 250 && Math.abs(totalDeltaY) > 25) {
      if (totalDeltaY < 0) {
        // 快速上滑 -> 一键扩大到最大展示区
        this.setData({
          commentsDrawerHeight: expandedH,
          isDrawerExpanded: true
        });
        return;
      } else {
        // 快速下滑
        if (currentH > standardH + 40) {
          this.setData({
            commentsDrawerHeight: standardH,
            isDrawerExpanded: false
          });
        } else {
          this.onToggleCommentsDrawer();
        }
        return;
      }
    }

    // 根据最终高度自动吸附分档
    const midpoint = (standardH + expandedH) / 2;
    if (currentH >= midpoint) {
      // 达到或超过中间线，自动平滑扩大展示全屏评论区
      this.setData({
        commentsDrawerHeight: expandedH,
        isDrawerExpanded: true
      });
    } else if (currentH < standardH - 80) {
      // 下拉过多直接收起关闭
      this.onToggleCommentsDrawer();
    } else {
      // 吸附恢复为标准高度
      this.setData({
        commentsDrawerHeight: standardH,
        isDrawerExpanded: false
      });
    }
  },

  // 点击顶部手柄快速切换：在「标准展示」与「扩大展示」之间一键切换
  onTapDrawerHandle() {
    const winH = this.windowHeight || 667;
    const standardH = this.standardDrawerHeight || Math.round(winH * 0.62);
    const expandedH = this.expandedDrawerHeight || Math.min(Math.round(winH * 0.90), winH - (this.statusBarHeight || 20) - 8);

    if (this.data.isDrawerExpanded) {
      this.setData({
        commentsDrawerHeight: standardH,
        isDrawerExpanded: false
      });
    } else {
      this.setData({
        commentsDrawerHeight: expandedH,
        isDrawerExpanded: true
      });
    }
  },

  // 加载当前题目的公开评论（严格按当前题目 ID 进行独立加载，绝不跨题目混淆）
  loadCommentsForCurrentQuestion(targetIndex) {
    const idx = typeof targetIndex === 'number' ? targetIndex : this.data.currentIndex;
    const currentQ = this.data.questions && this.data.questions[idx];
    if (!currentQ) {
      this.setData({ commentList: [] });
      return;
    }

    const qId = currentQ.id;

    // 先从本地提取只属于该题目的评论（严格按 questionId 过滤）
    const localNotes = this.getLocalCommentsForQuestion(qId);

    if (!CONFIG.USE_MOCK && typeof qId === 'number') {
      request({
        url: `/api/v1/questions/${qId}/comments`,
        needAuth: false
      }).then((res) => {
        const list = res && res.list ? res.list : (Array.isArray(res) ? res : []);
        const userInfo = wx.getStorageSync('user_info') || {};
        const currentUid = userInfo.id;
        const repliesStore = wx.getStorageSync('user_comment_replies') || {};

        const backendMapped = list.map((item) => {
          const author = item.author_name || (item.user && item.user.nickname) || '备考学员';
          const isMine = Boolean(currentUid && item.user_id === currentUid);
          let timeText = '刚刚';
          if (item.created_at) {
            try {
              const d = new Date(item.created_at);
              const now = new Date();
              const diffSec = Math.floor((now - d) / 1000);
              if (diffSec < 60) timeText = '刚刚';
              else if (diffSec < 3600) timeText = `${Math.floor(diffSec / 60)}分钟前`;
              else if (diffSec < 86400) timeText = `${Math.floor(diffSec / 3600)}小时前`;
              else timeText = `${d.getMonth() + 1}月${d.getDate()}日`;
            } catch (e) {}
          }

          // 组装并映射后端真实子回复
          const backendReplies = (item.replies || []).map(r => {
            const replyAuthor = r.author_name || (r.user && r.user.nickname) || '考友';
            const isReplyMine = Boolean(currentUid && r.user_id === currentUid);
            return {
              id: r.id,
              userId: r.user_id,
              author: replyAuthor,
              avatarText: replyAuthor.slice(0, 2),
              avatarUrl: (r.user && r.user.avatar_url) || r.author_avatar || '',
              time: r.created_at ? '刚刚' : '刚刚',
              content: r.content,
              replyToAuthor: r.reply_to_author || author,
              replyToUserId: r.reply_to_user_id,
              isMine: isReplyMine
            };
          });

          return {
            id: item.id,
            userId: item.user_id,
            author: author,
            avatarText: author.slice(0, 2),
            avatarUrl: item.author_avatar || '',
            time: timeText,
            content: item.content,
            likes: item.like_count || 0,
            isLiked: Boolean(item.is_liked),
            isMine: isMine,
            visibility: item.visibility || 'public',
            replies: backendReplies
          };
        });

        this.setData({ commentList: backendMapped });
      }).catch((err) => {
        console.log('[Quiz] 获取题目评论异常，使用本地独立缓存:', err);
        this.setData({ commentList: localNotes });
      });
      return;
    }

    this.setData({ commentList: localNotes });
  },

  getLocalCommentsForQuestion(qId) {
    if (!qId) return [];
    try {
      const userInfo = wx.getStorageSync('user_info') || {};
      const myName = userInfo.nickname || '备考学员';
      const stored = wx.getStorageSync('user_study_notes_list') || [];
      const repliesStore = wx.getStorageSync('user_comment_replies') || {};
      // 严格按 questionId 单题匹配，不同题目拥有不同评论！
      const matched = stored.filter(item => String(item.questionId) === String(qId));
      return matched.map(n => {
        const subReplies = repliesStore[String(n.id)] || n.replies || [];
        return {
          id: n.id,
          author: n.author || myName,
          avatarText: (n.author || myName).slice(0, 2),
          time: n.date || '刚刚',
          content: n.content,
          likes: n.likes || 0,
          isLiked: false,
          isMine: true,
          visibility: n.visibility || 'public',
          replies: subReplies
        };
      });
    } catch (e) {
      return [];
    }
  },

  // 点赞评论
  onLikeComment(e) {
    const id = e.currentTarget.dataset.id;
    const commentList = this.data.commentList.map(item => {
      if (item.id === id) {
        const isLiked = !item.isLiked;
        return {
          ...item,
          isLiked: isLiked,
          likes: isLiked ? item.likes + 1 : Math.max(0, item.likes - 1)
        };
      }
      return item;
    });
    this.setData({ commentList });

    // 真实后端模式：同步点赞
    if (!CONFIG.USE_MOCK && typeof id === 'number') {
      request({
        url: `/api/v1/notes/${id}/like`,
        method: 'POST'
      }).catch((err) => {
        console.log('[Quiz] 点赞切换异常:', err);
      });
    }
  },

  // 切换新发布评论的可见范围 (public / private)
  onSelectNewCommentVisibility(e) {
    const vis = e.currentTarget.dataset.vis;
    if (vis && (vis === 'public' || vis === 'private')) {
      this.setData({ newCommentVisibility: vis });
    }
  },

  // 切换已有个人评论的可见性 (公开 / 仅自己可见)
  onToggleItemVisibility(e) {
    const id = e.currentTarget.dataset.id;
    const currentItem = this.data.commentList.find(item => item.id === id && item.isMine);
    if (!currentItem) return;

    const targetVis = currentItem.visibility === 'private' ? 'public' : 'private';

    // 真实后端模式：提交后端进行合规校验与可见性变更
    if (!CONFIG.USE_MOCK && typeof id === 'number') {
      wx.showLoading({ title: '正在切换...' });
      request({
        url: `/api/v1/notes/${id}`,
        method: 'PUT',
        data: { visibility: targetVis }
      }).then(() => {
        wx.hideLoading();
        const commentList = this.data.commentList.map(item => {
          if (item.id === id && item.isMine) {
            return {
              ...item,
              visibility: targetVis
            };
          }
          return item;
        });
        this.setData({ commentList });
        const tip = targetVis === 'private' ? '已转为仅自己可见' : '已转为公开可见';
        wx.showToast({ title: tip, icon: 'none' });
      }).catch((err) => {
        wx.hideLoading();
        console.log('[Quiz] 切换可见性异常:', err);
        const errMsg = (err && (err.message || err.msg)) || '切换可见性失败';
        wx.showModal({
          title: '操作失败',
          content: errMsg,
          showCancel: false,
          confirmText: '我知道了',
          confirmColor: '#ba1a1a'
        });
      });
      return;
    }

    // 离线/Mock模式
    const commentList = this.data.commentList.map(item => {
      if (item.id === id && item.isMine) {
        return {
          ...item,
          visibility: targetVis
        };
      }
      return item;
    });
    this.setData({ commentList });
    const tip = targetVis === 'private' ? '已转为仅自己可见' : '已转为公开可见';
    wx.showToast({ title: tip, icon: 'none' });
  },

  // 评论输入
  onCommentInput(e) {
    this.setData({ newCommentText: e.detail.value });
  },

  // 开始对某条公开评论进行回复
  onStartReply(e) {
    const { id, author, userId } = e.currentTarget.dataset;
    if (!id || !author) return;

    this.setData({
      replyTarget: {
        id: id,
        author: author,
        userId: userId ? Number(userId) : undefined
      },
      newCommentVisibility: 'public',
      isInputFocused: true
    });
  },

  // 取消当前回复状态
  onCancelReply() {
    this.setData({
      replyTarget: null,
      isInputFocused: false
    });
  },

  // 删除某条子回复
  onDeleteSubReply(e) {
    const { parentId, id } = e.currentTarget.dataset;
    wx.showModal({
      title: '确认删除',
      content: '确定要删除这条回复吗？',
      confirmColor: '#ba1a1a',
      confirmText: '删除',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          const commentList = this.data.commentList.map(item => {
            if (String(item.id) === String(parentId)) {
              return {
                ...item,
                replies: (item.replies || []).filter(r => String(r.id) !== String(id))
              };
            }
            return item;
          });
          this.setData({ commentList });

          // 同步到本地子回复存储
          try {
            const repliesStore = wx.getStorageSync('user_comment_replies') || {};
            if (repliesStore[String(parentId)]) {
              repliesStore[String(parentId)] = repliesStore[String(parentId)].filter(r => String(r.id) !== String(id));
              wx.setStorageSync('user_comment_replies', repliesStore);
            }
          } catch (err) {}

          // 真实后端模式同步删除
          if (!CONFIG.USE_MOCK && typeof id === 'number') {
            request({
              url: `/api/v1/notes/${id}`,
              method: 'DELETE'
            }).catch(() => {});
          }

          wx.showToast({ title: '已删除回复', icon: 'success' });
        }
      }
    });
  },

  // 发布评论或回复
  onSubmitComment() {
    const text = this.data.newCommentText.trim();
    if (!text) {
      wx.showToast({ title: '请输入内容', icon: 'none' });
      return;
    }
    
    const currentQ = this.data.questions[this.data.currentIndex] || {};
    const replyTarget = this.data.replyTarget;
    const userInfo = wx.getStorageSync('user_info') || {};
    const myName = userInfo.nickname || '备考学员';
    const myAvatarUrl = userInfo.avatarUrl || userInfo.avatar_url || '';
    const myAvatarText = myName.slice(0, 2);

    // A. 处于对某条评论的“回复”模式
    if (replyTarget) {
      const parentIdNum = parseInt(replyTarget.id, 10);
      const bankIdNum = parseInt(currentQ.bank_id || this.data.bankId || this.data.subjectId, 10) || 1;

      // 真实后端模式：直接由后端服务执行合规校验（含全量敏感词拦截）与持久化
      if (!CONFIG.USE_MOCK && typeof currentQ.id === 'number') {
        wx.showLoading({ title: '正在发布回复...' });
        request({
          url: '/api/v1/notes',
          method: 'POST',
          data: {
            bank_id: bankIdNum,
            question_id: currentQ.id,
            content: text,
            visibility: 'public',
            parent_id: parentIdNum > 0 ? parentIdNum : undefined,
            reply_to_author: replyTarget.author,
            reply_to_user_id: replyTarget.userId
          }
        }).then((savedNote) => {
          wx.hideLoading();
          const newSubReply = {
            id: (savedNote && savedNote.id) ? savedNote.id : Date.now(),
            userId: userInfo.id,
            author: myName,
            avatarText: myAvatarText,
            avatarUrl: myAvatarUrl,
            time: '刚刚',
            content: text,
            replyToAuthor: replyTarget.author,
            replyToUserId: replyTarget.userId,
            isMine: true
          };

          const updated = this.data.commentList.map(c => {
            if (String(c.id) === String(replyTarget.id)) {
              return {
                ...c,
                replies: [...(c.replies || []), newSubReply]
              };
            }
            return c;
          });

          this.setData({
            commentList: updated,
            newCommentText: '',
            replyTarget: null,
            isInputFocused: false
          });

          // 同步到本地子回复存储
          try {
            const repliesStore = wx.getStorageSync('user_comment_replies') || {};
            const key = String(replyTarget.id);
            repliesStore[key] = [...(repliesStore[key] || []), newSubReply];
            wx.setStorageSync('user_comment_replies', repliesStore);
          } catch (e) {}

          wx.showToast({ title: '回复已成功发布', icon: 'success' });
        }).catch((err) => {
          wx.hideLoading();
          console.log('[Quiz] 提交回复失败:', err);
          const errMsg = (err && (err.message || err.msg)) || '提交回复失败';
          wx.showModal({
            title: '发布失败',
            content: errMsg,
            showCancel: false,
            confirmText: '我知道了',
            confirmColor: '#ba1a1a'
          });
        });
        return;
      }

      // 离线/Mock模式
      const newSubReply = {
        id: Date.now(),
        userId: userInfo.id,
        author: myName,
        avatarText: myAvatarText,
        avatarUrl: myAvatarUrl,
        time: '刚刚',
        content: text,
        replyToAuthor: replyTarget.author,
        replyToUserId: replyTarget.userId,
        isMine: true
      };

      const updated = this.data.commentList.map(c => {
        if (String(c.id) === String(replyTarget.id)) {
          return {
            ...c,
            replies: [...(c.replies || []), newSubReply]
          };
        }
        return c;
      });

      this.setData({
        commentList: updated,
        newCommentText: '',
        replyTarget: null,
        isInputFocused: false
      });
      wx.showToast({ title: '回复已成功发布', icon: 'success' });
      return;
    }

    // B. 发布全新的主评论/学习笔记
    const visibility = this.data.newCommentVisibility || 'public';
    const bankIdNum = parseInt(currentQ.bank_id || this.data.bankId || this.data.subjectId, 10) || 1;
    const now = new Date();
    const dateStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;

    // 智能解析题目标准题库名称
    let currentBankTitle = this.data.examTitle;
    const qTitle = currentQ.title || '';
    if (qTitle.includes('心肺复苏') || qTitle.includes('CPR') || qTitle.includes('水银体温计') || qTitle.includes('青霉素过敏') || bankIdNum === 2) {
      currentBankTitle = '2023年护士执业资格考试';
    } else if (qTitle.includes('会计') || qTitle.includes('核算与监督') || bankIdNum === 3) {
      currentBankTitle = '初级会计实务 - 核心考点';
    } else if (!currentBankTitle || currentBankTitle.includes('错题') || currentBankTitle.includes('收藏') || currentBankTitle.includes('全真模拟') || bankIdNum === 1) {
      currentBankTitle = '项目管理基础考试';
    }

    // 真实后端模式：直接由后端服务执行合规校验（含全量敏感词拦截）与持久化
    if (!CONFIG.USE_MOCK && typeof currentQ.id === 'number') {
      wx.showLoading({ title: '正在发布评论...' });
      request({
        url: '/api/v1/notes',
        method: 'POST',
        data: {
          bank_id: bankIdNum,
          question_id: currentQ.id,
          content: text,
          visibility: visibility
        }
      }).then((savedNote) => {
        wx.hideLoading();
        const newComment = {
          id: (savedNote && savedNote.id) ? savedNote.id : Date.now(),
          userId: userInfo.id,
          author: myName,
          avatarText: myAvatarText,
          avatarUrl: myAvatarUrl,
          time: '刚刚',
          content: text,
          likes: 0,
          isLiked: false,
          isMine: true,
          visibility: visibility,
          replies: []
        };

        this.setData({
          commentList: [newComment, ...this.data.commentList],
          newCommentText: ''
        });

        // 同步到学习笔记/评论存储
        try {
          const stored = wx.getStorageSync('user_study_notes_list') || [];
          const noteItem = {
            id: 'cmt_' + newComment.id,
            bankId: bankIdNum,
            bankTitle: currentBankTitle,
            questionId: currentQ.id || (this.data.currentIndex + 1),
            questionIndex: this.data.currentIndex,
            questionTitle: currentQ.title || '题目评论',
            content: text,
            visibility: visibility,
            date: dateStr,
            time: `${dateStr} ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
          };
          wx.setStorageSync('user_study_notes_list', [noteItem, ...stored]);
        } catch (e) {
          console.log('同步评论到学习笔记异常', e);
        }

        const successTip = visibility === 'private' ? '已发布（仅自己可见）' : '公开评论已发布';
        wx.showToast({ title: successTip, icon: 'success' });
      }).catch((err) => {
        wx.hideLoading();
        console.log('[Quiz] 提交笔记失败:', err);
        const errMsg = (err && (err.message || err.msg)) || '提交评论失败';
        wx.showModal({
          title: '发布失败',
          content: errMsg,
          showCancel: false,
          confirmText: '我知道了',
          confirmColor: '#ba1a1a'
        });
      });
      return;
    }

    // 离线/Mock模式
    const newComment = {
      id: Date.now(),
      userId: userInfo.id,
      author: myName,
      avatarText: myAvatarText,
      avatarUrl: myAvatarUrl,
      time: '刚刚',
      content: text,
      likes: 0,
      isLiked: false,
      isMine: true,
      visibility: visibility,
      replies: []
    };
    this.setData({
      commentList: [newComment, ...this.data.commentList],
      newCommentText: ''
    });

    try {
      const stored = wx.getStorageSync('user_study_notes_list') || [];
      const noteItem = {
        id: 'cmt_' + newComment.id,
        bankId: bankIdNum,
        bankTitle: currentBankTitle,
        questionId: currentQ.id || (this.data.currentIndex + 1),
        questionIndex: this.data.currentIndex,
        questionTitle: currentQ.title || '题目评论',
        content: text,
        visibility: visibility,
        date: dateStr,
        time: `${dateStr} ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
      };
      wx.setStorageSync('user_study_notes_list', [noteItem, ...stored]);
    } catch (e) {}

    const successTip = visibility === 'private' ? '已发布（仅自己可见）' : '公开评论已发布';
    wx.showToast({ title: successTip, icon: 'success' });
  },

  // 删除自己发布的评论 (弹出确认弹窗)
  onDeleteComment(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '确认删除',
      content: '确定要删除这条评论吗？',
      confirmColor: '#ba1a1a',
      confirmText: '删除',
      cancelText: '取消',
      success: (res) => {
        if (res.confirm) {
          const commentList = this.data.commentList.filter(item => item.id !== id);
          this.setData({ commentList });

          // 真实后端模式：同步删除笔记
          if (!CONFIG.USE_MOCK && typeof id === 'number') {
            request({
              url: `/api/v1/notes/${id}`,
              method: 'DELETE'
            }).catch((err) => {
              console.log('[Quiz] 云端删除笔记异常:', err);
            });
          }

          // 同步从学习笔记/评论存储中删除
          try {
            const stored = wx.getStorageSync('user_study_notes_list') || [];
            const filtered = stored.filter(item => item.id !== ('cmt_' + id) && item.id !== id);
            wx.setStorageSync('user_study_notes_list', filtered);
          } catch (e) {
            console.log('同步删除笔记异常', e);
          }

          wx.showToast({ title: '评论已删除', icon: 'success' });
        }
      }
    });
  },

  // 交卷
  onSubmitExam() {
    const total = this.data.questions.length;
    const answeredCount = Object.keys(this.data.userAnswers).length;
    
    wx.showModal({
      title: '确认交卷',
      content: `共 ${total} 题，已作答 ${answeredCount} 题。确认要现在交卷吗？`,
      confirmColor: '#0058bc',
      success: (res) => {
        if (res.confirm) {
          if (!CONFIG.USE_MOCK && this.data.bankId && !isNaN(Number(this.data.bankId))) {
            request({
              url: '/api/v1/practice/submit-exam',
              method: 'POST',
              data: {
                bank_id: Number(this.data.bankId),
                total_seconds: this.data.durationSeconds,
                answers: this.data.userAnswers
              }
            })
              .then((result) => {
                wx.showToast({ title: `交卷成功！得分: ${Math.round(result.score)}分`, icon: 'success' });
              })
              .catch((err) => {
                console.log('[Quiz] 交卷上报异常:', err);
                wx.showToast({ title: '交卷成功！', icon: 'success' });
              });
          } else {
            wx.showToast({ title: '交卷成功！', icon: 'success' });
          }
          this.setData({ showQuestionDrawer: false });
        }
      }
    });
  },

  checkIsVipBank(bankId, title) {
    if (!bankId && !title) return false;
    try {
      const customLibs = wx.getStorageSync('custom_libraries') || [];
      const foundInCustom = customLibs.find(l => String(l.id) === String(bankId) || l.title === title);
      if (foundInCustom) {
        return Boolean(foundInCustom.isVip || foundInCustom.is_vip);
      }
    } catch (e) {}

    // 预设官方题库中的 VIP 题库：项目管理基础考试 (IT互联网) 和 初级会计实务 - 核心考点 (财经类)
    if (String(bankId) === '1' || String(bankId) === '3') return true;
    if (title && (title.includes('项目管理') || title.includes('会计') || title.includes('VIP'))) return true;
    return false;
  }
});
