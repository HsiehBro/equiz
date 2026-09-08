const { request, CONFIG } = require('../../utils/request.js');
const NOTES_STORAGE_KEY = 'user_study_notes_list';
const CANCELLED_STORAGE_KEY = 'messages_cancelled_reply_ids';
const READ_STORAGE_KEY = 'messages_read_reply_ids';
const DELETED_NOTICES_STORAGE_KEY = 'messages_deleted_notice_ids';

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
    systemNotices: [],
    replies: [],
    hasCancelledMessages: false,
    activeSwipeId: null,
    activeSwipeType: null
  },

  touchStartX: 0,
  touchStartY: 0,
  startOffset: 0,
  isSwipingHorizontal: false,

  onLoad() {
    this.initNavBar();
    this.loadSystemNotices();
    this.loadReplies();
  },

  onShow() {
    this.resetSwipeOffsets();
    this.loadSystemNotices();
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
   * 加载系统与审核通知（排除公开评论回复通知，该类通知由回复消息列表专门呈现）
   */
  loadSystemNotices() {
    const userInfo = wx.getStorageSync('user_info') || {};
    const isAdmin = userInfo.role === 'admin';
    const currentUserId = userInfo.id || userInfo.open_id || userInfo.dev_user_id || (isAdmin ? 'dev_user_001' : 'dev_user_003');
    const deletedNoticeKey = `${DELETED_NOTICES_STORAGE_KEY}_${currentUserId}`;
    const deletedIds = new Set(wx.getStorageSync(deletedNoticeKey) || []);

    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/notifications', needAuth: true })
        .then((res) => {
          const list = Array.isArray(res) ? res : (res && res.list ? res.list : []);
          // 仅筛选系统与审核类型通知（排除 comment_reply，且过滤掉已删除通知）
          const systemOnly = list.filter(item => item.type !== 'comment_reply' && !deletedIds.has(String(item.id)));
          if (systemOnly.length > 0) {
            const formatted = systemOnly.map(item => ({
              id: item.id,
              type: item.type,
              title: item.title,
              content: item.content,
              time: item.created_at ? item.created_at.slice(0, 16).replace('T', ' ') : '刚刚'
            }));
            this.setData({ systemNotices: formatted });
            return;
          }
          this.loadLocalNotices(isAdmin, deletedIds);
        })
        .catch(() => {
          this.loadLocalNotices(isAdmin, deletedIds);
        });
      return;
    }

    this.loadLocalNotices(isAdmin, deletedIds);
  },

  loadLocalNotices(isAdmin, deletedIdsSet) {
    const userInfo = wx.getStorageSync('user_info') || {};
    const currentUserId = userInfo.id || userInfo.open_id || userInfo.dev_user_id || (isAdmin ? 'dev_user_001' : 'dev_user_003');
    const deletedIds = deletedIdsSet || new Set(wx.getStorageSync(`${DELETED_NOTICES_STORAGE_KEY}_${currentUserId}`) || []);

    let notices = [];
    if (isAdmin) {
      // 管理员：仅查看系统管理员专属的微信服务通知（有用户上传/申请公开题库提醒）
      const adminNotices = wx.getStorageSync('admin_system_notifications') || [];
      notices = [...adminNotices];
    } else {
      // 普通学员/VIP用户：谁申请那么通过/驳回审批的消息只出现在谁的消息中心！绝不出现在其他人的消息中心
      const userKey = `user_system_notifications_${currentUserId}`;
      let userNotices = wx.getStorageSync(userKey) || [];

      // 兼容历史老数据（若旧数据里带有对应当前用户ID）
      if (userNotices.length === 0) {
        const legacyNotices = wx.getStorageSync('user_system_notifications') || [];
        userNotices = legacyNotices.filter(n => !n.userId || String(n.userId) === String(currentUserId));
      }
      notices = [...userNotices];
    }

    // 严格过滤掉已删除的通知
    notices = notices.filter(n => !deletedIds.has(String(n.id)));

    if (notices.length === 0 && deletedIds.size === 0) {
      if (isAdmin) {
        notices = [
          {
            id: 'admin_demo_1',
            type: 'admin_pending',
            title: '【微信服务通知】有新的公开题库待审批',
            content: '系统管理员您好，用户上传或申请公开的题库已在个人中心手动审批栏就绪，请及时审核。',
            time: '刚刚'
          }
        ];
      } else {
        notices = [
          {
            id: 'user_demo_1',
            type: 'approval_pass',
            title: '【系统通知】用户权限与题库规则',
            content: '普通用户可拥有 2 个私有题库，公开题库需经管理员审核；VIP用户可拥有 20 个私有题库并畅刷所有 VIP 题库！',
            time: '刚刚'
          }
        ];
      }
    }

    this.setData({ systemNotices: notices });
  },

  /**
   * 动态加载真实的考友互动回复消息
   * 核心准则：仅展示其他用户实际针对当前用户公开评论发表的真实回复，严禁使用任何写死的假数据！
   */
  loadReplies() {
    try {
      const cancelledIds = wx.getStorageSync(CANCELLED_STORAGE_KEY) || [];
      const cancelledSet = new Set(cancelledIds.map(String));
      const readIds = wx.getStorageSync(READ_STORAGE_KEY) || [];
      const readSet = new Set(readIds.map(String));

      // 1. 真实后端模式：从服务端 /api/v1/notifications 获取当前用户收到的真实 comment_reply 通知
      if (!CONFIG.USE_MOCK) {
        request({ url: '/api/v1/notifications', needAuth: true })
          .then(res => {
            const list = Array.isArray(res) ? res : (res && res.list ? res.list : []);
            const replyNotifs = list.filter(n => n.type === 'comment_reply');
            
            const replies = replyNotifs
              .filter(bn => !cancelledSet.has(String(bn.id)))
              .map(bn => {
                const idStr = String(bn.id);
                let userName = bn.replier_name;
                if (!userName && bn.title) {
                  userName = bn.title.replace(' 回复了你的公开评论', '').trim();
                }
                if (!userName) userName = '考友';

                let timeStr = '刚刚';
                if (bn.created_at) {
                  try {
                    const d = new Date(bn.created_at);
                    const now = new Date();
                    const diffSec = Math.floor((now - d) / 1000);
                    if (diffSec < 60) timeStr = '刚刚';
                    else if (diffSec < 3600) timeStr = `${Math.floor(diffSec / 60)}分钟前`;
                    else if (diffSec < 86400) timeStr = `${Math.floor(diffSec / 3600)}小时前`;
                    else timeStr = `${d.getMonth() + 1}月${d.getDate()}日`;
                  } catch (e) {
                    timeStr = bn.created_at.slice(0, 16).replace('T', ' ');
                  }
                }

                return {
                  id: idStr,
                  rawId: bn.id,
                  userName: userName,
                  avatarUrl: bn.replier_avatar || '',
                  avatarLetter: userName ? userName[0] : '友',
                  time: timeStr,
                  content: bn.content,
                  isRead: Boolean(bn.is_read) || readSet.has(idStr),
                  swipeOffset: 0,
                  isSwiping: false,
                  isRemoving: false,
                  note: {
                    id: bn.related_id,
                    title: bn.question_title || '公开评论讨论',
                    bankId: bn.bank_id || 1,
                    bankTitle: bn.bank_title || (bn.bank_id == 2 ? '2023年护士执业资格考试' : '项目管理基础考试'),
                    questionId: bn.question_id,
                    content: bn.original_content || '',
                    label: '你的公开评论'
                  }
                };
              });

            this.setData({
              replies: replies,
              hasCancelledMessages: cancelledIds.length > 0,
              activeSwipeId: null
            });
          })
          .catch((err) => {
            console.error('拉取服务端回复通知失败', err);
            this.loadLocalReplies(cancelledSet, readSet, cancelledIds);
          });
        return;
      }

      // 2. 本地模拟模式：仅从本地真实产生的回复通知读取，不使用写死的模板假数据
      this.loadLocalReplies(cancelledSet, readSet, cancelledIds);
    } catch (e) {
      console.error('加载回复消息失败', e);
      this.setData({ replies: [], activeSwipeId: null });
    }
  },

  loadLocalReplies(cancelledSet, readSet, cancelledIds) {
    const dynamicNotifs = wx.getStorageSync('user_reply_notifications') || [];
    const replies = dynamicNotifs
      .filter(dn => dn && dn.id && !cancelledSet.has(String(dn.id)))
      .map(dn => {
        const idStr = String(dn.id);
        const userName = dn.userName || '考友';
        return {
          id: idStr,
          rawId: dn.id,
          userName: userName,
          avatarUrl: dn.avatarUrl || '',
          avatarLetter: dn.avatarLetter || (userName ? userName[0] : '友'),
          time: dn.time || '刚刚',
          content: dn.content || '',
          isRead: readSet.has(idStr) || Boolean(dn.isRead),
          swipeOffset: 0,
          isSwiping: false,
          isRemoving: false,
          note: dn.note || {
            id: 'cmt_default',
            title: '公开评论讨论',
            bankId: '1',
            questionId: 1,
            label: '你的公开评论'
          }
        };
      });

    this.setData({
      replies: replies,
      hasCancelledMessages: cancelledIds.length > 0,
      activeSwipeId: null
    });
  },

  onNavBack() {
    const pages = getCurrentPages();
    if (pages.length > 1) {
      wx.navigateBack({ delta: 1 });
    } else {
      wx.redirectTo({ url: '/pages/index/index' });
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
    const type = e.currentTarget.dataset.type || (this.data.systemNotices.some(n => String(n.id) === String(id)) ? 'notice' : 'reply');
    this.touchStartX = touch.clientX;
    this.touchStartY = touch.clientY;
    this.isSwipingHorizontal = false;
    this.activeSwipeType = type;

    const list = type === 'notice' ? this.data.systemNotices : this.data.replies;
    const currentItem = list.find(r => String(r.id) === String(id));
    this.startOffset = (currentItem && currentItem.swipeOffset < -20) ? -88 : 0;

    // 若当前有其他已展开的项，立即自动收起
    if (this.data.activeSwipeId && (String(this.data.activeSwipeId) !== String(id) || this.data.activeSwipeType !== type)) {
      this.resetSwipeOffsets();
    }
  },

  onTouchMove(e) {
    if (!e.touches || e.touches.length === 0) return;
    const touch = e.touches[0];
    const id = e.currentTarget.dataset.id;
    const type = e.currentTarget.dataset.type || this.activeSwipeType || 'reply';
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

    const listKey = type === 'notice' ? 'systemNotices' : 'replies';
    const list = this.data[listKey] || [];
    const currentItem = list.find(r => String(r.id) === String(id));
    if (!currentItem) return;

    let newOffset = (this.startOffset || 0) + deltaX;

    // 限制左滑阻尼（最多 -108px）与右滑阻尼（最多 0px，严禁向右露出空白）
    if (newOffset < -88) {
      newOffset = -88 + (newOffset + 88) * 0.25;
    } else if (newOffset > 0) {
      newOffset = 0;
    }

    const updated = list.map(r => {
      if (String(r.id) === String(id)) {
        return { ...r, swipeOffset: newOffset, isSwiping: true };
      }
      return r;
    });

    this.setData({ [listKey]: updated });
  },

  onTouchEnd(e) {
    const id = e.currentTarget.dataset.id;
    const type = e.currentTarget.dataset.type || this.activeSwipeType || 'reply';
    this.isSwipingHorizontal = false;
    const listKey = type === 'notice' ? 'systemNotices' : 'replies';
    const list = this.data[listKey] || [];
    const currentItem = list.find(r => String(r.id) === String(id));
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

    const updated = list.map(r => {
      if (String(r.id) === String(id)) {
        return { ...r, swipeOffset: targetOffset, isSwiping: false };
      }
      return { ...r, swipeOffset: 0, isSwiping: false };
    });

    const otherKey = type === 'notice' ? 'replies' : 'systemNotices';
    const otherList = (this.data[otherKey] || []).map(r => ({ ...r, swipeOffset: 0, isSwiping: false }));

    this.setData({
      [listKey]: updated,
      [otherKey]: otherList,
      activeSwipeId: newActiveId,
      activeSwipeType: newActiveId ? type : null
    });
  },

  /**
   * 重置全部滑动项为初始闭合状态
   */
  resetSwipeOffsets() {
    const updatedNotices = (this.data.systemNotices || []).map(r => ({
      ...r,
      swipeOffset: 0,
      isSwiping: false
    }));
    const updatedReplies = (this.data.replies || []).map(r => ({
      ...r,
      swipeOffset: 0,
      isSwiping: false
    }));
    this.setData({
      systemNotices: updatedNotices,
      replies: updatedReplies,
      activeSwipeId: null,
      activeSwipeType: null
    });
  },

  /**
   * 从消息中心删除该条系统与审核提醒
   */
  onDeleteSystemNotice(e) {
    const id = e.currentTarget.dataset.id;
    const item = (this.data.systemNotices || []).find(n => String(n.id) === String(id));
    if (!item) return;

    // 1. 触发高度折叠渐隐动画
    const updatedWithRemoving = this.data.systemNotices.map(n => {
      if (String(n.id) === String(id)) {
        return { ...n, isRemoving: true };
      }
      return n;
    });
    this.setData({ systemNotices: updatedWithRemoving });

    // 2. 动画结束后从列表中剔除并同步存储与服务端
    setTimeout(() => {
      const remaining = (this.data.systemNotices || []).filter(n => String(n.id) !== String(id));
      this.setData({
        systemNotices: remaining,
        activeSwipeId: null,
        activeSwipeType: null
      });

      // 持久化记录到已删除集合 (按用户隔离)
      const userInfo = wx.getStorageSync('user_info') || {};
      const isAdmin = userInfo.role === 'admin';
      const currentUserId = userInfo.id || userInfo.open_id || userInfo.dev_user_id || (isAdmin ? 'dev_user_001' : 'dev_user_003');

      try {
        const deletedNoticeKey = `${DELETED_NOTICES_STORAGE_KEY}_${currentUserId}`;
        const deletedIds = wx.getStorageSync(deletedNoticeKey) || [];
        if (!deletedIds.includes(String(id))) {
          deletedIds.push(String(id));
          wx.setStorageSync(deletedNoticeKey, deletedIds);
        }

        // 同步从本地存储的通知列表中清除
        if (isAdmin) {
          const adminNotices = wx.getStorageSync('admin_system_notifications') || [];
          wx.setStorageSync('admin_system_notifications', adminNotices.filter(n => String(n.id) !== String(id)));
        } else {
          const userKey = `user_system_notifications_${currentUserId}`;
          const userNotices = wx.getStorageSync(userKey) || [];
          wx.setStorageSync(userKey, userNotices.filter(n => String(n.id) !== String(id)));
        }
      } catch (err) {
        console.error('本地删除通知缓存异常', err);
      }

      // 真实后端模式：同步请求后端删除接口
      const numericId = Number(id);
      if (!CONFIG.USE_MOCK && !isNaN(numericId) && numericId > 0) {
        request({
          url: `/api/v1/notifications/${numericId}`,
          method: 'DELETE',
          needAuth: true
        }).catch(err => {
          console.log('[Messages] 后端删除通知异常:', err);
        });
      }

      wx.showToast({
        title: '已删除提醒',
        icon: 'success',
        duration: 1800
      });
    }, 220);
  },

  /**
   * 从消息中心取消展示该条消息
   */
  onCancelDisplay(e) {
    const id = e.currentTarget.dataset.id;
    const item = this.data.replies.find(r => String(r.id) === String(id));
    if (!item) return;

    // 触发高度折叠渐隐动画
    const updatedWithRemoving = this.data.replies.map(r => {
      if (String(r.id) === String(id)) {
        return { ...r, isRemoving: true };
      }
      return r;
    });
    this.setData({ replies: updatedWithRemoving });

    // 动画结束后从列表中剔除并存入本地缓存
    setTimeout(() => {
      const remaining = this.data.replies.filter(r => String(r.id) !== String(id));
      this.setData({
        replies: remaining,
        activeSwipeId: null,
        activeSwipeType: null
      });

      try {
        const cancelledIds = wx.getStorageSync(CANCELLED_STORAGE_KEY) || [];
        if (!cancelledIds.includes(String(id))) {
          cancelledIds.push(String(id));
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
    const bankTitle = note.bankTitle || (note.bankId == 2 ? '2023年护士执业资格考试' : '项目管理基础考试');
    const bankId = note.bankId || '1';
    const qId = note.questionId;
    const qIndex = typeof note.questionIndex === 'number' ? note.questionIndex : (qId > 0 ? qId - 1 : 0);

    if (qId) {
      // 优先直接跳转至对应题目的试题评论互动区
      wx.navigateTo({
        url: `/pages/quiz/quiz?bankId=${encodeURIComponent(bankId)}&questionId=${qId}&index=${qIndex}&title=${encodeURIComponent(bankTitle)}&openComments=true&highlightCommentId=${encodeURIComponent(noteId)}`
      });
    } else {
      // 跳转到学习笔记与题目评论页面，并精准定位高亮该条笔记
      wx.navigateTo({
        url: `/pages/notes/notes?noteId=${encodeURIComponent(noteId)}&title=${encodeURIComponent(bankTitle)}&bankId=${encodeURIComponent(bankId)}`
      });
    }
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
      itemList: [`回复 @${item.userName}`, '查看对应的试题与讨论', item.isRead ? '标记为未读' : '标记为已读'],
      success: (res) => {
        if (res.tapIndex === 0) {
          // 直接跳转前往做题页评论区，并激活对该用户的回复
          if (item.note) {
            const qId = item.note.questionId || 1;
            const qIndex = typeof item.note.questionIndex === 'number' ? item.note.questionIndex : (qId > 0 ? qId - 1 : 0);
            const bankTitle = item.note.bankTitle || (item.note.bankId == 2 ? '2023年护士执业资格考试' : '项目管理基础考试');
            wx.navigateTo({
              url: `/pages/quiz/quiz?bankId=${encodeURIComponent(item.note.bankId || '1')}&questionId=${qId}&index=${qIndex}&title=${encodeURIComponent(bankTitle)}&openComments=true&replyToCommentId=${encodeURIComponent(item.note.id || '')}&replyToAuthor=${encodeURIComponent(item.userName)}`
            });
          } else {
            wx.showToast({ title: `正在回复 @${item.userName}`, icon: 'none' });
          }
        } else if (res.tapIndex === 1) {
          if (item.note) {
            const qId = item.note.questionId || 1;
            const qIndex = typeof item.note.questionIndex === 'number' ? item.note.questionIndex : (qId > 0 ? qId - 1 : 0);
            const bankTitle = item.note.bankTitle || (item.note.bankId == 2 ? '2023年护士执业资格考试' : '项目管理基础考试');
            wx.navigateTo({
              url: `/pages/quiz/quiz?bankId=${encodeURIComponent(item.note.bankId || '1')}&questionId=${qId}&index=${qIndex}&title=${encodeURIComponent(bankTitle)}&openComments=true`
            });
          }
        } else if (res.tapIndex === 2) {
          const newIsRead = !item.isRead;
          const updated = this.data.replies.map(r => r.id === item.id ? { ...r, isRead: newIsRead } : r);
          this.setData({ replies: updated });

          if (!CONFIG.USE_MOCK && item.rawId && newIsRead) {
            request({
              url: `/api/v1/notifications/${item.rawId}/read`,
              method: 'POST'
            }).catch(() => {});
          }

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
