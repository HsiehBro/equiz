const NOTES_STORAGE_KEY = 'user_study_notes_list';
const CANCELLED_STORAGE_KEY = 'messages_cancelled_reply_ids';
const READ_STORAGE_KEY = 'messages_read_reply_ids';

// 默认用户学习笔记（若本地无任何笔记缓存时用于初始化数据源）
const DEFAULT_NOTES = [
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

// 已知预置公开笔记的互动回复
const KNOWN_REPLIES_MAP = {
  'note_1': {
    userName: 'Alex Chen',
    avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuDH_v2QU2wrTDmoEkfGnCW5INSBf3OpFz-iikHSduzhexfi8KXHoZkanaCyTcSXbd9nbZjkqm6kYEiw2hml5IpcUrhcwVnX06NryXNjJ50X8balOGeOSU7rv4t95D0mfa19nq0BOdRl50RRe34N9O7GOGWSXDD5Osnzk6SmpaL8oqi2whezjsGgNneYXj27iaeoJAvruCQFZEG32Oi2kAgWjSo8T7if_W7cC9PAfMl-i1nhLprUfRt3',
    avatarLetter: '',
    time: '10分钟前',
    content: '感谢分享！关于“认知负荷理论”的这部分笔记总结得非常到位，对我帮助很大。'
  },
  'note_2': {
    userName: 'Sarah Wang',
    avatarUrl: 'https://lh3.googleusercontent.com/aida-public/AB6AXuDlV-JhzqlzJxQyj1RAOLodkjPQ5ifaQhYttwZpHsyoI1K-hfqWvL1C6ObBfjwKOPaVc4xhRcuvrYByEvzseyhnjmg43KsNyVAQ-ci__pZluIq4ldEqjuzCXLZfpEXxlgRcpttCXELTReK7zzxr0XJtrhT0VkRHC73kgISIDi8i-HPIH-kfTLN_dZMQdFiHB__7jhYH4C6i7eP7ZasZasypN_PbDBb8m1Jsbz79BLJqkBr4sziY2kmZ',
    avatarLetter: '',
    time: '2小时前',
    content: '第三个知识点的公式推导好像有点跳跃，能详细解释一下第二步是怎么来的吗？'
  },
  'note_3': {
    userName: 'Liu Ming',
    avatarUrl: '',
    avatarLetter: 'L',
    time: '昨天 14:30',
    content: '马克了，考前必看系列。'
  },
  'cmt_1': {
    userName: '高项通关学者',
    avatarUrl: '',
    avatarLetter: '高',
    time: '昨天 10:20',
    content: '关键路径总浮动时间为 0 这个特征总结得很精炼！上周做真题就考了这个知识点，差点掉坑里。'
  },
  'cmt_2': {
    userName: '敏捷项目教练',
    avatarUrl: '',
    avatarLetter: 'M',
    time: '前天 16:45',
    content: '快速跟进和赶工的对比很清晰，一个增加风险，一个增加成本，经典常考考点，赞！'
  }
};

const DYNAMIC_REPLIES_POOL = [
  {
    userName: '考研必过君',
    avatarUrl: '',
    avatarLetter: 'K',
    time: '刚刚',
    content: '这个解题思路总结太赞了！刚好在复习这部分，重点很突出，已经收藏了！'
  },
  {
    userName: '学霸小周',
    avatarUrl: '',
    avatarLetter: '周',
    time: '15分钟前',
    content: '楼主分析得很透彻，比辅导书上的官方解析通俗易懂多了，感谢分享！'
  },
  {
    userName: '备考先锋',
    avatarUrl: '',
    avatarLetter: '锋',
    time: '40分钟前',
    content: '马克备查，期待楼主多更新一些相关高频考点的解题心得！'
  }
];

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    hasMore: true,
    activityMessage: {
      title: '活动消息',
      desc: '邀请好友注册 EduPrep，即可赢取高达 30 天的高级会员时长！快来参与吧。',
      time: '刚刚',
      isPinned: true
    },
    replies: [],
    hasCancelledMessages: false,
    activeSwipeId: null
  },

  touchStartX: 0,
  touchStartY: 0,
  startOffset: 0,
  isSwipingHorizontal: false,

  onLoad() {
    this.initNavBar();
    this.loadReplies();
  },

  onShow() {
    this.resetSwipeOffsets();
    // 每次显示时重新拉取用户真实公开笔记（若在笔记页切换了公开/私密状态或删减了笔记，即时同步）
    this.loadReplies();
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

  /**
   * 动态加载真实公开笔记的回复消息
   * 核心准则：仅展示处于“公开状态” (visibility === 'public') 的学习笔记/评论的互动消息！
   */
  loadReplies() {
    try {
      // 1. 读取真实用户学习笔记列表（优先读取本地存储，未初始化时注入预置数据）
      let allNotes = wx.getStorageSync(NOTES_STORAGE_KEY);
      if (!allNotes || !Array.isArray(allNotes) || allNotes.length === 0) {
        allNotes = DEFAULT_NOTES;
        try {
          wx.setStorageSync(NOTES_STORAGE_KEY, DEFAULT_NOTES);
        } catch (e) {}
      }

      // 2. 严格筛选：只有公开状态 (visibility === 'public') 的笔记才会有公开回复消息！
      // 若用户在笔记页面将某篇笔记设为 private 仅自己可见，则此处严格过滤排除！
      const publicNotes = allNotes.filter(item => item && item.visibility === 'public');

      // 3. 读取已取消展示的消息 ID
      const cancelledIds = wx.getStorageSync(CANCELLED_STORAGE_KEY) || [];
      const cancelledSet = new Set(cancelledIds);

      // 4. 读取已读状态集合（默认第三条 Liu Ming 为已读，与原型保持一致）
      const readIds = wx.getStorageSync(READ_STORAGE_KEY) || ['msg_note_3'];
      const readSet = new Set(readIds);

      // 5. 动态为每个真实的公开笔记映射最新用户回复
      const replies = [];
      publicNotes.forEach((note, idx) => {
        const replyId = `msg_${note.id}`;
        if (cancelledSet.has(replyId)) {
          return; // 已被用户左滑取消展示
        }

        const noteKey = String(note.id);
        const replyTpl = KNOWN_REPLIES_MAP[noteKey] || DYNAMIC_REPLIES_POOL[idx % DYNAMIC_REPLIES_POOL.length];

        replies.push({
          id: replyId,
          userName: replyTpl.userName,
          avatarUrl: replyTpl.avatarUrl || '',
          avatarLetter: replyTpl.avatarLetter || '',
          time: replyTpl.time || '刚刚',
          content: replyTpl.content,
          isRead: readSet.has(replyId),
          swipeOffset: 0,
          isSwiping: false,
          isRemoving: false,
          note: {
            id: note.id,
            title: note.questionTitle || note.title || (note.bankTitle + '：学习笔记'),
            bankId: note.bankId || '1',
            bankTitle: note.bankTitle || '通用题库',
            questionId: note.questionId,
            questionIndex: note.questionIndex,
            content: note.content,
            visibility: note.visibility,
            label: '你的笔记'
          }
        });
      });

      this.setData({
        replies: replies,
        hasCancelledMessages: cancelledIds.length > 0,
        activeSwipeId: null
      });
    } catch (e) {
      console.error('加载真实公开笔记回复失败', e);
      this.setData({ replies: [], activeSwipeId: null });
    }
  },

  onNavBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack({ delta: 1 });
    } else {
      wx.reLaunch({ url: '/pages/index/index' });
    }
  },

  onTapActivity() {
    wx.showModal({
      title: '邀请礼遇',
      content: '邀请好友注册 EduPrep，即可赢取高达 30 天的高级会员时长！快去分享给身边的考友吧。',
      confirmText: '立即邀请',
      confirmColor: '#0058bc',
      success: (res) => {
        if (res.confirm) {
          wx.showToast({ title: '已生成邀请链接', icon: 'success' });
        }
      }
    });
  },

  // =========================================================================
  // 核心交互：左滑手势与取消展示
  // =========================================================================

  onTouchStart(e) {
    if (!e.touches || e.touches.length === 0) return;
    const touch = e.touches[0];
    const id = e.currentTarget.dataset.id;
    this.touchStartX = touch.clientX;
    this.touchStartY = touch.clientY;
    this.isSwipingHorizontal = false;

    const currentItem = this.data.replies.find(r => r.id === id);
    this.startOffset = (currentItem && currentItem.swipeOffset < -20) ? -88 : 0;

    // 若当前有其他已展开的项，立即自动收起
    if (this.data.activeSwipeId && this.data.activeSwipeId !== id) {
      this.resetSwipeOffsets();
    }
  },

  onTouchMove(e) {
    if (!e.touches || e.touches.length === 0) return;
    const touch = e.touches[0];
    const id = e.currentTarget.dataset.id;
    const deltaX = touch.clientX - this.touchStartX;
    const deltaY = touch.clientY - this.touchStartY;

    // 判断手势方向：垂直位移大则放行页面原生滚动，不拦截
    if (!this.isSwipingHorizontal) {
      if (Math.abs(deltaY) > Math.abs(deltaX) && Math.abs(deltaY) > 6) {
        return;
      }
      if (Math.abs(deltaX) > 6) {
        this.isSwipingHorizontal = true;
      } else {
        return;
      }
    }

    const currentItem = this.data.replies.find(r => r.id === id);
    if (!currentItem) return;

    let newOffset = (this.startOffset || 0) + deltaX;

    // 限制左滑阻尼（最多 -108px）与右滑阻尼（最多 0px，严禁向右露出空白）
    if (newOffset < -88) {
      newOffset = -88 + (newOffset + 88) * 0.25;
    } else if (newOffset > 0) {
      newOffset = 0;
    }

    const updated = this.data.replies.map(r => {
      if (r.id === id) {
        return { ...r, swipeOffset: newOffset, isSwiping: true };
      }
      return r;
    });

    this.setData({ replies: updated });
  },

  onTouchEnd(e) {
    const id = e.currentTarget.dataset.id;
    this.isSwipingHorizontal = false;
    const currentItem = this.data.replies.find(r => r.id === id);
    if (!currentItem) return;

    let targetOffset = 0;
    let newActiveId = null;

    // 左滑超过 36px 即吸附展开 88px 操作区
    if (currentItem.swipeOffset < -36) {
      targetOffset = -88;
      newActiveId = id;
    } else {
      targetOffset = 0;
      newActiveId = null;
    }

    const updated = this.data.replies.map(r => {
      if (r.id === id) {
        return { ...r, swipeOffset: targetOffset, isSwiping: false };
      }
      return { ...r, swipeOffset: 0, isSwiping: false };
    });

    this.setData({
      replies: updated,
      activeSwipeId: newActiveId
    });
  },

  /**
   * 重置全部滑动项为初始闭合状态
   */
  resetSwipeOffsets() {
    const updated = this.data.replies.map(r => ({
      ...r,
      swipeOffset: 0,
      isSwiping: false
    }));
    this.setData({
      replies: updated,
      activeSwipeId: null
    });
  },

  /**
   * 从消息中心取消展示该条消息
   */
  onCancelDisplay(e) {
    const id = e.currentTarget.dataset.id;
    const item = this.data.replies.find(r => r.id === id);
    if (!item) return;

    // 触发高度折叠渐隐动画
    const updatedWithRemoving = this.data.replies.map(r => {
      if (r.id === id) {
        return { ...r, isRemoving: true };
      }
      return r;
    });
    this.setData({ replies: updatedWithRemoving });

    // 动画结束后从列表中剔除并存入本地缓存
    setTimeout(() => {
      const remaining = this.data.replies.filter(r => r.id !== id);
      this.setData({
        replies: remaining,
        activeSwipeId: null
      });

      try {
        const cancelledIds = wx.getStorageSync(CANCELLED_STORAGE_KEY) || [];
        if (!cancelledIds.includes(id)) {
          cancelledIds.push(id);
          wx.setStorageSync(CANCELLED_STORAGE_KEY, cancelledIds);
        }
      } catch (err) {
        console.error('保存取消展示状态异常', err);
      }

      wx.showToast({
        title: '已取消展示',
        icon: 'success',
        duration: 1800
      });
    }, 220);
  },

  /**
   * 恢复所有已取消展示的消息
   */
  onRestoreReplies() {
    try {
      wx.removeStorageSync(CANCELLED_STORAGE_KEY);
    } catch (e) {}

    this.loadReplies();

    wx.showToast({
      title: '已恢复所有消息展示',
      icon: 'success'
    });
  },

  /**
   * 跳转前往我的学习笔记页面
   */
  onGoToNotes() {
    wx.navigateTo({
      url: '/pages/notes/notes'
    });
  },

  // =========================================================================
  // 核心交互：点击消息链接跳转至具体评论/学习笔记
  // =========================================================================

  onTapNoteLink(e) {
    const note = e.currentTarget.dataset.note;
    if (!note) return;

    // 若有卡片处于滑动展开态，先收起
    if (this.data.activeSwipeId) {
      this.resetSwipeOffsets();
    }

    const noteId = note.id || '';
    const title = note.title || '';
    const bankId = note.bankId || '';

    // 跳转到学习笔记与题目评论页面，并精准定位高亮该条笔记
    wx.navigateTo({
      url: `/pages/notes/notes?noteId=${encodeURIComponent(noteId)}&title=${encodeURIComponent(title)}&bankId=${encodeURIComponent(bankId)}`
    });
  },

  /**
   * 点击消息卡片非链接区域
   */
  onTapReplyCard(e) {
    // 如果处于滑动展开状态，点击卡片优先收起
    if (this.data.activeSwipeId) {
      this.resetSwipeOffsets();
      return;
    }

    const item = e.currentTarget.dataset.item;
    if (!item) return;

    wx.showActionSheet({
      itemList: [`回复 @${item.userName}`, '查看对应的公开笔记', item.isRead ? '标记为未读' : '标记为已读'],
      success: (res) => {
        if (res.tapIndex === 0) {
          wx.showToast({ title: `回复 @${item.userName}`, icon: 'none' });
        } else if (res.tapIndex === 1) {
          if (item.note) {
            wx.navigateTo({
              url: `/pages/notes/notes?noteId=${encodeURIComponent(item.note.id || '')}&title=${encodeURIComponent(item.note.title || '')}&bankId=${encodeURIComponent(item.note.bankId || '')}`
            });
          }
        } else if (res.tapIndex === 2) {
          const newIsRead = !item.isRead;
          const updated = this.data.replies.map(r => r.id === item.id ? { ...r, isRead: newIsRead } : r);
          this.setData({ replies: updated });

          try {
            const readIds = wx.getStorageSync(READ_STORAGE_KEY) || [];
            const readSet = new Set(readIds);
            if (newIsRead) {
              readSet.add(item.id);
            } else {
              readSet.delete(item.id);
            }
            wx.setStorageSync(READ_STORAGE_KEY, Array.from(readSet));
          } catch (err) {}

          wx.showToast({ title: newIsRead ? '已设为已读' : '已设为未读', icon: 'success' });
        }
      }
    });
  },

  onLoadMore() {
    wx.showLoading({ title: '加载中...' });
    setTimeout(() => {
      wx.hideLoading();
      this.setData({ hasMore: false });
      wx.showToast({ title: '已加载全部消息', icon: 'none' });
    }, 500);
  }
});
