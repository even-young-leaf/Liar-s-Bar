# Liar's Bar - 服务端

基于 Go + WebSocket 的多人在线卡牌游戏服务端，支持实时对战、AI玩家、匹配系统等功能。

## 技术栈

- **语言**: Go 1.21
- **Web框架**: Gin
- **WebSocket**: Gorilla WebSocket
- **数据库**: MySQL 8.0
- **缓存**: Redis
- **容器化**: Docker + Docker Compose

## 项目架构

```
backend/
├── cmd/server/          # 应用入口
│   └── main.go
├── internal/
│   ├── config/          # 配置管理
│   ├── controller/      # HTTP控制器
│   ├── database/        # 数据库初始化
│   ├── game/            # 游戏核心逻辑
│   │   └── engine.go    # 游戏引擎、规则、状态机
│   ├── logger/          # 日志工具
│   ├── match/           # 匹配系统
│   ├── middleware/      # 中间件（CORS、JWT认证）
│   ├── model/           # 数据模型
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   ├── utils/           # 工具函数（Redis等）
│   └── websocket/       # WebSocket实时通信
│       ├── hub.go       # 连接管理中心
│       └── room.go      # 游戏房间逻辑（含AI策略）
└── go.mod
deploy/
├── Dockerfile.backend
├── Dockerfile.frontend
├── Dockerfile.ai
└── nginx.conf
```

## 核心功能

### 1. 用户系统
- 用户注册/登录（JWT认证）
- 用户资料管理
- ELO积分系统
- 游戏统计数据

### 2. 匹配系统
- 快速匹配（支持1~4个真人 + AI补位至4人）
- 优先凑齐4个真人，超时后自动填充AI玩家
- 匹配超时处理（默认10秒）
- 角色选择（4个角色）

### 3. 游戏系统
- 实时卡牌游戏逻辑
- 多轮游戏支持
- 出牌、质疑、惩罚机制
- AI玩家决策系统
- 角色技能系统

### 4. 房间管理
- 房间创建/加入/离开
- 实时状态同步
- 自动清理机制：
  - 即时清理：所有真人玩家离开时
  - 定时清理：每20分钟清理超时房间
- WebSocket断连自动处理

### 5. WebSocket通信
- 实时消息推送
- 心跳检测（Ping 54秒/次，Pong 60秒超时，写入10秒超时）
- 断线重连支持（通过 RECONNECT 事件恢复游戏状态）
- 玩家在线状态管理（断连后标记 IsOnline=false）

## 快速开始

### 环境要求

- Docker >= 20.10
- Docker Compose >= 2.0

### 1. 克隆项目

```bash
git clone https://github.com/even-young-leaf/Liar-s-Bar.git
cd Liar-s-Bar
```

### 2. 启动服务

```bash
docker compose up -d
```

服务将在以下端口启动：
- **后端API**: http://localhost:8082
- **MySQL**: localhost:3307
- **Redis**: localhost:6379

### 3. 查看日志

```bash
# 查看所有服务日志
docker compose logs -f

# 只查看后端日志
docker compose logs -f backend
```

### 4. 停止服务

```bash
docker compose down
```

## API文档

### 认证相关

#### 注册
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "test_user",
  "password": "password123",
  "nickname": "测试用户"
}
```

#### 登录
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "test_user",
  "password": "password123"
}

Response:
{
  "code": 0,
  "token": "eyJhbGc...",
  "user": {
    "id": 1,
    "nickname": "测试用户",
    "username": "test_user"
  }
}
```

### 用户相关

所有用户接口需要在Header中携带JWT Token：
```
Authorization: Bearer <token>
```

#### 获取个人资料
```http
GET /api/v1/user/profile
```

#### 获取用户状态
```http
GET /api/v1/user/status
```

#### 更新个人资料
```http
PUT /api/v1/user/profile
Content-Type: application/json

{
  "nickname": "新昵称",
  "avatar_url": "https://example.com/avatar.png"
}
```

#### 获取游戏统计
```http
GET /api/v1/user/stats
```

### 匹配相关

#### 开始匹配
```http
POST /api/v1/match/start
Content-Type: application/json

{
  "character_id": "scubby"  # 可选: scubby, foxy, bristle, tor
}
```

#### 取消匹配
```http
POST /api/v1/match/cancel
```

