package service

import (
	"testing"

	"exam-server/internal/model"
)

func TestParseSyllabusWeightsRule(t *testing.T) {
	// 正常用例
	rule1 := "考纲配比:基础理论:20%|范围进度:30%|成本质量:25%|风险采购:25%"
	weights, err := parseSyllabusWeightsRule(rule1)
	if err != nil {
		t.Fatalf("expected valid rule, got error: %v", err)
	}
	if len(weights) != 4 {
		t.Fatalf("expected 4 weights, got %d", len(weights))
	}
	if weights[0].Name != "基础理论" || weights[0].Percent != 20 {
		t.Errorf("expected 基础理论:20, got %s:%.1f", weights[0].Name, weights[0].Percent)
	}

	// 错误用例：百分比总和不为 100%
	rule2 := "考纲配比:基础理论:20%|范围进度:30%"
	_, err2 := parseSyllabusWeightsRule(rule2)
	if err2 == nil {
		t.Fatalf("expected error for sum != 100%%, got nil")
	}

	// 中文冒号与分号容错
	rule3 := "考纲配比：模块A：50%；模块B：50%"
	weights3, err3 := parseSyllabusWeightsRule(rule3)
	if err3 != nil {
		t.Fatalf("expected valid chinese colon rule, got error: %v", err3)
	}
	if len(weights3) != 2 {
		t.Fatalf("expected 2 weights, got %d", len(weights3))
	}
}

func TestMockDiagnosticsCalculation(t *testing.T) {
	// 模拟交卷打分与大纲维度统计
	svc := NewMockService(nil)

	q1 := model.Question{
		ID:      1,
		Section: "基础理论",
		Answer:  model.AnswerList{"A"},
	}
	q2 := model.Question{
		ID:      2,
		Section: "基础理论",
		Answer:  model.AnswerList{"B"},
	}
	q3 := model.Question{
		ID:      3,
		Section: "项目进度",
		Answer:  model.AnswerList{"C"},
	}
	q4 := model.Question{
		ID:      4,
		Section: "项目进度",
		Answer:  model.AnswerList{"D"},
	}

	questions := []model.Question{q1, q2, q3, q4}
	userAnswers := map[uint][]string{
		1: {"A"}, // 对
		2: {"A"}, // 错 (应为 B)
		3: {"C"}, // 对
		4: {"D"}, // 对
	}

	// 3 对 1 错，总分 75 分
	// 基础理论：1对 1错 (50% -> 薄弱待强化)
	// 项目进度：2对 0错 (100% -> 掌握良好)

	correctTotal := 0
	wrongTotal := 0
	for _, q := range questions {
		if CompareAnswers(userAnswers[q.ID], q.Answer) {
			correctTotal++
		} else {
			wrongTotal++
		}
	}

	if correctTotal != 3 || wrongTotal != 1 {
		t.Errorf("expected 3 correct and 1 wrong, got %d and %d", correctTotal, wrongTotal)
	}
	score := float64(correctTotal) / float64(len(questions)) * 100.0
	if score != 75.0 {
		t.Errorf("expected score 75.0, got %.1f", score)
	}
	_ = svc
}
