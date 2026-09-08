package service

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"exam-server/internal/model"

	"github.com/signintech/gopdf"
)

// FindChineseFontPath 查找系统或项目可用的中文字体文件
func FindChineseFontPath() (string, error) {
	candidates := []string{
		os.Getenv("PDF_FONT_PATH"),
		"C:/Windows/Fonts/simhei.ttf",
		"C:/Windows/Fonts/simsun.ttc",
		"C:/Windows/Fonts/simfang.ttf",
		"C:/Windows/Fonts/msyh.ttc",
		"C:/Windows/Fonts/Deng.ttf",
		"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"assets/fonts/simhei.ttf",
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			return path, nil
		}
	}

	return "", errors.New("未找到可用的中文字体文件，请确认系统已安装中文字体或设置环境变量 PDF_FONT_PATH")
}

// GenerateErrorsPDF 为指定科目的错题列表生成背题模式排版的 PDF
func GenerateErrorsPDF(bankName string, errorsList []model.UserError) ([]byte, error) {
	fontPath, err := FindChineseFontPath()
	if err != nil {
		return nil, err
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	fontName := "custom_chinese"
	if err := pdf.AddTTFFont(fontName, fontPath); err != nil {
		return nil, fmt.Errorf("加载中文字体失败: %w", err)
	}

	const (
		pageWidth    = 595.28
		pageHeight   = 841.89
		leftMargin   = 40.0
		rightMargin  = 40.0
		contentWidth = pageWidth - leftMargin - rightMargin // 515.28
		topMargin    = 45.0
		bottomMargin = 50.0
		maxY         = pageHeight - bottomMargin
	)

	curY := topMargin

	// 换页检查辅助闭包
	ensureSpace := func(neededHeight float64) {
		if curY+neededHeight > maxY {
			pdf.AddPage()
			curY = topMargin
		}
	}

	// 1. 文档首页顶端标题栏
	_ = pdf.SetFont(fontName, "", 16)
	pdf.SetTextColor(26, 26, 26) // #1a1a1a

	title := fmt.Sprintf("《%s · 错题背题集》", bankName)
	titleLines, _ := pdf.SplitText(title, contentWidth)
	for _, line := range titleLines {
		pdf.SetX(leftMargin)
		pdf.SetY(curY)
		_ = pdf.Cell(nil, line)
		curY += 24.0
	}

	// 副标题与元信息栏
	_ = pdf.SetFont(fontName, "", 9)
	pdf.SetTextColor(100, 100, 100) // #646464
	metaText := fmt.Sprintf("导出时间: %s  |  错题总数: %d 题  |  复习模式: 背题模式（标明答案与考点解析）",
		time.Now().Format("2006-01-02 15:04"), len(errorsList))
	pdf.SetX(leftMargin)
	pdf.SetY(curY)
	_ = pdf.Cell(nil, metaText)
	curY += 16.0

	// 顶部横向分割装饰线
	pdf.SetStrokeColor(0, 88, 188) // 品牌主色 #0058bc
	pdf.SetLineWidth(1.5)
	pdf.Line(leftMargin, curY, leftMargin+contentWidth, curY)
	curY += 18.0

	// 2. 循环输出每道错题（背题模式）
	for qIdx, errItem := range errorsList {
		q := errItem.Question
		if q == nil {
			continue
		}

		// 构建题型徽标与标签文本
		qType := q.Type
		if qType == "" {
			qType = "单选"
		}

		tagItems := []string{fmt.Sprintf("【%s题】", qType)}
		if q.Difficulty != "" {
			tagItems = append(tagItems, fmt.Sprintf("难度: %s", q.Difficulty))
		}
		if q.KnowledgePoint != "" {
			tagItems = append(tagItems, fmt.Sprintf("考点: %s", q.KnowledgePoint))
		}
		if q.Section != "" {
			tagItems = append(tagItems, fmt.Sprintf("章节: %s", q.Section))
		}
		tagsLine := strings.Join(tagItems, "  ")

		// 预先分割题干
		qTitleFull := fmt.Sprintf("%d. %s", qIdx+1, q.Title)
		_ = pdf.SetFont(fontName, "", 11)
		stemLines, _ := pdf.SplitText(qTitleFull, contentWidth)

		// 检查题头所需空间（标签 + 题干）
		neededHeaderSpace := 18.0 + float64(len(stemLines))*17.0 + 10.0
		ensureSpace(neededHeaderSpace)

		// 绘制题目标签行
		_ = pdf.SetFont(fontName, "", 9.5)
		pdf.SetTextColor(0, 88, 188) // 蓝色徽标
		pdf.SetX(leftMargin)
		pdf.SetY(curY)
		_ = pdf.Cell(nil, tagsLine)
		curY += 16.0

		// 绘制题干内容
		_ = pdf.SetFont(fontName, "", 11)
		pdf.SetTextColor(30, 30, 30) // 深灰黑色
		for _, sLine := range stemLines {
			pdf.SetX(leftMargin)
			pdf.SetY(curY)
			_ = pdf.Cell(nil, sLine)
			curY += 17.0
		}
		curY += 6.0

		// 选项集合及正确答案映射
		correctMap := make(map[string]bool)
		for _, ansKey := range q.Answer {
			correctMap[strings.ToUpper(strings.TrimSpace(ansKey))] = true
		}

		// 绘制选项列表（在背题模式下直观标出正确选项）
		_ = pdf.SetFont(fontName, "", 10)
		for _, opt := range q.Options {
			isCorrect := correctMap[strings.ToUpper(strings.TrimSpace(opt.Key))]

			prefix := " [   ] "
			if isCorrect {
				prefix = " [✔ 正确] "
			}

			optText := fmt.Sprintf("%s%s. %s", prefix, opt.Key, opt.Text)
			optLines, _ := pdf.SplitText(optText, contentWidth-10)

			ensureSpace(float64(len(optLines)) * 16.0)

			for _, oLine := range optLines {
				pdf.SetX(leftMargin + 6.0)
				pdf.SetY(curY)

				if isCorrect {
					pdf.SetTextColor(16, 124, 65) // 绿色高亮 #107c41
				} else {
					pdf.SetTextColor(60, 60, 60)
				}

				_ = pdf.Cell(nil, oLine)
				curY += 16.0
			}
			curY += 2.0
		}
		curY += 6.0

		// 绘制背题模式常驻【考点与精析】卡片
		analysisList := []string{}

		// 格式化正确答案
		var answerDisplay string
		if len(q.Answer) > 0 {
			answerDisplay = strings.Join(q.Answer, ", ")
		} else {
			answerDisplay = "见解析"
		}
		analysisList = append(analysisList, fmt.Sprintf("【正确答案】：%s", answerDisplay))

		analysisContent := strings.TrimSpace(q.Analysis)
		if analysisContent == "" {
			analysisContent = "暂无详细试题解析"
		}
		analysisList = append(analysisList, fmt.Sprintf("【试题解析】：%s", analysisContent))

		if strings.TrimSpace(q.KnowledgeDetail) != "" {
			analysisList = append(analysisList, fmt.Sprintf("【知识拓展】：%s", strings.TrimSpace(q.KnowledgeDetail)))
		}

		// 计算精析卡片内部所有换行
		_ = pdf.SetFont(fontName, "", 9.5)
		cardInnerWidth := contentWidth - 20.0
		type cardLineItem struct {
			text      string
			isAnsLine bool
		}
		var cardLines []cardLineItem

		for i, block := range analysisList {
			lines, _ := pdf.SplitText(block, cardInnerWidth)
			for _, l := range lines {
				cardLines = append(cardLines, cardLineItem{
					text:      l,
					isAnsLine: (i == 0),
				})
			}
			if i < len(analysisList)-1 {
				// 块与块之间留微小间距
				cardLines = append(cardLines, cardLineItem{text: ""})
			}
		}

		cardHeight := float64(len(cardLines))*15.0 + 16.0 // 上下 padding 8
		ensureSpace(cardHeight + 10.0)

		// 绘制卡片背景矩形（浅灰底纹与浅灰边框）
		pdf.SetFillColor(246, 248, 250)   // #f6f8fa
		pdf.SetStrokeColor(225, 230, 235) // #e1e6eb
		pdf.SetLineWidth(0.8)
		pdf.RectFromUpperLeftWithStyle(leftMargin, curY, contentWidth, cardHeight, "FD")

		cardTextY := curY + 9.0
		for _, cl := range cardLines {
			if cl.text != "" {
				pdf.SetX(leftMargin + 10.0)
				pdf.SetY(cardTextY)

				if cl.isAnsLine {
					pdf.SetTextColor(16, 124, 65) // 答案行绿色加重
				} else {
					pdf.SetTextColor(60, 60, 60)
				}
				_ = pdf.Cell(nil, cl.text)
			}
			cardTextY += 15.0
		}
		curY += cardHeight + 14.0

		// 每道题末尾的浅色分割线
		pdf.SetStrokeColor(230, 230, 230)
		pdf.SetLineWidth(0.6)
		pdf.Line(leftMargin, curY, leftMargin+contentWidth, curY)
		curY += 14.0
	}

	return pdf.GetBytesPdf(), nil
}
