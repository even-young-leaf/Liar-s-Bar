# Liar's Bar 项目审查报告

## 📋 审查日期：2026-07-06

---

## ✅ 项目优点

### 1. 架构设计
- ✅ 前后端分离架构清晰
- ✅ 微服务化设计（Backend、Frontend、AI Service、Gateway）
- ✅ 使用 Docker Compose 编排，易于部署
- ✅ 后端采用分层架构（Controller → Service → Repository → Model）
- ✅ WebSocket 实现实时通信

### 2. 技术栈选择
- ✅ Go 1.21 + Gin 框架（性能好）
- ✅ Vue 3 + Vite（现代前端）
- ✅ MySQL 8.0 + Redis 7（稳定可靠）
- ✅ GORM ORM（自动迁移）
- ✅ JWT 认证

### 3. 代码质量
- ✅ Go 代码结构规范，符合标准项目布局
- ✅ 前端使用 Pinia 状态管理
- ✅ 有单元测试（engine_test.go）
- ✅ 有集成测试脚本（mobile_smoke_test.py）

### 4. 文档
- ✅ API 文档完整（api.md, mobile-api.md）
- ✅ 架构文档（architecture.md）
- ✅ 数据库文档（database.md）
- ✅ 游戏规则文档（game_rules.md）
- ✅ 环境变量示例文件（.env.example）

---

## ⚠️ 需要改进的地方

### 1. 安全问题
- ⚠️ **docker-compose.yml 中硬编码敏感信息**
  - MySQL 密码直接写在 docker-compose.yml 中
  - JWT_SECRET 暴露在配置文件
  - 应该使用 `.env` 文件或 Docker secrets
- ⚠️ **CORS 配置可能过于宽松**（需检查 backend/main.go）
- ⚠️ **没有 API 速率限制**（容易被 DDoS）
- ⚠️ **WebSocket 连接缺少身份验证检查**（需审查）

### 2. 数据库设计
- ⚠️ **缺少索引优化**
  - room_players 表的 room_id, user_id 没有联合索引
  - game_actions 表的 game_id 应该加索引（查询频繁）
- ⚠️ **缺少外键约束**
  - 当前只用 GORM 关联，数据库层面没有强制外键
- ⚠️ **缺少软删除**
  - 重要数据（User, Game）删除后无法恢复
  - 建议加 deleted_at 字段

### 3. 错误处理
- ⚠️ **没有统一的错误码规范**
- ⚠️ **缺少日志聚合方案**（生产环境难以调试）
- ⚠️ **panic 恢复机制不完善**

### 4. 性能问题
- ⚠️ **WebSocket 连接管理没有做负载均衡准备**
  - 当前设计只支持单机部署
  - 需要 Redis Pub/Sub 或消息队列做跨实例通信
- ⚠️ **游戏状态全在内存，重启丢失**
- ⚠️ **N+1 查询问题**（需检查 service 层的查询逻辑）


### 5. Git 项目规范
- ⚠️ **缺少 README.md**（项目入口文档）
- ⚠️ **缺少 LICENSE 文件**
- ⚠️ **缺少 CONTRIBUTING.md**（贡献指南）
- ⚠️ **commit 信息不规范**（需要 Conventional Commits）
- ⚠️ **没有 Git hooks**（pre-commit 检查）
- ⚠️ **没有 CI/CD 配置**（GitHub Actions / GitLab CI）

### 6. 测试覆盖率
- ⚠️ **单元测试覆盖率低**
  - 只有 engine_test.go，缺少 service/repository 层测试
- ⚠️ **缺少集成测试自动化**
  - mobile_smoke_test.py 是手动脚本
- ⚠️ **没有 E2E 测试**
- ⚠️ **没有测试覆盖率报告**

### 7. 代码规范
- ⚠️ **缺少 Go linter 配置**（golangci-lint）
- ⚠️ **缺少前端 ESLint/Prettier 配置**
- ⚠️ **注释不足**（复杂逻辑缺少说明）
- ⚠️ **magic number 过多**（硬编码数字）


## 🔧 具体改进建议

### 1. 安全加固（优先级：🔴 高）

