/**
 * utils/studyStats.js
 * 个人中心学习数据看板统一状态管理模块
 * 核心指标：
 * 1. 累计打卡 (checkInDays)：初始为0，与学习规划题库绑定，完成每日目标后打卡+1，更换题库清空为0。
 * 2. 累计答题 (totalQuestions)：初始为0，无论刷哪个题库答对或答错都+1，模拟考试严格不计入。
 * 3. 平均正确率 (accuracyRate)：初始为100，由累计答对数 / 累计答题数 * 100 计算所得。
 */

const { request, CONFIG } = require('./request.js');

const STATS_KEY = 'user_study_stats';
const PLAN_KEY = 'user_study_plan';

/**
 * 获取今日日期字符串 YYYY-MM-DD
 */
function getTodayDateStr() {
  const now = new Date();
  const yyyy = now.getFullYear();
  const mm = String(now.getMonth() + 1).padStart(2, '0');
  const dd = String(now.getDate()).padStart(2, '0');
  return `${yyyy}-${mm}-${dd}`;
}

/**
 * 获取默认统计数据
 */
function getDefaultStats() {
  return {
    totalQuestions: 0,
    correctQuestions: 0,
    accuracyRate: 100
  };
}

/**
 * 获取当前学习统计数据
 */
function getStudyStats() {
  let stats = getDefaultStats();
  try {
    const cached = wx.getStorageSync(STATS_KEY);
    if (cached && typeof cached.totalQuestions === 'number') {
      stats = {
        totalQuestions: cached.totalQuestions || 0,
        correctQuestions: cached.correctQuestions || 0,
        accuracyRate: cached.totalQuestions > 0 
          ? Math.round((cached.correctQuestions / cached.totalQuestions) * 100) 
          : 100
      };
    }
  } catch (e) {
    console.warn('[studyStats] 读取本地 stats 异常:', e);
  }

  // 从当前学习规划读取绑定的累计打卡天数与关联题库
  let checkInDays = 0;
  let bankId = '';
  let bankTitle = '';
  let dailyGoal = 30;
  let todayCount = 0;
  let lastCheckInDate = '';

  try {
    const plan = wx.getStorageSync(PLAN_KEY) || {};
    bankId = String(plan.bankId || (plan.bank && plan.bank.id) || '');
    bankTitle = plan.bankTitle || (plan.bank && plan.bank.title) || '';
    checkInDays = parseInt(plan.checkInDays, 10) || 0;
    dailyGoal = parseInt(plan.dailyGoal, 10) || 30;
    lastCheckInDate = plan.lastCheckInDate || '';

    const todayStr = getTodayDateStr();
    if (plan.lastPracticeDate === todayStr) {
      todayCount = parseInt(plan.todayCount, 10) || 0;
    } else {
      todayCount = 0;
    }
  } catch (e) {
    console.warn('[studyStats] 读取本地 plan 异常:', e);
  }

  return {
    ...stats,
    checkInDays,
    bankId,
    bankTitle,
    dailyGoal,
    todayCount,
    lastCheckInDate
  };
}

/**
 * 记录普通练习单题作答结果（模拟考试不得调用该函数）
 * @param {string|number} bankId - 当前刷题所属题库ID
 * @param {boolean} isCorrect - 当前题目是否回答正确
 * @returns {object} { justCheckedIn: boolean, checkInDays: number, dailyGoal: number, stats: object }
 */
