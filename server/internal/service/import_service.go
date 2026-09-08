package service

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"exam-server/internal/model"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ImportService struct {
	db           *gorm.DB
	previewCache map[string]*CachedPreview
	cacheMutex   sync.RWMutex
}

type CachedPreview struct {
	Questions       []model.Question
	SyllabusWeights model.SyllabusWeightsList
	CreatedAt       time.Time
	FileName        string
}

func NewImportService(db *gorm.DB) *ImportService {
	s := &ImportService{
		db:           db,
		previewCache: make(map[string]*CachedPreview),
	}

	// 定期清理过期预览缓存（每小时清理一次，保留 2 小时）
	go func() {
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			s.cacheMutex.Lock()
			now := time.Now()
			for token, item := range s.previewCache {
				if now.Sub(item.CreatedAt) > time.Hour*2 {
					delete(s.previewCache, token)
				}
			}
			s.cacheMutex.Unlock()
		}
	}()

	return s
}

type ImportPreviewResult struct {
	PreviewToken        string                    `json:"preview_token"`
	FileName            string                    `json:"file_name"`
	TotalQuestions      int                       `json:"total_questions"`
	SingleChoiceCount   int                       `json:"single_choice_count"`
	MultipleChoiceCount int                       `json:"multiple_choice_count"`
	HasAnalysis         bool                      `json:"has_analysis"`
	SyllabusWeights     model.SyllabusWeightsList `json:"syllabus_weights"`
	SampleQuestions     []model.Question          `json:"sample_questions"`
}

// 解析第一行声明的考纲百分比规则 (如: "考纲配比:基础理论:20%|范围进度:30%|成本质量:25%|风险采购:25%")
func parseSyllabusWeightsRule(ruleStr string) (model.SyllabusWeightsList, error) {
	parts := strings.SplitN(ruleStr, ":", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(ruleStr, "：", 2)
	}
	if len(parts) < 2 {
		return nil, errors.New("考纲配比格式错误，应为：考纲配比:分类A:20%|分类B:30%...")
	}

	raw := strings.TrimSpace(parts[1])
	items := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '|' || r == ';' || r == '；' || r == ',' || r == '，'
	})

	var list model.SyllabusWeightsList
	totalPercent := 0.0

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		kv := strings.SplitN(item, ":", 2)
		if len(kv) < 2 {
			kv = strings.SplitN(item, "：", 2)
		}
		if len(kv) < 2 {
			return nil, fmt.Errorf("考纲项格式错误 [%s]，应为 分类名:百分比%%", item)
		}
		name := strings.TrimSpace(kv[0])
		valStr := strings.TrimSpace(kv[1])
		valStr = strings.TrimSuffix(valStr, "%")
		pct, err := strconv.ParseFloat(valStr, 64)
		if err != nil || pct <= 0 {
			return nil, fmt.Errorf("考纲项 [%s] 百分比数值无效: %s", name, kv[1])
		}
		list = append(list, model.SyllabusWeightItem{
			Name:    name,
			Percent: pct,
		})
		totalPercent += pct
	}

	if len(list) == 0 {
		return nil, errors.New("未能识别到有效的考纲配比规则")
	}

	if math.Abs(totalPercent-100.0) > 0.5 {
		return nil, fmt.Errorf("各考纲配比百分比之和必须为 100%%，当前总和为 %.1f%%", totalPercent)
	}

	return list, nil
}

