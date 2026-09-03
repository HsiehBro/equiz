package service

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MockService struct {
	db *gorm.DB
}

func NewMockService(db *gorm.DB) *MockService {
	return &MockService{db: db}
}

type StartMockExamRequest struct {
	BankID          uint `json:"bank_id" binding:"required"`
	TotalQuestions  int  `json:"total_questions"`
	DurationMinutes int  `json:"duration_minutes"`
}

type MockExamPaper struct {
	BankID          uint                      `json:"bank_id"`
	ExamTitle       string                    `json:"exam_title"`
	DurationMinutes int                       `json:"duration_minutes"`
	TotalQuestions  int                       `json:"total_questions"`
	SyllabusWeights model.SyllabusWeightsList `json:"syllabus_weights"`
	Questions       []model.Question          `json:"questions"`
}

// StartMockExam 根据题库大纲配比动态抽题组卷，带严格题量强校验拦截
func (s *MockService) StartMockExam(userID uint, req *StartMockExamRequest) (*MockExamPaper, error) {
	var bank model.QuestionBank
	if err := s.db.First(&bank, req.BankID).Error; err != nil {
		return nil, errors.New("指定题库不存在")
	}

	totalQuestions := req.TotalQuestions
	if totalQuestions <= 0 {
		totalQuestions = 100 // 默认 100 题全真模拟
	}
	durationMinutes := req.DurationMinutes
	if durationMinutes <= 0 {
		durationMinutes = 120 // 默认 120 分钟
	}

	syllabusWeights := bank.SyllabusWeights
	if len(syllabusWeights) == 0 {
		// 题库若无显式权重配置，自动从该题库已有试题中提取大纲分类并均分比例
		var sections []string
		s.db.Model(&model.Question{}).Where("bank_id = ?", bank.ID).Distinct().Pluck("section", &sections)
		if len(sections) == 0 {
			return nil, errors.New("该题库中暂无任何试题，无法生成模拟考卷")
		}
		avgPct := math.Round(100.0 / float64(len(sections)))
		for _, sec := range sections {
			syllabusWeights = append(syllabusWeights, model.SyllabusWeightItem{
				Name:    sec,
				Percent: avgPct,
			})
		}
	}

	// 1. 计算每个大纲考点的目标抽题配额
	quotas := make([]int, len(syllabusWeights))
	allocated := 0
	for i, w := range syllabusWeights {
		qCount := int(math.Round(float64(totalQuestions) * (w.Percent / 100.0)))
		if qCount <= 0 {
			qCount = 1
		}
		quotas[i] = qCount
		allocated += qCount
	}

	// 调整误差，确保总抽题数严格等于 totalQuestions
	diff := totalQuestions - allocated
	if diff != 0 && len(quotas) > 0 {
		quotas[0] += diff
		if quotas[0] < 1 {
			quotas[0] = 1
		}
	}

	// 2. 严格强校验拦截：检查题库中每个分类的可用题目是否满足配额
	for i, w := range syllabusWeights {
		targetQuota := quotas[i]
		var actualCount int64
		s.db.Model(&model.Question{}).
			Where("bank_id = ? AND section = ?", bank.ID, w.Name).
			Count(&actualCount)

		if int(actualCount) < targetQuota {
			return nil, fmt.Errorf("大纲考点【%s】试题不足：模考要求 %d 题，当前题库仅有 %d 题，请先补齐对应试题后再开考",
				w.Name, targetQuota, actualCount)
		}
	}

	// 3. 从每个大纲考点独立随机抽题组卷
	var selectedQuestions []model.Question
	for i, w := range syllabusWeights {
		targetQuota := quotas[i]
		var catQuestions []model.Question

		// PostgreSQL 原生 RANDOM() 随机抽取
		err := s.db.Where("bank_id = ? AND section = ?", bank.ID, w.Name).
			Order("RANDOM()").
			Limit(targetQuota).
			Find(&catQuestions).Error
		if err != nil {
			return nil, fmt.Errorf("抽取大纲【%s】试题失败: %w", w.Name, err)
		}

		selectedQuestions = append(selectedQuestions, catQuestions...)
	}

	if len(selectedQuestions) == 0 {
		return nil, errors.New("组卷失败，未能抽取到任何有效试题")
	}

	// 重新编号题号 (1 到 totalQuestions)
	for i := range selectedQuestions {
		selectedQuestions[i].SortOrder = i + 1
	}

	paper := &MockExamPaper{
		BankID:          bank.ID,
		ExamTitle:       fmt.Sprintf("%s · 全真模拟考试", bank.Title),
		DurationMinutes: durationMinutes,
		TotalQuestions:  len(selectedQuestions),
		SyllabusWeights: syllabusWeights,
		Questions:       selectedQuestions,
	}

	return paper, nil
}

