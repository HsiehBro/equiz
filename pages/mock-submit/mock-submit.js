// pages/mock-submit/mock-submit.js
const { request, CONFIG } = require('../../utils/request.js');
const app = getApp();

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    capsuleRightPadding: 16,

    // 考试元数据
    examTitle: '全真模拟考试',
    durationMinutes: 120,
    remainingSeconds: 4845, // 默认 01:20:45
    countdownText: '01:20:45',
    timerInterval: null,

    // 筛选状态: 'all' | 'answered' | 'unanswered'
    activeFilter: 'all',

    // 题目数据
    totalQuestions: 60,
    answeredCount: 58,
    unansweredCount: 2,
    currentIndex: 57, // 默认第 58 题为当前焦点题
    questionList: [],
    displayQuestionList: [],

    // 试题完整实体与用户作答
    questions: [],
    userAnswers: {}
  },

  onLoad(options) {
    this.initSystemInfo();
    this.initSessionData(options);
    this.startCountdown();
  },

  onUnload() {
    this.clearTimer();
  },

  /**
   * 初始化系统导航栏与状态栏高度，避让微信胶囊按钮
   */
  initSystemInfo() {
    try {
      const windowInfo = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
      const statusBarHeight = windowInfo.statusBarHeight || 20;

      let navBarHeight = 44;
      let capsuleRightPadding = 16;
      if (wx.getMenuButtonBoundingClientRect) {
        const menu = wx.getMenuButtonBoundingClientRect();
        if (menu && menu.top) {
          navBarHeight = (menu.top - statusBarHeight) * 2 + menu.height;
        }
        if (menu && menu.left && windowInfo.windowWidth) {
          capsuleRightPadding = windowInfo.windowWidth - menu.left + 10;
        }
      }

      this.setData({
        statusBarHeight,
        navBarHeight,
        capsuleRightPadding
      });
    } catch (e) {
      console.warn('获取系统高度异常:', e);
    }
  },

  /**
   * 从 globalData 或路由传参加载考场实时状态
   */
  initSessionData(options) {
    const session = (app.globalData && app.globalData.currentExamSession) ? app.globalData.currentExamSession : null;

    let examTitle = '全真模拟考试';
    let durationMinutes = 120;
    let remainingSeconds = 4845; // 01:20:45
    let questions = [];
    let userAnswers = {};
    let currentIndex = 57;

    if (session) {
      examTitle = session.examTitle || examTitle;
      durationMinutes = session.durationMinutes || durationMinutes;
      remainingSeconds = session.remainingSeconds !== undefined ? session.remainingSeconds : remainingSeconds;
      questions = session.questions || [];
      userAnswers = session.userAnswers || {};
      currentIndex = session.currentIndex !== undefined ? session.currentIndex : currentIndex;
    } else {
      // 缺省构建 60 道题的标准演示数据集 (匹配 Stitch 原型 58/60 与 14, 42 未作答)
      const total = 60;
      for (let i = 0; i < total; i++) {
        const qId = `q_${i + 1}`;
        questions.push({
          id: qId,
          title: `[第 ${i + 1} 题] 模拟考试专业综合试题`,
          type: '单选题',
          correctAnswer: 'A',
          options: [
            { label: 'A', text: '选项 A 说明' },
            { label: 'B', text: '选项 B 说明' },
            { label: 'C', text: '选项 C 说明' },
            { label: 'D', text: '选项 D 说明' }
          ]
        });

        // 模拟第 14 题 (index 13) 和第 42 题 (index 41) 未作答，其余已答
        if (i !== 13 && i !== 41) {
          userAnswers[qId] = 'A';
        }
      }
      currentIndex = 57; // 第 58 题 (index 57)
    }

    const totalQuestions = questions.length || 60;

    // 构建题目矩阵状态
    const questionList = questions.map((q, idx) => {
      const ans = userAnswers[q.id];
      const isAnswered = Array.isArray(ans) ? ans.length > 0 : !!ans;
      const isCurrent = idx === currentIndex;
      return {
        id: q.id,
        index: idx,
        number: idx + 1,
        isAnswered,
        isCurrent
      };
    });

    const answeredCount = questionList.filter((item) => item.isAnswered).length;
    const unansweredCount = totalQuestions - answeredCount;

    this.setData({
      examTitle,
      durationMinutes,
      remainingSeconds,
      countdownText: this.formatTime(remainingSeconds),
      questions,
      userAnswers,
      currentIndex,
      totalQuestions,
      answeredCount,
      unansweredCount,
      questionList,
      displayQuestionList: questionList
    }, () => {
      this.updateDisplayQuestions();
    });
  },

  /**
   * 切换筛选状态 ('all' | 'answered' | 'unanswered')
   */
  onSwitchFilter(e) {
    const filter = e.currentTarget.dataset.filter;
    if (this.data.activeFilter === filter && filter !== 'all') {
      this.setData({ activeFilter: 'all' }, () => {
        this.updateDisplayQuestions();
      });
    } else {
      this.setData({ activeFilter: filter }, () => {
        this.updateDisplayQuestions();
      });
    }
  },

  /**
   * 根据当前 activeFilter 过滤显示的题目列表
   */
  updateDisplayQuestions() {
    const { questionList, activeFilter } = this.data;
    let list = questionList;

    if (activeFilter === 'answered') {
      list = questionList.filter((item) => item.isAnswered);
    } else if (activeFilter === 'unanswered') {
      list = questionList.filter((item) => !item.isAnswered);
    }

    this.setData({
      displayQuestionList: list
    });
  },

  /**
   * 启动秒级倒计时保持与考场同步
   */
  startCountdown() {
    this.clearTimer();

    this.data.timerInterval = setInterval(() => {
      let secs = this.data.remainingSeconds - 1;
      if (secs <= 0) {
        this.clearTimer();
        this.setData({
          remainingSeconds: 0,
          countdownText: '00:00:00'
        });
        this.onTimeExpired();
        return;
      }

      this.setData({
        remainingSeconds: secs,
        countdownText: this.formatTime(secs)
      });

      // 同步回 globalData
      if (app.globalData && app.globalData.currentExamSession) {
        app.globalData.currentExamSession.remainingSeconds = secs;
      }
    }, 1000);
  },

  /**
   * 清除计时器
   */
  clearTimer() {
    if (this.data.timerInterval) {
      clearInterval(this.data.timerInterval);
      this.data.timerInterval = null;
    }
  },

  /**
   * 格式化时间为 HH:MM:SS
   */
  formatTime(seconds) {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;

    const pad = (n) => (n < 10 ? '0' + n : n);
    return `${pad(h)}:${pad(m)}:${pad(s)}`;
  },

  /**
   * 倒计时自然结束
   */
  onTimeExpired() {
    wx.showModal({
      title: '考试时间已结束',
      content: '模拟考试规定时长已用尽，系统将自动为您提交当前答题卡并计算成绩。',
      showCancel: false,
      confirmText: '查看成绩',
      confirmColor: '#0058bc',
      success: () => {
        this.onConfirmSubmit();
      }
    });
  },

  /**
   * 点击题目序号跳转回考场对应题目
   */
  onJumpToQuestion(e) {
    const index = parseInt(e.currentTarget.dataset.index, 10);
    if (!isNaN(index) && index >= 0 && index < this.data.totalQuestions) {
      if (app.globalData) {
        app.globalData.jumpToIndex = index;
      }
      this.onReturnToExam();
    }
  },

  /**
   * 顶部左侧返回或返回作答按钮
   */
  onNavBack() {
    this.onReturnToExam();
  },

  /**
   * 返回考场继续作答
   */
  onReturnToExam() {
    this.clearTimer();
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack();
    } else {
      wx.redirectTo({
        url: '/pages/mock-run/mock-run'
      });
    }
  },

  /**
   * 确认并提交试卷 (Confirm & Submit Exam)
   */
  onConfirmSubmit() {
    this.clearTimer();

    const total = this.data.questions.length;
    const timeUsedSecs = Math.max(1, this.data.durationMinutes * 60 - this.data.remainingSeconds);
    const session = (app && app.globalData && app.globalData.currentExamSession) ? app.globalData.currentExamSession : null;

    if (!CONFIG.USE_MOCK) {
      wx.showLoading({ title: '正在提交试卷打分...', mask: true });

      // 格式化用户答案映射
      const userAnswersPayload = {};
      Object.keys(this.data.userAnswers).forEach((key) => {
        const val = this.data.userAnswers[key];
        userAnswersPayload[key] = Array.isArray(val) ? val : (val ? [val] : []);
      });

      // 组装试题实体
      const questionsPayload = this.data.questions.map((q) => {
        let realId = q.id;
        if (typeof realId !== 'number') {
          realId = parseInt(String(realId).replace(/\D/g, ''), 10) || 0;
        }
        let ans = [];
        if (q.rawQuestion && q.rawQuestion.answer) {
          ans = q.rawQuestion.answer;
        } else if (Array.isArray(q.correctAnswer)) {
          ans = q.correctAnswer;
        } else if (q.correctAnswer) {
          ans = [q.correctAnswer];
        }
        return {
          id: realId,
          title: q.title,
          type: (q.type && q.type.includes('多选')) ? '多选' : '单选',
          section: q.section || '默认大纲分类',
          options: q.options ? q.options.map(o => ({ key: o.label || o.key, text: o.text })) : [],
          answer: ans
        };
      });

      request({
        url: '/api/v1/mock/submit',
        method: 'POST',
        data: {
          bank_id: (session && session.bankId) ? session.bankId : 1,
          exam_title: this.data.examTitle || '全真模拟考试',
          time_used_seconds: timeUsedSecs,
          total_questions: total,
          questions: questionsPayload,
          user_answers: userAnswersPayload
        }
      })
        .then((record) => {
          wx.hideLoading();

          // 同步错题至本地错题集缓存
          try {
            const errorsMap = wx.getStorageSync('user_errors_map') || {};
            const bankId = (session && session.bankId) ? session.bankId : 1;
            questionsPayload.forEach(q => {
              const userAns = userAnswersPayload[q.id] || [];
              const correctAns = q.answer || [];
              const userJoined = userAns.slice().sort().join('');
              const corrJoined = correctAns.slice().sort().join('');
              const isCorr = (userJoined === corrJoined) && userJoined.length > 0;
              if (!isCorr && q.id > 0) {
                errorsMap[q.id] = {
                  question_id: q.id,
                  bank_id: Number(bankId) || 1,
                  question: q,
                  wrong_count: (errorsMap[q.id] ? errorsMap[q.id].wrong_count : 0) + 1,
                  is_mastered: false,
                  last_wrong_at: new Date().toISOString()
                };
              }
            });
            wx.setStorageSync('user_errors_map', errorsMap);
          } catch (e) {}

          if (app && app.globalData) {
            app.globalData.currentExamSession = null;
            app.globalData.jumpToIndex = null;
          }
          wx.showToast({ title: '交卷成功', icon: 'success' });
          setTimeout(() => {
            wx.redirectTo({
              url: `/pages/mock-record/mock-record?id=${record.id}`
            });
          }, 400);
        })
        .catch((err) => {
          wx.hideLoading();
          console.warn('[MockSubmit] 后端评分异常，启用本地降级:', err);
          this.submitLocalFallback();
        });
      return;
    }

    this.submitLocalFallback();
  },

  submitLocalFallback() {
    const total = this.data.questions.length;
    let correctCount = 0;

    this.data.questions.forEach((q) => {
      const ans = this.data.userAnswers[q.id];
      if (q.type === '单选题') {
        if (ans === q.correctAnswer) {
          correctCount++;
        }
      } else {
        // 多选题
        if (Array.isArray(ans) && Array.isArray(q.correctAnswer)) {
          if (ans.length === q.correctAnswer.length && ans.every((val, idx) => val === q.correctAnswer[idx])) {
            correctCount++;
          }
        }
      }
    });

    const score = Math.round((correctCount / total) * 100);
    const accuracy = Math.round((correctCount / total) * 100);
    const timeUsedSecs = this.data.durationMinutes * 60 - this.data.remainingSeconds;
    const timeUsedMins = Math.max(1, Math.ceil(timeUsedSecs / 60));
    const isPass = score >= 60;

    let newRecord = null;
    try {
      const now = new Date();
      const dateStr = `${now.getFullYear()}年${now.getMonth() + 1}月${now.getDate()}日`;
      newRecord = {
        id: 'rec-' + Date.now(),
        date: dateStr,
        title: this.data.examTitle,
        scoreText: `得分: ${score} / 100 分 · ${isPass ? '及格' : '未及格'}`,
        score: score,
        total: 100,
        timeUsed: `${timeUsedMins} 分钟`,
        accuracy: `${accuracy}%`,
        isCompleted: true
      };

      const existingRecords = wx.getStorageSync('mock_exam_records') || [];
      existingRecords.unshift(newRecord);
      wx.setStorageSync('mock_exam_records', existingRecords);
    } catch (e) {
      console.warn('保存模考历史记录异常:', e);
    }

    if (app.globalData) {
      app.globalData.currentExamSession = null;
      app.globalData.jumpToIndex = null;
    }

    wx.showToast({
      title: '交卷成功，生成报告中...',
      icon: 'success',
      duration: 1000
    });

    setTimeout(() => {
      const targetUrl = `/pages/mock-record/mock-record?id=${newRecord ? newRecord.id : ''}&score=${score}&total=100&timeUsed=${encodeURIComponent(timeUsedMins + ' 分钟')}&accuracy=${encodeURIComponent(accuracy + '%')}&title=${encodeURIComponent(this.data.examTitle)}&date=${encodeURIComponent(newRecord ? newRecord.date : '')}`;
      wx.redirectTo({
        url: targetUrl
      });
    }, 600);
  }
});
