package service

import (
	"math"
	"testing"
)

func TestStudyStatsAccuracyCalculation(t *testing.T) {
	calcAccuracy := func(total, correct int) int {
		if total <= 0 {
			return 100 // 初始为 100
		}
		return int(math.Round((float64(correct) / float64(total)) * 100))
	}

	// 1. 初始为 0 题时，平均准确率应为 100%
	if rate := calcAccuracy(0, 0); rate != 100 {
		t.Errorf("expected 100 for 0 answered, got %d", rate)
	}

	// 2. 答 1 题，答对 1 题 -> 100%
	if rate := calcAccuracy(1, 1); rate != 100 {
		t.Errorf("expected 100 for 1 correct of 1, got %d", rate)
	}

	// 3. 答 2 题，答对 1 题 -> 50%
	if rate := calcAccuracy(2, 1); rate != 50 {
		t.Errorf("expected 50 for 1 correct of 2, got %d", rate)
	}

	// 4. 答 3 题，答对 2 题 -> 67%
	if rate := calcAccuracy(3, 2); rate != 67 {
		t.Errorf("expected 67 for 2 correct of 3, got %d", rate)
	}

	// 5. 答 1280 题，答对 1088 题 -> 85%
	if rate := calcAccuracy(1280, 1088); rate != 85 {
		t.Errorf("expected 85 for 1088 correct of 1280, got %d", rate)
	}
}

func TestCheckInTriggerLogic(t *testing.T) {
	shouldCheckIn := func(todayCount, dailyGoal int, lastCheckInDate, todayDate string) bool {
		return todayCount >= dailyGoal && lastCheckInDate != todayDate
	}

	today := "2026-09-07"
	yesterday := "2026-09-06"

	// 场景 1: 未达标 -> 不打卡
	if shouldCheckIn(29, 30, yesterday, today) {
		t.Errorf("expected false for todayCount < dailyGoal")
	}

	// 场景 2: 刚达标且今天尚未打卡 -> 触发打卡
	if !shouldCheckIn(30, 30, yesterday, today) {
		t.Errorf("expected true for todayCount == dailyGoal and not checked in today")
	}

	// 场景 3: 超额达标且今天已打卡 -> 不重复触发打卡
	if shouldCheckIn(35, 30, today, today) {
		t.Errorf("expected false for already checked in today")
	}
}