// ParseExcelPreview 解析上传的 Excel/CSV 文件，生成预览统计与缓存 Token
func (s *ImportService) ParseExcelPreview(r io.Reader, fileName string) (*ImportPreviewResult, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	var rows [][]string
	var err error

	if ext == ".csv" {
		br := bufio.NewReader(r)
		// 自动识别并剥离 UTF-8 BOM (0xEF, 0xBB, 0xBF)，避免首字段带引号时引发 CSV 规范解析报错
		if bom, err := br.Peek(3); err == nil && len(bom) >= 3 && bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
			_, _ = br.Discard(3)
		}
		reader := csv.NewReader(br)
		rows, err = reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("CSV 解析失败: %w", err)
		}
	} else {
		f, err := excelize.OpenReader(r)
		if err != nil {
			return nil, fmt.Errorf("Excel 文件损坏或格式不支持: %w", err)
		}
		defer f.Close()

		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, errors.New("Excel 工作簿中无有效工作表")
		}

		rows, err = f.GetRows(sheets[0])
		if err != nil {
			return nil, fmt.Errorf("读取工作表失败: %w", err)
		}
	}

	if len(rows) < 2 {
		return nil, errors.New("导入文件行数过少，至少需包含表头和一道试题")
	}

	// 检查第一行是否为考纲权重声明
	var syllabusWeights model.SyllabusWeightsList
	startRowIdx := 1
	headerRow := rows[0]

	firstCell := ""
	if len(rows[0]) > 0 {
		firstCell = strings.TrimSpace(strings.TrimPrefix(rows[0][0], "\ufeff"))
	}
	if strings.HasPrefix(firstCell, "考纲配比") || strings.HasPrefix(firstCell, "考点配比") || strings.HasPrefix(firstCell, "大纲占比") || strings.HasPrefix(firstCell, "考纲占比") {
		weights, err := parseSyllabusWeightsRule(firstCell)
		if err != nil {
			return nil, err
		}
		syllabusWeights = weights
		startRowIdx = 2
		if len(rows) < 3 {
			return nil, errors.New("导入文件行数过少，需包含考纲配比声明、表头及试题明细")
		}
		headerRow = rows[1]
	}

	// 表头映射定位
	colMap := make(map[string]int)
	for idx, colName := range headerRow {
		cleaned := strings.TrimSpace(strings.TrimPrefix(colName, "\ufeff"))
		colMap[cleaned] = idx
	}

	// 智能识别表头列索引
	findCol := func(candidates ...string) int {
		for _, c := range candidates {
			for key, idx := range colMap {
				if strings.Contains(key, c) {
					return idx
				}
			}
		}
		return -1
	}

	titleIdx := findCol("题目", "题干", "试题内容", "Question")
	if titleIdx == -1 {
		titleIdx = 0 // 容错回退第1列
	}

	ansIdx := findCol("正确答案", "答案", "Answer")
	analysisIdx := findCol("解析", "详解", "题目解析", "考点解析", "Analysis")
	sectionIdx := findCol("大纲考点", "考点分类", "大纲分类", "章节", "模块", "Section")
	diffIdx := findCol("难度", "Difficulty")
	pointIdx := findCol("考点", "知识点", "Knowledge")

	// 选项列识别
	optionCols := []struct {
		Key string
		Idx int
	}{
		{"A", findCol("选项A", "选项 A", "A")},
		{"B", findCol("选项B", "选项 B", "B")},
		{"C", findCol("选项C", "选项 C", "C")},
		{"D", findCol("选项D", "选项 D", "D")},
		{"E", findCol("选项E", "选项 E", "E")},
		{"F", findCol("选项F", "选项 F", "F")},
	}

	var parsedQuestions []model.Question
	singleCount := 0
	multiCount := 0
	hasAnalysisCount := 0

	for rowIdx := startRowIdx; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if len(row) == 0 {
			continue
		}

		getVal := func(idx int) string {
			if idx >= 0 && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}

		qTitle := getVal(titleIdx)
		if qTitle == "" {
			continue // 跳过空题目
		}

		// 提取选项
		var options model.OptionsList
		for _, opt := range optionCols {
			if opt.Idx >= 0 {
				txt := getVal(opt.Idx)
				if txt != "" {
					txt = strings.TrimPrefix(txt, opt.Key+".")
					txt = strings.TrimPrefix(txt, opt.Key+"、")
					txt = strings.TrimPrefix(txt, opt.Key+":")
					txt = strings.TrimSpace(txt)
					options = append(options, model.OptionItem{
						Key:  opt.Key,
						Text: txt,
					})
				}
			}
		}

		// 提取答案
		rawAns := strings.ToUpper(getVal(ansIdx))
		var answers model.AnswerList
		for _, ch := range rawAns {
			if ch >= 'A' && ch <= 'F' {
				charStr := string(ch)
				already := false
				for _, a := range answers {
					if a == charStr {
						already = true
						break
					}
				}
				if !already {
					answers = append(answers, charStr)
				}
			}
		}

		// 单选/多选完全根据正确答案中的字母数量决定：字母数 > 1 为多选，字母数 <= 1 为单选
		qType := "单选"
		if len(answers) > 1 {
			qType = "多选"
			multiCount++
		} else {
			singleCount++
		}

		qAnalysis := getVal(analysisIdx)
		if qAnalysis != "" {
			hasAnalysisCount++
		}

		qSection := getVal(sectionIdx)
		if qSection == "" {
			qSection = "默认大纲分类"
		}

		qDiff := getVal(diffIdx)
		if qDiff == "" {
			qDiff = "中等"
		}

		// 去除知识点列后，知识点设为空；若文件兼容旧格式则读取
		qPoint := ""
		if pointIdx >= 0 {
			qPoint = getVal(pointIdx)
		}

		question := model.Question{
			Type:           qType,
			Section:        qSection,
			Title:          qTitle,
			Options:        options,
			Answer:         answers,
			Difficulty:     qDiff,
			KnowledgePoint: qPoint,
			Analysis:       qAnalysis,
			SortOrder:      len(parsedQuestions) + 1,
		}

		parsedQuestions = append(parsedQuestions, question)
	}

	if len(parsedQuestions) == 0 {
		return nil, errors.New("未能成功解析出任何有效试题，请检查 Excel 列名与内容")
	}

	// 统计试题中各大纲考点的题量分布
	sectionCountMap := make(map[string]int)
	for _, q := range parsedQuestions {
		sectionCountMap[q.Section]++
	}

	// 若未在第一行显式声明考纲配比，根据试题明细的自然题量占比自动生成固化配比
	if len(syllabusWeights) == 0 {
		totalQ := float64(len(parsedQuestions))
		var autoWeights model.SyllabusWeightsList
		for secName, cnt := range sectionCountMap {
			pct := math.Round((float64(cnt) / totalQ) * 100)
			if pct <= 0 {
				pct = 1
			}
			autoWeights = append(autoWeights, model.SyllabusWeightItem{
				Name:    secName,
				Percent: pct,
			})
		}
		syllabusWeights = autoWeights
	} else {
		// 显式声明了考纲配比时，严格校验每个考点是否有对应题目
		for _, w := range syllabusWeights {
			cnt := sectionCountMap[w.Name]
			if cnt == 0 {
				return nil, fmt.Errorf("考纲分类【%s】在试题明细中未找到任何试题，请核对“大纲考点”列名称是否一致", w.Name)
			}
		}
	}

	// 生成预览 Token 并缓存
	previewToken := uuid.New().String()
	s.cacheMutex.Lock()
	s.previewCache[previewToken] = &CachedPreview{
		Questions:       parsedQuestions,
		SyllabusWeights: syllabusWeights,
		CreatedAt:       time.Now(),
		FileName:        fileName,
	}
	s.cacheMutex.Unlock()

	// 截取前 3 题作为样例
	sampleCount := min(3, len(parsedQuestions))
	samples := make([]model.Question, sampleCount)
	copy(samples, parsedQuestions[:sampleCount])

	return &ImportPreviewResult{
		PreviewToken:        previewToken,
		FileName:            fileName,
		TotalQuestions:      len(parsedQuestions),
		SingleChoiceCount:   singleCount,
		MultipleChoiceCount: multiCount,
		HasAnalysis:         hasAnalysisCount > 0,
		SyllabusWeights:     syllabusWeights,
		SampleQuestions:     samples,
	}, nil
}

