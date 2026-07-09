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
│   ├── match/           # 匹配系统
│   ├── middleware/      # 中间件（CORS、JWT认证）
│   ├── model/           # 数据模型
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   ├── utils/           # 工具函数
│   └── websocket/       # WebSocket实时通信
│       ├── hub.go       # 连接管理中心
│       └── room.go      # 游戏房间逻辑
└── Dockerfile.backend
```

## 核心功能

### 1. 用户系统
- 用户注册/登录（JWT认证）
- 用户资料管理
- ELO积分系统
- 游戏统计数据

### 2. 匹配系统
- 快速匹配（1v3 AI）
- 自动填充AI玩家
- 匹配超时处理
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
- 心跳检测（60秒超时）
- 自动重连支持
- 玩家在线状态管理

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
- **后端API**: http://localhost:8080
- **MySQL**: localhost:3306
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
  "data": {
    "token": "eyJhbGc...",
    "user": { ... }
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

### WebSocket连接

```
ws://localhost:8080/ws?token=<jwt_token>
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
    "card_indices": [0, 1, 2],
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
    IsAlive          bool      // 是否存活
    Hand             []Card    // 手牌（隐私数据）
    HandCount        int       // 手牌数量（公开）
    PunishmentCount  int       // 惩罚计数（俄罗斯轮盘子弹数）
    CharacterID      string    // 角色ID
    SkillUsed        bool      // 技能是否已使用
    ChallengeUsed    int       // 本轮已质疑次数
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

系统实现了4种不同的AI策略类型，每种策略有独特的行为模式：

```go
type AIStrategyType int
const (
    AIStrategyConservative AIStrategyType = iota  // 保守型
    AIStrategyAggressive                          // 激进型
    AIStrategyBalanced                            // 平衡型
    AIStrategyRandom                              // 随机型
)
```

**策略特点对比：**

| 策略类型 | 出牌数量 | 说谎倾向 | 质疑倾向 | 特点 |
|---------|---------|---------|---------|------|
| 保守型 | 1-2张 | 低(10-30%) | 高(+30%) | 稳扎稳打，少说谎，多质疑 |
| 激进型 | 2-3张 | 高(40-60%) | 低(-30%) | 快速出牌，敢于说谎 |
| 平衡型 | 1-2张 | 中(20-40%) | 中 | 灵活应变，综合考虑 |
| 随机型 | 随机 | 随机 | 随机 | 不可预测的行为 |

#### AI决策流程

**1. 出牌决策**

AI出牌分为三个步骤：

**步骤1：决定出牌数量**
```go
func (ai *aiStrategy) decidePlayCount() int {
    handSize := len(ai.player.Hand)
    
    switch ai.strategyType {
    case AIStrategyConservative:
        // 保守：优先出1张，手牌多时出2张
        if handSize >= 4 { return 2 }
        return 1
        
    case AIStrategyAggressive:
        // 激进：优先出2-3张，快速清空手牌
        if handSize >= 5 { return 3 }
        if handSize >= 3 { return 2 }
        return 1
        
    case AIStrategyBalanced:
        // 平衡：根据手牌灵活调整
        if handSize >= 5 { return 2 }
        return 1
    }
}
```

**步骤2：选择出哪些牌**

AI会优先查找目标牌和万能牌：
```go
func (ai *aiStrategy) selectCards(count int) []int {
    targetCard := ai.gameState.TargetCard
    
    // 1. 优先收集目标牌
    for i, card := range ai.player.Hand {
        if card == targetCard || card == game.Wild {
            targetIndices = append(targetIndices, i)
        }
    }
    
    // 2. 如果目标牌足够，直接出真牌
    if len(targetIndices) >= count {
        return targetIndices[:count]
    }
    
    // 3. 目标牌不够，决定是否说谎
    if ai.shouldLie() {
        // 说谎：混合出目标牌+非目标牌
        return mixed_cards
    } else {
        // 不说谎：只出现有的目标牌
        return targetIndices
    }
}
```

**步骤3：判断是否说谎**

说谎概率基于多个因素动态计算：

```go
func (ai *aiStrategy) shouldLie() bool {
    baseProbability := 0.3  // 基础概率30%
    
    // 因素1：手牌中目标牌数量（目标牌越少越可能说谎）
    targetCount := ai.countTargetCards()
    if targetCount == 0 {
        baseProbability = 0.8  // 没有目标牌，80%说谎
    } else if targetCount <= 2 {
        baseProbability = 0.5  // 目标牌少，50%说谎
    }
    
    // 因素2：手牌总数（手牌多时更倾向说谎快速出牌）
    if len(ai.player.Hand) >= 5 {
        baseProbability += 0.1
    }
    
    // 因素3：策略类型调整
    switch ai.strategyType {
    case AIStrategyConservative:
        baseProbability *= 0.5  // 保守型减半
    case AIStrategyAggressive:
        baseProbability *= 1.5  // 激进型增加50%
    }
    
    return rand.Float64() < baseProbability
}
```

