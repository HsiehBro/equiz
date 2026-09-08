# 智能刷题助手 (Smart Quiz Assistant)

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/WeChat-MiniProgram-07C160?style=flat-square&logo=wechat" alt="WeChat MiniProgram" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791?style=flat-square&logo=postgresql" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Kubernetes-Cloud%20Native-326CE5?style=flat-square&logo=kubernetes" alt="Kubernetes" />
  <img src="https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square" alt="License" />
</p>

> 基于 **微信原生小程序** + **Go (Gin + GORM)** 企业级分层架构打造的现代化考研/考证/职业考试智能刷题助手。支持双镜像解耦容器化与 Kubernetes 云原生一键部署，具备离线本地 Mock 与云端全链路联调双重开发能力。

---

## 目录
- [功能亮点](#-功能亮点)
- [系统架构与技术栈](#-系统架构与技术栈)
- [项目目录结构](#-项目目录结构)
- [开发者本地调试指南](#-开发者本地调试指南)
- [生产与开源部署指南](#-生产与开源部署指南)
  - [方案 A：Kubernetes 云原生一键部署 (推荐)](#方案-akubernetes-云原生一键部署-推荐)
  - [方案 B：Docker / 源码直接运行](#方案-bdocker--源码直接运行)
- [环境变量与配置说明](#-环境变量与配置说明)
- [开源协议](#-开源协议)

---

## 🌟 功能亮点

### 1. 核心刷题与智能判定引擎
- **单题自适应练习**：支持单选、多选与解析展示，提交即刻判定并返回考点归纳。
- **全真模考系统**：模拟考试倒计时、答题卡快速跳转、交卷评分诊断报告与耗时统计。
- **智能错题闭环**：答错题目自动沉淀至错题集，支持针对性攻克与“标记掌握”一键移出，支持生成错题集 PDF。
- **试题收藏夹**：收藏高频重点题，支持多题库快捷筛选。

### 2. 学习规划与目标督学
- **个性化每日目标**：支持针对特定题库制定每日答题数量，动态计算预计通关天数。
- **连续打卡激励**：自动识别每日刷题进度，打卡状态置顶于首页与个人中心。

### 3. 互动社区与试题笔记
- **题目笔记与精选评论**：学员可记录个人私密笔记或发布公开笔记。
- **点赞互动流**：精选评论点赞防重，热度排序聚合考友实战解法。

### 4. VIP 会员与社交裂变
- **会员权益分级**：支持连续包月/包季/包年及终身会员体系，精确控制 VIP 题库访问权限。
- **邀请奖励闭环**：生成个人专属推广码，被邀请人首购绑定后自动为邀请者发放 VIP 体验时长奖励。

### 5. 题库导入与内容合规
- **Excel/CSV 批量导入**：服务端自动校验格式，支持先预览统计后持久化入库。
- **审核流与安全过滤**：私有题库支持一键申请公开，接入微信安全接口与敏感词过滤，管理员专属审批工作流。

---

## 🛠 系统架构与技术栈

- **前端 (Frontend)**：微信小程序原生框架、Tailwind CSS (`weapp-tailwindcss`)、Gulp、PostCSS
- **后端 (Backend)**：Go 1.24、Gin Web Framework、GORM ORM、Viper 配置中心、JWT 鉴权中间件
- **数据库 (Database)**：PostgreSQL 16 (支持 JSONB 高性能存储与时区自适应)
- **部署与运维 (DevOps)**：Docker 多阶段轻量镜像 (Alpine ~25MB)、Kubernetes (StatefulSet/Deployment/Ingress/Job)

---

## 📂 项目目录结构

```text
├── pages/                  # 微信小程序各页面源码 (首页、答题、模考、错题、规划、会员等)
├── utils/                  # 小程序工具库 (网络请求封装、打卡统计算法等)
├── config.js               # 小程序全局环境切换配置 (Mock模式 / 真实后端IP)
├── DESIGN.md               # 官方设计系统与 UI/UX 规范定义
├── deploy/
│   └── kubernetes/         # Kubernetes 全套生产级编排清单
│       ├── 00-namespace.yaml
│       ├── 01-postgres-secret.yaml
│       ├── 02-postgres-pvc.yaml
│       ├── 03-postgres-deployment.yaml
│       ├── 04-server-configmap.yaml
│       ├── 05-server-secret.yaml
│       ├── 06-server-deployment.yaml
│       ├── 07-server-ingress.yaml
│       ├── 08-seed-job.yaml
│       └── kustomization.yaml
├── server/                 # Go 后端工程根目录
│   ├── cmd/
│   │   ├── api/            # API 服务启动入口
│   │   ├── seed/           # 100 道真实考题初始化注入工具
│   │   └── e2e/            # Go 原生 20 项端到端全链路自动化集成测试套件
│   ├── configs/            # 后端默认配置文件 (config.yaml)
│   ├── internal/           # 企业级分层业务包 (handler, service, model, router, middleware)
│   ├── Dockerfile          # 生产级多阶段精简构建镜像文件
│   └── .dockerignore
└── DEPLOYMENT.md           # 详细云原生部署与运维手册
```

---

## 💻 开发者本地调试指南

### 1. 前端独立纯 Mock 调试 (无需后端与数据库)
若只需调整小程序 UI 交互，无需启动 Go 服务与数据库：
1. 打开根目录下 `config.js`，设置：
   ```javascript
   const CONFIG = {
     USE_MOCK: true,               // 开启前端离线 Mock 模式
     ENABLE_DEV_MOCK_LOGIN: true,  // 开启一键身份切换弹窗
   };
   module.exports = CONFIG;
   ```
2. 使用 **微信开发者工具** 导入本项目根目录，点击即可进行全页面免密体验与走查。

---

### 2. 本地前后端全链路联调

#### 第一步：启动 PostgreSQL
确保本地或虚拟机运行 PostgreSQL（默认端口 `5432`，库名 `equiz_db`）。

#### 第二步：启动 Go 后端并注入初始题库
进入 `server` 目录：
```bash
# 1. 灌入 100 道官方真实考题与章节矩阵
go run cmd/seed/main.go

# 2. 启动 API 服务 (监听 127.0.0.1:8080)
go run cmd/api/main.go
```

#### 第三步：执行本地全链路自动化测试
项目提供开箱即用的原生集成测试工具，一键验证 20 项业务全流程与 401/403/400 边界：
```bash
go run cmd/e2e/main.go -url=http://127.0.0.1:8080
```

#### 第四步：切换小程序为真实后端模式
修改根目录下 `config.js`：
```javascript
const CONFIG = {
  API_BASE_URL: 'http://127.0.0.1:8080', // 或局域网/虚拟机 IP
  USE_MOCK: false                        // 直连后端数据库
};
```
在微信开发者工具右上角勾选 **「详情 -> 本地设置 -> 不校验合法域名、web-view (业务域名)、TLS 版本以及 HTTPS 证书」** 即可畅通联调。

---

## 🚀 生产与开源部署指南

本项目采用 **后端微服务与 PostgreSQL 双镜像完全解耦** 架构，并已通过 12-Factor 标准化改造。

### 方案 A：Kubernetes 云原生一键部署 (推荐)

#### 1. 构建并推送后端镜像
```bash
# 构建紧凑安全的 Alpine 后端镜像
docker build -t your-registry/exam-server:latest -f server/Dockerfile server/
docker push your-registry/exam-server:latest
```

#### 2. 一键应用 K8s 清单
```bash
# 使用 Kustomize 一键部署 Namespace、PVC、PostgreSQL、Go API 与 Service
kubectl apply -k deploy/kubernetes/
```

#### 3. 运行题库初始化 Job
```bash
# 执行 Kubernetes Job 自动完成 100 题入库
kubectl apply -f deploy/kubernetes/08-seed-job.yaml
```

#### 4. 配置 HTTPS 与 Ingress 域名
编辑 `deploy/kubernetes/07-server-ingress.yaml`，填入您的已备案域名与 TLS 证书 Secret，执行：
```bash
kubectl apply -f deploy/kubernetes/07-server-ingress.yaml
```

---

### 方案 B：Docker / 源码直接运行

若使用轻量云主机，可直接编译 Linux 二进制运行：
```bash
cd server
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api-server ./cmd/api
CONFIG_PATH=/etc/exam-server/config.yaml ./api-server
```

---

## ⚙️ 环境变量与配置说明

系统支持通过配置文件（`config.yaml`）或标准容器环境变量无缝注入（环境变量优先级高于配置文件）：

| 环境变量名 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `SERVER_PORT` | `8080` | API 监听端口 |
| `SERVER_MODE` | `debug` | 运行模式 (`debug` / `release`) |
| `DATABASE_HOST` | `127.0.0.1` | PostgreSQL 服务名或 IP |
| `DATABASE_PORT` | `5432` | 数据库端口 |
| `DATABASE_USER` | `postgres` | 数据库连接用户名 |
| `DATABASE_PASSWORD`| - | 数据库连接密码 |
| `DATABASE_DBNAME` | `exam_prep` | 业务数据库名称 |
| `JWT_SECRET` | 随机默认值 | 用户 JWT 鉴权密钥 (生产务必重置) |
| `WECHAT_APP_ID` | - | 微信小程序 AppID |
| `WECHAT_APP_SECRET`| - | 微信小程序 AppSecret |

---

## 📄 开源协议

本项目采用 [MIT License](LICENSE) 授权，欢迎社区提交 Issue 与 PR 共同完善！
