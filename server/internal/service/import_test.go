package service

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestCompareAnswers(t *testing.T) {
	// 单选题测试
	if !CompareAnswers([]string{"A"}, []string{"A"}) {
		t.Errorf("expected A == A")
	}
	if CompareAnswers([]string{"A"}, []string{"B"}) {
		t.Errorf("expected A != B")
	}

	// 多选题乱序测试
	if !CompareAnswers([]string{"B", "A", "C"}, []string{"A", "B", "C"}) {
		t.Errorf("expected BAC == ABC")
	}
	if CompareAnswers([]string{"A", "B"}, []string{"A", "B", "C"}) {
		t.Errorf("expected AB != ABC")
	}

	// 大小写和空格容错
	if !CompareAnswers([]string{" a ", "b"}, []string{"A", "B"}) {
		t.Errorf("expected case insensitive match")
	}
}

func TestExcelImportParser(t *testing.T) {
	// 动态构造一个测试用内存 Excel 文件
	f := excelize.NewFile()
	sheet := "Sheet1"

	// 表头
	f.SetCellValue(sheet, "A1", "题目")
	f.SetCellValue(sheet, "B1", "选项A")
	f.SetCellValue(sheet, "C1", "选项B")
	f.SetCellValue(sheet, "D1", "选项C")
	f.SetCellValue(sheet, "E1", "选项D")
	f.SetCellValue(sheet, "F1", "正确答案")
	f.SetCellValue(sheet, "G1", "解析")

	// 第 1 题（单选）
	f.SetCellValue(sheet, "A2", "敏捷开发中Scrum框架的核心事件不包括？")
	f.SetCellValue(sheet, "B2", "冲刺规划会")
	f.SetCellValue(sheet, "C2", "每日站会")
	f.SetCellValue(sheet, "D2", "冲刺评审会")
	f.SetCellValue(sheet, "E2", "年度战略会")
	f.SetCellValue(sheet, "F2", "D")
	f.SetCellValue(sheet, "G2", "Scrum 包含五大事件，不包含年度战略会。")

	// 第 2 题（多选）
	f.SetCellValue(sheet, "A3", "常见敏捷三大角色包括？")
	f.SetCellValue(sheet, "B3", "产品负责人 (PO)")
	f.SetCellValue(sheet, "C3", "Scrum Master")
	f.SetCellValue(sheet, "D3", "开发团队")
	f.SetCellValue(sheet, "E3", "项目总监")
	f.SetCellValue(sheet, "F3", "A,B,C")
	f.SetCellValue(sheet, "G3", "三大角色由 PO、SM 和开发团队组成。")

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		t.Fatalf("failed to write excel buffer: %v", err)
	}

	svc := NewImportService(nil)
	preview, err := svc.ParseExcelPreview(buf, "test_exam.xlsx")
	if err != nil {
		t.Fatalf("ParseExcelPreview failed: %v", err)
	}

	if preview.TotalQuestions != 2 {
		t.Errorf("expected 2 questions, got %d", preview.TotalQuestions)
	}
	if preview.SingleChoiceCount != 1 {
		t.Errorf("expected 1 single choice, got %d", preview.SingleChoiceCount)
	}
	if preview.MultipleChoiceCount != 1 {
		t.Errorf("expected 1 multiple choice, got %d", preview.MultipleChoiceCount)
	}
	if !preview.HasAnalysis {
		t.Errorf("expected has analysis true")
	}
	if preview.PreviewToken == "" {
		t.Errorf("expected preview token not empty")
	}
}

func TestStandardCSVTemplate(t *testing.T) {
	csvData := GetStandardTemplateCSV()
	if len(csvData) == 0 {
		t.Fatalf("GetStandardTemplateCSV returned empty string")
	}

	// 验证包含 UTF-8 BOM
	if !bytes.HasPrefix([]byte(csvData), []byte("\xEF\xBB\xBF")) {
		t.Errorf("expected CSV to start with UTF-8 BOM")
	}

	svc := NewImportService(nil)
	buf := bytes.NewBufferString(csvData)
	preview, err := svc.ParseExcelPreview(buf, "标准题库导入模板.csv")
	if err != nil {
		t.Fatalf("ParseExcelPreview failed on standard CSV template: %v", err)
	}

	if preview.TotalQuestions != 5 {
		t.Errorf("expected 5 questions in standard template, got %d", preview.TotalQuestions)
	}
	if preview.SingleChoiceCount != 3 {
		t.Errorf("expected 3 single choice questions, got %d", preview.SingleChoiceCount)
	}
	if preview.MultipleChoiceCount != 2 {
		t.Errorf("expected 2 multiple choice questions, got %d", preview.MultipleChoiceCount)
	}
	if !preview.HasAnalysis {
		t.Errorf("expected has analysis to be true")
	}
	if len(preview.SyllabusWeights) == 0 {
		t.Errorf("expected non-empty syllabus weights")
	}

	// 验证单选题和多选题根据答案字母数量正确识别
	if len(preview.SampleQuestions) >= 3 {
		// 第 1 题答案为 D (1个字母) -> 必须识别为单选
		if preview.SampleQuestions[0].Type != "单选" {
			t.Errorf("expected Q1 to be 单选, got %s", preview.SampleQuestions[0].Type)
		}
		// 第 2 题答案为 ABC (3个字母) -> 必须识别为多选
		if preview.SampleQuestions[1].Type != "多选" {
			t.Errorf("expected Q2 to be 多选, got %s", preview.SampleQuestions[1].Type)
		}
		// 第 3 题答案为 B (1个字母) -> 必须识别为单选
		if preview.SampleQuestions[2].Type != "单选" {
			t.Errorf("expected Q3 to be 单选, got %s", preview.SampleQuestions[2].Type)
		}
	}
}

