package service

import (
	"fmt"
	"testing"

	"exam-server/internal/model"

	"github.com/signintech/gopdf"
)

func TestGopdfChinese(t *testing.T) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	fontPath := "C:/Windows/Fonts/simhei.ttf"
	err := pdf.AddTTFFont("simhei", fontPath)
	if err != nil {
		t.Fatalf("AddTTFFont err: %v", err)
	}
	err = pdf.SetFont("simhei", "", 12)
	if err != nil {
		t.Fatalf("SetFont err: %v", err)
	}

	text := "这是一个用于测试中文字符自动换行的长文本。马克思主义哲学包括辩证唯物主义和历史唯物主义，是无产阶级的科学世界观和方法论。"
	lines, err := pdf.SplitText(text, 200)
	if err != nil {
		t.Fatalf("SplitText err: %v", err)
	}
	fmt.Printf("SplitText produced %d lines:\n", len(lines))

	pdf.SetFillColor(245, 247, 250)
	pdf.SetStrokeColor(218, 224, 233)
	pdf.RectFromUpperLeftWithStyle(40, 50, 515, 60, "FD")

	pdf.SetTextColor(0, 88, 188)
	pdf.SetX(50)
	pdf.SetY(60)
	pdf.Cell(nil, "【单选题】 1. 唯物辩证法的核心是什么？")

	pdf.SetStrokeColor(200, 200, 200)
	pdf.Line(40, 120, 555, 120)

	pdfBytes := pdf.GetBytesPdf()
	if len(pdfBytes) == 0 {
		t.Fatalf("Expected non-empty pdf bytes")
	}
	fmt.Printf("Generated PDF size: %d bytes\n", len(pdfBytes))
}

func TestGenerateErrorsPDF(t *testing.T) {
	mockErrors := []model.UserError{
		{
			ID:         1,
			UserID:     1,
			BankID:     1,
			QuestionID: 101,
			WrongCount: 3,
			Question: &model.Question{
				ID:             101,
				BankID:         1,
				Type:           "单选",
				Difficulty:     "中等",
				KnowledgePoint: "唯物辩证法核心",
				Section:        "第一章 哲学基本问题",
				Title:          "唯物辩证法的实质和核心是什么？",
				Options: model.OptionsList{
					{Key: "A", Text: "对立统一规律"},
					{Key: "B", Text: "质量互变规律"},
					{Key: "C", Text: "否定之否定规律"},
					{Key: "D", Text: "普遍联系规律"},
				},
				Answer:          model.AnswerList{"A"},
				Analysis:        "对立统一规律揭示了普遍联系的根本内容和事物内部发展的动力，是唯物辩证法的实质和核心。",
				KnowledgeDetail: "对立统一规律提供了人们认识世界和改造世界的根本方法，即矛盾分析法。",
			},
		},
		{
			ID:         2,
			UserID:     1,
			BankID:     1,
			QuestionID: 102,
			WrongCount: 1,
			Question: &model.Question{
				ID:             102,
				BankID:         1,
				Type:           "多选",
				Difficulty:     "较难",
				KnowledgePoint: "商品二因素",
				Section:        "第二章 资本主义形成",
				Title:          "下列关于商品使用价值与价值的关系，表述正确的有：",
				Options: model.OptionsList{
					{Key: "A", Text: "使用价值是价值的物质承担者"},
					{Key: "B", Text: "有使用价值的东西一定有价值"},
					{Key: "C", Text: "价值寓于使用价值之中"},
					{Key: "D", Text: "二者相互排斥，不可兼得"},
				},
				Answer:          model.AnswerList{"A", "C", "D"},
				Analysis:        "使用价值是商品的自然属性，价值是商品的社会属性。使用价值是价值的物质承担者，价值存在于使用价值之中。对生产者或消费者而言，二者不可兼得。",
				KnowledgeDetail: "有使用价值的物品如果没有凝聚人类一般劳动（如空气、阳光），则不具有价值，故B项错误。",
			},
		},
	}

	pdfData, err := GenerateErrorsPDF("马克思主义基本原理", mockErrors)
	if err != nil {
		t.Fatalf("GenerateErrorsPDF failed: %v", err)
	}

	if len(pdfData) == 0 {
		t.Fatalf("GenerateErrorsPDF returned empty data")
	}

	fmt.Printf("Successfully generated errors PDF, size: %d bytes\n", len(pdfData))
}