#### 查询匹配状态
```http
GET /api/v1/match/status
```

### 房间相关

#### 获取房间列表
```http
GET /api/v1/rooms
```

#### 获取房间详情
```http
GET /api/v1/rooms/:id
```

#### 创建房间
```http
POST /api/v1/rooms
Content-Type: application/json

{
  "name": "我的房间"
}
```

#### 加入房间
```http
POST /api/v1/rooms/:id/join
```

#### 离开房间
```http
POST /api/v1/rooms/:id/leave
```

### 大厅与角色

#### 获取大厅信息
```http
GET /api/v1/lobby
```
返回在线人数、匹配队列长度、活跃房间列表。

#### 获取角色列表
```http
GET /api/v1/characters
```

### 游戏历史

#### 获取游戏历史
```http
GET /api/v1/history
```

#### 获取游戏详情
```http
GET /api/v1/games/:id
```

### WebSocket连接

```
ws://localhost:8082/ws?token=<jwt_token>
```

#### 加入房间
```json
{
  "type": "PLAYER_JOIN",
  "payload": {
    "room_id": 123
  }
}
```

#### 准备
```json
{
  "type": "PLAYER_READY"
}
```

#### 出牌
```json
{
  "type": "PLAY_CARD",
  "payload": {
    "card_ids": [0, 1, 2],
    "claim": "A"
  }
}
```

#### 质疑
```json
{
  "type": "CHALLENGE"
}
```

#### Pass（没有手牌时）
```json
{
  "type": "PASS"
}
```

## 游戏规则

### 基础规则
1. 4名玩家，每人初始6张手牌（A/K/Q/J各6张，共24张）
2. 每轮有指定的目标牌（按顺序：A → K → Q → J → A...）
3. 玩家必须声称出目标牌，但可以说谎
4. 其他玩家可以质疑，质疑成功/失败者接受惩罚
5. 惩罚：俄罗斯轮盘（随机生死）
6. 最后存活的玩家获胜

### 角色技能
- **Scubby**: 拥有万能牌（Wild Card），可当任意牌使用
- **Foxy**: 可查看一名玩家的手牌（每局一次）
- **Bristle**: 可质疑两次（其他角色只能质疑一次）
- **Tor**: 50%概率减少一次惩罚，30%概率免疫一次死亡

## 环境变量

可通过环境变量覆盖默认配置：

```bash
# 数据库配置
DB_HOST=mysql
DB_PORT=3306
DB_USER=liarsbar
DB_PASSWORD=liarsbar123
DB_NAME=liars_bar

# Redis配置
REDIS_ADDR=redis:6379
REDIS_PASSWORD=

# JWT配置
JWT_SECRET=liars-bar-secret-key-2024

# 服务器配置
SERVER_PORT=8080

# AI服务配置
AI_SERVICE_URL=http://ai-service:8000
AI_ENABLED=true
```

## 数据库管理

### 连接数据库
```bash
docker compose exec mysql mysql -uliarsbar -pliarsbar123 liars_bar
```

### 查看房间状态
```sql
SELECT id, room_status, current_players, created_at FROM rooms;
```

### 查看用户统计
```sql
SELECT username, nickname, elo_rating, total_games, total_wins FROM users ORDER BY elo_rating DESC LIMIT 10;
```

### 清理超时房间
```sql
DELETE FROM rooms WHERE created_at < DATE_SUB(NOW(), INTERVAL 20 MINUTE);
```

## 开发调试

### 本地编译
```bash
cd backend
go build -o server ./cmd/server/
./server
```

### 运行测试
```bash
cd backend
go test ./...
```

### 查看日志
```bash
# 实时查看后端日志
docker compose logs -f backend

# 查看最近100行
docker compose logs backend --tail=100

# 筛选特定关键词
docker compose logs backend | grep "ERROR"
```

### 重启服务
```bash
# 重启后端
docker compose restart backend

# 重新编译并重启
docker compose build backend && docker compose restart backend
```

## 故障排除

### 1. 数据库连接失败
```bash
# 检查MySQL是否启动
docker compose ps mysql

# 查看MySQL日志
docker compose logs mysql
```

### 2. Redis连接失败
```bash
# 检查Redis是否启动
docker compose ps redis

# 测试Redis连接
docker compose exec redis redis-cli ping
```