**2. 质疑决策**

AI质疑决策考虑多个智能因素：

```go
func (ai *aiStrategy) shouldChallenge() bool {
    baseProbability := 0.0
    
    // 因素1：对方出牌数量 vs 我的目标牌数量
    claimedCount := len(lastPlay.Cards)  // 对方声称出了几张
    myTargetCount := ai.countTargetCards()  // 我手里有几张目标牌
    
    if claimedCount >= 3 {
        // 对方出3张目标牌，可疑度高
        if myTargetCount >= 4 {
            baseProbability = 0.7  // 我有4张，对方出3张，70%质疑
        } else if myTargetCount >= 3 {
            baseProbability = 0.5  // 我有3张，50%质疑
        }
    } else if claimedCount == 2 {
        if myTargetCount >= 5 {
            baseProbability = 0.6  // 对方出2张，我有5张，60%质疑
        } else if myTargetCount >= 4 {
            baseProbability = 0.5
        } else if myTargetCount >= 3 {
            baseProbability = 0.3
        }
    } else {
        // 对方只出1张，很少质疑
        if myTargetCount >= 5 {
            baseProbability = 0.2
        }
    }
    
    // 因素2：对方手牌数量（手牌少可能着急出牌容易说谎）
    if targetPlayer.HandCount <= 2 && claimedCount >= 2 {
        baseProbability += 0.15
    }
    
    // 因素3：对方历史说谎率
    if targetPlayer.PlayCount > 0 {
        lieRate := float64(targetPlayer.LieCount) / float64(targetPlayer.PlayCount)
        if lieRate > 0.5 {
            baseProbability += 0.1  // 对方经常说谎，增加质疑
        }
    }
    
    // 因素4：策略类型调整
    switch ai.strategyType {
    case AIStrategyConservative:
        baseProbability *= 1.3  // 保守型更倾向质疑
    case AIStrategyAggressive:
        baseProbability *= 0.7  // 激进型较少质疑
    }
    
    return rand.Float64() < baseProbability
}
```

**质疑决策的核心逻辑：**
1. **牌面推理**：如果我手里有很多目标牌，对方也声称出很多，那对方很可能在说谎
2. **行为分析**：追踪对方的历史说谎率，对"老骗子"提高警惕
3. **处境分析**：手牌少的玩家更着急，更可能冒险说谎
4. **策略差异**：不同AI性格对质疑的态度不同

#### AI行为示例

**场景1：保守型AI，手里有5张目标牌A**
- 对方声称出3张A → 质疑概率70% × 1.3(保守加成) = 91%，极大概率质疑
- 对方声称出1张A → 质疑概率20% × 1.3 = 26%，较少质疑

**场景2：激进型AI，手里只有1张目标牌K**
- 决定出2张牌 → 目标牌不够 → 说谎概率50% × 1.5(激进加成) = 75%
- 很可能出1张K + 1张其他牌，并声称都是K

**场景3：平衡型AI分析对手**
- 对手历史：出牌5次，说谎3次（说谎率60%）
- 对手当前：剩余2张手牌，声称出2张Q
- 我手里有4张Q
- 基础质疑概率50% + 历史说谎加成10% = 60%，中等概率质疑

### WebSocket实时通信系统

#### Hub架构

Hub是整个WebSocket系统的中心枢纽，负责管理所有客户端连接和游戏房间：

```go
type Hub struct {
    clients    map[uint]*Client           // userID -> Client
    rooms      map[uint]*GameRoom         // roomID -> GameRoom
    Register   chan *Client               // 注册新连接
    Unregister chan *Client               // 断开连接
    Broadcast  chan BroadcastMessage      // 全局广播
    RoomService *service.RoomService      // 房间数据库服务
    OnGameOver  func(roomID, winnerID uint, players []*game.Player)  // 游戏结束回调
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
    Send     chan Message    // 发送队列
    Hub      *Hub
    RoomID   uint
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
    
    // 设置60秒心跳超时
    c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.Conn.SetPongHandler(func(string) error {
        c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })
    
    for {
        var msg Message
        if err := c.Conn.ReadJSON(&msg); err != nil {
            break  // 连接断开
        }
        c.Hub.RouteMessage(c, msg)  // 路由消息
    }
}
```