#### 1.1 敏感信息管理
```bash
# 创建 .env 文件（已有 .env.example，需实际使用）
cp .env.example .env

# docker-compose.yml 改为使用环境变量
environment:
  MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
  MYSQL_PASSWORD: ${MYSQL_PASSWORD}
  JWT_SECRET: ${JWT_SECRET}
```

#### 1.2 添加 API 速率限制
```go
// backend/internal/middleware/rate_limit.go
import "github.com/ulule/limiter/v3"

func RateLimit() gin.HandlerFunc {
    rate := limiter.Rate{
        Period: 1 * time.Minute,
        Limit:  100, // 每分钟100次
    }
    // ... 实现
}
```

#### 1.3 WebSocket 身份验证
```go
// 在 WebSocket 握手时验证 JWT token
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
    token := c.Query("token")
    if !h.authService.ValidateToken(token) {
        c.JSON(401, gin.H{"error": "unauthorized"})
        return
    }
    // ...
}
```


### 2. 数据库优化（优先级：🟡 中）

#### 2.1 添加索引
```go
// backend/internal/model/room.go
type RoomPlayer struct {
    ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
    RoomID    uint      `gorm:"index:idx_room_user,priority:1" json:"room_id"`
    UserID    uint      `gorm:"index:idx_room_user,priority:2" json:"user_id"`
    IsAI      bool      `gorm:"default:false" json:"is_ai"`
    SeatIndex int       `json:"seat_index"`
    JoinTime  time.Time `json:"join_time"`
}

// backend/internal/model/game.go
type GameAction struct {
    ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
    GameID   uint   `gorm:"index" json:"game_id"` // 添加索引
    // ...
}
```

#### 2.2 添加软删除
```go
// backend/internal/model/user.go
import "gorm.io/gorm"

type User struct {
    // ... 原有字段
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
```

#### 2.3 添加外键约束
```go
// backend/internal/model/room.go
type Room struct {
    // ...
    HostUserID uint  `json:"host_user_id"`
    Host       User  `gorm:"foreignKey:HostUserID;constraint:OnDelete:SET NULL"`
}
```


### 3. 错误处理与监控（优先级：🟡 中）

#### 3.1 统一错误码
```go
// backend/internal/apierror/codes.go
package apierror

const (
    ErrCodeInvalidRequest = 1001
    ErrCodeUnauthorized   = 1002
    ErrCodeRoomFull       = 2001
    ErrCodeGameNotFound   = 2002
    // ...
)

type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}
```

#### 3.2 添加日志框架
```go
// 使用 zap 或 logrus
import "go.uber.org/zap"

func InitLogger() (*zap.Logger, error) {
    cfg := zap.NewProductionConfig()
    cfg.OutputPaths = []string{"stdout", "logs/app.log"}
    return cfg.Build()
}
```

#### 3.3 Panic 恢复中间件
```go
// backend/internal/middleware/recovery.go
func Recovery(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                logger.Error("panic recovered", zap.Any("error", err))
                c.JSON(500, gin.H{"error": "internal server error"})
            }
        }()
        c.Next()
    }
}
```


### 4. 性能优化（优先级：🟡 中）

#### 4.1 WebSocket 分布式支持
```go
// 使用 Redis Pub/Sub 做消息广播
type RedisEventBus struct {
    client *redis.Client
}

func (r *RedisEventBus) Publish(channel string, message interface{}) {
    data, _ := json.Marshal(message)
    r.client.Publish(context.Background(), channel, data)
}

func (r *RedisEventBus) Subscribe(channel string, handler func([]byte)) {
    pubsub := r.client.Subscribe(context.Background(), channel)
    ch := pubsub.Channel()
    for msg := range ch {
        handler([]byte(msg.Payload))
    }
}
```

#### 4.2 游戏状态持久化
```go
// 关键游戏状态存入 Redis，重启可恢复
type GameStateCache struct {
    redis *redis.Client
}

func (g *GameStateCache) SaveGameState(gameID uint, state *GameState) error {
    data, _ := json.Marshal(state)
    return g.redis.Set(ctx, fmt.Sprintf("game:%d", gameID), data, 24*time.Hour).Err()
}
```