type ConfirmImportRequest struct {
	PreviewToken string `json:"preview_token" binding:"required"`
	Title        string `json:"title" binding:"required"`
	Category     string `json:"category"`
	CategoryID   uint   `json:"category_id"`
	Description  string `json:"description"`
	Visibility   string `json:"visibility"` // 'public' (公开) | 'private' (私有)
}

// ConfirmImport 确认导入并正式入库
func (s *ImportService) ConfirmImport(userID uint, req *ConfirmImportRequest) (*model.QuestionBank, error) {
	s.cacheMutex.RLock()
	cached, exists := s.previewCache[req.PreviewToken]
	s.cacheMutex.RUnlock()

	if !exists {
		return nil, errors.New("导入预览会话已过期，请重新上传文件")
	}

	bankTitle := strings.TrimSpace(req.Title)
	if bankTitle == "" {
		bankTitle = strings.TrimSuffix(cached.FileName, filepath.Ext(cached.FileName))
	}
	if bankTitle == "" {
		bankTitle = "导入题库"
	}

	// 题库名称全局唯一校验：若与系统中已有题库冲突，强行拦截并要求用户修改
	var existingCount int64
	if err := s.db.Model(&model.QuestionBank{}).Where("LOWER(TRIM(title)) = LOWER(TRIM(?))", bankTitle).Count(&existingCount).Error; err == nil && existingCount > 0 {
		return nil, fmt.Errorf("题库名称「%s」已存在，题库名称须全局唯一，请修改题库名称后重新提交", bankTitle)
	}

	categoryID := req.CategoryID
	categoryName := strings.TrimSpace(req.Category)

	// 若传了 category_id，校准 categoryName；若仅传了 categoryName，反查 category_id
	if categoryID > 0 {
		var cat model.Category
		if err := s.db.First(&cat, categoryID).Error; err == nil {
			categoryName = cat.Name
		} else {
			categoryID = 1
			categoryName = "综合"
		}
	} else if categoryName != "" {
		var cat model.Category
		if err := s.db.Where("LOWER(TRIM(name)) = LOWER(TRIM(?))", categoryName).First(&cat).Error; err == nil {
			categoryID = cat.ID
			categoryName = cat.Name
		} else {
			categoryID = 1
			categoryName = "综合"
		}
	} else {
		categoryID = 1
		categoryName = "综合"
	}

	visibility := strings.TrimSpace(req.Visibility)
	if visibility != "public" && visibility != "private" {
		visibility = "private"
	}

	// 查找用户信息以进行权限与配额校验
	var user model.User
	if userID > 0 {
		_ = s.db.First(&user, userID).Error
	}
	if user.Nickname == "" {
		user.Nickname = "备考学员"
	}

	reviewStatus := "approved"
	if visibility == "private" {
		// 1. 私有题库配额限制校验：普通用户最多2个，VIP用户最多20个，管理员无限制
		var privateCount int64
		if err := s.db.Model(&model.QuestionBank{}).Where("creator_id = ? AND visibility = 'private'", userID).Count(&privateCount).Error; err == nil {
			if user.Role == "user" && privateCount >= 2 {
				return nil, errors.New("普通用户最多拥有2个私有题库，升级VIP会员可拥有20个私有题库！")
			} else if user.Role == "vip" && privateCount >= 20 {
				return nil, errors.New("VIP用户最多拥有20个私有题库，已达上限！")
			}
		}
	} else if visibility == "public" {
		// 2. 公开题库单次上传限制校验：每次只能上传一个，由admin审批通过后才能再次上传
		var pendingCount int64
		if err := s.db.Model(&model.QuestionBank{}).Where("creator_id = ? AND visibility = 'public' AND review_status = 'pending'", userID).Count(&pendingCount).Error; err == nil {
			if pendingCount > 0 {
				return nil, errors.New("公开题库每次只能上传一个，您当前已有题库正在等待管理员审批，审批通过后方可再次上传公开题库！")
			}
		}
		// 公开题库需管理员手动审批，初始状态为待审核 (pending)
		reviewStatus = "pending"
	}

	var newBank model.QuestionBank

	err := s.db.Transaction(func(tx *gorm.DB) error {
		newBank = model.QuestionBank{
			Title:           bankTitle,
			Category:        categoryName,
			CategoryID:      categoryID,
			Description:     req.Description,
			TotalCount:      len(cached.Questions),
			IsOfficial:      false,
			Visibility:      visibility,
			ReviewStatus:    reviewStatus,
			CreatorID:       userID,
			CreatorName:     user.Nickname,
			SyllabusWeights: cached.SyllabusWeights, // 固化考纲配比，用户后期不可调整
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := tx.Create(&newBank).Error; err != nil {
			return fmt.Errorf("创建题库记录失败: %w", err)
		}

		// 若为待审批公开题库，自动生成管理员微信服务通知提醒
		if visibility == "public" && reviewStatus == "pending" {
			adminNotice := model.SystemNotification{
				UserID:    0, // 0 表示面向管理员系统通知
				Type:      "admin_pending",
				Title:     "【微信服务通知】有新的公开题库待审批",
				Content:   fmt.Sprintf("用户「%s」提交了新的公开题库《%s》（共 %d 道题），请前往个人中心手动审批栏进行审核。", user.Nickname, bankTitle, len(cached.Questions)),
				RelatedID: newBank.ID,
				IsRead:    false,
				CreatedAt: time.Now(),
			}
			if err := tx.Create(&adminNotice).Error; err != nil {
				return fmt.Errorf("记录管理员审批通知失败: %w", err)
			}
		}

		// 批量插入试题
		questionsToInsert := make([]model.Question, len(cached.Questions))
		for i, q := range cached.Questions {
			q.BankID = newBank.ID
			q.SortOrder = i + 1
			questionsToInsert[i] = q
		}

		// 分批每 100 条插入，避免 SQL 占位符超标
		batchSize := 100
		for i := 0; i < len(questionsToInsert); i += batchSize {
			end := i + batchSize
			if end > len(questionsToInsert) {
				end = len(questionsToInsert)
			}
			if err := tx.Create(questionsToInsert[i:end]).Error; err != nil {
				return fmt.Errorf("批量插入试题失败: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 清理缓存
	s.cacheMutex.Lock()
	delete(s.previewCache, req.PreviewToken)
	s.cacheMutex.Unlock()

	return &newBank, nil
}

// GetStandardTemplateCSV 返回标准导入格式的 CSV 模板内容（带 UTF-8 BOM 标识，兼容 Excel / WPS 与系统解析）
func GetStandardTemplateCSV() string {
	return "\xEF\xBB\xBF" +
		"\"题目\",\"选项A\",\"选项B\",\"选项C\",\"选项D\",\"正确答案\",\"解析\",\"大纲考点\",\"难度\"\r\n" +
		"\"敏捷开发中Scrum框架的核心事件不包括以下哪一项？\",\"冲刺规划会 (Sprint Planning)\",\"每日站会 (Daily Scrum)\",\"冲刺评审会 (Sprint Review)\",\"季度战略规划会 (Quarterly Strategic Planning)\",\"D\",\"Scrum框架规范了五个核心事件：冲刺、冲刺规划会、每日站会、冲刺评审会和冲刺回顾会。季度战略规划会属于企业高层治理，不属于Scrum规范事件。\",\"敏捷项目管理\",\"简单\"\r\n" +
		"\"敏捷项目团队中的三大核心角色包括哪些？\",\"产品负责人 (PO)\",\"Scrum主管 (Scrum Master)\",\"跨职能开发团队 (Developers)\",\"项目管理办公室主管 (PMO Director)\",\"ABC\",\"Scrum指南明确定义了三大角色：产品负责人(PO)、Scrum主管(SM)和开发团队。PMO不属于Scrum内部核心角色。\",\"敏捷项目管理\",\"中等\"\r\n" +
		"\"在关系型数据库中，关于主键约束（Primary Key）特性的描述错误的是？\",\"主键列的数据必须在整张表中保持唯一\",\"复合主键允许其中某一个字段存储NULL值\",\"一张数据表最多只能定义一个主键\",\"主键可以由单列或多个列组合而成\",\"B\",\"主键具有唯一性和非空性（NOT NULL）双重约束。无论是单列主键还是复合主键，其所有构成列均绝对不允许包含NULL值。\",\"数据库技术\",\"中等\"\r\n" +
		"\"计算机网络中，HTTP协议默认采用的传输层协议以及标准端口号是？\",\"UDP 协议 80 端口\",\"TCP 协议 80 端口\",\"TCP 协议 443 端口\",\"UDP 协议 8080 端口\",\"B\",\"HTTP（超文本传输协议）默认使用面向连接且可靠的TCP协议传输，标准端口为80；HTTPS默认标准端口为443。\",\"计算机网络\",\"简单\"\r\n" +
		"\"软件工程测试方法中，下列属于常见黑盒测试用例设计技术的有？\",\"等价类划分法\",\"边界值分析法\",\"判定/条件覆盖法\",\"错误推测法\",\"ABD\",\"等价类划分、边界值分析、因果图和错误推测均属于黑盒测试技术；而判定/条件覆盖属于白盒测试（逻辑覆盖测试）方法。\",\"软件测试\",\"困难\"\r\n"
}

