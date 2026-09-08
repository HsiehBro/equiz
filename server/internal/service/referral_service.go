package service

import (
	"fmt"
	"math/rand"
	"time"

	"exam-server/internal/model"

	"gorm.io/gorm"
)

type ReferralService struct {
	db *gorm.DB
}

func NewReferralService(db *gorm.DB) *ReferralService {
	return &ReferralService{db: db}
}

// PlanMeta 维护套餐对应的天数与价格
type PlanMeta struct {
	Name  string
	Days  int
	Price float64
}

var VIPPlanConfig = map[string]PlanMeta{
	"monthly":   {Name: "连续包月会员", Days: 30, Price: 29.0},
	"quarterly": {Name: "连续包季会员", Days: 90, Price: 68.0},
	"yearly":    {Name: "连续包年会员", Days: 365, Price: 198.0},
	"lifetime":  {Name: "终身畅学卡", Days: 36500, Price: 398.0},
}

// BindReferral 绑定被邀请人与邀请人的归属关系（首购保护）
func (s *ReferralService) BindReferral(inviteeID uint, inviterID uint) error {
	// 1. 防自邀检查
	if inviteeID == inviterID {
		return fmt.Errorf("不能邀请自己")
	}

	// 2. 检查邀请人是否存在且非永久会员（永久会员不参与活动）
	var inviter model.User
	if err := s.db.First(&inviter, inviterID).Error; err != nil {
		return fmt.Errorf("邀请人不存在")
	}
	if inviter.IsLifetimeVIP {
		return fmt.Errorf("永久会员不可参与活动推广")
	}

	// 3. 检查被邀请人是否存在
	var invitee model.User
	if err := s.db.First(&invitee, inviteeID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 4. 首购保护：若被邀请人此前已经是 VIP 或永久 VIP，则不再享受被邀请资格
	if invitee.IsLifetimeVIP || (invitee.VIPExpire != nil && invitee.VIPExpire.After(time.Now())) {
		return fmt.Errorf("您已是 VIP 会员，无需重复绑定邀请关系")
	}

	// 5. 检查是否已被任何人绑定过（单用户唯一被邀请锁定）
	var existing model.UserReferral
	if err := s.db.Where("invitee_id = ?", inviteeID).First(&existing).Error; err == nil {
		// 已存在绑定记录，保持幂等成功，不再重复创建
		return nil
	}

	// 6. 防刷互邀拦截：A 邀了 B，则 B 不能反向绑定 A
	var reverse model.UserReferral
	if err := s.db.Where("inviter_id = ? AND invitee_id = ?", inviteeID, inviterID).First(&reverse).Error; err == nil {
		return fmt.Errorf("双方不能互刷邀请奖励")
	}

	// 7. 创建锁定记录
	ref := model.UserReferral{
		InviterID: inviterID,
		InviteeID: inviteeID,
		Status:    "bound",
	}

	if err := s.db.Create(&ref).Error; err != nil {
		return fmt.Errorf("绑定邀请关系失败: %w", err)
	}

	return nil
}

type VIPPurchaseResult struct {
	User       *model.User `json:"user"`
	RewardDays int         `json:"reward_days"`
	OrderNo    string      `json:"order_no"`
}

// ProcessVIPPurchase 处理 VIP 开通结算与邀请发奖联动
func (s *ReferralService) ProcessVIPPurchase(userID uint, planID string) (*VIPPurchaseResult, error) {
	plan, ok := VIPPlanConfig[planID]
	if !ok {
		return nil, fmt.Errorf("无效的 VIP 会员套餐")
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var user model.User
	if err := tx.First(&user, userID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("用户不存在")
	}

	if user.IsLifetimeVIP {
		tx.Rollback()
		return nil, fmt.Errorf("您已是终身永久 VIP，无需购买套餐")
	}

	now := time.Now()
	isLifetime := planID == "lifetime"

	// 1. 更新当前购买者的基础 VIP 权益
	if isLifetime {
		user.IsLifetimeVIP = true
		user.Role = "vip"
		user.VIPExpire = nil
	} else {
		user.Role = "vip"
		var newExpire time.Time
		if user.VIPExpire != nil && user.VIPExpire.After(now) {
			newExpire = user.VIPExpire.AddDate(0, 0, plan.Days)
		} else {
			newExpire = now.AddDate(0, 0, plan.Days)
		}
		user.VIPExpire = &newExpire
	}

	// 2. 生成订单记录
	randNum := rand.Intn(9000) + 1000
	orderNo := fmt.Sprintf("VIP%d%04d", now.Unix(), randNum)
	order := model.VIPOrder{
		OrderNo: orderNo,
		UserID:  userID,
		PlanID:  planID,
		Amount:  plan.Price,
		Status:  "paid",
	}
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建订单失败: %w", err)
	}

	// 3. 检查并结算邀请发奖联动
	rewardDays := 0
	var referral model.UserReferral
	if err := tx.Where("invitee_id = ? AND status = 'bound'", userID).First(&referral).Error; err == nil {
		// 计算返奖天数：终身卡购买封顶赠送 365 天；周期卡赠送同等天数
		if isLifetime {
			rewardDays = 365
		} else {
			rewardDays = plan.Days
		}

		// A. 给邀请人加时长 (若邀请人已经是永久 VIP 则跳过)
		var inviter model.User
		if err := tx.First(&inviter, referral.InviterID).Error; err == nil {
			if !inviter.IsLifetimeVIP {
				var inviterExpire time.Time
				if inviter.VIPExpire != nil && inviter.VIPExpire.After(now) {
					inviterExpire = inviter.VIPExpire.AddDate(0, 0, rewardDays)
				} else {
					inviterExpire = now.AddDate(0, 0, rewardDays)
				}
				inviter.VIPExpire = &inviterExpire
				inviter.Role = "vip"
				if err := tx.Save(&inviter).Error; err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("更新邀请人时长失败: %w", err)
				}
			}

			// 发送系统通知给邀请人
			inviterNotify := model.SystemNotification{
				UserID:  referral.InviterID,
				Type:    "activity",
				Title:   "🎁 邀请好友福利已到账！",
				Content: fmt.Sprintf("太棒了！您邀请的好友【%s】已成功开通【%s】，已自动为您增加 %d 天 VIP 会员时长！", user.Nickname, plan.Name, rewardDays),
			}
			tx.Create(&inviterNotify)
		}

		// B. 给被邀请人（双方同享）：如果开通的是周期卡，再额外获赠同等时长
		if !isLifetime && rewardDays > 0 {
			extraExpire := user.VIPExpire.AddDate(0, 0, rewardDays)
			user.VIPExpire = &extraExpire
		}

		// C. 标记邀请关系完成
		referral.Status = "rewarded"
		referral.PlanID = planID
		referral.RewardDays = rewardDays
		referral.RewardedAt = &now
		if err := tx.Save(&referral).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("更新邀请状态失败: %w", err)
		}

		// 发送系统通知给被邀请人
		inviteeNotify := model.SystemNotification{
			UserID:  user.ID,
			Type:    "activity",
			Title:   "🎉 邀请专享同等时长已生效！",
			Content: fmt.Sprintf("您通过好友邀请成功开通【%s】，专属同等时长福利已累加生效至您的账号！", plan.Name),
		}
		tx.Create(&inviteeNotify)
	}

	// 4. 保存购买者信息
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("保存用户信息失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &VIPPurchaseResult{
		User:       &user,
		RewardDays: rewardDays,
		OrderNo:    orderNo,
	}, nil
}