#### 4.3 预加载优化
```go
// 使用 GORM Preload 避免 N+1 查询
func (r *RoomRepository) GetRoomWithPlayers(roomID uint) (*Room, error) {
    var room Room
    err := r.db.Preload("Players.User").First(&room, roomID).Error
    return &room, err
}
```


### 5. Git 项目规范化（优先级：🔴 高）

#### 5.1 创建 README.md
```markdown
# Liar's Bar - 在线桌游平台

## 项目简介
基于 Go + Vue3 的实时多人在线狼人杀风格桌游

## 技术栈
- Backend: Go 1.21 + Gin + GORM
- Frontend: Vue 3 + Vite + Pinia
- Database: MySQL 8.0 + Redis 7
- AI Service: Python FastAPI
- Deployment: Docker Compose

## 快速开始
1. 克隆仓库
2. 配置 .env 文件
3. 运行 `docker-compose up -d`
4. 访问 http://localhost:8081

## API 文档
见 docs/api.md

## 架构文档
见 docs/architecture.md
```

#### 5.2 添加 LICENSE
```bash
# 选择合适的开源协议，例如 MIT License
# 如果是商业项目，添加 proprietary license
```

#### 5.3 Conventional Commits
```bash
# 安装 commitizen
npm install -g commitizen cz-conventional-changelog

# .czrc 配置
{
  "path": "cz-conventional-changelog"
}

# Commit 格式
feat: 添加房间匹配功能
fix: 修复 WebSocket 断线重连问题
docs: 更新 API 文档
refactor: 重构游戏引擎代码
test: 添加用户服务单元测试
```


#### 5.4 添加 Git Hooks
```bash
# .husky/pre-commit
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

# Go 代码格式化
cd backend && gofmt -w . && go vet ./...

# 前端 lint
cd ../frontend && npm run lint

# 运行测试
cd ../backend && go test ./...
```

#### 5.5 GitHub Actions CI/CD
```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: cd backend && go test -v -cover ./...
  
  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: cd frontend && npm ci && npm run test
  
  docker-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: docker-compose build
```


### 6. 测试覆盖率提升（优先级：🟡 中）

#### 6.1 添加 Service 层单元测试
```go
// backend/internal/service/room_service_test.go
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockRoomRepository struct {
    mock.Mock
}

func TestCreateRoom(t *testing.T) {
    mockRepo := new(MockRoomRepository)
    service := NewRoomService(mockRepo)
    
    // 测试创建房间逻辑
    mockRepo.On("Create", mock.Anything).Return(nil)
    err := service.CreateRoom(1, "测试房间", 4)
    assert.NoError(t, err)
}
```

#### 6.2 集成测试自动化
```go
// backend/internal/test/integration_test.go
func TestRoomFlow(t *testing.T) {
    // 启动测试服务器
    router := setupTestRouter()
    
    // 1. 创建房间
    w := performRequest(router, "POST", "/api/rooms", createRoomPayload)
    assert.Equal(t, 200, w.Code)
    
    // 2. 加入房间
    // 3. 开始游戏
    // ...
}
```

#### 6.3 测试覆盖率报告
```bash
# Makefile
test-coverage:
    cd backend && go test -coverprofile=coverage.out ./...
    cd backend && go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: backend/coverage.html"
```


### 7. 代码规范化（优先级：🟢 低）

#### 7.1 Go Linter 配置
```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - structcheck
    - varcheck
    - ineffassign
    - deadcode

run:
  timeout: 5m
  tests: true

issues:
  exclude-use-default: false
```

#### 7.2 前端 ESLint + Prettier
```javascript
// frontend/.eslintrc.js
module.exports = {
  extends: [
    'plugin:vue/vue3-recommended',
    '@vue/eslint-config-typescript',
    '@vue/eslint-config-prettier'
  ],
  rules: {
    'vue/multi-word-component-names': 'off',
    'no-console': process.env.NODE_ENV === 'production' ? 'warn' : 'off'
  }
}

// frontend/.prettierrc
{
  "semi": false,
  "singleQuote": true,
  "trailingComma": "none"
}
```