2. **WritePump（写入泵）**：持续发送消息到客户端
```go
func (c *Client) WritePump() {
    ticker := time.NewTicker(54 * time.Second)  // 心跳间隔
    defer func() {
        ticker.Stop()
        c.Conn.Close()
    }()
    
    for {
        select {
        case message, ok := <-c.Send:
            if !ok {
                return  // 通道关闭
            }
            c.Conn.WriteJSON(message)
            
        case <-ticker.C:
            // 发送心跳Ping
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

**心跳机制：**
- 客户端每54秒发送一次Ping
- 服务端60秒未收到Pong则认为连接断开
- 防止TCP连接僵死和代理超时

#### 房间事件循环

每个GameRoom运行独立的事件循环处理游戏逻辑：

```go
type GameRoom struct {
    ID       uint
    Players  map[uint]*game.Player
    State    *game.GameState
    Events   chan GameEvent        // 事件队列（缓冲256）
    Hub      *Hub
    closed   bool
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
- `PASS`：跳过/没有手牌
- `AI_ACTION`：AI行动
- `PLAYER_LEAVE`：玩家离开

**事件处理特点：**
- 单线程串行处理，避免竞态条件
- 256容量的缓冲通道，应对短时消息爆发
- 通道满时丢弃消息并记录日志

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
                h.RoomService.DeleteRoom(id)
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

```go
type MatchService struct {
    waitingPlayers map[uint]*MatchRequest   // 等待匹配的玩家
    hub            *Hub
    timeout        time.Duration            // 10秒超时
}

func (m *MatchService) StartMatch(userID uint, characterID string) {
    // 1. 创建匹配请求
    req := &MatchRequest{
        UserID:      userID,
        CharacterID: characterID,
        CreatedAt:   time.Now(),
    }
    m.waitingPlayers[userID] = req
    
    // 2. 立即创建房间（1v3 AI模式）
    room := m.hub.CreateRoom(fmt.Sprintf("Room-%d", userID))
    
    // 3. 玩家加入房间
    room.HandleEvent(GameEvent{
        Type:     "PLAYER_JOIN",
        PlayerID: userID,
        Payload:  characterPayload,
    })
    
    // 4. 填充3个AI玩家
    m.fillAIPlayers(room, 3)
    
    // 5. 设置超时清理
    go func() {
        time.Sleep(m.timeout)
        if !room.allPlayersReady() {
            m.hub.DestroyRoom(room.ID)
        }
    }()
}
```

**匹配特点：**
- 当前版本采用1v3 AI模式（1个真人玩家 + 3个AI）
- 匹配成功后立即进入房间
- 10秒内未准备则清理房间
- AI玩家随机分配策略类型和角色

#### AI填充策略

```go
func (m *MatchService) fillAIPlayers(room *GameRoom, count int) {
    aiCharacters := []string{"scubby", "foxy", "bristle", "tor"}
    
    for i := 0; i < count; i++ {
        aiID := uint(1000000 + rand.Intn(900000))  // AI玩家ID：1000000-1899999
        characterID := aiCharacters[rand.Intn(len(aiCharacters))]
        
        room.HandleEvent(GameEvent{
            Type:     "PLAYER_JOIN",
            PlayerID: aiID,
            AIPlayer: true,
            Payload:  characterPayload,
        })
        
        // AI自动准备
        room.HandleEvent(GameEvent{
            Type:     "PLAYER_READY",
            PlayerID: aiID,
        })
    }
}
```

### ELO积分系统

#### 积分计算公式

采用标准ELO算法：

```go
func calculateELO(winnerELO, loserELO float64) (newWinner, newLoser float64) {
    K := 32.0  // K因子
    
    // 期望胜率
    expectedWinner := 1.0 / (1.0 + math.Pow(10, (loserELO-winnerELO)/400))
    expectedLoser := 1.0 / (1.0 + math.Pow(10, (winnerELO-loserELO)/400))
    
    // 新积分
    newWinner = winnerELO + K * (1.0 - expectedWinner)
    newLoser = loserELO + K * (0.0 - expectedLoser)
    
    return newWinner, newLoser
}
```

**积分规则：**
- 初始积分：1000
- K因子：32（积分变化幅度）
- 胜利：+分（战胜高手加分多）
- 失败：-分（输给弱手扣分多）
- 只对真人玩家计算（AI不参与积分）

### 数据持久化

#### 游戏结果记录

```go
func (u *UserService) RecordGameResult(winnerID uint, players []*game.Player) {
    for _, p := range players {
        if p.IsAI {
            continue  // 跳过AI
        }
        
        isWinner := p.ID == winnerID
        
        // 更新统计
        repo.IncrementGames(p.ID)
        if isWinner {
            repo.IncrementWins(p.ID)
        }
        
        // 更新ELO（仅对真人玩家）
        if isWinner {
            newELO := calculateELOGain(p)
            repo.UpdateELO(p.ID, newELO)
        }
    }
}
```

#### 数据库表结构

**users表**
```sql
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    nickname VARCHAR(50),
    elo_rating FLOAT DEFAULT 1000,
    total_games INT DEFAULT 0,
    total_wins INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**rooms表**
```sql
CREATE TABLE rooms (
    id INT PRIMARY KEY AUTO_INCREMENT,
    room_uuid VARCHAR(36) UNIQUE,
    room_status VARCHAR(20),
    current_players INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**game_history表**
```sql
CREATE TABLE game_history (
    id INT PRIMARY KEY AUTO_INCREMENT,
    room_id INT,
    winner_id INT,
    players JSON,
    duration INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## License

MIT License

## 贡献者

- [@even-young-leaf](https://github.com/even-young-leaf)
- [@mywww0517](https://github.com/mywww0517)