### 3. WebSocket连接失败
- 检查JWT token是否有效
- 确认后端服务正常运行
- 查看浏览器Console错误信息

### 4. 房间累积未清理
```bash
# 手动触发清理（需要修改代码将清理间隔改为1分钟）
# 或直接从数据库清理
docker compose exec mysql mysql -uliarsbar -pliarsbar123 liars_bar -e "DELETE FROM rooms;"
```

## 性能优化

- 使用Redis缓存热点数据
- WebSocket心跳保持连接活跃
- 定时清理僵尸房间（每20分钟）
- 数据库连接池管理
- Goroutine并发处理游戏逻辑

## 安全措施

- JWT Token认证
- 密码bcrypt加密存储
- SQL注入防护（使用ORM）
- CORS跨域配置
- WebSocket连接超时保护

## 项目状态

当前版本实现的功能：
- ✅ 用户注册/登录系统
- ✅ JWT认证中间件
- ✅ 快速匹配系统
- ✅ AI玩家填充
- ✅ WebSocket实时通信
- ✅ 完整游戏逻辑
- ✅ 4种角色技能
- ✅ 房间自动清理机制
- ✅ ELO积分系统
- ✅ 游戏历史记录

## 技术实现详解

### 游戏引擎核心 (engine.go)

#### 数据结构设计

**卡牌系统**
```go
type Card string
const (
    Ace   Card = "A"
    King  Card = "K"
    Queen Card = "Q"
    Jack  Card = "J"
    Wild  Card = "W"  // Scubby角色的万能牌
)
```

**游戏阶段状态机**
```go
type GamePhase string
const (
    PhaseWaiting    GamePhase = "WAITING"     // 等待玩家加入
    PhaseMatched    GamePhase = "MATCHED"     // 匹配完成
    PhasePlaying    GamePhase = "PLAYING"     // 出牌阶段
    PhaseChallenge  GamePhase = "CHALLENGE"   // 质疑阶段
    PhasePunishment GamePhase = "PUNISHMENT"  // 惩罚阶段
    PhaseRoundEnd   GamePhase = "ROUND_END"   // 回合结束
    PhaseGameOver   GamePhase = "GAME_OVER"   // 游戏结束
)
```

**玩家状态**
```go
type Player struct {
    ID               uint      // 玩家ID
    Nickname         string    // 昵称
    IsAI             bool      // 是否AI玩家
    IsOnline         bool      // 是否在线（断连后为false）
    IsAlive          bool      // 是否存活
    IsReady          bool      // 是否已准备
    AITakeover       bool      // 是否被AI托管
    SeatIndex        int       // 座位编号（0-3）
    Hand             []Card    // 手牌（隐私数据，JSON序列化时隐藏）
    HandCount        int       // 手牌数量（公开）
    PunishmentCount  int       // 惩罚计数（俄罗斯轮盘子弹数）
    PlayCount        int       // 累计出牌次数
    LieCount         int       // 累计说谎次数
    ChallengeCount   int       // 累计质疑次数
    ChallengeSuccess int       // 累计质疑成功次数
    CharacterID      string    // 角色ID
    CharacterName    string    // 角色名称
    SkillUsed        bool      // 技能是否已使用（每局重置一次）
    ChallengeUsed    int       // 本轮已质疑次数（每轮重置）
}
```

#### 核心游戏逻辑

**1. 发牌机制**
- 24张牌组成（A/K/Q/J各6张）
- Fisher-Yates洗牌算法确保随机性
- 每人初始6张手牌
- Scubby角色额外获得1张万能牌（Wild Card）

```go
func NewDeck() []Card {
    deck := make([]Card, 0, 24)
    cards := []Card{Ace, King, Queen, Jack}
    for _, c := range cards {
        for i := 0; i < 6; i++ {
            deck = append(deck, c)
        }
    }
    rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
    return deck
}
```

**2. 出牌验证流程**
```
玩家出牌 → 验证回合 → 验证手牌索引 → 提取选中的牌 
→ 判断是否说谎 → 更新手牌 → 进入质疑阶段
```

关键逻辑：
- 必须声称当前目标牌（TargetCard）
- 检查实际出的牌是否与声称一致
- Wild牌（万能牌）可以当作任何牌
- 说谎判定：`if card != targetCard && card != Wild { truthful = false }`