type ReferralItem struct {
	InviteeID       uint       `json:"invitee_id"`
	InviteeNickname string     `json:"invitee_nickname"`
	InviteeAvatar   string     `json:"invitee_avatar"`
	Status          string     `json:"status"`
	PlanName        string     `json:"plan_name"`
	RewardDays      int        `json:"reward_days"`
	RewardedAt      *time.Time `json:"rewarded_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ReferralStatsResponse struct {
	TotalRewardDays  int            `json:"total_reward_days"`
	TotalInviteCount int            `json:"total_invite_count"`
	List             []ReferralItem `json:"list"`
}

// maskNickname 对学员昵称做脱敏处理 (如 "小明同学" -> "小**学")
func maskNickname(name string) string {
	runes := []rune(name)
	l := len(runes)
	if l <= 1 {
		return name
	}
	if l == 2 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + "*" + string(runes[l-1])
}

// GetReferralStats 查询邀请人的累计奖励统计与好友列表
func (s *ReferralService) GetReferralStats(userID uint) (*ReferralStatsResponse, error) {
	var referrals []model.UserReferral
	err := s.db.Preload("Invitee").Where("inviter_id = ?", userID).Order("created_at DESC").Find(&referrals).Error
	if err != nil {
		return nil, fmt.Errorf("查询邀请记录失败: %w", err)
	}

	totalDays := 0
	successCount := 0
	list := make([]ReferralItem, 0, len(referrals))

	for _, ref := range referrals {
		nickname := "学员"
		avatar := ""
		if ref.Invitee != nil {
			nickname = maskNickname(ref.Invitee.Nickname)
			avatar = ref.Invitee.AvatarURL
		}

		planName := ""
		if p, ok := VIPPlanConfig[ref.PlanID]; ok {
			planName = p.Name
		}

		if ref.Status == "rewarded" {
			totalDays += ref.RewardDays
			successCount++
		}

		list = append(list, ReferralItem{
			InviteeID:       ref.InviteeID,
			InviteeNickname: nickname,
			InviteeAvatar:   avatar,
			Status:          ref.Status,
			PlanName:        planName,
			RewardDays:      ref.RewardDays,
			RewardedAt:      ref.RewardedAt,
			CreatedAt:       ref.CreatedAt,
		})
	}

	return &ReferralStatsResponse{
		TotalRewardDays:  totalDays,
		TotalInviteCount: successCount,
		List:             list,
	}, nil
}
