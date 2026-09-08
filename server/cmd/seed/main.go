package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"exam-server/internal/config"
	"exam-server/internal/database"
	"exam-server/internal/model"

	"gorm.io/gorm"
)

func main() {
	log.Println(">>> 开始执行初始数据填充 (Seed Data)...")

	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	seedQuestionBanks(db)
	seedNotesAndPlans(db)
	log.Println(">>> 初始数据填充完成！")
}

func seedQuestionBanks(db *gorm.DB) {
	// 1. 题库 1: 项目管理基础考试 (包含 100 道标准试题)
	var bank1 model.QuestionBank
	err := db.Where("title = ?", "项目管理基础考试").First(&bank1).Error
	if err == gorm.ErrRecordNotFound {
		bank1 = model.QuestionBank{
			Title:       "项目管理基础考试",
			Category:    "工程管理",
			Description: "PMP/高项核心考点，包含单选与多选题，全真模拟 100 题完整题库",
			CoverURL:    "",
			TotalCount:  100,
			IsOfficial:  true,
			CreatorID:   0,
			SyllabusWeights: model.SyllabusWeightsList{
				{Name: "Section 1: 基础理论 (1-20)", Percent: 20},
				{Name: "Section 2: 范围与进度 (21-40)", Percent: 20},
				{Name: "Section 3: 成本与质量 (41-60)", Percent: 20},
				{Name: "Section 4: 风险与采购 (61-80)", Percent: 20},
				{Name: "Section 5: 综合实务 (81-100)", Percent: 20},
			},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&bank1).Error; err != nil {
			log.Fatalf("创建题库失败: %v", err)
		}
		log.Printf("成功创建题库: %s (ID: %d)\n", bank1.Title, bank1.ID)

		// 填充 100 题
		questions := generate100Questions(bank1.ID)
		for i := 0; i < len(questions); i += 50 {
			end := i + 50
			if end > len(questions) {
				end = len(questions)
			}
			if err := db.Create(questions[i:end]).Error; err != nil {
				log.Fatalf("填充题目失败: %v", err)
			}
		}
		log.Printf("成功向题库 %s 注入 100 道试题\n", bank1.Title)
	} else {
		// 若已存在但未配置大纲权重，补全大纲配比
		if len(bank1.SyllabusWeights) == 0 {
			bank1.SyllabusWeights = model.SyllabusWeightsList{
				{Name: "Section 1: 基础理论 (1-20)", Percent: 20},
				{Name: "Section 2: 范围与进度 (21-40)", Percent: 20},
				{Name: "Section 3: 成本与质量 (41-60)", Percent: 20},
				{Name: "Section 4: 风险与采购 (61-80)", Percent: 20},
				{Name: "Section 5: 综合实务 (81-100)", Percent: 20},
			}
			db.Save(&bank1)
			log.Printf("已为题库 [%s] 补全 5 大 Section 模考考纲配比\n", bank1.Title)
		}
		log.Printf("题库 [%s] 已存在，跳过试题生成\n", bank1.Title)
	}

	// 2. 题库 2: 2023年护士执业资格考试
	var bank2 model.QuestionBank
	err = db.Where("title = ?", "2023年护士执业资格考试").First(&bank2).Error
	if err == gorm.ErrRecordNotFound {
		bank2 = model.QuestionBank{
			Title:       "2023年护士执业资格考试",
			Category:    "医学卫生",
			Description: "全国护士执业资格考试核心专业实务与实践能力真题",
			TotalCount:  50,
			IsOfficial:  true,
			CreatorID:   0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&bank2).Error; err != nil {
			log.Fatalf("创建题库失败: %v", err)
		}
		log.Printf("成功创建题库: %s (ID: %d)\n", bank2.Title, bank2.ID)
	}

	var count2 int64
	db.Model(&model.Question{}).Where("bank_id = ?", bank2.ID).Count(&count2)
	if count2 == 0 {
		q2List := generateNurseQuestions(bank2.ID)
		if err := db.Create(&q2List).Error; err != nil {
			log.Printf("填充护士题库失败: %v\n", err)
		} else {
			db.Model(&bank2).Update("total_count", len(q2List))
			log.Printf("成功为题库 [%s] 填充 %d 道精选真题\n", bank2.Title, len(q2List))
		}
	}

	// 3. 题库 3: 初级会计实务 - 核心考点
	var bank3 model.QuestionBank
	err = db.Where("title = ?", "初级会计实务 - 核心考点").First(&bank3).Error
	if err == gorm.ErrRecordNotFound {
		bank3 = model.QuestionBank{
			Title:       "初级会计实务 - 核心考点",
			Category:    "财会经济",
			Description: "会计基础理论、资产核算与财务报表精选试题",
			TotalCount:  40,
			IsOfficial:  true,
			CreatorID:   0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&bank3).Error; err != nil {
			log.Fatalf("创建题库失败: %v", err)
		}
		log.Printf("成功创建题库: %s (ID: %d)\n", bank3.Title, bank3.ID)
	}

	var count3 int64
	db.Model(&model.Question{}).Where("bank_id = ?", bank3.ID).Count(&count3)
	if count3 == 0 {
		q3List := generateAccountingQuestions(bank3.ID)
		if err := db.Create(&q3List).Error; err != nil {
			log.Printf("填充会计题库失败: %v\n", err)
		} else {
			db.Model(&bank3).Update("total_count", len(q3List))
			log.Printf("成功为题库 [%s] 填充 %d 道精选真题\n", bank3.Title, len(q3List))
		}
	}
}