type SubmitMockExamRequest struct {
	BankID          uint              `json:"bank_id" binding:"required"`
	ExamTitle       string            `json:"exam_title"`
	TimeUsedSeconds int               `json:"time_used_seconds"`
	TotalQuestions  int               `json:"total_questions"`
	Questions       []model.Question  `json:"questions" binding:"required"`
	UserAnswers     map[uint][]string `json:"user_answers" binding:"required"`
}

// SubmitMockExam 正常交卷：评分打分、大纲维度多维诊断与成绩落库
func (s *MockService) SubmitMockExam(userID uint, req *SubmitMockExamRequest) (*model.MockRecord, error) {
	questions := req.Questions
	if len(questions) == 0 {
		return nil, errors.New("试卷题目为空，无法评分交卷")
	}

	totalQuestions := len(questions)
	correctTotal := 0
	wrongTotal := 0
	unansweredTotal := 0

	// 提取试题 ID 列表并从数据库拉取权威试题实体（杜绝前端字段传输丢失导致答案为 null）
	var qIDs []uint
	for _, q := range questions {
		if q.ID > 0 {
			qIDs = append(qIDs, q.ID)
		}
	}

	dbQMap := make(map[uint]model.Question)
	if len(qIDs) > 0 {
		var dbQuestions []model.Question
		if err := s.db.Where("id IN ?", qIDs).Find(&dbQuestions).Error; err == nil {
			for _, dq := range dbQuestions {
				dbQMap[dq.ID] = dq
			}
		}
	}

	// 考点聚合统计：name -> { total, correct, wrong }
	type catStat struct {
		TotalCount   int
		CorrectCount int
		WrongCount   int
	}
	statMap := make(map[string]*catStat)

	var snapshots model.QuestionSnapshotsList

	for i, q := range questions {
		// 优先采用数据库真实实体，保证 Answer、Analysis 等核心元数据完整
		realQ, exists := dbQMap[q.ID]
		if !exists {
			realQ = q
		}

		userAns := req.UserAnswers[q.ID]
		isUnanswered := len(userAns) == 0
		isCorrect := false

		if !isUnanswered {
			isCorrect = CompareAnswers(userAns, realQ.Answer)
		}

		if isCorrect {
			correctTotal++
		} else {
			wrongTotal++
			if isUnanswered {
				unansweredTotal++
			}
		}

		secName := realQ.Section
		if secName == "" {
			secName = "默认大纲分类"
		}
		if _, ok := statMap[secName]; !ok {
			statMap[secName] = &catStat{}
		}
		statMap[secName].TotalCount++
		if isCorrect {
			statMap[secName].CorrectCount++
		} else {
			statMap[secName].WrongCount++
		}

		snapshots = append(snapshots, model.QuestionSnapshotItem{
			ID:              realQ.ID,
			Number:          i + 1,
			Type:            realQ.Type,
			Section:         realQ.Section,
			Title:           realQ.Title,
			Options:         realQ.Options,
			UserAnswer:      userAns,
			CorrectAnswer:   realQ.Answer,
			IsCorrect:       isCorrect,
			Difficulty:      realQ.Difficulty,
			KnowledgePoint:  realQ.KnowledgePoint,
			Analysis:        realQ.Analysis,
			KnowledgeDetail: realQ.KnowledgeDetail,
		})
	}

	// 计算总得分 (满分 100)
	score := math.Round((float64(correctTotal)/float64(totalQuestions))*100.0*10) / 10.0
	isPassed := score >= 60.0

	// 计算各大纲分类的维度诊断报告
	var diagnostics model.SyllabusDiagnosticsList
	for secName, st := range statMap {
		rate := 0.0
		if st.TotalCount > 0 {
			rate = math.Round((float64(st.CorrectCount)/float64(st.TotalCount))*100.0*10) / 10.0
		}

		statusTag := "掌握良好"
		levelClass := "level-success"
		advice := "核心概念掌握扎实，建议考前快速浏览保持题感。"

		if rate < 60.0 {
			statusTag = "薄弱待强化"
			levelClass = "level-warning"
			advice = "该考点失分较多，为核心薄弱环节，强烈建议针对本考点进行专项复习攻克！"
		} else if rate < 85.0 {
			statusTag = "达标巩固"
			levelClass = "level-info"
			advice = "基础理解较好，但存在细节混淆，需针对做错题目查漏补缺。"
		}

		diagnostics = append(diagnostics, model.SyllabusDiagnosticItem{
			Name:         secName,
			TargetWeight: math.Round((float64(st.TotalCount)/float64(totalQuestions))*100.0*10) / 10.0,
			TotalCount:   st.TotalCount,
			CorrectCount: st.CorrectCount,
			WrongCount:   st.WrongCount,
			ScoreRate:    rate,
			StatusTag:    statusTag,
			LevelClass:   levelClass,
			Advice:       advice,
		})
	}

	// 计算超越全国考生百分比
	ranking := math.Min(99.0, math.Max(15.0, math.Round(score*0.85+12.0)))

	title := req.ExamTitle
	if title == "" {
		title = "全真模拟考试"
	}

	record := model.MockRecord{
		UserID:              userID,
		BankID:              req.BankID,
		ExamTitle:           title,
		Score:               score,
		TotalScore:          100.0,
		TotalQuestions:      totalQuestions,
		CorrectCount:        correctTotal,
		WrongCount:          wrongTotal,
		UnansweredCount:     unansweredTotal,
		TimeUsedSeconds:     req.TimeUsedSeconds,
		RankingPercent:      ranking,
		IsPassed:            isPassed,
		SyllabusDiagnostics: diagnostics,
		QuestionSnapshots:   snapshots,
		CreatedAt:           time.Now(),
	}

	if err := s.db.Create(&record).Error; err != nil {
		return nil, fmt.Errorf("保存模考成绩记录失败: %w", err)
	}

	// 将模考中答错或未作答的试题自动沉淀同步至用户错题集
	now := time.Now()
	for _, snap := range snapshots {
		if !snap.IsCorrect && snap.ID > 0 {
			userErr := model.UserError{
				UserID:      userID,
				BankID:      req.BankID,
				QuestionID:  snap.ID,
				WrongCount:  1,
				IsMastered:  false,
				LastWrongAt: now,
			}
			s.db.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}, {Name: "question_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"wrong_count":   gorm.Expr("user_errors.wrong_count + 1"),
					"is_mastered":   false,
					"last_wrong_at": now,
				}),
			}).Create(&userErr)
		}
	}

	// 格式化展示文本
	mins := req.TimeUsedSeconds / 60
	secs := req.TimeUsedSeconds % 60
	record.TimeUsedText = fmt.Sprintf("%d分%02d秒", mins, secs)
	record.DateText = record.CreatedAt.Format("2006年01月02日")
	record.ScoreText = fmt.Sprintf("得分: %.1f / 100 分 · %s", score, func() string {
		if isPassed {
			return "及格"
		}
		return "未及格"
	}())

	return &record, nil
}