#### 7.3 消除 Magic Number
```go
// backend/internal/constants/game.go
package constants

const (
    DefaultMaxPlayers     = 4
    DefaultEloRating      = 1000
    MaxCardsInHand        = 6
    MinPlayersToStart     = 2
    GameTimeoutMinutes    = 30
    WebSocketPingInterval = 30 * time.Second
)
```


## 🏗️ 重构建议

### 是否需要大规模重构？
**结论：❌ 不需要**

当前项目架构合理，代码结构清晰，**不建议大规模重构**。只需要**渐进式优化**。

### 需要重构的局部模块

#### 1. WebSocket 连接管理模块
**当前问题：**
- 连接管理逻辑分散
- 没有统一的消息格式
- 缺少心跳检测

**建议重构：**
```go
// backend/internal/websocket/manager.go
type ConnectionManager struct {
    connections map[uint]*Connection // userID -> Connection
    broadcast   chan *Message
    register    chan *Connection
    unregister  chan *Connection
    eventBus    *RedisEventBus // 新增：支持分布式
}

type Message struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
    RoomID  uint        `json:"room_id,omitempty"`
}
```

#### 2. 游戏状态机
**当前问题：**
- 状态转换逻辑复杂
- 缺少状态验证

**建议重构：**
```go
// backend/internal/engine/state_machine.go
type GameState int

const (
    StateWaiting GameState = iota
    StateDealing
    StatePlaying
    StateChallenging
    StateRevealing
    StateFinished
)

type StateMachine struct {
    currentState GameState
    allowedTransitions map[GameState][]GameState
}

func (sm *StateMachine) CanTransition(to GameState) bool {
    allowed := sm.allowedTransitions[sm.currentState]
    for _, s := range allowed {
        if s == to {
            return true
        }
    }
    return false
}
```


#### 3. 配置管理
**当前问题：**
- 配置分散在多处
- 缺少配置验证

**建议重构：**
```go
// backend/internal/config/config.go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    JWT      JWTConfig
    Game     GameConfig
}

type GameConfig struct {
    MaxPlayers        int           `env:"GAME_MAX_PLAYERS" envDefault:"4"`
    MinPlayers        int           `env:"GAME_MIN_PLAYERS" envDefault:"2"`
    TurnTimeout       time.Duration `env:"GAME_TURN_TIMEOUT" envDefault:"30s"`
    RoomExpireTime    time.Duration `env:"ROOM_EXPIRE_TIME" envDefault:"24h"`
}

func (c *Config) Validate() error {
    if c.Game.MaxPlayers < c.Game.MinPlayers {
        return errors.New("max_players must >= min_players")
    }
    // ...
}
```

### 不需要重构的部分
✅ **Model 层**：数据模型设计合理，字段完整  
✅ **Controller 层**：路由结构清晰  
✅ **Repository 层**：数据访问封装良好  
✅ **前端组件**：Vue 3 组件化合理  


## 📊 优先级排序与实施路线

### 第一阶段：安全与规范化（1-2 周）🔴 高优先级

#### 必须立即完成
1. **敏感信息迁移到 .env**
   - 从 docker-compose.yml 移除硬编码密码
   - 使用环境变量注入
   - 预计时间：2 小时

2. **创建 README.md**
   - 项目说明、快速开始、API 文档链接
   - 预计时间：3 小时

3. **添加 .gitignore 完善**
   - 确保 .env 文件被忽略
   - 添加日志文件、临时文件规则
   - 预计时间：30 分钟

4. **API 速率限制**
   - 防止 DDoS 攻击
   - 使用 gin 中间件实现
   - 预计时间：4 小时

5. **WebSocket 身份验证**
   - 握手时验证 JWT token
   - 预计时间：3 小时

### 第二阶段：数据库优化（1 周）🟡 中优先级

1. **添加数据库索引**
   - room_players 联合索引
   - game_actions 索引
   - 预计时间：2 小时

2. **添加软删除**
   - User、Game、Room 表
   - 预计时间：3 小时

