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

## License

MIT License

## 贡献者

- [@even-young-leaf](https://github.com/even-young-leaf)
- [@mywww0517](https://github.com/mywww0517)