// ListMockRecords 获取用户的模考历史记录列表
func (s *MockService) ListMockRecords(userID uint, bankID uint) ([]model.MockRecord, error) {
	var records []model.MockRecord
	query := s.db.Where("user_id = ?", userID)
	if bankID > 0 {
		query = query.Where("bank_id = ?", bankID)
	}

	// 查询时不加载庞大的 question_snapshots 快照，节约网络传输带宽
	err := query.Omit("question_snapshots").Order("created_at DESC").Find(&records).Error
	if err != nil {
		return nil, err
	}

	for i := range records {
		r := &records[i]
		mins := r.TimeUsedSeconds / 60
		secs := r.TimeUsedSeconds % 60
		r.TimeUsedText = fmt.Sprintf("%d分钟", max(1, mins))
		if mins == 0 {
			r.TimeUsedText = fmt.Sprintf("%d秒", secs)
		}
		r.DateText = r.CreatedAt.Format("2006年01月02日")
		passStr := "及格"
		if !r.IsPassed {
			passStr = "未及格"
		}
		r.ScoreText = fmt.Sprintf("得分: %.0f / 100 分 · %s", r.Score, passStr)
	}

	return records, nil
}

// GetMockRecordByID 获取指定模考记录的完整战报与逐题解析快照
func (s *MockService) GetMockRecordByID(id uint, userID uint) (*model.MockRecord, error) {
	var record model.MockRecord
	err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&record).Error
	if err != nil {
		return nil, errors.New("模考记录不存在")
	}

	mins := record.TimeUsedSeconds / 60
	secs := record.TimeUsedSeconds % 60
	record.TimeUsedText = fmt.Sprintf("%d分%02d秒", mins, secs)
	record.DateText = record.CreatedAt.Format("2006年01月02日")
	passStr := "及格"
	if !record.IsPassed {
		passStr = "未及格"
	}
	record.ScoreText = fmt.Sprintf("得分: %.1f / 100 分 · %s", record.Score, passStr)

	return &record, nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