**3. 质疑机制**
- 质疑阶段所有其他玩家可选择质疑或Pass
- Bristle角色可质疑2次，其他角色1次
- 质疑结果立即揭示被质疑玩家的实际出牌
- 失败方接受俄罗斯轮盘惩罚

**4. 俄罗斯轮盘惩罚系统**

惩罚机制：
- 惩罚计数累积，每次惩罚相当于向左轮枪添加一发子弹
- 6个弹仓位，随机击发（1-6号弹仓）
- 子弹数越多，死亡概率越高
- Tor角色特殊保护：
  - 50%概率减少一次惩罚计数（不增加子弹）
  - 30%概率免疫死亡（即使击中也存活）

```go
bulletSlots := min(player.PunishmentCount, 6)  // 最多6发子弹
hitChamber := rand.Intn(6) + 1                 // 随机选择1-6号弹仓
survived := hitChamber > bulletSlots           // 击中编号大于子弹数则存活
```

**5. 回合循环与新回合机制**
- 目标牌按顺序循环：A → K → Q → J → A...
- 当所有存活玩家手牌打光时自动开启新回合
- 重新洗牌发牌（存活玩家各6张）
- 重置质疑次数和技能使用状态

**6. 胜利条件**
- 当存活玩家数 ≤ 1 时游戏结束
- 最后存活的玩家获胜

### AI决策系统 (room.go)

#### AI策略类型

系统实现了4种不同的AI策略类型，每个AI玩家在创建时随机选择一种：

```go
type AIStrategyType int
const (
    AIStrategyConservative AIStrategyType = iota  // 保守型
    AIStrategyAggressive                          // 激进型
    AIStrategyBalanced                            // 平衡型
    AIStrategyRandom                              // 随机型
)

func newAIStrategy(player *game.Player, state *game.GameState) *aiStrategy {
    strategyType := AIStrategyType(rand.Intn(4))  // 随机选择
    return &aiStrategy{player: player, state: state, strategyType: strategyType}
}
```

**策略特点对比：**

| 策略类型 | 出牌数量 | 选牌方式 | 质疑倾向 | 特点 |
|---------|---------|---------|---------|------|
| 保守型 | 1-2张 | 优先真牌+Wild，不足时补其他牌 | 高(×1.3) | 稳扎稳打，少说谎，多质疑 |
| 激进型 | 2-3张 | 50%概率混入非目标牌 | 低(×0.7) | 快速出牌，敢于说谎 |
| 平衡型 | 1-3张 | 优先真牌+Wild，不足时补其他牌 | 中(×1.0) | 灵活应变，综合考虑 |
| 随机型 | 完全随机 | 随机打乱手牌后选择 | 完全随机 | 不可预测的行为 |

#### AI出牌决策

AI出牌分为两个步骤：

**步骤1：决定出牌数量 (`decidePlayCount`)**

```go
func (ai *aiStrategy) decidePlayCount() int {
    handSize := len(ai.player.Hand)
    if handSize == 0 { return 1 }  // 防护：调用方会跳过空手

    switch ai.strategyType {
    case AIStrategyConservative:
        // 保守：手牌≤2出1张，手牌≤4出1-2张，否则1-2张
        if handSize <= 2 { return 1 }
        return rand.Intn(2) + 1   // 1-2张

    case AIStrategyAggressive:
        // 激进：手牌≤2全出，手牌≤4出2-3张，否则2-3张
        if handSize <= 2 { return handSize }
        return rand.Intn(2) + 2   // 2-3张

    case AIStrategyBalanced:
        // 平衡：手牌≤2出1张，手牌≤4出1-2张，否则1-3张
        if handSize <= 2 { return 1 }
        if handSize <= 4 { return rand.Intn(2) + 1 }
        return rand.Intn(3) + 1   // 1-3张

    case AIStrategyRandom:
        // 随机：1~min(3, handSize) 张
        maxPlay := min(3, handSize)
        return rand.Intn(maxPlay) + 1
    }
    return 1
}
```

**步骤2：选择出哪些牌 (`selectCards`)**

AI 根据策略类型用不同方式选牌，**没有独立的 `shouldLie()` 函数**——说谎决策隐含在选牌逻辑中：

