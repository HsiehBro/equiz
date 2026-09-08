# 智能刷题助手 - Go 后端服务 (Gin + GORM + PostgreSQL)

本目录为微信小程序「智能刷题助手」配套的高性能 Go 后端 API 服务，基于标准企业级分层架构开发。

---

## 1. 快速启动指南

### 第一步：配置虚拟机 PostgreSQL 连接
打开 `configs/config.yaml`（或复制 `configs/config.example.yaml`）：
```yaml
database:
  host: 192.168.1.100  # 替换为您的虚拟机实际 IP 地址
  port: 5432
  user: postgres
  password: your_password  # 替换为虚拟机 PostgreSQL 密码
  dbname: exam_prep        # 虚拟机中创建的数据库名
  sslmode: disable
```

> **提示**：首次连接时，后端会自动执行 GORM `AutoMigrate`，在 PostgreSQL 中自动创建 `users`, `question_banks`, `questions`, `user_records`, `user_errors`, `user_favorites` 6 张数据表。

---

### 第二步：一键注入官方题库与 100 道真实真题 (Seed)
在 `server` 目录下执行：
```bash
go run cmd/seed/main.go
```
执行后将自动注入：
1. **项目管理基础考试**：完整 100 道真题矩阵（包含单选题、多选题、解析、考点精析与各 Section 章节划分）。
2. **2023年护士执业资格考试**
3. **初级会计实务 - 核心考点**

---

### 第三步：启动 API 服务
在 `server` 目录下执行：
```bash
go run cmd/api/main.go
```
启动成功后默认监听在 `http://127.0.0.1:8080`：
- 服务探活地址：`http://127.0.0.1:8080/health`
- 接口基础路径：`http://127.0.0.1:8080/api/v1`

---

## 2. 微信小程序前端环境切换

在小程序根目录下的 `config.js` 中：
```javascript
const CONFIG = {
  // 后端服务地址 (可配置为 127.0.0.1:8080 或局域网 IP)
  API_BASE_URL: 'http://127.0.0.1:8080',

  // 设为 false: 开启全量真实后端与虚拟机 PostgreSQL 联调
  // 设为 true:  随时切回离线本地 Mock 数据模式进行独立 UI 测试
  USE_MOCK: false,
};
```

---

## 3. 核心 API 端点一览

| 模块 | 请求方式 | 路径 | 说明 |
| :--- | :--- | :--- | :--- |
| **探活** | `GET` | `/health` | 服务健康检查 |
| **认证** | `POST` | `/api/v1/auth/mock-login` | 开发者一键免密快速登录换发 JWT |
| | `POST` | `/api/v1/auth/wechat-login` | 微信标准 `code2Session` 登录 |
| | `GET` | `/api/v1/auth/profile` | 获取当前登录用户信息 |
| **题库** | `GET` | `/api/v1/banks` | 题库列表（含做题进度与动态统计） |
| | `GET` | `/api/v1/banks/:id` | 题库详情 |
| | `DELETE`| `/api/v1/banks/:id` | 移除用户自定义导入题库 |
| | `GET` | `/api/v1/banks/:id/questions` | 获取指定题库下的全部试题数据 |
| **刷题** | `POST` | `/api/v1/practice/submit-single` | 单题作答判定（自动记录对错与错题集） |
| | `POST` | `/api/v1/practice/submit-exam` | 整卷交卷评分与耗时统计 |
| **错题** | `GET` | `/api/v1/errors` | 获取用户当前未掌握错题列表 |
| | `POST` | `/api/v1/errors/:questionId/master` | 标记掌握（移出错题集） |
| **收藏** | `GET` | `/api/v1/favorites` | 获取用户收藏题目列表 |
| | `POST` | `/api/v1/favorites/toggle` | 切换收藏/取消收藏状态 |
| **导入** | `POST` | `/api/v1/import/preview` | 上传 Excel/CSV，服务端校验并返回统计预览 |
| | `POST` | `/api/v1/import/confirm` | 确认题库名称并批量持久化入库 |
| **笔记** | `GET` | `/api/v1/notes` | 获取用户个人学习笔记列表（支持按题库/公开私密筛选） |
| | `POST` | `/api/v1/notes` | 发布题目笔记或评论（指定 public 或 private） |
| | `PUT` | `/api/v1/notes/:id` | 修改笔记内容或可见权限 |
| | `DELETE`| `/api/v1/notes/:id` | 删除个人笔记 |
| | `POST` | `/api/v1/notes/:id/like` | 笔记/评论点赞与取消点赞切换 |
| | `GET` | `/api/v1/questions/:id/comments` | 获取题目公开精选评论列表（按赞数排序） |
| **规划** | `GET` | `/api/v1/plans` | 获取用户全部题库规划及动态进度数据 |
| | `GET` | `/api/v1/plans/active` | 获取当前生效的主目标规划（用于首页与个人中心卡片） |
| | `POST` | `/api/v1/plans` | 创建或保存题库学习规划与每日目标（可设置顶） |
| | `DELETE`| `/api/v1/plans/:id` | 删除指定学习规划 |
| **分类** | `GET` | `/api/v1/categories` | 获取可用题库分类列表（含动态题库数与VIP标识） |
| | `POST` | `/api/v1/categories` | 新建题库分类（仅系统管理员） |
| | `PUT` | `/api/v1/categories/:id` | 编辑题库分类名称/图标/排序（仅管理员） |
| | `DELETE`| `/api/v1/categories/:id` | 删除题库分类（关联题库自动重定向归入「综合」） |
| | `PUT` | `/api/v1/banks/:id/category` | 修改题库所属分类（创建者与管理员可用） |

