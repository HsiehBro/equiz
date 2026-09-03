// pages/mock-exam/mock-exam.js
const { request, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    showDrawer: false,
    daysRemaining: 14,

    // 模考大纲与范围列表
    syllabusList: [
      { id: 'full', name: '全科综合模拟考 (Full Syllabus)', defaultDuration: 120, defaultQuestions: 100 },
      { id: 'quantitative', name: '数量关系专项模考 (Quantitative Focus)', defaultDuration: 60, defaultQuestions: 50 },
      { id: 'verbal', name: '言语理解专项模考 (Verbal Focus)', defaultDuration: 60, defaultQuestions: 50 },
      { id: 'professional', name: '专业综合强化模考 (Professional Focus)', defaultDuration: 90, defaultQuestions: 80 }
    ],
    selectedSyllabusIndex: 0,

    // 考试参数
    durationMinutes: 120,
    questionCount: 100,

    // 查看全部历史记录开关
    showAllRecords: false,

    // 模拟考试历史记录数据集
    records: [
      {
        id: 'rec-001',
        date: '2023年10月24日',
        title: '全科综合模拟考',
        scoreText: '得分: 85 / 100 分 · 及格',
        score: 85,
        total: 100,
        timeUsed: '104 分钟',
        accuracy: '85%',
        isCompleted: false
      },
      {
        id: 'rec-002',
        date: '2023年10月20日',
        title: '数量关系专项模考',
        scoreText: '已完成 · 78 分',
        score: 78,
        total: 100,
        timeUsed: '48 分钟',
        accuracy: '78%',
        isCompleted: true
      },
      {
        id: 'rec-003',
        date: '2023年10月15日',
        title: '言语理解专项模考',
        scoreText: '得分: 92 / 100 分 · 优秀',
        score: 92,
        total: 100,
        timeUsed: '42 分钟',
        accuracy: '92%',
        isCompleted: false
      },
      {
        id: 'rec-004',
        date: '2023年10月08日',
        title: '全科综合模拟考',
        scoreText: '得分: 79 / 100 分 · 及格',
        score: 79,
        total: 100,
        timeUsed: '118 分钟',
        accuracy: '79%',
        isCompleted: false
      }
    ],
    displayRecords: []
  },

  onLoad() {
    this.initSystemInfo();
    this.loadHistoryRecords();
  },

  onShow() {
    this.loadHistoryRecords();
  },

  /**
   * 从后端接口或本地缓存加载模考历史记录
   */
  loadHistoryRecords() {
    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/mock/records' })
        .then((records) => {
          if (Array.isArray(records) && records.length > 0) {
            const formatted = records.map((r) => ({
              id: String(r.id),
              date: r.date_text || '近期',
              title: r.exam_title,
              scoreText: r.score_text || `得分: ${r.score} 分`,
              score: r.score,
              total: r.total_score || 100,
              timeUsed: r.time_used_text || `${Math.round(r.time_used_seconds / 60)} 分钟`,
              accuracy: `${Math.round((r.correct_count / (r.total_questions || 1)) * 100)}%`,
              isCompleted: true
            }));
            this.setData({ records: formatted }, () => {
              this.updateDisplayRecords();
            });
            return;
          }
          this.loadHistoryRecordsFallback();
        })
        .catch((err) => {
          console.warn('[MockExam] 加载云端模考记录失败，降级本地:', err);
          this.loadHistoryRecordsFallback();
        });
    } else {
      this.loadHistoryRecordsFallback();
    }
  },

  loadHistoryRecordsFallback() {
    try {
      const userRecords = wx.getStorageSync('mock_exam_records') || [];
      const defaultPresets = [
        {
          id: 'rec-preset-001',
          date: '2023年10月24日',
          title: '全科综合模拟考',
          scoreText: '得分: 85 / 100 分 · 及格',
          score: 85,
          total: 100,
          timeUsed: '104 分钟',
          accuracy: '85%',
          isCompleted: false
        },
        {
          id: 'rec-preset-002',
          date: '2023年10月20日',
          title: '数量关系专项模考',
          scoreText: '已完成 · 78 分',
          score: 78,
          total: 100,
          timeUsed: '48 分钟',
          accuracy: '78%',
          isCompleted: true
        }
      ];

      const combinedRecords = [...userRecords, ...defaultPresets];
      this.setData({
        records: combinedRecords
      }, () => {
        this.updateDisplayRecords();
      });
    } catch (e) {
      console.warn('加载模考记录异常:', e);
      this.updateDisplayRecords();
    }
  },

  /**
   * 初始化系统导航栏与状态栏高度
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
      console.warn('获取导航栏系统高度异常:', e);
    }
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
        url: '/pages/index/index'
      });
    }
  },

  /**
   * 模考类型范围变更
   */
  onSyllabusChange(e) {
    const index = parseInt(e.detail.value, 10);
    const item = this.data.syllabusList[index];
    if (item) {
      this.setData({
        selectedSyllabusIndex: index,
        durationMinutes: item.defaultDuration,
        questionCount: item.defaultQuestions
      });

      wx.showToast({
        title: `已应用「${item.name}」推荐参数`,
        icon: 'none',
        duration: 1500
      });
    }
  },

  /**
   * 考试时长输入
   */
  onDurationInput(e) {
    const val = parseInt(e.detail.value, 10);
    this.setData({
      durationMinutes: isNaN(val) ? '' : val
    });
  },

  /**
   * 题目数量输入
   */
  onQuestionCountInput(e) {
    const val = parseInt(e.detail.value, 10);
    this.setData({
      questionCount: isNaN(val) ? '' : val
    });
  },

  /**
   * 开始全真模拟考试
   */
  onStartSimulation() {
    const { syllabusList, selectedSyllabusIndex, durationMinutes, questionCount } = this.data;
    const currentSyllabus = syllabusList[selectedSyllabusIndex];

    const duration = parseInt(durationMinutes, 10);
    const count = parseInt(questionCount, 10);

    if (!duration || duration < 10 || duration > 300) {
      wx.showToast({
        title: '请输入有效的考试时长 (10-300 分钟)',
        icon: 'none'
      });
      return;
    }

    if (!count || count < 5 || count > 200) {
      wx.showToast({
        title: '请输入有效的题目数量 (5-200 题)',
        icon: 'none'
      });
      return;
    }

    wx.showModal({
      title: '进入全真模拟考场',
      content: `考试类型: ${currentSyllabus.name}\n考试时长: ${duration} 分钟\n题目总数: ${count} 题\n\n进入考场后将正式启动倒计时，请保持专注！`,
      confirmText: '开始答题',
      confirmColor: '#0058bc',
      cancelText: '再检查下',
      success: (res) => {
        if (res.confirm) {
          if (!CONFIG.USE_MOCK) {
            wx.showLoading({ title: '按考纲组卷中...', mask: true });
            request({
              url: '/api/v1/mock/start',
              method: 'POST',
              data: {
                bank_id: 1, // 默认关联项目管理官方题库
                total_questions: count,
                duration_minutes: duration
              }
            })
              .then((paper) => {
                wx.hideLoading();
                const app = getApp();
                if (app) {
                  app.globalData.currentExamSession = {
                    bankId: paper.bank_id,
                    examTitle: paper.exam_title || currentSyllabus.name,
                    durationMinutes: paper.duration_minutes || duration,
                    remainingSeconds: (paper.duration_minutes || duration) * 60,
                    questions: paper.questions,
                    userAnswers: {},
                    currentIndex: 0
                  };
                }
                wx.navigateTo({
                  url: `/pages/mock-run/mock-run?bankId=${paper.bank_id}&examTitle=${encodeURIComponent(paper.exam_title || currentSyllabus.name)}&duration=${duration}&count=${count}`
                });
              })
              .catch((err) => {
                wx.hideLoading();
                wx.showModal({
                  title: '开考条件不满足',
                  content: err.message || '大纲考点试题不足，无法生成试卷',
                  showCancel: false,
                  confirmColor: '#ba1a1a'
                });
              });
          } else {
            wx.navigateTo({
              url: `/pages/mock-run/mock-run?examTitle=${encodeURIComponent(currentSyllabus.name)}&duration=${duration}&count=${count}`
            });
          }
        }
      }
    });
  },

  /**
   * 更新历史记录展示列表
   */
  updateDisplayRecords() {
    const { records, showAllRecords } = this.data;
    const displayRecords = showAllRecords ? records : records.slice(0, 2);
    this.setData({ displayRecords });
  },

  /**
   * 展开/收起全部历史记录
   */
  onToggleViewAll() {
    this.setData({
      showAllRecords: !this.data.showAllRecords
    }, () => {
      this.updateDisplayRecords();
    });
  },

  /**
   * 查看历史记录详情 -> 跳转至模考记录与分析页
   */
  onTapRecord(e) {
    const record = e.currentTarget.dataset.record;
    if (!record) return;

    const url = `/pages/mock-record/mock-record?id=${encodeURIComponent(record.id || '')}&title=${encodeURIComponent(record.title || '')}&score=${record.score || 85}&total=${record.total || 100}&timeUsed=${encodeURIComponent(record.timeUsed || '')}&date=${encodeURIComponent(record.date || '')}&accuracy=${encodeURIComponent(record.accuracy || '')}`;
    wx.navigateTo({
      url
    });
  },

  /**
   * 底部标签栏切换 (3 项)
   */
  onSwitchTab(e) {
    const tab = e.currentTarget.dataset.tab;
    if (tab === 'analysis') {
      return; // 当前页面
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