```go
func (ai *aiStrategy) selectCards(playCount int) []int {
    targetCards, wildCards, otherCards := ai.analyzeHand()

    switch ai.strategyType {
    case AIStrategyConservative:
        // 优先打真牌+Wild，不够再用其他牌补（不得已时才说谎）
        indices = append(indices, targetCards...)
        indices = append(indices, wildCards...)
        indices = append(indices, otherCards...)

    case AIStrategyAggressive:
        // 50%概率混入非目标牌（积极说谎），WILD最后打
        for len(indices) < playCount {
            if rand.Float64() < 0.5 && len(targetCards) > 0 {
                indices = append(indices, targetCards[0])
            } else if len(otherCards) > 0 {
                indices = append(indices, otherCards[0])  // 混入假牌
            } else if len(targetCards) > 0 || len(wildCards) > 0 {
                // 没有假牌可混时才打真牌/Wild
            }
        }

    case AIStrategyBalanced:
        // 与保守型类似：优先真牌+Wild，不足时补其他牌

    case AIStrategyRandom:
        // 完全随机：打乱所有手牌，取前 playCount 张
    }

    return indices[:playCount]
}
```

**手牌分析 (`analyzeHand`)**：将手牌分为目标牌（匹配 `TargetCard`）、Wild牌、其他牌三类，供 `selectCards` 使用。

#### AI质疑决策

```go
func (ai *aiStrategy) shouldChallenge() bool {
    if ai.state.LastPlay == nil { return false }

    // Bristle角色检查：可质疑2次，其他1次
    maxChallenges := 1
    if ai.player.CharacterID == game.CharacterBristle {
        maxChallenges = 2
    }
    if ai.player.ChallengeUsed >= maxChallenges { return false }

    // 统计自己手里的目标牌数量（含Wild）
    myTargetCount := countTargetAndWildCards(ai.player.Hand, ai.state.TargetCard)
    claimedCount := len(ai.state.LastPlay.CardIDs)
    targetPlayer := ai.state.GetPreviousPlayer()

    baseProbability := 0.0

    // 因素1：对方出牌数量 vs 自己手牌数量
    if claimedCount >= 3 {
        if myTargetCount >= 4      { baseProbability = 0.8 }
        else if myTargetCount >= 3 { baseProbability = 0.6 }
        else if myTargetCount >= 2 { baseProbability = 0.3 }
    } else if claimedCount >= 2 {
        if myTargetCount >= 4      { baseProbability = 0.5 }
        else if myTargetCount >= 3 { baseProbability = 0.3 }
    } else {
        if myTargetCount >= 5      { baseProbability = 0.2 }
    }

    // 因素2：对方手牌少且出得多 → 可疑
    if targetPlayer != nil && targetPlayer.HandCount <= 2 && claimedCount >= 2 {
        baseProbability += 0.15
    }

    // 因素3：对方历史说谎率 > 50% → 更可疑
    if targetPlayer != nil && targetPlayer.PlayCount > 0 {
        lieRate := float64(targetPlayer.LieCount) / float64(targetPlayer.PlayCount)
        if lieRate > 0.5 { baseProbability += 0.1 }
    }

    // 因素4：策略类型调整
    switch ai.strategyType {
    case AIStrategyConservative: baseProbability *= 1.3
    case AIStrategyAggressive:   baseProbability *= 0.7
    case AIStrategyRandom:       baseProbability = rand.Float64()  // 完全随机
    }

    // 限制概率上限为 90%
    if baseProbability > 0.9 { baseProbability = 0.9 }

    return rand.Float64() < baseProbability
}
```

**质疑决策的核心逻辑：**
1. **牌面推理**：如果我手里目标牌很多，对方也声称出很多 → 对方很可能在说谎
2. **行为分析**：追踪对方历史说谎率，说谎率 > 50% 时提升质疑概率
3. **处境分析**：手牌 ≤ 2 且出牌 ≥ 2 的玩家更可能冒险说谎
4. **策略差异**：不同 AI 性格对质疑的态度不同（×1.3 / ×0.7 / ×1.0 / 随机）

#### AI 回合执行流程

