package service

import (
	"testing"
)

func TestCheckSensitiveContent(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantHit     bool
		wantWordSub string
	}{
		{"Normal study comment", "这道题很有深度，知识点总结得很到位，谢谢分享！", false, ""},
		{"Empty string", "", false, ""},
		// 涉政 / 邪教
		{"Political Falun", "宣传法轮功危害社会", true, "法轮功"},
		{"Political Falun Obfuscated", "法*轮*功 组织", true, "法轮功"},
		{"Political Leader", "李 洪 志 邪教教主", true, "李洪志"},
		{"Political Split", "坚决打击台独分裂势力", true, "台独"},
		// 涉黄 / 低俗
		{"Porn Adult Site", "找同城小姐包夜约炮加v", true, "找小姐"},
		{"Porn Video Site", "看片网址www.xxx", true, "看片网址"},
		{"Porn Obfuscated", "约-泡 软件", true, "约泡"},
		// 涉枪 / 涉毒
		{"Contraband Gun", "私自买枪买手枪被查", true, "买枪"},
		{"Contraband Drug", "海*洛*因 纯度高", true, "海洛因"},
		// 考试作弊
		{"Exam cheating direct", "需要代考请联系我", true, "代考"},
		{"Exam cheating with space", "专业 替 考 包 过", true, "替考"},
		{"Cheating obfuscated", "想买*答*案私聊", true, "买答案"},
		// 辱骂 / 网暴
		{"Profanity English", "This question is so shit", true, "shit"},
		{"Profanity Chinese", "真是一个大煞笔", true, "煞笔"},
		{"Profanity Obfuscated", "出题人真是个 傻 逼", true, "傻逼"},
		// 诈骗 / 引流
		{"Illegal scam", "高薪兼职刷单月入过万", true, "兼职刷单"},
		{"Illegal fake diploma", "办假证办理毕业证加微信", true, "办假证"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit, word := CheckSensitiveContent(tt.input)
			if hit != tt.wantHit {
				t.Errorf("CheckSensitiveContent(%q) hit = %v, want %v", tt.input, hit, tt.wantHit)
			}
			if tt.wantHit && word == "" {
				t.Errorf("CheckSensitiveContent(%q) hitWord is empty", tt.input)
			}
		})
	}
}
