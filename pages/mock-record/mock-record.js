// pages/mock-record/mock-record.js
const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,

    // 模考记录信息
    record: {
      id: '',
      title: '全科综合模拟考',
      score: 85,
      total: 100,
      timeUsed: '45分 12秒',
      date: '2023年10月24日',
      accuracy: '85%'
    },
    rankingPercent: 85,

    // 大纲考点多维诊断数据集
    syllabusDiagnostics: [],

    // 筛选状态: 'all' | 'correct' | 'incorrect'
    activeFilter: 'all',

    // 试题解析列表
    analysisQuestions: [],
    displayQuestions: [],
    correctCount: 0,
    incorrectCount: 0
  },

  onLoad(options) {
    this.initSystemInfo();
    this.initRecordData(options);
  },

  /**
   * 初始化状态栏与导航栏高度
   */
  initSystemInfo() {
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
      console.warn('获取系统高度异常:', e);
    }
  },

  /**
   * 初始化模考记录数据与逐题解析
   */
  initRecordData(options) {
    if (options.id && !isNaN(parseInt(options.id, 10)) && !CONFIG.USE_MOCK) {
      wx.showLoading({ title: '加载成绩分析...', mask: true });
      request({ url: `/api/v1/mock/records/${options.id}` })
        .then((rec) => {
          wx.hideLoading();
          if (rec) {
            const formattedQuestions = (rec.question_snapshots || []).map((q) => ({
              id: q.id,
              number: q.number,
              title: q.title,
              type: q.type,
              section: q.section,
              isCorrect: q.is_correct,
              userAnswer: Array.isArray(q.user_answer) ? (q.user_answer.length > 0 ? q.user_answer.join(', ') : '未作答') : (q.user_answer || '未作答'),
              correctAnswer: Array.isArray(q.correct_answer) ? (q.correct_answer.length > 0 ? q.correct_answer.join(', ') : '暂无') : (q.correct_answer || '暂无'),
              analysis: q.analysis || '暂无题目解析',
              knowledgePoint: q.knowledge_point || q.section || '综合考点'
            }));

            let correctCount = 0;
            let incorrectCount = 0;
            formattedQuestions.forEach((q) => {
              if (q.isCorrect) {
                correctCount++;
              } else {
                incorrectCount++;
              }
            });

            this.setData({
              record: {
                id: String(rec.id),
                title: rec.exam_title,
                score: rec.score,
                total: rec.total_score || 100,
                timeUsed: rec.time_used_text || `${Math.round(rec.time_used_seconds / 60)} 分钟`,
                date: rec.date_text || '今天',
                accuracy: `${Math.round((rec.correct_count / (rec.total_questions || 1)) * 100)}%`
              },
              rankingPercent: rec.ranking_percent || 85,
              syllabusDiagnostics: rec.syllabus_diagnostics || [],
              analysisQuestions: formattedQuestions,
              correctCount,
              incorrectCount
            }, () => {
              this.applyFilter();
            });
            return;
          }
          this.initRecordDataFallback(options);
        })
        .catch((err) => {
          wx.hideLoading();
          console.warn('[MockRecord] 拉取云端战报失败，使用本地兜底:', err);
          this.initRecordDataFallback(options);
        });
      return;
    }

    this.initRecordDataFallback(options);
  },

  initRecordDataFallback(options) {
    let score = options.score ? parseInt(options.score, 10) : 85;
    let total = options.total ? parseInt(options.total, 10) : 100;
    let timeUsed = options.timeUsed ? decodeURIComponent(options.timeUsed) : '45分 12秒';
    let date = options.date ? decodeURIComponent(options.date) : '2023年10月24日';
    let title = options.title ? decodeURIComponent(options.title) : '全科综合模拟考';
    let accuracy = options.accuracy ? decodeURIComponent(options.accuracy) : `${score}%`;

    // 如果传入了 id，尝试从 storage 查找完整记录
    if (options.id) {
      try {
        const records = wx.getStorageSync('mock_exam_records') || [];
        const target = records.find((r) => r.id === options.id);
        if (target) {
          score = target.score !== undefined ? target.score : score;
          total = target.total || 100;
          timeUsed = target.timeUsed || timeUsed;
          date = target.date || date;
          title = target.title || title;
          accuracy = target.accuracy || accuracy;
        }
      } catch (e) {
        console.warn('读取本地记录异常:', e);
      }
    }

    // 计算排名超越百分比
    const rankingPercent = Math.min(99, Math.max(15, Math.round(score * 0.9 + 10)));

    // 默认兜底大纲诊断卡片
    const defaultDiagnostics = [
      { name: 'Section 1: 基础理论', target_weight: 20, total_count: 20, correct_count: 18, score_rate: 90.0, status_tag: '掌握良好', level_class: 'level-success', advice: '核心概念掌握扎实，建议考前快速浏览保持题感。' },
      { name: 'Section 2: 范围与进度', target_weight: 20, total_count: 20, correct_count: 16, score_rate: 80.0, status_tag: '达标巩固', level_class: 'level-info', advice: '基础理解良好，但需注意进度网络图关键路径细节。' },
      { name: 'Section 3: 成本与质量', target_weight: 20, total_count: 20, correct_count: 10, score_rate: 50.0, status_tag: '薄弱待强化', level_class: 'level-warning', advice: '该模块失分较多，挣值分析与质量工具为薄弱考点，需重点复习！' },
      { name: 'Section 4: 风险与采购', target_weight: 20, total_count: 20, correct_count: 17, score_rate: 85.0, status_tag: '掌握良好', level_class: 'level-success', advice: '合同类型与风险应对策略掌握熟练。' },
      { name: 'Section 5: 综合实务', target_weight: 20, total_count: 20, correct_count: 14, score_rate: 70.0, status_tag: '达标巩固', level_class: 'level-info', advice: '情景题审题需更细致，紧扣变更管理流程。' }
    ];

    this.setData({
      syllabusDiagnostics: defaultDiagnostics
    });

    // 构建符合 Stitch 原型的高质量逐题解析数据
    const sampleQuestions = [
      {
        id: 'rec_q1',
        number: 1,
        title: '在完全竞争市场模型中，以下哪一项最准确地描述了经济供求曲线之间的动态交互关系？',
        type: '单选题',
        isCorrect: true,
        userAnswer: 'A',
        correctAnswer: 'A',
        analysis: '【考点解析】在完全竞争市场中，个别厂商是价格的接受者而非制定者。市场均衡价格由全行业的总供给曲线与总需求曲线的交点共同决定。',
        knowledgePoint: '微观经济学 · 市场结构与均衡价格'
      },
      {
        id: 'rec_q2',
        number: 2,
        title: '在持续高通货膨胀（恶性通胀）时期，采用历史成本会计计量属性的主要局限性是什么？',
        type: '单选题',
        isCorrect: false,
        userAnswer: 'B',
        correctAnswer: 'D',
        analysis: '【考点解析】在恶性通胀时期，币值大幅贬值，历史成本无法反映资产与负债现时公允价值与重置成本，从而导致资产低估、虚增当期利润。',
        knowledgePoint: '财务会计 · 计量属性与物价变动会计'
      },
      {
        id: 'rec_q3',
        number: 3,
        title: '在组织行为学领导理论中，变革型领导（Transformational Leader）与事务型领导的核心区别特征是什么？',
        type: '单选题',
        isCorrect: true,
        userAnswer: 'C',
        correctAnswer: 'C',
        analysis: '【考点解析】变革型领导通过描绘愿景、智力激发与个性化关怀激励下属超越自我利益，而事务型领导主要依赖契约交易、奖惩交换维持常规运转。',
        knowledgePoint: '管理学原理 · 领导行为与激励理论'
      },
      {
        id: 'rec_q4',
        number: 4,
        title: '在网络安全体系架构中，以下哪一项最准确地阐述了“最小权限原则 (Principle of Least Privilege)”的核心要求？',
        type: '单选题',
        isCorrect: true,
        userAnswer: 'A',
        correctAnswer: 'A',
        analysis: '【考点解析】最小权限原则要求任何主体（用户或进程）仅被授予完成其合法任务所必需的最小范围权限，从而最大限度降低越权攻击面。',
        knowledgePoint: '系统架构 · 零信任与安全设计'
      },
      {
        id: 'rec_q5',
        number: 5,
        title: '在现代关系型数据库事务隔离级别中，哪种级别可以防止“脏读”和“不可重复读”，但仍可能出现“幻读”？',
        type: '单选题',
        isCorrect: true,
        userAnswer: 'C',
        correctAnswer: 'C',
        analysis: '【考点解析】根据 ANSI SQL 标准，可重复读 (Repeatable Read) 隔离级别通过行锁或 MVCC 确保事务内多次读取同一记录一致，但不能完全防止范围查询插入引起的幻读。',
        knowledgePoint: '数据库系统 · ACID 事务与并发控制'
      },
      {
        id: 'rec_q6',
        number: 6,
        title: '在微服务架构设计中，为防止局部下游依赖故障引发级联雪崩效应，应优先采用哪些弹性高可用设计模式？(多选)',
        type: '多选题',
        isCorrect: false,
        userAnswer: 'A, C',
        correctAnswer: 'A, C, D',
        analysis: '【考点解析】熔断器模式、舱壁资源隔离以及提供优雅降级（Fallback）均为高可用防御雪崩的标准模式。本题少选了 D 选项。',
        knowledgePoint: '微服务治理 · 高可用与容错架构'
      },
      {
        id: 'rec_q7',
        number: 7,
        title: '根据分布式系统的 CAP 定理，在发生网络分区 (P) 的情况下，分布式系统必须在以下哪两者之间做出权衡？',
        type: '单选题',
        isCorrect: true,
        userAnswer: 'A',
        correctAnswer: 'A',
        analysis: '【考点解析】CAP 定理指出，在分布式数据存储系统中，网络分区 (P) 是必然存在的客观现实，因此系统只能在一阶一致性 (C) 和高可用性 (A) 之间二选一。',
        knowledgePoint: '分布式系统 · CAP 定理与 BASE 理论'
      },
      {
        id: 'rec_q8',
        number: 8,
        title: '在大型高防互联网架构中，以下哪些属于应对超大规模 DDoS 攻击的有效防御策略？(多选)',
        type: '多选题',
        isCorrect: true,
        userAnswer: 'A, B, D',
        correctAnswer: 'A, B, D',
        analysis: '【考点解析】Anycast BGP 泛播路由调度、API 智能限流与云端流量清洗中心可有效分散并清洗海量攻击报文。C 选项关闭 TLS 会引入严重明文泄露隐患，非正确做法。',
        knowledgePoint: '网络工程 · DDoS 防御与边缘计算'
      }
    ];

    let correctCount = 0;
    let incorrectCount = 0;
    sampleQuestions.forEach((q) => {
      if (q.isCorrect) {
        correctCount++;
      } else {
        incorrectCount++;
      }
    });

    this.setData({
      record: {
        id: options.id || 'rec-current',
        title,
        score,
        total,
        timeUsed,
        date,
        accuracy
      },
      rankingPercent,
      analysisQuestions: sampleQuestions,
      correctCount,
      incorrectCount
    }, () => {
      this.applyFilter();
    });
  },

  /**
   * 切换筛选状态
   */
  onSwitchFilter(e) {
    const filter = e.currentTarget.dataset.filter;
    this.setData({
      activeFilter: filter
    }, () => {
      this.applyFilter();
    });
  },

  /**
   * 重新统计正确数与错误数
   */
  calculateStats() {
    const { analysisQuestions } = this.data;
    let correctCount = 0;
    let incorrectCount = 0;
    (analysisQuestions || []).forEach((q) => {
      if (q.isCorrect) {
        correctCount++;
      } else {
        incorrectCount++;
      }
    });
    this.setData({
      correctCount,
      incorrectCount
    });
  },

  /**
   * 执行筛选过滤
   */
  applyFilter() {
    const { analysisQuestions, activeFilter } = this.data;
    let list = analysisQuestions;

    if (activeFilter === 'correct') {
      list = analysisQuestions.filter((q) => q.isCorrect);
    } else if (activeFilter === 'incorrect') {
      list = analysisQuestions.filter((q) => !q.isCorrect);
    }

    this.setData({
      displayQuestions: list
    });
  },

  /**
   * 查看单题解析 -> 直接跳转到对应题库对应题目的背题模式
   */
  onViewQuestionDetail(e) {
    const item = e.currentTarget.dataset.item;
    if (!item) return;

    const targetIndex = item.number ? Math.max(0, item.number - 1) : 0;
    const examTitle = this.data.record.title || '模拟考试';

    wx.navigateTo({
      url: `/pages/quiz/quiz?mode=recite&index=${targetIndex}&title=${encodeURIComponent(examTitle)}`
    });
  },

  /**
   * 返回模考首页
   */
  onBackToHome() {
    wx.redirectTo({
      url: '/pages/mock-exam/mock-exam'
    });
  },

  /**
   * 顶部左侧返回
   */
  onNavBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack();
    } else {
      wx.reLaunch({
        url: '/pages/mock-exam/mock-exam'
      });
    }
  },

  /**
   * 底部标签栏切换
   */
  onSwitchTab(e) {
    const tab = e.currentTarget.dataset.tab;
    if (tab === 'analysis') {
      wx.redirectTo({
        url: '/pages/mock-exam/mock-exam'
      });
      return;
    }

    if (tab === 'study') {
      wx.reLaunch({
        url: '/pages/index/index'
      });
    } else if (tab === 'profile') {
      wx.reLaunch({
        url: '/pages/profile/profile'
      });
    }
  }
});