3. **添加外键约束**
   - 确保数据一致性
   - 预计时间：4 小时


### 第三阶段：测试与监控（2 周）🟡 中优先级

1. **统一错误码系统**
   - 创建 apierror 包
   - 定义错误码常量
   - 预计时间：1 天

2. **添加日志框架**
   - 使用 zap 或 logrus
   - 配置日志级别和输出
   - 预计时间：1 天

3. **Service 层单元测试**
   - 覆盖核心业务逻辑
   - 使用 testify/mock
   - 预计时间：3 天

4. **集成测试自动化**
   - 完整的 API 流程测试
   - 预计时间：2 天

5. **测试覆盖率报告**
   - 配置 Makefile
   - 目标：60% 以上
   - 预计时间：1 天

### 第四阶段：性能与扩展性（2-3 周）🟡 中优先级

1. **Redis Pub/Sub 消息广播**
   - 支持多实例部署
   - WebSocket 跨节点通信
   - 预计时间：1 周

2. **游戏状态持久化**
   - 关键状态存 Redis
   - 重启可恢复
   - 预计时间：3 天

3. **N+1 查询优化**
   - 使用 GORM Preload
   - 检查所有 service 层查询
   - 预计时间：2 天


### 第五阶段：CI/CD 与代码规范（1 周）🟢 低优先级

1. **GitHub Actions CI/CD**
   - 自动化测试
   - Docker 镜像构建
   - 预计时间：2 天

2. **Git Hooks 配置**
   - pre-commit 格式检查
   - commit-msg 规范验证
   - 预计时间：1 天

3. **Linter 配置**
   - golangci-lint
   - ESLint + Prettier
   - 预计时间：1 天

4. **常量提取**
   - 消除 magic number
   - 创建 constants 包
   - 预计时间：2 天

---

## 📝 实施检查清单

### 立即执行（本周内）
- [ ] 将敏感信息迁移到 .env 文件
- [ ] 创建 README.md
- [ ] 添加 API 速率限制中间件
- [ ] WebSocket 添加 JWT 验证
- [ ] 完善 .gitignore

### 短期目标（1 个月内）
- [ ] 数据库添加索引优化
- [ ] 实现软删除机制
- [ ] 添加外键约束
- [ ] 统一错误码系统
- [ ] 集成 zap 日志框架
- [ ] Service 层单元测试（覆盖率 > 60%）

### 中期目标（2-3 个月内）
- [ ] Redis Pub/Sub 分布式消息
- [ ] 游戏状态 Redis 持久化
- [ ] CI/CD 流水线配置
- [ ] Git Hooks 规范化
- [ ] 代码 Linter 配置

### 长期优化（持续进行）
- [ ] 代码注释完善
- [ ] API 文档自动生成（Swagger）
- [ ] 性能监控（Prometheus + Grafana）
- [ ] 错误追踪（Sentry）
- [ ] E2E 测试

---

## 🎯 总结与建议

### 项目整体评价
**评分：7.5/10** ⭐⭐⭐⭐

**优点：**
- ✅ 架构设计合理，微服务化清晰
- ✅ 技术栈现代化，符合主流趋势
- ✅ 文档相对完整
- ✅ Docker 化部署方便

**主要问题：**
- ⚠️ 安全性不足（硬编码密码、缺少速率限制）
- ⚠️ 测试覆盖率低
- ⚠️ 缺少 Git 项目规范（README、CI/CD）
- ⚠️ 性能优化空间大（单机部署限制）

### 核心建议
1. **不需要大规模重构**，当前架构可持续发展
2. **优先解决安全问题**，这是生产环境的前提
3. **逐步提升测试覆盖率**，保证代码质量
4. **建立 Git 规范**，方便团队协作
5. **预留扩展性设计**（Redis Pub/Sub），为未来水平扩展做准备

### 下一步行动
1. 先完成"立即执行"清单（1 周内）
2. 按优先级依次推进
3. 每个阶段完成后进行代码审查
4. 保持小步迭代，避免一次性改动过大

---

**审查人：AI Assistant**  
**审查日期：2026-07-06**  
**项目状态：可继续开发，需优化**