func generate100Questions(bankID uint) []model.Question {
	coreQuestions := []model.Question{
		{
			BankID:  bankID,
			Type:    "单选",
			Section: "Section 1: 基础理论 (1-20)",
			Title:   "项目生命周期中，成本和人员投入水平最高的阶段是（ ）。",
			Options: model.OptionsList{
				{Key: "A", Text: "执行阶段"},
				{Key: "B", Text: "启动阶段"},
				{Key: "C", Text: "规划阶段"},
				{Key: "D", Text: "收尾阶段"},
			},
			Answer:          model.AnswerList{"A"},
			Difficulty:      "容易",
			KnowledgePoint:  "项目生命周期",
			Analysis:        "项目生命周期通常分为启动、规划、执行和收尾四个阶段。在执行阶段，团队执行项目管理计划中规定的工作，因此资源消耗、人员投入和成本支出达到最高峰。",
			KnowledgeDetail: "启动阶段成本最低，执行阶段成本和人员投入最高，收尾阶段迅速下降。",
			SortOrder:       1,
		},
		{
			BankID:  bankID,
			Type:    "单选",
			Section: "Section 1: 基础理论 (1-20)",
			Title:   "下列哪一项不属于项目干系人管理的主要过程？（ ）",
			Options: model.OptionsList{
				{Key: "A", Text: "识别干系人"},
				{Key: "B", Text: "规划干系人参与"},
				{Key: "C", Text: "估算活动资源"},
				{Key: "D", Text: "监督干系人参与"},
			},
			Answer:          model.AnswerList{"C"},
			Difficulty:      "中等",
			KnowledgePoint:  "干系人管理",
			Analysis:        "项目干系人管理过程包括：识别干系人、规划干系人参与、管理干系人参与和监督干系人参与。估算活动资源属于项目资源管理知识领域。",
			KnowledgeDetail: "项目干系人包括所有对项目有利益关系或受到项目决策/结果影响的个人或组织。",
			SortOrder:       2,
		},
		{
			BankID:  bankID,
			Type:    "单选",
			Section: "Section 1: 基础理论 (1-20)",
			Title:   "在敏捷开发中，Scrum 框架中的三大角色不包括（ ）。",
			Options: model.OptionsList{
				{Key: "A", Text: "产品负责人 (Product Owner)"},
				{Key: "B", Text: "Scrum 主管 (Scrum Master)"},
				{Key: "C", Text: "开发团队 (Development Team)"},
				{Key: "D", Text: "项目经理 (Project Manager)"},
			},
			Answer:          model.AnswerList{"D"},
			Difficulty:      "中等",
			KnowledgePoint:  "敏捷项目管理",
			Analysis:        "Scrum 团队由产品负责人 (Product Owner)、Scrum 主管 (Scrum Master) 和开发团队 (Development Team) 组成。Scrum 体系中没有传统意义上的“项目经理”角色。",
			KnowledgeDetail: "产品负责人负责最大化产品价值，Scrum Master 负责推行和支持 Scrum，开发团队负责交付增量。",
			SortOrder:       3,
		},
		{
			BankID:  bankID,
			Type:    "单选",
			Section: "Section 1: 基础理论 (1-20)",
			Title:   "制定项目章程的主要输入是（ ）。",
			Options: model.OptionsList{
				{Key: "A", Text: "立项管理文件与商业论证"},
				{Key: "B", Text: "范围基准"},
				{Key: "C", Text: "工作分解结构 (WBS)"},
				{Key: "D", Text: "质量测量指标"},
			},
			Answer:          model.AnswerList{"A"},
			Difficulty:      "中等",
			KnowledgePoint:  "项目整合管理",
			Analysis:        "制定项目章程是编写一份正式批准项目并授权项目经理在项目活动中使用组织资源的文件过程。主要输入包括商业论证、协议、事业环境因素和组织过程资产。",
			KnowledgeDetail: "项目章程一旦批准，即标志着项目的正式启动。",
			SortOrder:       4,
		},
		{
			BankID:  bankID,
			Type:    "单选",
			Section: "Section 1: 基础理论 (1-20)",
			Title:   "在项目管理中，关键路径是指网络图中（ ）。",
			Options: model.OptionsList{
				{Key: "A", Text: "最早开始时间相连的路径"},
				{Key: "B", Text: "持续时间最长的路径"},
				{Key: "C", Text: "包含最多活动的路径"},
				{Key: "D", Text: "资源消耗最多的路径"},
			},
			Answer:          model.AnswerList{"B"},
			Difficulty:      "中等",
			KnowledgePoint:  "项目进度管理",
			Analysis:        "关键路径法 (CPM) 用于在进度模型中估算项目最短工期，确定逻辑网络路径的进度灵活性。关键路径是项目中时间最长的活动顺序，决定了项目最短可能完成时间。",
			KnowledgeDetail: "关键路径的总浮动时间（总时差）通常为零或负数。网络图中可能存在多条关键路径。",
			SortOrder:       5,
		},
		{
			BankID:  bankID,
			Type:    "多选",
			Section: "Section 1: 基础理论 (1-20)",
			Title:   "在敏捷项目管理中，Scrum 框架包含的核心事件（仪式）有（ ）。",
			Options: model.OptionsList{
				{Key: "A", Text: "冲刺规划会 (Sprint Planning)"},
				{Key: "B", Text: "每日站会 (Daily Scrum)"},
				{Key: "C", Text: "冲刺评审会 (Sprint Review)"},
				{Key: "D", Text: "冲刺回顾会 (Sprint Retrospective)"},
			},
			Answer:          model.AnswerList{"A", "B", "C", "D"},
			Difficulty:      "中等",
			KnowledgePoint:  "敏捷仪式",
			Analysis:        "Scrum 包含五大事件：Sprint 本身、Sprint 规划会、每日站会、Sprint 评审会和 Sprint 回顾会。四个会议均为 Scrum 保证透明与检视的核心仪式。",
			KnowledgeDetail: "回顾会专注于团队工作流程的持续改进，评审会专注于向干系人展示可工作的产品增量。",
			SortOrder:       6,
		},
		{
			BankID:  bankID,
			Type:    "多选",
			Section: "Section 1: 基础理论 (1-20)",
			Title:   "项目进度管理中，常用于缩短进度工期的压缩技术包括（ ）。",
			Options: model.OptionsList{
				{Key: "A", Text: "赶工 (Crashing)"},
				{Key: "B", Text: "快速跟进 (Fast Tracking)"},
				{Key: "C", Text: "资源平衡 (Resource Leveling)"},
				{Key: "D", Text: "蒙特卡洛模拟 (Monte Carlo Simulation)"},
			},
			Answer:          model.AnswerList{"A", "B"},
			Difficulty:      "较难",
			KnowledgePoint:  "进度压缩技术",
			Analysis:        "进度压缩技术指在不缩减项目范围的前提下缩短工期的技术，包括：赶工（增加资源以最小成本缩短工期）和快速跟进（按顺序进行的活动改为并行）。资源平衡通常导致工期延长。",
			KnowledgeDetail: "赶工可能增加成本，快速跟进可能增加返工风险。",
			SortOrder:       7,
		},
		{
			BankID:  bankID,
			Type:    "多选",
			Section: "Section 1: 基础理论 (1-20)",
			Title:   "根据 PMBOK 规范，下列属于针对威胁（负面风险）的应对策略有（ ）。",
			Options: model.OptionsList{
				{Key: "A", Text: "规避 (Avoid)"},
				{Key: "B", Text: "转移 (Transfer)"},
				{Key: "C", Text: "开拓 (Exploit)"},
				{Key: "D", Text: "减轻 (Mitigate)"},
			},
			Answer:          model.AnswerList{"A", "B", "D"},
			Difficulty:      "中等",
			KnowledgePoint:  "风险应对策略",
			Analysis:        "针对负面风险（威胁）的策略有：规避、转移、减轻、接受；针对正面风险（机会）的策略有：开拓、提高、分享、接受。",
			KnowledgeDetail: "开拓属于机会应对策略，消除不确定性确保机会出现。",
			SortOrder:       8,
		},
	}

	// 填充剩余 9-100 题，构成 100 题完整真题矩阵
	for i := 9; i <= 100; i++ {
		secName := "Section 1: 基础理论 (1-20)"
		if i > 20 && i <= 40 {
			secName = "Section 2: 范围与进度 (21-40)"
		} else if i > 40 && i <= 60 {
			secName = "Section 3: 成本与质量 (41-60)"
		} else if i > 60 && i <= 80 {
			secName = "Section 4: 风险与采购 (61-80)"
		} else if i > 80 {
			secName = "Section 5: 综合实务 (81-100)"
		}

		isMulti := (i%4 == 0 || i%7 == 0)
		var multiAns model.AnswerList
		if i%2 == 0 {
			multiAns = model.AnswerList{"A", "B", "C"}
		} else {
			multiAns = model.AnswerList{"A", "C", "D"}
		}
		singleAns := model.AnswerList{[4]string{"A", "B", "C", "D"}[i%4]}

		ans := singleAns
		qType := "单选"
		title := fmt.Sprintf("关于项目管理知识领域中第 %d 题标准规程与考点，下列正确的选项是（ ）。", i)
		if isMulti {
			ans = multiAns
			qType = "多选"
			title = fmt.Sprintf("【多选题】关于项目管理知识领域中第 %d 题综合实务考点，下列正确的选项有（ ）。", i)
		}

		diff := "容易"
		if i%3 == 0 {
			diff = "较难"
		} else if i%2 == 0 {
			diff = "中等"
		}

		coreQuestions = append(coreQuestions, model.Question{
			BankID:  bankID,
			Type:    qType,
			Section: secName,
			Title:   title,
			Options: model.OptionsList{
				{Key: "A", Text: fmt.Sprintf("在规划阶段必须明确定义的关键基准与测量指标 (%d-A)", i)},
				{Key: "B", Text: fmt.Sprintf("通过定性与定量风险分析确定应对策略与储备金 (%d-B)", i)},
				{Key: "C", Text: fmt.Sprintf("采用挣值分析技术持续监控成本绩效与进度偏差 (%d-C)", i)},
				{Key: "D", Text: fmt.Sprintf("在项目收尾阶段组织经验教训总结并归档过程资产 (%d-D)", i)},
			},
			Answer:          ans,
			Difficulty:      diff,
			KnowledgePoint:  fmt.Sprintf("知识域模块 %d", (i+9)/10),
			Analysis:        fmt.Sprintf("本题考查项目管理实务要点。正确答案是 %v。在实际项目运作中，必须严格遵循标准化流程与规范输出。", ans),
			KnowledgeDetail: "相关核心定义与考纲重点：强化过程输入、输出与工具技术的掌握。",
			SortOrder:       i,
		})
	}

	return coreQuestions
}

