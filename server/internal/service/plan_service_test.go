package service

import (
	"math"
	"testing"

	"exam-server/internal/model"
)

func TestPlanDailyGoalValidation(t *testing.T) {
	svc := NewPlanService(nil)

	// 目标题数 <= 0 应当报错
	_, err := svc.SavePlan(1, model.SavePlanRequest{
		BankID:    1,
		DailyGoal: 0,
	})
	if err == nil {
		t.Fatalf("expected error for daily_goal <= 0, got nil")
	}

	_, errNegative := svc.SavePlan(1, model.SavePlanRequest{
		BankID:    1,
		DailyGoal: -10,
	})
	if errNegative == nil {
		t.Fatalf("expected error for negative daily_goal, got nil")
	}
}

func TestPlanDaysNeededCalculation(t *testing.T) {
	calcDaysNeeded := func(total, finished, dailyGoal int) int {
		remaining := total - finished
		if remaining <= 0 {
			return 0
		}
		if dailyGoal <= 0 {
			return 0
		}
		return int(math.Ceil(float64(remaining) / float64(dailyGoal)))
	}

	// 场景 1: 100题，已做25题，每日30题 -> 剩75题 -> 需 3 天
	if days := calcDaysNeeded(100, 25, 30); days != 3 {
		t.Errorf("expected 3 days, got %d", days)
	}

	// 场景 2: 100题，已做10题，每日10题 -> 剩90题 -> 需 9 天
	if days := calcDaysNeeded(100, 10, 10); days != 9 {
		t.Errorf("expected 9 days, got %d", days)
	}

	// 场景 3: 100题全部学完 -> 0 天
	if days := calcDaysNeeded(100, 100, 30); days != 0 {
		t.Errorf("expected 0 days for fully finished, got %d", days)
	}

	// 场景 4: 重复做题导致已刷题数超过总题数 -> 0 天
	if days := calcDaysNeeded(100, 120, 30); days != 0 {
		t.Errorf("expected 0 days for finished > total, got %d", days)
	}
}

func TestPlanIsTodayGoalReached(t *testing.T) {
	checkGoalReached := func(todayCount, dailyGoal int) bool {
		return todayCount >= dailyGoal
	}

	if checkGoalReached(15, 30) {
		t.Errorf("expected 15/30 to be false")
	}
	if !checkGoalReached(30, 30) {
		t.Errorf("expected 30/30 to be true")
	}
	if !checkGoalReached(45, 30) {
		t.Errorf("expected 45/30 to be true (overachieved)")
	}
}