function recordAnswer(bankId, isCorrect) {
  let stats = getDefaultStats();
  try {
    const cached = wx.getStorageSync(STATS_KEY);
    if (cached && typeof cached.totalQuestions === 'number') {
      stats = cached;
    }
  } catch (e) {}

  // 1. 累计答题 +1，答对则答对数 +1
  stats.totalQuestions = (stats.totalQuestions || 0) + 1;
  if (isCorrect) {
    stats.correctQuestions = (stats.correctQuestions || 0) + 1;
  }
  stats.accuracyRate = stats.totalQuestions > 0 
    ? Math.round((stats.correctQuestions / stats.totalQuestions) * 100) 
    : 100;

  try {
    wx.setStorageSync(STATS_KEY, stats);
  } catch (e) {
    console.warn('[studyStats] 保存本地 stats 异常:', e);
  }

  // 2. 检查与更新当前学习规划进度及打卡状态
  let justCheckedIn = false;
  let checkInDays = 0;
  let dailyGoal = 30;

  try {
    const plan = wx.getStorageSync(PLAN_KEY) || null;
    if (plan) {
      const planBankId = String(plan.bankId || (plan.bank && plan.bank.id) || '');
      dailyGoal = parseInt(plan.dailyGoal, 10) || 30;
      checkInDays = parseInt(plan.checkInDays, 10) || 0;
      const todayStr = getTodayDateStr();

      // 仅当用户刷的题库与当前激活规划题库一致时，累计今日进度并触发打卡判定
      if (planBankId && String(bankId) === planBankId) {
        if (plan.lastPracticeDate !== todayStr) {
          plan.todayCount = 0;
          plan.lastPracticeDate = todayStr;
        }

        plan.todayCount = (plan.todayCount || 0) + 1;

        // 判定是否达成今日目标且今天尚未打卡
        if (plan.todayCount >= dailyGoal && plan.lastCheckInDate !== todayStr) {
          plan.lastCheckInDate = todayStr;
          checkInDays = checkInDays + 1;
          plan.checkInDays = checkInDays;
          justCheckedIn = true;
        }

        wx.setStorageSync(PLAN_KEY, plan);
      }
    }
  } catch (e) {
    console.warn('[studyStats] 更新本地规划打卡异常:', e);
  }

  return {
    justCheckedIn,
    checkInDays,
    dailyGoal,
    stats: {
      ...stats,
      checkInDays
    }
  };
}

/**
 * 当学习规划题库变更时，清空累计打卡天数，重新从0累计
 * @param {string|number} newBankId - 新题库ID
 * @param {string} newBankTitle - 新题库名称
 */
function resetCheckInForNewBank(newBankId, newBankTitle) {
  try {
    const plan = wx.getStorageSync(PLAN_KEY) || {};
    plan.bankId = String(newBankId);
    if (newBankTitle) {
      plan.bankTitle = newBankTitle;
    }
    // 核心规则：清空累计打卡天数，重新从 0 累计
    plan.checkInDays = 0;
    plan.todayCount = 0;
    plan.lastCheckInDate = '';
    plan.lastPracticeDate = getTodayDateStr();

    wx.setStorageSync(PLAN_KEY, plan);
  } catch (e) {
    console.warn('[studyStats] 重置累计打卡异常:', e);
  }
}

/**
 * 联网模式下拉取服务端最新学习统计数据并同步本地
 */
function fetchStatsFromServer() {
  if (CONFIG.USE_MOCK) {
    return Promise.resolve(getStudyStats());
  }

  return request({ url: '/api/v1/practice/stats', needAuth: true })
    .then((res) => {
      if (res) {
        const stats = {
          totalQuestions: res.total_questions !== undefined ? res.total_questions : 0,
          correctQuestions: res.correct_questions !== undefined ? res.correct_questions : 0,
          accuracyRate: res.accuracy_rate !== undefined ? res.accuracy_rate : 100
        };
        wx.setStorageSync(STATS_KEY, stats);

        if (res.check_in_days !== undefined) {
          const plan = wx.getStorageSync(PLAN_KEY) || {};
          plan.checkInDays = res.check_in_days;
          wx.setStorageSync(PLAN_KEY, plan);
        }
      }
      return getStudyStats();
    })
    .catch((err) => {
      console.warn('[studyStats] 从服务端拉取 stats 失败，降级本地数据:', err);
      return getStudyStats();
    });
}

module.exports = {
  getStudyStats,
  recordAnswer,
  resetCheckInForNewBank,
  fetchStatsFromServer,
  getTodayDateStr
};