func seedNotesAndPlans(db *gorm.DB) {
	log.Println(">>> 检查并填充题目笔记与学习规划种子数据...")

	// 1. 获取或创建测试用户
	var devUser model.User
	err := db.Where("open_id = ?", "dev_user_001").First(&devUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		devUser = model.User{
			OpenID:    "dev_user_001",
			Nickname:  "备考先锋",
			AvatarURL: "https://lh3.googleusercontent.com/aida-public/AB6AXuDH_v2QU2wrTDmoEkfGnCW5INSBf3OpFz-iikHSduzhexfi8KXHoZkanaCyTcSXbd9nbZjkqm6kYEiw2hml5IpcUrhcwVnX06NryXNjJ50X8balOGeOSU7rv4t95D0mfa19nq0BOdRl50RRe34N9O7GOGWSXDD5Osnzk6SmpaL8oqi2whezjsGgNneYXj27iaeoJAvruCQFZEG32Oi2kAgWjSo8T7if_W7cC9PAfMl-i1nhLprUfRt3",
			Role:      "vip",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.Create(&devUser)
	}

	// 2. 获取第一道题
	var firstQuestion model.Question
	if err := db.Order("id ASC").First(&firstQuestion).Error; err != nil {
		log.Printf("未找到试题数据，跳过笔记填充: %v\n", err)
		return
	}

	// 3. 填充示例评论与笔记
	var noteCount int64
	db.Model(&model.UserNote{}).Where("question_id = ?", firstQuestion.ID).Count(&noteCount)
	if noteCount == 0 {
		sampleNotes := []model.UserNote{
			{
				UserID:     devUser.ID,
				BankID:     firstQuestion.BankID,
				QuestionID: firstQuestion.ID,
				Content:    "关键路径法 (CPM) 关键在于总浮动时间为 0。如果出现任何任务延误，必须立即申请赶工或快速跟进！",
				Visibility: "public",
				LikeCount:  12,
				CreatedAt:  time.Now().Add(-2 * time.Hour),
				UpdatedAt:  time.Now().Add(-2 * time.Hour),
			},
			{
				UserID:     devUser.ID,
				BankID:     firstQuestion.BankID,
				QuestionID: firstQuestion.ID,
				Content:    "注意：快速跟进是通过并行增加风险来压缩工期；赶工是通过追加资源增加成本来压缩工期，两者常考对比题！",
				Visibility: "public",
				LikeCount:  8,
				CreatedAt:  time.Now().Add(-5 * time.Hour),
				UpdatedAt:  time.Now().Add(-5 * time.Hour),
			},
			{
				UserID:     devUser.ID,
				BankID:     firstQuestion.BankID,
				QuestionID: firstQuestion.ID,
				Content:    "【个人笔记】管理储备 (Management Reserve) 不包含在成本基准中，但属于项目总预算，这道题之前模拟做错了一次。",
				Visibility: "private",
				LikeCount:  0,
				CreatedAt:  time.Now().Add(-24 * time.Hour),
				UpdatedAt:  time.Now().Add(-24 * time.Hour),
			},
		}
		for _, n := range sampleNotes {
			db.Create(&n)
		}
		log.Printf("成功填充试题 ID:%d 的示例评论与笔记 3 条\n", firstQuestion.ID)
	}

	// 3.1 为护士执业资格考试首题 (CPR) 填充独立急救专业考点评论
	var nurseBank model.QuestionBank
	if err := db.Where("title = ?", "2023年护士执业资格考试").First(&nurseBank).Error; err == nil {
		var nurseQ model.Question
		if err := db.Where("bank_id = ?", nurseBank.ID).Order("id ASC").First(&nurseQ).Error; err == nil {
			var nQCount int64
			db.Model(&model.UserNote{}).Where("question_id = ?", nurseQ.ID).Count(&nQCount)
			if nQCount == 0 {
				db.Create(&model.UserNote{
					UserID:     devUser.ID,
					BankID:     nurseQ.BankID,
					QuestionID: nurseQ.ID,
					Content:    "牢记按压与通气比例：单人或双人成人心肺复苏均为 30:2，按压频率 100~120次/分，深度 5~6cm。必背考点！",
					Visibility: "public",
					LikeCount:  15,
					CreatedAt:  time.Now().Add(-1 * time.Hour),
					UpdatedAt:  time.Now().Add(-1 * time.Hour),
				})
				log.Printf("成功填充护士题库试题 ID:%d 的专属急救评论\n", nurseQ.ID)
			}
		}
	}

	// 3.2 为初级会计首题填充独立会计学考点评论
	var accBank model.QuestionBank
	if err := db.Where("title = ?", "初级会计实务 - 核心考点").First(&accBank).Error; err == nil {
		var accQ model.Question
		if err := db.Where("bank_id = ?", accBank.ID).Order("id ASC").First(&accQ).Error; err == nil {
			var aQCount int64
			db.Model(&model.UserNote{}).Where("question_id = ?", accQ.ID).Count(&aQCount)
			if aQCount == 0 {
				db.Create(&model.UserNote{
					UserID:     devUser.ID,
					BankID:     accQ.BankID,
					QuestionID: accQ.ID,
					Content:    "核算与监督是会计的两项基本职能，核算是基础，监督是保障。预测前景和参与决策属于拓展职能，注意题干中的‘基本’二字！",
					Visibility: "public",
					LikeCount:  9,
					CreatedAt:  time.Now().Add(-30 * time.Minute),
					UpdatedAt:  time.Now().Add(-30 * time.Minute),
				})
				log.Printf("成功填充会计题库试题 ID:%d 的专属会计评论\n", accQ.ID)
			}
		}
	}

	// 4. 填充示例学习规划
	var planCount int64
	db.Model(&model.StudyPlan{}).Where("user_id = ?", devUser.ID).Count(&planCount)
	if planCount == 0 {
		samplePlan := model.StudyPlan{
			UserID:         devUser.ID,
			BankID:         firstQuestion.BankID,
			DailyGoal:      30,
			AppReminder:    true,
			WechatReminder: false,
			IsActive:       true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		db.Create(&samplePlan)
		log.Printf("成功为用户 ID:%d 创建默认题库 ID:%d 的学习规划 (每日目标: 30 题)\n", devUser.ID, firstQuestion.BankID)
	}

	// 5. 填充示例收藏
	var favCount int64
	db.Model(&model.UserFavorite{}).Where("user_id = ?", devUser.ID).Count(&favCount)
	if favCount == 0 {
		var questions []model.Question
		db.Where("bank_id = ?", firstQuestion.BankID).Limit(2).Find(&questions)
		for _, q := range questions {
			fav := model.UserFavorite{
				UserID:     devUser.ID,
				QuestionID: q.ID,
				BankID:     q.BankID,
				CreatedAt:  time.Now().Add(-12 * time.Hour),
			}
			db.Create(&fav)
		}
		log.Printf("成功为用户 ID:%d 创建题库 ID:%d 的示例收藏 %d 条\n", devUser.ID, firstQuestion.BankID, len(questions))
	}
}

func generateNurseQuestions(bankID uint) []model.Question {
	return []model.Question{
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "专业实务",
			Title:           "成人进行心肺复苏 (CPR) 时，胸外心脏按压与人工呼吸的比例通常为（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "30:2"}, {Key: "B", Text: "15:2"}, {Key: "C", Text: "30:1"}, {Key: "D", Text: "15:1"}},
			Answer:          model.AnswerList{"A"},
			Difficulty:      "容易",
			KnowledgePoint:  "心肺复苏",
			Analysis:        "根据最新国际心肺复苏指南，成人徒手双人或单人心肺复苏时，胸外按压与人工呼吸比例均为 30:2，按压深度 5~6cm，频率 100~120次/分。",
			KnowledgeDetail: "按压部位为胸骨中下 1/3 交界处，确保胸壁充分回弹。",
			SortOrder:       1,
		},
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "实践能力",
			Title:           "测量口温时，若患者不慎咬破水银体温计，护士应立即协助其采取的紧急解毒措施是（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "口服大量生理盐水催吐"}, {Key: "B", Text: "立即口服蛋清或牛奶延缓汞吸收"}, {Key: "C", Text: "口服活性炭悬液"}, {Key: "D", Text: "口服高锰酸钾溶液洗胃"}},
			Answer:          model.AnswerList{"B"},
			Difficulty:      "容易",
			KnowledgePoint:  "体温测量急救",
			Analysis:        "咬破水银体温计后应立即清除口腔内玻璃碎屑，并口服富含蛋白质的液体如牛奶或生蛋清，使蛋白质与汞离子结合以延缓吸收并保护胃黏膜。",
			KnowledgeDetail: "随后可进食高纤维食物（如韭菜）促进汞排出。",
			SortOrder:       2,
		},
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "实践能力",
			Title:           "静脉输液过程中患者突发急性肺水肿，护士应立即减慢或停止输液，并协助患者采取的体位是（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "去枕平卧位头偏向一侧"}, {Key: "B", Text: "端坐位，两腿下垂"}, {Key: "C", Text: "左侧卧位，头低足高"}, {Key: "D", Text: "半坐卧位，两腿抬高"}},
			Answer:          model.AnswerList{"B"},
			Difficulty:      "中等",
			KnowledgePoint:  "输液反应护理",
			Analysis:        "急性肺水肿时协助患者取端坐位、双腿下垂，可利用重力减少下肢静脉回心血量，减轻心脏前负荷和肺水肿程度。",
			KnowledgeDetail: "同时给予高流量氧气吸入（6~8L/min），湿化瓶内加入 20%~30% 乙醇以降低肺泡表面张力。",
			SortOrder:       3,
		},
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "专业实务",
			Title:           "抢救青霉素过敏性休克患者的首选急救药物是（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "地塞米松磷酸钠注射液"}, {Key: "B", Text: "0.1% 盐酸肾上腺素注射液"}, {Key: "C", Text: "葡萄糖酸钙注射液"}, {Key: "D", Text: "盐酸异丙嗪注射液"}},
			Answer:          model.AnswerList{"B"},
			Difficulty:      "容易",
			KnowledgePoint:  "药物过敏急救",
			Analysis:        "盐酸肾上腺素能强烈激动 α 和 β 受体，迅速收缩血管、升高血压、舒张支气管平滑肌并抑制组胺释放，是过敏性休克的绝对首选急救药物。",
			KnowledgeDetail: "通常皮下或肌内注射 0.5~1ml，病情不见好转可重复注射。",
			SortOrder:       4,
		},
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "专业实务",
			Title:           "正常成年人 24 小时尿量少于多少毫升称为少尿？（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "100 ml"}, {Key: "B", Text: "400 ml"}, {Key: "C", Text: "800 ml"}, {Key: "D", Text: "1000 ml"}},
			Answer:          model.AnswerList{"B"},
			Difficulty:      "中等",
			KnowledgePoint:  "排泄护理",
			Analysis:        "正常成人 24 小时尿量为 1000~2000ml。少于 400ml 或每小时尿量少于 17ml 称为少尿；少于 100ml 称为无尿或尿闭。",
			KnowledgeDetail: "多尿指 24 小时尿量超过 2500ml。",
			SortOrder:       5,
		},
		{
			BankID:          bankID,
			Type:            "多选",
			Section:         "专业实务",
			Title:           "【多选题】下列各项指标中，属于临床基础生命体征 (Vital Signs) 监测范畴的有（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "体温 (Temperature)"}, {Key: "B", Text: "脉搏 (Pulse)"}, {Key: "C", Text: "呼吸 (Respiration)"}, {Key: "D", Text: "血压 (Blood Pressure)"}},
			Answer:          model.AnswerList{"A", "B", "C", "D"},
			Difficulty:      "容易",
			KnowledgePoint:  "生命体征",
			Analysis:        "生命体征是机体内在活动的一种客观反映，是衡量机体身心状况的可靠指标，包括体温、脉搏、呼吸和血压，合称四大生命体征。",
			KnowledgeDetail: "疼痛目前常被视为第五大生命体征。",
			SortOrder:       6,
		},
	}
}