真人出牌后 → 进入质疑阶段 → AI 玩家逐个决策（挑战 or Pass）→ 如果全部 Pass → 轮到下一位玩家出牌 → 如果当前玩家是 AI → 延迟 1 秒后自动出牌：

```go
func (r *GameRoom) processAITurns() {
    // 质疑阶段：AI逐个决策
    if r.State.Phase == game.PhaseChallenge {
        r.processAIChallengePhase()
        return
    }
    // 出牌阶段：如果当前玩家是AI且有手牌，延迟1秒后执行
    if currentPlayer.IsAI && len(currentPlayer.Hand) > 0 {
        time.Sleep(1 * time.Second)
        r.executeAITurn()
    }
    // 如果无手牌，自动跳过
}
```

### WebSocket实时通信系统

#### Hub架构

Hub是整个WebSocket系统的中心枢纽，负责管理所有客户端连接和游戏房间：

```go
type Hub struct {
    Clients    map[uint]*Client           // userID -> Client
    Rooms      map[uint]*GameRoom         // roomID -> GameRoom
    Register   chan *Client               // 注册新连接（缓冲256）
    Unregister chan *Client               // 断开连接（缓冲256）
    mu         sync.RWMutex               // 读写锁

    // 房间数据库清理接口（由 service.RoomService 实现）
    RoomService interface {
        CleanupStaleRooms(maxAgeMinutes int) (int, error)
    }

    // 游戏结束回调（在 main.go 中设置，游戏结束时调用一次）
    OnGameOver func(roomID uint, winnerID uint, players []*game.Player)
}
```

**核心职责：**
1. **连接管理**：维护所有在线玩家的WebSocket连接
2. **房间管理**：创建、销毁、查找游戏房间
3. **消息路由**：将消息路由到正确的房间和玩家
4. **定时清理**：自动清理超时房间（每20分钟）

#### Client连接管理

每个WebSocket连接对应一个Client实例：

```go
type Client struct {
    UserID   uint
    Username string
    Nickname string
    Conn     *websocket.Conn
    Hub      *Hub
    Send     chan []byte          // 发送队列（缓冲256）
    RoomID   uint
    IsAI     bool
    mu       sync.Mutex
}
```

**双向消息泵：**

1. **ReadPump（读取泵）**：持续读取客户端消息
```go
func (c *Client) ReadPump() {
    defer func() {
        c.Hub.Unregister <- c
        c.Conn.Close()
    }()
    
    // 限制消息大小（4096字节）
    c.Conn.SetReadLimit(4096)
    // 设置60秒心跳超时
    c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.Conn.SetPongHandler(func(string) error {
        c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })
    
    for {
        _, message, err := c.Conn.ReadMessage()
        if err != nil {
            break  // 连接断开
        }
        
        var msg WSMessage
        json.Unmarshal(message, &msg)
        
        // PLAYER_JOIN 特殊处理：加入房间
        if msg.Type == "PLAYER_JOIN" {
            roomID := parseRoomID(msg.Payload)
            c.Hub.JoinRoom(roomID, c)
            continue
        }
        
        // 其他消息路由到当前房间
        if c.RoomID > 0 {
            c.Hub.RouteMessage(c.RoomID, c.UserID, msg)
        }
    }
}
```

2. **WritePump（写入泵）**：持续发送消息到客户端
```go
func (c *Client) WritePump() {
    ticker := time.NewTicker(54 * time.Second)  // 心跳间隔 = pongWait * 9/10
    defer func() {
        ticker.Stop()
        c.Conn.Close()
    }()
    
    for {
        select {
        case message, ok := <-c.Send:
            c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            c.Conn.WriteMessage(websocket.TextMessage, message)
            
        case <-ticker.C:
            c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

**心跳机制：**
- 服务端每 54 秒（60s × 9/10）发送一次 Ping
- 客户端 60 秒内未回复 Pong 则认为连接断开
- 写入超时为 10 秒
- 防止 TCP 连接僵死和代理超时

#### 房间事件循环

每个GameRoom运行独立的事件循环处理游戏逻辑：

```go
type GameRoom struct {
    ID          uint
    Name        string
    Players     map[uint]*game.Player
    State       *game.GameState
    Events      chan GameEvent         // 事件队列（缓冲256）
    Hub         *Hub
    mu          sync.RWMutex
    aiService   *AIProxy               // AI服务代理（可选）
    turnTimer   *time.Timer
    turnTimeout time.Duration          // 回合超时（默认30秒）
    closed      bool
    CreatedAt   time.Time
    statsRecorded int32                // 原子操作，确保游戏结果只记录一次
}

