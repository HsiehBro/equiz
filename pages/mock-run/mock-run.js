// pages/mock-run/mock-run.js
const app = getApp();

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    capsuleRightPadding: 16,
    examTitle: '全真模拟考试',
    
    // 倒计时相关
    durationMinutes: 120,
    remainingSeconds: 7200, // 02:00:00
    countdownText: '02:00:00',
    isTimeLow: false, // 是否少于 5 分钟
    timerInterval: null,

    // 题目状态
    currentIndex: 0,
    progressPercent: 1.6,
    userAnswers: {}, // { [questionId]: 'A' | ['A', 'B'] }
    currentSelectedMap: {}, // { 'A': true, 'B': false, ... }
    answeredCount: 0,

    // 答题卡抽屉显隐
    showAnswerSheet: false,

    // 试题库列表
    questions: []
  },

  onLoad(options) {
    this.initSystemInfo();
    this.initExamParams(options);
    this.initQuestionBank(options);
    this.startCountdown();
  },

  onShow() {
    // 检查是否有来自交卷汇总确认页的题号跳转指令
    if (app && app.globalData) {
      if (app.globalData.currentExamSession && app.globalData.currentExamSession.remainingSeconds !== undefined) {
        const secs = app.globalData.currentExamSession.remainingSeconds;
        this.setData({
          remainingSeconds: secs,
          countdownText: this.formatTime(secs),
          isTimeLow: secs < 300
        });
      }

      if (app.globalData.jumpToIndex !== null && app.globalData.jumpToIndex !== undefined) {
        const targetIndex = app.globalData.jumpToIndex;
        app.globalData.jumpToIndex = null;
        if (targetIndex >= 0 && targetIndex < this.data.questions.length) {
          this.setData({
            currentIndex: targetIndex,
            progressPercent: (((targetIndex + 1) / this.data.questions.length) * 100).toFixed(1)
          }, () => {
            this.updateCurrentSelectedMap();
          });
        }
      }
    }
  },

  onUnload() {
    this.clearTimer();
  },

  /**
   * 初始化状态栏与导航栏高度，并避让微信右上角胶囊按钮
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
   * 解析路由传参并初始化考试配置
   */
  initExamParams(options) {
    const title = options.examTitle ? decodeURIComponent(options.examTitle) : '全真模拟考';
    const duration = parseInt(options.duration, 10) || 120;
    const totalSecs = duration * 60;

    this.setData({
      examTitle: title,
      durationMinutes: duration,
      remainingSeconds: totalSecs,
      countdownText: this.formatTime(totalSecs)
    });
  },

  /**
   * 启动秒级倒计时
   */
  startCountdown() {
    this.clearTimer();

    this.data.timerInterval = setInterval(() => {
      let secs = this.data.remainingSeconds - 1;
      if (secs <= 0) {
        this.clearTimer();
        this.setData({
          remainingSeconds: 0,
          countdownText: '00:00:00',
          isTimeLow: true
        });
        this.onTimeExpired();
        return;
      }

      const isTimeLow = secs < 300; // 少于 5 分钟预警
      this.setData({
        remainingSeconds: secs,
        countdownText: this.formatTime(secs),
        isTimeLow
      });
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
   * 倒计时结束强制交卷
   */
  onTimeExpired() {
    wx.showModal({
      title: '考试时间已结束',
      content: '模拟考试规定时长已用尽，系统将自动为您提交当前答题卡并计算成绩。',
      showCancel: false,
      confirmText: '查看成绩',
      confirmColor: '#0058bc',
      success: () => {
        this.calculateAndShowResult();
      }
    });
  },

  /**
   * 构建试题数据集 (包含单选题与多选题)
   */
  initQuestionBank(options) {
    const session = (app && app.globalData && app.globalData.currentExamSession) ? app.globalData.currentExamSession : null;
    if (session && session.questions && session.questions.length > 0) {
      const formatted = session.questions.map((q, idx) => ({
        id: q.id,
        number: idx + 1,
        title: q.title,
        context: q.section ? `【大纲考点: ${q.section}】` : '',
        section: q.section || '大纲考点',
        type: (q.type && q.type.includes('多选')) ? '多选题' : '单选题',
        correctAnswer: (q.type && q.type.includes('多选')) ? q.answer : (q.answer && q.answer[0] ? q.answer[0] : 'A'),
        options: (q.options || []).map(opt => ({
          label: opt.key || opt.label,
          text: opt.text
        })),
        rawQuestion: q
      }));
      this.setData({
        questions: formatted,
        progressPercent: ((1 / formatted.length) * 100).toFixed(1)
      });
      this.updateCurrentSelectedMap();
      return;
    }

    const count = parseInt(options.count, 10) || 60;
    
    // 基础试题池 (中文专业题库，单选题与多选题混合)
    const baseQuestions = [
      {
        id: 'q1',
        title: "在网络安全体系架构中，以下哪一项最准确地阐述了“最小权限原则 (Principle of Least Privilege)”的核心要求？",
        context: "请结合零信任 (Zero Trust) 架构中微服务间的动态细粒度鉴权机制进行分析。",
        type: "单选题",
        correctAnswer: "A",
        options: [
          { label: 'A', text: '任何用户、进程或服务仅应被赋予完成其指定职责所必需的最低访问权限级别。' },
          { label: 'B', text: '所有网络内部通信流量必须默认强制加密，以确保数据传输阶段的信息不泄露。' },
          { label: 'C', text: '系统管理员在登录和操作核心生产集群前，必须无条件通过多因素身份认证 (MFA)。' },
          { label: 'D', text: '软硬件系统在发生异常故障时应自动进入闭锁安全状态，默认拒绝一切后续访问请求。' }
        ]
      },
      {
        id: 'q2',
        title: "在主流关系型数据库的标准事务隔离级别中，哪一种级别可以有效防止“脏读”和“不可重复读”，但仍无法避免“幻读”现象？",
        context: "假定遵循 ANSI SQL 标准事务模型，并基于多版本并发控制 (MVCC) 实现。",
        type: "单选题",
        correctAnswer: "C",
        options: [
          { label: 'A', text: '读未提交 (Read Uncommitted)' },
          { label: 'B', text: '读已提交 (Read Committed)' },
          { label: 'C', text: '可重复读 (Repeatable Read)' },
          { label: 'D', text: '串行化 (Serializable)' }
        ]
      },
      {
        id: 'q3',
        title: "在设计高并发、高吞吐的分布式微服务架构时，通常采用哪种机制来实现业务跨服务的“最终一致性 (Eventual Consistency)”？",
        context: "结合基于消息中间件 (如 Kafka/RocketMQ) 与 Saga 分布式长事务补偿模式。",
        type: "单选题",
        correctAnswer: "B",
        options: [
          { label: 'A', text: '跨所有参与节点强制开启严格两阶段提交 (2PC) 阻塞式协议' },
          { label: 'B', text: '采用异步事件驱动消息发布与消费，配合失败补偿事务机制' },
          { label: 'C', text: '通过集中式全局分布式排他锁配合强同步 RPC 阻塞等待' },
          { label: 'D', text: '跨多个异地机房进行强一致性的法定人数多副本即时写入' }
        ]
      },
      {
        id: 'q4',
        title: "根据分布式系统著名的 CAP 定理，当集群遭遇网络分区故障 (Network Partition, P) 时，架构师必须在以下哪两个属性之间做出取舍？",
        context: "请结合大规模分布式数据库与分布式缓存的高可用部署实践进行考量。",
        type: "单选题",
        correctAnswer: "A",
        options: [
          { label: 'A', text: '数据一致性 (Consistency) 与 系统可用性 (Availability)' },
          { label: 'B', text: '系统处理性能 (Performance) 与 数据持久性 (Durability)' },
          { label: 'C', text: '事务原子性 (Atomicity) 与 事务隔离性 (Isolation)' },
          { label: 'D', text: '网络响应延迟 (Latency) 与 集群水平可扩展性 (Scalability)' }
        ]
      },
      {
        id: 'q5',
        title: "在常见的内存缓存系统 (如 Redis) 的淘汰策略中，哪种策略会优先淘汰“最长时间未被访问”的数据项？",
        context: "分析缓存容量达到上限时的内存回收算法。",
        type: "单选题",
        correctAnswer: "A",
        options: [
          { label: 'A', text: 'LRU (Least Recently Used，最近最少使用淘汰算法)' },
          { label: 'B', text: 'LFU (Least Frequently Used，最不经常使用淘汰算法)' },
          { label: 'C', text: 'FIFO (First In First Out，先进先出淘汰算法)' },
          { label: 'D', text: 'Random (随机淘汰算法)' }
        ]
      },
      {
        id: 'q6',
        title: "在大型高防互联网架构中，以下哪些属于应对超大规模 DDoS (分布式拒绝服务) 攻击的有效防御策略？(多选)",
        context: "结合边缘网络防护、CDN 分发加速与云端流量清洗中心进行综合研判。",
        type: "多选题",
        correctAnswer: ["A", "B", "D"],
        options: [
          { label: 'A', text: '在全球分布式边缘接入点 (PoP) 部署 Anycast BGP 泛播路由技术进行流量分散' },
          { label: 'B', text: '在 API 网关层部署智能动态限流与行为式验证码 (CAPTCHA) 识别异常特征' },
          { label: 'C', text: '在反向代理接入层直接禁用 HTTPS/TLS 加密以节省服务器握手 CPU 开销' },
          { label: 'D', text: '引入云端流量清洗中心 (Scrubbing Center) 并在 CDN 边缘缓存所有静态资源' }
        ]
      },
      {
        id: 'q7',
        title: "在现代企业级 Web 基础设施中，反向代理 (Reverse Proxy) 服务器的核心职责是什么？",
        context: "评估在后端业务集群前端部署 NGINX 或 Envoy 进行流量调度的典型应用架构。",
        type: "单选题",
        correctAnswer: "C",
        options: [
          { label: 'A', text: '单向保护终端客户端免受来自互联网的恶意流量扫描与攻击' },
          { label: 'B', text: '帮助内部局域网用户绕过企业防火墙直接访问外部公网应用' },
          { label: 'C', text: '接收客户端请求并智能转发至后端服务集群，提供 SSL 卸载与负载均衡' },
          { label: 'D', text: '替代权威 DNS 根服务器提供全球顶级域名的权威递归解析' }
        ]
      },
      {
        id: 'q8',
        title: "在微服务架构设计中，为防止局部下游依赖服务故障引发级联崩溃（雪崩效应），应优先采用哪些弹性高可用设计模式？(多选)",
        context: "参考高可用稳定性建设中的故障隔离、熔断器与容错降级规范。",
        type: "多选题",
        correctAnswer: ["A", "C", "D"],
        options: [
          { label: 'A', text: '熔断器模式 (Circuit Breaker Pattern)' },
          { label: 'B', text: '无间隔且无退避算法的无限级联同步递归重试' },
          { label: 'C', text: '舱壁资源隔离模式 (Bulkhead Pattern)' },
          { label: 'D', text: '优雅降级策略与提供备用 Fallback 托底数据' }
        ]
      }
    ];

    // 生成填充至指定 count 题量
    const fullQuestions = [];
    for (let i = 0; i < count; i++) {
      const base = baseQuestions[i % baseQuestions.length];
      fullQuestions.push({
        id: `q_${i + 1}`,
        title: `[第 ${i + 1} 题] ${base.title}`,
        context: base.context,
        type: base.type,
        correctAnswer: base.correctAnswer,
        options: base.options
      });
    }

    this.setData({
      questions: fullQuestions,
      progressPercent: ((1 / fullQuestions.length) * 100).toFixed(1)
    });

    this.updateCurrentSelectedMap();
  },

  /**
   * 更新当前题目的选中映射状态
   */
  updateCurrentSelectedMap() {
    const currentQ = this.data.questions[this.data.currentIndex];
    if (!currentQ) return;

    const answer = this.data.userAnswers[currentQ.id];
    const map = {};

    if (currentQ.type === '单选题') {
      if (typeof answer === 'string') {
        map[answer] = true;
      }
    } else {
      // 多选题
      if (Array.isArray(answer)) {
        answer.forEach((item) => {
          map[item] = true;
        });
      }
    }

    this.setData({
      currentSelectedMap: map
    });
  },

  /**
   * 用户选择选项 (无震动，支持单选与多选)
   */
  onSelectOption(e) {
    const label = e.currentTarget.dataset.label;
    const currentQ = this.data.questions[this.data.currentIndex];
    if (!currentQ) return;

    let newAnswers = { ...this.data.userAnswers };

    if (currentQ.type === '单选题') {
      // 单选题：点击已选中的选项可取消选中，点击其他选项则切换
      if (newAnswers[currentQ.id] === label) {
        delete newAnswers[currentQ.id];
      } else {
        newAnswers[currentQ.id] = label;
      }
    } else {
      // 多选题：多选切换 (可自由选中与取消)
      let selectedArr = Array.isArray(newAnswers[currentQ.id]) ? [...newAnswers[currentQ.id]] : [];
      const idx = selectedArr.indexOf(label);
      if (idx > -1) {
        selectedArr.splice(idx, 1);
      } else {
        selectedArr.push(label);
      }
      selectedArr.sort();

      if (selectedArr.length > 0) {
        newAnswers[currentQ.id] = selectedArr;
      } else {
        delete newAnswers[currentQ.id];
      }
    }

    // 统计已作答题数
    const answeredCount = Object.keys(newAnswers).filter((k) => {
      const v = newAnswers[k];
      return Array.isArray(v) ? v.length > 0 : !!v;
    }).length;

    this.setData({
      userAnswers: newAnswers,
      answeredCount
    }, () => {
      this.updateCurrentSelectedMap();
    });
  },

  /**
   * 上一题
   */
  onPrevQuestion() {
    if (this.data.currentIndex > 0) {
      const nextIndex = this.data.currentIndex - 1;
      this.setData({
        currentIndex: nextIndex,
        progressPercent: (((nextIndex + 1) / this.data.questions.length) * 100).toFixed(1)
      }, () => {
        this.updateCurrentSelectedMap();
      });
    }
  },

  /**
   * 下一题 / 结束
   */
  onNextQuestion() {
    if (this.data.currentIndex < this.data.questions.length - 1) {
      const nextIndex = this.data.currentIndex + 1;
      this.setData({
        currentIndex: nextIndex,
        progressPercent: (((nextIndex + 1) / this.data.questions.length) * 100).toFixed(1)
      }, () => {
        this.updateCurrentSelectedMap();
      });
    } else {
      // 最后一题提示交卷
      this.onTapSubmit();
    }
  },

  /**
   * 打开/关闭答题卡抽屉
   */
  onToggleAnswerSheet() {
    this.setData({
      showAnswerSheet: !this.data.showAnswerSheet
    });
  },

  /**
   * 答题卡点击序号跳转
   */
  onJumpToQuestion(e) {
    const index = parseInt(e.currentTarget.dataset.index, 10);
    if (!isNaN(index) && index >= 0 && index < this.data.questions.length) {
      this.setData({
        currentIndex: index,
        showAnswerSheet: false,
        progressPercent: (((index + 1) / this.data.questions.length) * 100).toFixed(1)
      }, () => {
        this.updateCurrentSelectedMap();
      });
    }
  },

  /**
   * 点击顶部关闭退出 (不暂存进度，直接放弃退出考场)
   */
  onTapClose() {
    wx.showModal({
      title: '确认退出考场？',
      content: '退出后将不保留本次答题进度与作答记录，且不会记录在模考历史中。确定要退出吗？',
      cancelText: '继续答题',
      confirmText: '退出考场',
      confirmColor: '#ba1a1a',
      success: (res) => {
        if (res.confirm) {
          this.clearTimer();
          // 不保存任何答题进度与历史记录，直接返回
          wx.navigateBack();
        }
      }
    });
  },

  /**
   * 点击交卷 -> 跳转至「模拟考试-交卷确认与答题卡汇总」页面
   */
  onTapSubmit() {
    this.setData({ showAnswerSheet: false });

    // 将考场实时状态同步至 globalData
    if (app && app.globalData) {
      app.globalData.currentExamSession = {
        examTitle: this.data.examTitle,
        durationMinutes: this.data.durationMinutes,
        remainingSeconds: this.data.remainingSeconds,
        questions: this.data.questions,
        userAnswers: this.data.userAnswers,
        currentIndex: this.data.currentIndex
      };
    }

    wx.navigateTo({
      url: '/pages/mock-submit/mock-submit'
    });
  },

  /**
   * 计算得分并展示考场成绩报告 (只有正常交卷才保存到历史记录)
   */
  calculateAndShowResult() {
    this.clearTimer();
    this.setData({ showAnswerSheet: false });

    const total = this.data.questions.length;
    let correctCount = 0;

    this.data.questions.forEach((q) => {
      const ans = this.data.userAnswers[q.id];
      if (q.type === '单选题') {
        if (ans === q.correctAnswer) {
          correctCount++;
        }
      } else {
        // 多选题判断
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
    const resultTitle = isPass ? '🎉 恭喜通过全真模拟考试！' : '✍️ 模拟考试结束，再接再厉！';

    // 只有正常交卷才将成绩保存到模考历史记录中
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