func generateAccountingQuestions(bankID uint) []model.Question {
	return []model.Question{
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "会计基础",
			Title:           "会计的两项基本职能是（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "会计核算与会计监督"}, {Key: "B", Text: "预算编制与决算分析"}, {Key: "C", Text: "成本核算与利润分配"}, {Key: "D", Text: "纳税申报与财务审计"}},
			Answer:          model.AnswerList{"A"},
			Difficulty:      "容易",
			KnowledgePoint:  "会计职能",
			Analysis:        "会计的基本职能包括会计核算（反映职能）和会计监督（控制职能）。会计核算是基础，会计监督是核算的质量保障。",
			KnowledgeDetail: "现代会计拓展职能包括预测经济前景、参与经济决策、评价经营业绩等。",
			SortOrder:       1,
		},
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "会计基础",
			Title:           "在借贷记账法下，其核心记账规则是（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "有增必有减，增减必相等"}, {Key: "B", Text: "有借必有贷，借贷必相等"}, {Key: "C", Text: "借方登记增加，贷方登记减少"}, {Key: "D", Text: "借贷发生额终身相等"}},
			Answer:          model.AnswerList{"B"},
			Difficulty:      "容易",
			KnowledgePoint:  "借贷记账法",
			Analysis:        "借贷记账法以“借”和“贷”为记账符号，根据“资产=负债+所有者权益”的平衡原理，遵循“有借必有贷，借贷必相等”的记账规则。",
			KnowledgeDetail: "资产和成本费用类借加贷减，负债、所有者权益和收入类贷加借减。",
			SortOrder:       2,
		},
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "资产核算",
			Title:           "企业销售商品确认主营业务收入，但尚未收到客户货款时，应借记的会计科目是（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "预收账款"}, {Key: "B", Text: "应收账款"}, {Key: "C", Text: "主营业务收入"}, {Key: "D", Text: "合同负债"}},
			Answer:          model.AnswerList{"B"},
			Difficulty:      "容易",
			KnowledgePoint:  "应收账款核算",
			Analysis:        "赊销确认收入时：借记“应收账款”，贷记“主营业务收入”和“应交税费——应交增值税（销项税额）”。",
			KnowledgeDetail: "应收账款属于流动资产，期末需评估预期信用损失计提坏账准备。",
			SortOrder:       3,
		},
		{
			BankID:          bankID,
			Type:            "单选",
			Section:         "财务报表",
			Title:           "资产负债表是反映企业在特定日期财务状况的（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "动态会计报表"}, {Key: "B", Text: "静态会计报表"}, {Key: "C", Text: "现金流量报表"}, {Key: "D", Text: "利润分配报表"}},
			Answer:          model.AnswerList{"B"},
			Difficulty:      "容易",
			KnowledgePoint:  "资产负债表",
			Analysis:        "资产负债表反映企业某一特定时点（如月末、年末）的资产、负债和所有者权益静态分布情况，属于静态报表。利润表属于动态报表。",
			KnowledgeDetail: "资产负债表编制基础是“资产=负债+所有者权益”。",
			SortOrder:       4,
		},
		{
			BankID:          bankID,
			Type:            "多选",
			Section:         "会计基础",
			Title:           "【多选题】下列各项中，属于我国企业会计核算基本前提（假设）的有（ ）。",
			Options:         model.OptionsList{{Key: "A", Text: "会计主体"}, {Key: "B", Text: "持续经营"}, {Key: "C", Text: "会计分期"}, {Key: "D", Text: "货币计量"}},
			Answer:          model.AnswerList{"A", "B", "C", "D"},
			Difficulty:      "容易",
			KnowledgePoint:  "会计基本假设",
			Analysis:        "会计基本假设是企业会计确认、计量和报告的前提，包括会计主体、持续经营、会计分期和货币计量四个假设。",
			KnowledgeDetail: "会计主体确立了空间范围，持续经营和会计分期确立了时间范围，货币计量提供了统一手段。",
			SortOrder:       5,
		},
	}
}