func (r *GameRoom) eventLoop() {
    for evt := range r.Events {
        r.processEvent(evt)  // 串行处理事件
    }
}
```

**支持的事件类型：**
- `PLAYER_JOIN`：玩家加入房间
- `SET_CHARACTER`：选择角色
- `PLAYER_READY`：准备开始
- `START_GAME`：开始游戏
- `PLAY_CARD`：出牌
- `CHALLENGE`：质疑
- `PASS`：跳过质疑/没有手牌时自动跳过
- `USE_SKILL`：使用角色技能（Foxy查看手牌）
- `CHAT`：聊天消息
- `AI_ACTION`：AI行动触发
- `RECONNECT`：玩家重连
- `GAME_OVER`：强制结束游戏
- `PLAYER_LEAVE`：玩家离开

**事件处理特点：**
- 单线程串行处理，避免竞态条件
- 256容量的缓冲通道，应对短时消息爆发
- 通道满时丢弃消息并记录日志
- 游戏开始后 AI 玩家通过 `processAITurns()` 自动驱动

#### 房间清理机制

系统实现了双重清理策略确保资源释放：

**1. 即时清理**

当所有真人玩家离开时立即触发：

```go
func (r *GameRoom) handlePlayerLeave(evt GameEvent) {
    delete(r.Players, evt.PlayerID)
    
    // 检查是否只剩AI玩家
    hasHuman := false
    for _, p := range r.Players {
        if !p.IsAI {
            hasHuman = true
            break
        }
    }
    
    // 没有真人玩家，立即结束游戏
    if !hasHuman {
        r.State.Phase = game.PhaseGameOver
        
        // 记录统计数据
        if r.Hub.OnGameOver != nil {
            r.Hub.OnGameOver(r.ID, 0, r.State.Players)
        }
        
        // 5秒后销毁房间
        go func() {
            time.Sleep(5 * time.Second)
            r.Hub.DestroyRoom(r.ID)
        }()
    }
}
```

**2. 定时清理**

每20分钟清理一次超时房间：

```go
func (h *Hub) StartCleanupTask(interval, maxAge time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            h.CleanupStaleRooms(maxAge)
        }
    }()
}

func (h *Hub) CleanupStaleRooms(maxAge time.Duration) {
    now := time.Now()
    h.mu.Lock()
    defer h.mu.Unlock()
    
    for id, room := range h.rooms {
        if now.Sub(room.CreatedAt) > maxAge {
            // 内存清理
            room.Close()
            delete(h.rooms, id)
            
            // 数据库清理
            if h.RoomService != nil {
                h.RoomService.CleanupStaleRooms(int(maxAge.Minutes()))
            }
        }
    }
}
```

清理触发条件：
- 即时清理：所有真人玩家离开
- 定时清理：房间创建超过20分钟

### 匹配系统

#### 快速匹配流程

系统使用匹配队列，支持多人匹配 + AI 补位：

```go
type MatchService struct {
    Hub         *websocket.Hub
    Config      *config.GameConfig
    Queue       []MatchEntry       // 匹配队列
    mu          sync.Mutex
    aiPlayerID  uint               // AI ID生成器（从100000开始递增）
    roomService *service.RoomService
}

