package service

import (
	"strings"
	"unicode"
)

// SensitiveCategories 按中国大陆内容安全治理标准分层归纳
var SensitiveCategories = map[string][]string{
	// 1. 涉政、意识形态与邪教分裂
	"POLITICAL": {
		"法轮功", "李洪志", "真善忍", "明慧网", "九评", "大纪元", "退党", "神韵艺术", "邪教",
		"分裂国家", "颠覆政权", "颠覆国家", "推翻政权", "暴动", "暴乱", "涉恐", "恐怖分子",
		"东突", "东突厥斯坦", "疆独", "藏独", "港独", "台独", "时代革命", "光复香港",
		"颜色革命", "反共", "反党", "打倒共产党", "推翻政府", "反华", "境外敌对势力",
		"六四事件", "天安门事件", "89学潮", "屠杀平民", "维权抗暴", "自焚",
	},

	// 2. 色情、低俗淫秽与招嫖
	"PORN": {
		"招嫖", "嫖娼", "卖淫", "找小姐", "兼职小姐", "同城兼职妹", "包夜", "全套服务", "特殊服务",
		"桑拿全套", "上门按摩", "同城交友约", "约炮", "约泡", "找约", "裸聊", "色情直播", "黄色直播",
		"色情网站", "黄色网站", "黄网", "看片网址", "看黄片", "黄色小电影", "成人电影", "成人片",
		"无码", "露点", "偷拍视频", "走光自拍", "艳照", "艳照门", "换妻", "群交", "迷奸", "迷药",
		"听话水", "乖乖水", "催情水", "春药", "迷魂药", "三唑仑", "失忆水", "口交", "手淫", "自慰",
		"阴茎", "阴道", "生殖器", "肉棒", "骚逼", "巨乳露点", "强奸", "轮奸",
	},

	// 3. 涉毒、涉暴、枪支管制与危险品
	"CONTRABAND": {
		"毒品", "贩毒", "吸毒", "大麻", "海洛因", "冰毒", "甲基苯丙胺", "K粉", "氯胺酮", "摇头丸",
		"麻古", "止咳水", "可卡因", "芬太尼", "恰特草", "致幻剂",
		"买枪", "手枪", "步枪", "气枪", "火药枪", "猎枪", "自制手枪", "铅弹", "子弹", "弹药",
		"管制刀具", "弓弩", "雷管", "炸药", "自制炸弹", "炸弹制作", "军用刺刀", "违禁品出售",
	},

	// 4. 诈骗、赌博、黑灰产与引流
	"SCAM": {
		"网络赌博", "境外赌博", "博彩网站", "在线博彩", "百家乐", "时时彩", "六合彩", "地下钱庄",
		"太阳城娱乐", "澳门博彩", "彩票预测", "包中特码", "兼职刷单", "刷单返利", "网店代刷",
		"高薪日结刷单", "办假证", "刻章办证", "办理毕业证", "做假学历", "假文凭", "办假身份证",
		"买卖银行卡", "收银行卡", "收u", "洗黑钱", "跑分洗钱", "高利贷", "套路贷", "裸贷",
		"黑客接单", "木马盗号", "呼死你", "短信轰炸", "撞库破解", "微信号购买",
		"加微信", "加我微信", "加威信", "加微", "加v", "加vx", "加qq", "微信同号", "威信同号", "私聊微信",
	},

	// 5. 考试作弊与学术不端
	"EXAM": {
		"代考", "替考", "枪手", "找枪手", "买答案", "卖答案", "买考题", "卖考题", "出售考题",
		"考前押密", "考前密卷", "考前答案", "真题泄露", "透题", "泄题", "考场作弊", "作弊器材",
		"隐形耳机作弊", "修改成绩", "改分", "花钱改成绩", "保过", "包过", "不过退款代考", "助考",
		"包拿证", "免考拿证", "替考包过", "押题包过",
	},

	// 6. 恶毒辱骂与网络暴力
	"ABUSE": {
		"傻逼", "煞笔", "沙比", "二逼", "二B", "2b", "sb", "SB", "操你", "草你", "操你妈", "草泥马",
		"操你全家", "日你妈", "日你妹", "妈的", "狗日的", "狗东西", "滚蛋", "去死吧", "白痴", "脑残",
		"弱智", "弱鸡", "低能儿", "垃圾题", "司马", "死全家", "死妈", "全家暴毙", "cnm", "CNM",
		"nmsl", "NMSL", "nt", "NT", "fuck", "shit", "bitch", "asshole",
	},
}

// DFANode 前缀树节点
type DFANode struct {
	Children map[rune]*DFANode
	IsEnd    bool
	Word     string
}

// DFATree 确定有穷自动机
type DFATree struct {
	Root *DFANode
}

func NewDFATree() *DFATree {
	return &DFATree{
		Root: &DFANode{
			Children: make(map[rune]*DFANode),
		},
	}
}

func (t *DFATree) AddWord(word string) {
	curr := t.Root
	runes := []rune(strings.ToLower(word))
	for _, r := range runes {
		if _, ok := curr.Children[r]; !ok {
			curr.Children[r] = &DFANode{
				Children: make(map[rune]*DFANode),
			}
		}
		curr = curr.Children[r]
	}
	curr.IsEnd = true
	curr.Word = word
}

// isNoiseRune 判断是否为跳跃噪音字符 (空格、常见中英文标点等)
func isNoiseRune(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '-', '_', '*', '.', ',', '!', '?', ':', ';', '~', '`', '#', '@', '$', '%', '^', '&', '+', '=', '/', '\\', '|':
		return true
	case '，', '。', '！', '？', '、', '：', '；', '“', '”', '‘', '’', '（', '）', '《', '》', '【', '】', '…', '·':
		return true
	}
	return false
}

// Search 检索文本是否命中敏感词
func (t *DFATree) Search(text string) (bool, string) {
	if text == "" {
		return false, ""
	}

	runes := []rune(strings.ToLower(text))
	n := len(runes)

	for i := 0; i < n; i++ {
		if isNoiseRune(runes[i]) {
			continue
		}

		curr := t.Root
		j := i
		matchedCount := 0

		for j < n {
			r := runes[j]

			// 若已匹配到至少一个前缀字符，遇到干扰噪音字符可跳过继续深入
			if isNoiseRune(r) {
				if matchedCount > 0 {
					j++
					continue
				} else {
					break
				}
			}

			if nextNode, ok := curr.Children[r]; ok {
				curr = nextNode
				matchedCount++
				if curr.IsEnd {
					return true, curr.Word
				}
				j++
			} else {
				break
			}
		}
	}

	return false, ""
}

var globalDFATree *DFATree

func init() {
	globalDFATree = NewDFATree()
	for _, list := range SensitiveCategories {
		for _, w := range list {
			w = strings.TrimSpace(w)
			if w != "" {
				globalDFATree.AddWord(w)
			}
		}
	}
}

// CheckSensitiveContent 检查文本中是否包含违规敏感词，返回是否存在及命中的词汇
func CheckSensitiveContent(text string) (bool, string) {
	return globalDFATree.Search(text)
}
