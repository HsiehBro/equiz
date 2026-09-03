const { request, uploadFile, CONFIG } = require('../../utils/request.js');

Page({
  data: {
    statusBarHeight: 20,
    navBarHeight: 44,
    showTemplateModal: false,
    showPreviewModal: false,
    libraryNameInput: '2024年综合执业考试模拟题库',
    nameConflictError: '',
    libraryVisibility: 'private',
    previewToken: '',
    previewData: {
      fileName: 'question_bank_2024.xlsx',
      totalQuestions: 50,
      singleChoiceCount: 40,
      multipleChoiceCount: 10,
      hasAnalysis: true
    }
  },

  onLoad() {
    this.initNavBar();
    this.loadExistingBanks();
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
   * 加载系统已有题库名称列表（用于全局唯一性排重与强校验）
   */
  loadExistingBanks() {
    this.existingBankTitles = [
      '项目管理基础考试',
      '2023年护士执业资格考试',
      '初级会计实务 - 核心考点'
    ];

    if (!CONFIG.USE_MOCK) {
      request({ url: '/api/v1/banks' })
        .then((banks) => {
          const list = Array.isArray(banks) ? banks : (banks && banks.list ? banks.list : []);
          list.forEach(b => {
            if (b.title && !this.existingBankTitles.some(t => t.toLowerCase() === b.title.trim().toLowerCase())) {
              this.existingBankTitles.push(b.title.trim());
            }
          });
        })
        .catch(() => {});
    }

    try {
      const customLibs = wx.getStorageSync('custom_libraries') || [];
      customLibs.forEach(c => {
        if (c.title && !this.existingBankTitles.some(t => t.toLowerCase() === c.title.trim().toLowerCase())) {
          this.existingBankTitles.push(c.title.trim());
        }
      });
    } catch (e) {}
  },

  /**
   * 检查题库名称是否与已有题库冲突 (全局唯一校验)
   */
  checkTitleConflict(name) {
    if (!name || !name.trim()) return false;
    const clean = name.trim().toLowerCase();
    const titles = this.existingBankTitles || [];
    return titles.some(t => (t || '').trim().toLowerCase() === clean);
  },

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

  onNavMore() {
    wx.showActionSheet({
      itemList: ['查看题库导入规范详情', '下载标准CSV模板', '常见格式错误排查'],
      success: (res) => {
        if (res.tapIndex === 0) {
          this.setData({ showTemplateModal: true });
        } else if (res.tapIndex === 1) {
          this.downloadTemplateCsv();
        } else if (res.tapIndex === 2) {
          wx.showModal({
            title: '错误排查',
            content: '请检查是否包含空行、多余换行符、或答案列字母非ABCD。',
            showCancel: false
          });
        }
      }
    });
  },

  /**
   * 生成包含标准导入规范样例数据的 CSV 字符串
   * 包含 UTF-8 BOM 标识 (\uFEFF)，确保在 Excel / WPS 打开时中文字符正常解析不乱码
   */
  getStandardTemplateCsv() {
    const rows = [
      ['题目', '选项A', '选项B', '选项C', '选项D', '正确答案', '解析', '大纲考点', '难度'],
      [
        '敏捷开发中Scrum框架的核心事件不包括以下哪一项？',
        '冲刺规划会 (Sprint Planning)',
        '每日站会 (Daily Scrum)',
        '冲刺评审会 (Sprint Review)',
        '季度战略规划会 (Quarterly Strategic Planning)',
        'D',
        'Scrum框架规范了五个核心事件：冲刺、冲刺规划会、每日站会、冲刺评审会和冲刺回顾会。季度战略规划会属于企业高层治理，不属于Scrum规范事件。',
        '敏捷项目管理',
        '简单'
      ],
      [
        '敏捷项目团队中的三大核心角色包括哪些？',
        '产品负责人 (PO)',
        'Scrum主管 (Scrum Master)',
        '跨职能开发团队 (Developers)',
        '项目管理办公室主管 (PMO Director)',
        'ABC',
        'Scrum指南明确定义了三大角色：产品负责人(PO)、Scrum主管(SM)和开发团队。PMO不属于Scrum内部核心角色。',
        '敏捷项目管理',
        '中等'
      ],
      [
        '在关系型数据库中，关于主键约束（Primary Key）特性的描述错误的是？',
        '主键列的数据必须在整张表中保持唯一',
        '复合主键允许其中某一个字段存储NULL值',
        '一张数据表最多只能定义一个主键',
        '主键可以由单列或多个列组合而成',
        'B',
        '主键具有唯一性和非空性（NOT NULL）双重约束。无论是单列主键还是复合主键，其所有构成列均绝对不允许包含NULL值。',
        '数据库技术',
        '中等'
      ],
      [
        '计算机网络中，HTTP协议默认采用的传输层协议以及标准端口号是？',
        'UDP 协议 80 端口',
        'TCP 协议 80 端口',
        'TCP 协议 443 端口',
        'UDP 协议 8080 端口',
        'B',
        'HTTP（超文本传输协议）默认使用面向连接且可靠的TCP协议传输，标准端口为80；HTTPS默认标准端口为443。',
        '计算机网络',
        '简单'
      ],
      [
        '软件工程测试方法中，下列属于常见黑盒测试用例设计技术的有？',
        '等价类划分法',
        '边界值分析法',
        '判定/条件覆盖法',
        '错误推测法',
        'ABD',
        '等价类划分、边界值分析、因果图和错误推测均属于黑盒测试技术；而判定/条件覆盖属于白盒测试（逻辑覆盖测试）方法。',
        '软件测试',
        '困难'
      ]
    ];

    const formatCell = (cell) => {
      const str = String(cell == null ? '' : cell);
      if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
        return `"${str.replace(/"/g, '""')}"`;
      }
      return `"${str}"`;
    };

    const csvBody = rows.map(row => row.map(formatCell).join(',')).join('\r\n');
    return '\uFEFF' + csvBody;
  },

  /**
   * 判断当前是否为 PC 环境 (Windows 微信、Mac 微信、微信开发者工具)
   */
  isPcEnvironment() {
    try {
      const info = wx.getSystemInfoSync ? wx.getSystemInfoSync() : {};
      const platform = (info.platform || '').toLowerCase();
      return platform === 'windows' || platform === 'mac' || platform === 'devtools';
    } catch (e) {
      return false;
    }
  },

  /**
   * 点击“下载标准模板”按钮：分 PC 端和手机端调用对应的另存为窗口
   */
  onDownloadTemplate() {
    this.downloadTemplateCsv();
  },

  downloadTemplateCsv() {
    const fileName = '标准题库导入模板.csv';
    const csvContent = this.getStandardTemplateCsv();

    wx.showLoading({
      title: '正在准备模板...',
      mask: true
    });

    try {
      const fs = wx.getFileSystemManager();
      const filePath = `${wx.env.USER_DATA_PATH}/${fileName}`;

      fs.writeFile({
        filePath: filePath,
        data: csvContent,
        encoding: 'utf8',
        success: () => {
          wx.hideLoading();

          if (this.isPcEnvironment()) {
            // 情况 1: PC 端打开小程序 -> 调用系统“另存为”窗口
            this.saveFileToPcDisk(filePath, csvContent);
          } else {
            // 情况 2: 手机端打开小程序 -> 调用手机文件系统另存为窗口
            this.saveFileToMobileFileSystem(filePath, fileName);
          }
        },
        fail: (err) => {
          wx.hideLoading();
          console.error('写入模板文件失败', err);
          wx.showToast({
            title: '生成模板文件失败',
            icon: 'none'
          });
        }
      });
    } catch (e) {
      wx.hideLoading();
      console.error('文件系统操作异常', e);
      wx.showToast({
        title: '下载异常',
        icon: 'none'
      });
    }
  },

  /**
   * PC 端：直接调用系统“另存为”窗口保存文件到用户磁盘
   */
  saveFileToPcDisk(filePath, csvContent) {
    if (typeof wx.saveFileToDisk === 'function') {
      wx.saveFileToDisk({
        filePath: filePath,
        success: () => {
          wx.showToast({
            title: '模板已保存到本地',
            icon: 'success'
          });
        },
        fail: (err) => {
          console.log('wx.saveFileToDisk 结果:', err);
          if (err && err.errMsg && err.errMsg.includes('cancel')) {
            // 用户主动点击系统另存为窗口的取消按钮，无需报错
            return;
          }
          if (err && err.errMsg && (err.errMsg.includes('开发者工具') || err.errMsg.includes('not support'))) {
            // 微信开发者工具未实现 saveFileToDisk 的底层 mock 桥接（仅在 PC 版微信客户端真机中生效）
            if (csvContent) {
              wx.setClipboardData({
                data: csvContent.replace(/^\uFEFF/, ''),
                success: () => {
                  wx.showModal({
                    title: '开发者工具调试提示',
                    content: '微信开发者工具模拟器尚未支持另存为 API 调试（请在真机或 PC 版微信客户端中验证原生另存为窗口）。\n\n已自动将标准 CSV 模板内容复制到剪贴板，可直接在电脑上粘贴保存。',
                    showCancel: false,
                    confirmText: '我知道了'
                  });
                }
              });
            } else {
              wx.showToast({
                title: '工具暂不支持另存为',
                icon: 'none'
              });
            }
            return;
          }
          wx.showToast({
            title: '另存为未完成',
            icon: 'none'
          });
        }
      });
    } else {
      wx.showToast({
        title: '当前微信版本过低，不支持另存为',
        icon: 'none'
      });
    }
  },

  /**
   * 手机端：调用文件系统另存为窗口 (打开文档预览并启用右上角菜单保存到手机)
   */
  saveFileToMobileFileSystem(filePath, fileName) {
    if (typeof wx.openDocument === 'function') {
      wx.openDocument({
        filePath: filePath,
        showMenu: true,
        success: () => {
          console.log('已调起手机端文件预览，用户可点击右上角菜单保存到手机');
        },
        fail: (openErr) => {
          console.warn('wx.openDocument 失败，尝试唤起转发保存', openErr);
          if (typeof wx.shareFileMessage === 'function') {
            wx.shareFileMessage({
              filePath: filePath,
              fileName: fileName,
              success: () => {
                wx.showToast({
                  title: '已转发保存',
                  icon: 'success'
                });
              },
              fail: (shareErr) => {
                console.error('wx.shareFileMessage fail:', shareErr);
              }
            });
          } else {
            wx.showToast({
              title: '无法调起文件系统窗口',
              icon: 'none'
            });
          }
        }
      });
    } else if (typeof wx.shareFileMessage === 'function') {
      wx.shareFileMessage({
        filePath: filePath,
        fileName: fileName,
        success: () => {
          wx.showToast({
            title: '已转发保存',
            icon: 'success'
          });
        },
        fail: (shareErr) => {
          console.error('wx.shareFileMessage fail:', shareErr);
        }
      });
    } else {
      wx.showToast({
        title: '当前微信版本不支持文件保存',
        icon: 'none'
      });
    }
  },

  onCloseTemplateModal() {
    this.setData({
      showTemplateModal: false
    });
  },

  onCopyTemplateLink() {
    const csvContent = this.getStandardTemplateCsv().replace(/^\uFEFF/, '');
    wx.setClipboardData({
      data: csvContent,
      success: () => {
        wx.showToast({
          title: '模板内容已复制',
          icon: 'success'
        });
        this.setData({
          showTemplateModal: false
        });
      }
    });
  },

  onUseSampleData() {
    const sampleTitle = '标准综合模拟试题(示例)';
    const isConflict = this.checkTitleConflict(sampleTitle);
    this.setData({
      showTemplateModal: false,
      libraryNameInput: sampleTitle,
      nameConflictError: isConflict ? `题库名称「${sampleTitle}」已存在，题库名称须全局唯一，请修改！` : '',
      previewData: {
        fileName: '标准题库导入模板.csv',
        totalQuestions: 5,
        singleChoiceCount: 3,
        multipleChoiceCount: 2,
        hasAnalysis: true
      },
      showPreviewModal: true
    });
  },

  onUploadFile() {
    // 优先调用微信聊天文件选择器
    if (wx.chooseMessageFile) {
      wx.chooseMessageFile({
        count: 1,
        type: 'file',
        extension: ['xlsx', 'csv', 'xls'],
        success: (res) => {
          if (res.tempFiles && res.tempFiles.length > 0) {
            const file = res.tempFiles[0];
            this.processFile(file.path, file.name || 'custom_question_bank.xlsx');
          }
        },
        fail: (err) => {
          if (err && err.errMsg && err.errMsg.indexOf('cancel') !== -1) {
            return;
          }
          this.processFileFallback('2024年度专业综合真题精选.xlsx');
        }
      });
    } else {
      this.processFileFallback('2024年度专业综合真题精选.xlsx');
    }
  },

  processFile(filePath, fileName) {
    if (!CONFIG.USE_MOCK && filePath) {
      wx.showLoading({ title: '服务端校验解析中...', mask: true });
      uploadFile({
        url: '/api/v1/import/preview',
        filePath: filePath,
        name: 'file'
      })
        .then((res) => {
          wx.hideLoading();
          const baseName = (fileName || res.file_name || '新导入题库').replace(/\.[^/.]+$/, "");
          const isConflict = this.checkTitleConflict(baseName);
          this.setData({
            previewToken: res.preview_token,
            libraryNameInput: baseName,
            nameConflictError: isConflict ? `题库名称「${baseName}」已存在，题库名称须全局唯一，请修改！` : '',
            previewData: {
              fileName: fileName || res.file_name,
              totalQuestions: res.total_questions,
              singleChoiceCount: res.single_choice_count,
              multipleChoiceCount: res.multiple_choice_count,
              hasAnalysis: res.has_analysis
            },
            showPreviewModal: true
          });
        })
        .catch((err) => {
          wx.hideLoading();
          wx.showModal({
            title: '解析失败',
            content: err.message || '文件格式不合规，请检查列名与数据',
            showCancel: false
          });
        });
    } else {
      this.processFileFallback(fileName || 'custom_bank.xlsx');
    }
  },

  processFileFallback(fileName) {
    wx.showLoading({
      title: '正在校验文件...',
      mask: true
    });

    setTimeout(() => {
      wx.hideLoading();
      const baseName = fileName.replace(/\.[^/.]+$/, "");
      const isConflict = this.checkTitleConflict(baseName);
      const isTemplate = fileName.includes('模板') || fileName.includes('template');
      this.setData({
        previewToken: '',
        libraryNameInput: baseName || '新导入题库',
        nameConflictError: isConflict ? `题库名称「${baseName}」已存在，题库名称须全局唯一，请修改！` : '',
        previewData: {
          fileName: fileName,
          totalQuestions: isTemplate ? 5 : 65,
          singleChoiceCount: isTemplate ? 3 : 50,
          multipleChoiceCount: isTemplate ? 2 : 15,
          hasAnalysis: true
        },
        showPreviewModal: true
      });
    }, 600);
  },

  onClosePreviewModal() {
    this.setData({
      showPreviewModal: false,
      nameConflictError: ''
    });
  },

  /**
   * 用户实时修改题库名称输入：动态检测冲突并给予提示
   */
  onLibraryNameInput(e) {
    const val = (e.detail.value || '').trim();
    const isConflict = this.checkTitleConflict(val);
    this.setData({
      libraryNameInput: e.detail.value,
      nameConflictError: isConflict ? `题库名称「${val}」已被占用，题库名称须全局唯一` : ''
    });
  },

  /**
   * 切换题库可见范围 (公开 / 私有)
   */
  onSelectVisibility(e) {
    const vis = e.currentTarget.dataset.visibility || 'private';
    this.setData({
      libraryVisibility: vis
    });
  },

  /**
   * 确认导入：强制要求题库名称全局唯一，并记录可见性
   */
  onConfirmImport() {
    const name = (this.data.libraryNameInput || '').trim();
    if (!name) {
      wx.showToast({ title: '请输入题库名称', icon: 'none' });
      return;
    }

    // 强校验拦截：若与已有题库名称冲突，强制要求修改，绝不允许重名导入
    if (this.checkTitleConflict(name)) {
      this.setData({
        nameConflictError: `题库名称「${name}」已存在，题库名称须全局唯一`
      });
      wx.showModal({
        title: '题库名称冲突',
        content: `系统中已存在题库「${name}」。\n题库名称须全局唯一，必须修改题库名称后方可导入！`,
        showCancel: false,
        confirmText: '去修改',
        confirmColor: '#ba1a1a'
      });
      return;
    }

    const total = this.data.previewData.totalQuestions || 50;
    const token = this.data.previewToken;
    const visibility = this.data.libraryVisibility || 'private';

    wx.showLoading({
      title: '正在导入入库...',
      mask: true
    });

    // 真实后端入库
    if (!CONFIG.USE_MOCK && token) {
      request({
        url: '/api/v1/import/confirm',
        method: 'POST',
        data: {
          preview_token: token,
          title: name,
          visibility: visibility
        }
      })
        .then((createdBank) => {
          wx.hideLoading();
          this.setData({ showPreviewModal: false, nameConflictError: '' });
          
          if (!this.existingBankTitles.includes(name)) {
            this.existingBankTitles.push(name);
          }

          wx.showModal({
            title: '导入成功',
            content: `题库「${name}」已成功导入，共 ${createdBank.total_count || total} 道题目。是否立即开始刷题？`,
            confirmText: '开始刷题',
            cancelText: '返回首页',
            confirmColor: '#0058bc',
            success: (res) => {
              if (res.confirm) {
                wx.navigateTo({
                  url: `/pages/quiz/quiz?bankId=${createdBank.id}&title=${encodeURIComponent(name)}`
                });
              } else {
                wx.navigateBack({
                  fail: () => {
                    wx.reLaunch({ url: '/pages/index/index' });
                  }
                });
              }
            }
          });
        })
        .catch((err) => {
          wx.hideLoading();
          const msg = err.message || '导入入库失败';
          if (msg.includes('已存在') || msg.includes('全局唯一') || msg.includes('冲突')) {
            this.setData({
              nameConflictError: msg
            });
            wx.showModal({
              title: '题库名称冲突',
              content: msg,
              showCancel: false,
              confirmText: '去修改',
              confirmColor: '#ba1a1a'
            });
          } else {
            wx.showToast({ title: msg, icon: 'none' });
          }
        });
      return;
    }

    // 离线/Mock 降级写入本地缓存
    setTimeout(() => {
      wx.hideLoading();
      this.setData({
        showPreviewModal: false,
        nameConflictError: ''
      });

      const userInfo = wx.getStorageSync('user_info') || {};
      const currentUserId = userInfo.id || userInfo.open_id || 'dev_user_001';

      const newLib = {
        id: 'lib_' + Date.now(),
        title: name,
        totalCount: total,
        progress: 0,
        lastPractice: '刚刚',
        visibility: visibility,
        creatorId: currentUserId,
        isPrimary: true
      };

      try {
        const existing = wx.getStorageSync('custom_libraries') || [];
        existing.unshift(newLib);
        wx.setStorageSync('custom_libraries', existing);
        if (!this.existingBankTitles.includes(name)) {
          this.existingBankTitles.push(name);
        }
      } catch (e) {
        console.log('保存题库缓存异常', e);
      }

      wx.showModal({
        title: '导入成功',
        content: `题库「${name}」已成功导入，共 ${total} 道题目。是否立即开始刷题？`,
        confirmText: '开始刷题',
        cancelText: '返回首页',
        confirmColor: '#0058bc',
        success: (res) => {
          if (res.confirm) {
            wx.navigateTo({
              url: '/pages/quiz/quiz'
            });
          } else {
            wx.navigateBack({
              fail: () => {
                wx.reLaunch({ url: '/pages/index/index' });
              }
            });
          }
        }
      });
    }, 600);
  },

  onCancel() {
    this.onNavBack();
  },

  preventDumbTap() {
    // 阻止弹窗点击冒泡
  }
});