type MatchEntry struct {
    UserID      uint
    Nickname    string
    CharacterID string
    JoinedAt    time.Time
}
```

**匹配逻辑**（每秒检查一次）：

```go
func (ms *MatchService) tryMatch() {
    // 按等待时间将队列分为两组
    var fresh, timedOut []MatchEntry
    for _, entry := range ms.Queue {
        if now.Sub(entry.JoinedAt) >= ms.Config.AIFillTimeout {
            timedOut = append(timedOut, entry)   // 已超时，愿意接受AI补位
        } else {
            fresh = append(fresh, entry)          // 仍在等待真人
        }
    }

    if len(fresh) >= 4 {
        // 凑齐4个真人，立即开局（无AI）
        selected = fresh[:4]
        fillAI = false
    } else if len(timedOut) > 0 {
        // 有人等超时，用队列所有人 + AI补位开局
        pool = fresh + timedOut
        fillAI = true
    } else {
        return  // 不够4人且无人超时，继续等待
    }

    go ms.createRoom(selected, fillAI)
}
```

**匹配特点：**
- 优先凑 4 个真人玩家开局
- 有玩家等待超时（默认 10 秒）后，用 AI 补足 4 人开局
- 支持 1~4 个真人玩家 + 0~3 个 AI 补位
- 匹配成功后通过 WebSocket 发送 `MATCH_FOUND` 消息通知玩家
- 房间创建后 2 秒自动开始游戏

#### AI填充

```go
// 在 createRoom 中为每个 AI 玩家分配：
gameRoom.Players[aiID] = &game.Player{
    ID:            aiID,          // ID 从 100000 递增
    Nickname:      "AI-Bot",
    SeatIndex:     humanCount + i,
    IsAlive:       true,
    IsOnline:      true,
    IsAI:          true,
    CharacterID:   game.CharacterScubby,   // 固定 Scubby
    CharacterName: "Scubby",
}
```

- AI ID 从 100000 开始递增生成
- AI 玩家角色固定为 Scubby
- AI 策略类型在创建 `aiStrategy` 时随机选择（4 种策略等概率）
- AI 玩家自动设为准备状态
- AI 出牌有 1 秒延迟，模拟思考时间

### ELO积分系统

#### 积分计算方式

采用固定分差制：

```go
func (s *UserService) RecordGameResult(winnerID uint, players []*game.Player) {
    for _, p := range players {
        if p.IsAI {
            continue  // 跳过AI，不参与积分
        }
        isWin := p.ID == winnerID
        // 更新游戏统计数据（包含说谎、质疑等细分统计）
        s.repo.IncrementGames(p.ID, isWin)
        s.repo.IncrementStats(p.ID, p.LieCount, p.ChallengeCount, p.ChallengeSuccess)
        if isWin {
            s.UpdateELO(p.ID, 20)   // 胜利 +20
        } else {
            s.UpdateELO(p.ID, -15)  // 失败 -15
        }
    }
}
```

**积分规则：**
- 初始积分：1000
- 胜利：+20 分
- 失败：-15 分
- 积分不会降到负数以下（最小为0）
- 只对真人玩家计算（AI不参与积分）

### 数据持久化

#### 游戏结果记录

游戏结束时，`RecordGameResult` 遍历所有玩家，跳过 AI，对真人玩家：
- 更新游戏场次（胜/负计数）
- 更新细分统计（说谎次数、质疑次数、质疑成功次数）
- 更新 ELO 积分（胜 +20，负 -15）

#### 数据库表结构

数据库使用 GORM AutoMigrate 自动创建，包含以下核心表：

**users表**
```sql
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(50) NOT NULL,
    avatar_url VARCHAR(255),
    email VARCHAR(100),
    elo_rating INT DEFAULT 1000,
    total_games INT DEFAULT 0,
    total_wins INT DEFAULT 0,
    total_losses INT DEFAULT 0,
    total_lies INT DEFAULT 0,
    total_challenges INT DEFAULT 0,
    total_successful_challenges INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'OFFLINE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
```

**rooms表**
```sql
CREATE TABLE rooms (
    id INT PRIMARY KEY AUTO_INCREMENT,
    room_uuid VARCHAR(64) UNIQUE,
    host_user_id INT,
    room_name VARCHAR(100),
    max_players INT DEFAULT 4,
    current_players INT DEFAULT 0,
    room_status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP NULL,
    finished_at TIMESTAMP NULL
);
```

**games表**
```sql
CREATE TABLE games (
    id INT PRIMARY KEY AUTO_INCREMENT,
    room_id INT,
    winner_id INT,
    player_count INT,
    round_count INT,
    duration INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**game_players表**
```sql
CREATE TABLE game_players (
    id INT PRIMARY KEY AUTO_INCREMENT,
    game_id INT,
    user_id INT,
    character_id VARCHAR(20),
    is_ai BOOLEAN DEFAULT FALSE,
    final_rank INT,
    is_alive BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## License

MIT License

## 贡献者

- [@even-young-leaf](https://github.com/even-young-leaf)
- [@mywww0517](https://github.com/mywww0517)
