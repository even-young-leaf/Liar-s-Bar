# Liar's Bar 移动端API文档

## 基础信息

**Base URL:** `http://your-server:8081/api/v1`

**认证方式:** JWT Bearer Token

所有需要认证的接口在请求头中添加：
```
Authorization: Bearer <token>
```

**统一响应格式:**
```json
{
  "code": 0,        // 0表示成功，非0表示错误
  "msg": "success", // 错误时返回错误信息
  "data": {}        // 响应数据
}
```

---

## 1. 认证接口

### 1.1 用户注册

**接口:** `POST /auth/register`

**请求体:**
```json
{
  "username": "player1",      // 3-50字符，仅支持字母数字下划线横线
  "password": "123456",       // 6-100字符
  "nickname": "玩家一号"      // 1-50字符，显示名称
}
```

**响应:**
```json
{
  "code": 0,
  "msg": "success"
}
```

---

### 1.2 用户登录

**接口:** `POST /auth/login`

**请求体:**
```json
{
  "username": "player1",
  "password": "123456"
}
```

**响应:**
```json
{
  "code": 0,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "player1",
    "nickname": "玩家一号"
  }
}
```

**说明:** 
- token需要保存，后续所有接口都需要
- token有效期7天

---

## 2. 用户信息接口

### 2.1 获取个人资料

**接口:** `GET /user/profile`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "username": "player1",
    "nickname": "玩家一号",
    "avatar_url": "https://...",
    "email": "player@example.com",
    "elo_rating": 1200,
    "total_games": 100,
    "total_wins": 35,
    "total_losses": 65,
    "total_lies": 200,
    "total_challenges": 50,
    "total_successful_challenges": 30,
    "status": "ONLINE",
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-07-07T10:00:00Z"
  }
}
```

---

### 2.2 修改个人资料

**接口:** `PUT /user/profile`

**需要认证:** ✅

**请求体:**
```json
{
  "nickname": "新昵称",
  "avatar_url": "https://example.com/avatar.jpg"
}
```

**响应:**
```json
{
  "code": 0,
  "msg": "success"
}
```

---

### 2.3 获取用户状态

**接口:** `GET /user/status`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "status": "IN_GAME",      // ONLINE/IN_QUEUE/IN_GAME
    "room_id": 123,           // 所在房间ID（0表示不在房间）
    "can_reconnect": true     // 是否可以重连
  }
}
```

**状态说明:**
- `ONLINE`: 在线但未在游戏中
- `IN_QUEUE`: 正在匹配队列中
- `IN_GAME`: 正在游戏中

---

### 2.4 获取游戏统计数据

**接口:** `GET /user/stats`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "total_games": 100,
    "total_wins": 35,
    "total_losses": 65,
    "win_rate": 0.35,                      // 胜率
    "elo_rating": 1200,
    "total_lies": 200,
    "total_challenges": 50,
    "total_successful_challenges": 30,
    "challenge_success_rate": 0.6,         // 质疑成功率
    "avg_rank": 2.3                        // 平均排名
  }
}
```

---

## 3. 角色系统

### 3.1 获取角色列表

**接口:** `GET /characters`

**需要认证:** ❌ (公开接口)

**响应:**
```json
{
  "code": 0,
  "data": [
    {
      "id": "scubby",
      "name": "Scubby",
      "description": "起手多一张万能牌",
      "skill": "开局额外获得1张WILD牌，可以代替任何牌型",
      "avatar_url": "/assets/characters/scubby.png"
    },
    {
      "id": "foxy",
      "name": "Foxy",
      "description": "可以偷看其他玩家手牌",
      "skill": "每局游戏可使用一次偷看技能，查看目标玩家的所有手牌3秒",
      "avatar_url": "/assets/characters/foxy.png"
    },
    {
      "id": "bristle",
      "name": "Bristle",
      "description": "每轮可以质疑两次",
      "skill": "不受每轮一次质疑的限制，每轮可以质疑最多2次",
      "avatar_url": "/assets/characters/bristle.png"
    },
    {
      "id": "tor",
      "name": "Tor",
      "description": "减少惩罚或免疫",
      "skill": "失败惩罚时有50%概率减少1发子弹或完全免疫惩罚",
      "avatar_url": "/assets/characters/tor.png"
    }
  ]
}
```

---

## 4. 匹配系统

### 4.1 开始匹配

**接口:** `POST /match/start`

**需要认证:** ✅

**请求体:**
```json
{
  "character_id": "scubby"    // 可选：scubby/foxy/bristle/tor
}
```

**响应:**
```json
{
  "code": 0,
  "msg": "matching started"
}
```

**说明:**
- 匹配成功后会通过WebSocket推送 `MATCH_FOUND` 消息
- 如果30秒内未匹配到4个真人，将自动使用AI补位开始游戏

---

### 4.2 取消匹配

**接口:** `POST /match/cancel`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "msg": "matching cancelled"
}
```

---

### 4.3 查询匹配状态

**接口:** `GET /match/status`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "status": "WAITING"    // NOT_MATCHING/WAITING/IN_ROOM
  }
}
```

---

## 5. 房间系统

### 5.1 创建房间

**接口:** `POST /rooms`

**需要认证:** ✅

**请求体:**
```json
{
  "name": "我的房间"
}
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 123,
    "room_name": "我的房间",
    "creator_id": 1,
    "status": "WAITING",
    "current_players": 1,
    "max_players": 4,
    "created_at": "2026-07-07T10:00:00Z"
  }
}
```

---

### 5.2 获取房间列表

**接口:** `GET /rooms`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": [
    {
      "id": 123,
      "room_name": "房间1",
      "creator_id": 1,
      "status": "WAITING",
      "current_players": 2,
      "max_players": 4,
      "created_at": "2026-07-07T10:00:00Z"
    }
  ]
}
```

**房间状态:**
- `WAITING`: 等待中，可加入
- `MATCHED`: 匹配成功，准备开始
- `PLAYING`: 游戏进行中
- `FINISHED`: 已结束

---

### 5.3 获取房间详情

**接口:** `GET /rooms/:id`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "room": {
      "id": 123,
      "room_name": "房间1",
      "creator_id": 1,
      "status": "WAITING",
      "current_players": 2,
      "max_players": 4
    },
    "players": [
      {
        "id": 1,
        "user_id": 1,
        "username": "player1",
        "nickname": "玩家1",
        "is_ready": true
      }
    ]
  }
}
```

---

### 5.4 加入房间

**接口:** `POST /rooms/:id/join`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "msg": "success"
}
```

**说明:** 加入成功后需要连接WebSocket进行游戏

---

### 5.5 离开房间

**接口:** `POST /rooms/:id/leave`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "msg": "success"
}
```

---

## 6. 大厅信息

### 6.1 获取大厅数据

**接口:** `GET /lobby`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "online_count": 150,     // 在线人数
    "queue_length": 12,      // 匹配队列长度
    "active_rooms": 35       // 活跃房间数
  }
}
```

---

## 7. 历史战绩

### 7.1 获取历史对局列表

**接口:** `GET /history`

**需要认证:** ✅

**查询参数:**
- `page`: 页码，默认1
- `page_size`: 每页数量，默认20，最大100

**示例:** `GET /history?page=1&page_size=10`

**响应:**
```json
{
  "code": 0,
  "data": {
    "games": [
      {
        "game_id": 1,
        "game_uuid": "550e8400-e29b-41d4-a716-446655440000",
        "room_id": 123,
        "winner_user_id": 1,
        "is_win": true,
        "final_rank": 1,
        "survived": true,
        "total_rounds": 5,
        "total_turns": 20,
        "ai_count": 1,
        "start_time": "2026-07-07T10:00:00Z",
        "end_time": "2026-07-07T10:15:00Z",
        "score_change": 20
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 10
  }
}
```

---

### 7.2 获取对局详情

**接口:** `GET /games/:id`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "game": {
      "id": 1,
      "game_uuid": "550e8400-e29b-41d4-a716-446655440000",
      "room_id": 123,
      "winner_user_id": 1,
      "total_rounds": 5,
      "total_turns": 20,
      "ai_count": 1,
      "start_time": "2026-07-07T10:00:00Z",
      "end_time": "2026-07-07T10:15:00Z"
    },
    "players": [
      {
        "id": 1,
        "game_id": 1,
        "user_id": 1,
        "username": "player1",
        "nickname": "玩家1",
        "avatar_url": "https://...",
        "is_ai": false,
        "final_rank": 1,
        "survived": true,
        "lie_count": 3,
        "challenge_count": 2,
        "challenge_success_count": 1,
        "punishment_count": 2,
        "bullets_fired": 1,
        "score_change": 20
      }
    ],
    "actions": [
      {
        "id": 1,
        "game_id": 1,
        "player_id": 1,
        "round_no": 1,
        "turn_no": 1,
        "action_type": "PLAY_CARD",
        "action_data": "{\"card_ids\":[0,1],\"claim\":\"A\"}",
        "created_at": "2026-07-07T10:01:00Z"
      }
    ]
  }
}
```

---

## 8. 管理接口

### 8.1 在线统计

**接口:** `GET /admin/online`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "online_count": 150
  }
}
```

---

### 8.2 房间统计

**接口:** `GET /admin/rooms`

**需要认证:** ✅

**响应:**
```json
{
  "code": 0,
  "data": {
    "total_rooms": 35,
    "waiting_rooms": 10,
    "playing_rooms": 25
  }
}
```

---

## 9. WebSocket实时通信

### 9.1 连接WebSocket

**URL:** `ws://your-server:8081/ws?token=<JWT_TOKEN>`

**连接方式:**
```javascript
const ws = new WebSocket('ws://your-server:8081/ws?token=' + token);
```

**说明:**
- 必须先通过HTTP接口登录获取token
- token通过URL参数传递
- 连接成功后会自动加入之前的房间（如果有）

---

### 9.2 消息格式

所有WebSocket消息都是JSON格式：

**客户端→服务端:**
```json
{
  "type": "消息类型",
  "payload": { /* 消息数据 */ }
}
```

**服务端→客户端:**
```json
{
  "type": "消息类型",
  "payload": { /* 消息数据 */ }
}
```

---

### 9.3 客户端发送的消息类型

#### 9.3.1 加入房间
```json
{
  "type": "PLAYER_JOIN",
  "payload": {
    "character_id": "scubby"
  }
}
```

#### 9.3.2 选择角色
```json
{
  "type": "SET_CHARACTER",
  "payload": {
    "character_id": "foxy"
  }
}
```

#### 9.3.3 准备
```json
{
  "type": "PLAYER_READY",
  "payload": {}
}
```

#### 9.3.4 出牌
```json
{
  "type": "PLAY_CARD",
  "payload": {
    "card_ids": [0, 1, 2],    // 手牌索引
    "claim": "A"              // 声称的牌型：A/K/Q/J
  }
}
```

#### 9.3.5 质疑
```json
{
  "type": "CHALLENGE",
  "payload": {
    "target_player_id": 2
  }
}
```

#### 9.3.6 跳过
```json
{
  "type": "PASS",
  "payload": {}
}
```

#### 9.3.7 发送聊天
```json
{
  "type": "CHAT",
  "payload": {
    "content": "你在说谎！"
  }
}
```

#### 9.3.8 使用技能（Foxy偷看）
```json
{
  "type": "USE_SKILL",
  "payload": {
    "target_player_id": 3
  }
}
```

#### 9.3.9 重连
```json
{
  "type": "RECONNECT",
  "payload": {}
}
```

---

### 9.4 服务端推送的消息类型

#### 9.4.1 匹配成功
```json
{
  "type": "MATCH_FOUND",
  "payload": {
    "room_id": 123,
    "room_name": "Match Room"
  }
}
```

#### 9.4.2 玩家加入
```json
{
  "type": "PLAYER_JOINED",
  "payload": {
    "player_id": 2,
    "nickname": "玩家2",
    "character_id": "foxy",
    "character_name": "Foxy"
  }
}
```

#### 9.4.3 玩家离开
```json
{
  "type": "PLAYER_LEFT",
  "payload": {
    "player_id": 2,
    "game_over": false,      // 如果为true表示游戏因此结束
    "reason": "玩家退出，游戏结束",
    "winner_id": 1
  }
}
```

#### 9.4.4 角色选择
```json
{
  "type": "CHARACTER_SELECTED",
  "payload": {
    "player_id": 1,
    "character_id": "bristle",
    "character_name": "Bristle"
  }
}
```

#### 9.4.5 玩家准备
```json
{
  "type": "PLAYER_READY",
  "payload": {
    "player_id": 1
  }
}
```

#### 9.4.6 房间状态更新
```json
{
  "type": "ROOM_STATE",
  "payload": {
    "id": 123,
    "name": "房间1",
    "phase": "WAITING",
    "players": [
      {
        "id": 1,
        "nickname": "玩家1",
        "is_ai": false,
        "is_online": true,
        "is_ready": true,
        "seat_index": 0,
        "character_id": "scubby",
        "character_name": "Scubby"
      }
    ],
    "player_count": 2,
    "max_players": 4,
    "ready_count": 1,
    "can_join": true
  }
}
```

#### 9.4.7 游戏开始
```json
{
  "type": "GAME_STARTED",
  "payload": {
    "phase": "PLAYING",
    "current_player": 0,
    "target_card": "A",
    "round": 1,
    "players": [
      {
        "id": 1,
        "nickname": "玩家1",
        "seat_index": 0,
        "hand_count": 6,
        "is_alive": true,
        "character_id": "scubby",
        "character_name": "Scubby"
      }
    ]
  }
}
```

#### 9.4.8 游戏状态更新
```json
{
  "type": "GAME_STATE",
  "payload": {
    "phase": "PLAYING",
    "current_player": 1,
    "target_card": "K",
    "round": 2,
    "hand": ["A", "A", "K", "Q", "J"],    // 自己的手牌
    "players": [
      {
        "id": 1,
        "nickname": "玩家1",
        "seat_index": 0,
        "hand_count": 5,
        "is_alive": true,
        "bullets": 0,
        "character_id": "scubby"
      }
    ],
    "last_play": {
      "player_id": 2,
      "card_count": 2,
      "claim": "K"
    },
    "legal_actions": ["PLAY_CARD", "CHALLENGE", "PASS"]
  }
}
```

#### 9.4.9 质疑结果
```json
{
  "type": "CHALLENGE_RESULT",
  "payload": {
    "challenger_id": 1,
    "target_id": 2,
    "success": true,
    "actual_cards": ["K", "Q"],
    "claimed_card": "K",
    "loser_id": 2
  }
}
```

#### 9.4.10 俄罗斯轮盘
```json
{
  "type": "RUSSIAN_ROULETTE",
  "payload": {
    "player_id": 2,
    "bullet_count": 2,
    "survived": true,
    "tor_reduced": false,
    "tor_immune": false,
    "failed_challenge": true
  }
}
```

#### 9.4.11 玩家淘汰
```json
{
  "type": "PLAYER_ELIMINATED",
  "payload": {
    "player_id": 2
  }
}
```

#### 9.4.12 游戏结束
```json
{
  "type": "GAME_OVER",
  "payload": {
    "winner_id": 1
  }
}
```

#### 9.4.13 聊天消息
```json
{
  "type": "CHAT",
  "payload": {
    "sender_id": 2,
    "sender_name": "玩家2",
    "content": "我没说谎！",
    "is_ai": false
  }
}
```

#### 9.4.14 技能使用
```json
{
  "type": "SKILL_USED",
  "payload": {
    "player_id": 1,
    "character_id": "foxy",
    "skill": "foxy_peek"
  }
}
```

#### 9.4.15 技能结果（私有消息）
```json
{
  "type": "SKILL_RESULT",
  "payload": {
    "skill": "foxy_peek",
    "target_player_id": 3,
    "hand": ["A", "K", "Q"],
    "duration_ms": 3000
  }
}
```

#### 9.4.16 错误消息
```json
{
  "type": "ERROR",
  "payload": {
    "msg": "not your turn"
  }
}
```

---

## 10. 游戏流程

### 10.1 完整游戏流程

1. **登录** → 调用 `POST /auth/login` 获取token
2. **连接WebSocket** → `ws://server/ws?token=xxx`
3. **开始匹配** → 调用 `POST /match/start`
4. **等待匹配** → 收到 `MATCH_FOUND` 消息
5. **加入房间** → 发送 `PLAYER_JOIN` 消息
6. **选择角色** → 发送 `SET_CHARACTER` 消息
7. **准备** → 发送 `PLAYER_READY` 消息
8. **游戏开始** → 收到 `GAME_STARTED` 消息
9. **游戏进行中**:
   - 收到 `GAME_STATE` 查看游戏状态
   - 轮到自己时发送 `PLAY_CARD`/`CHALLENGE`/`PASS`
   - 收到 `CHALLENGE_RESULT`/`RUSSIAN_ROULETTE`/`PLAYER_ELIMINATED`
10. **游戏结束** → 收到 `GAME_OVER` 消息

### 10.2 房间自建流程

1. **登录** → `POST /auth/login`
2. **创建房间** → `POST /rooms`
3. **连接WebSocket** → `ws://server/ws?token=xxx`
4. **发送PLAYER_JOIN** → 加入自己的房间
5. **等待其他玩家加入** → 收到 `PLAYER_JOINED`
6. **所有人准备** → 发送 `PLAYER_READY`
7. **游戏开始** → 4人准备后自动开始

---

## 11. 错误码

| code | 说明 |
|------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（token无效或过期） |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 12. 游戏规则

### 12.1 基本规则

- 4名玩家，每人初始6张手牌（Scubby为7张）
- 牌型：A、K、Q、J，每种13张，共52张
- WILD牌：万能牌，可以当作任何牌型
- 每轮有目标牌型，玩家出牌时声称出的是目标牌
- 出牌后下一位玩家可以选择：跟牌、质疑、跳过
- 质疑成功：说谎者受罚
- 质疑失败：质疑者受罚
- 惩罚：俄罗斯轮盘（1/6概率淘汰）
- 最后存活者获胜

### 12.2 角色技能

- **Scubby**: 起手多1张WILD牌
- **Foxy**: 每局可偷看一次其他玩家手牌（3秒）
- **Bristle**: 每轮可质疑2次（其他角色只能1次）
- **Tor**: 失败惩罚时50%概率减少1发子弹或完全免疫

---

## 13. 开发建议

### 13.1 WebSocket心跳

建议每30秒发送一次ping保持连接：
```javascript
setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: "PING", payload: {} }));
  }
}, 30000);
```

### 13.2 断线重连

网络断开后自动重连并发送 `RECONNECT` 消息：
```javascript
ws.onclose = () => {
  setTimeout(() => {
    reconnect();
  }, 3000);
};

function reconnect() {
  const newWs = new WebSocket('ws://server/ws?token=' + token);
  newWs.onopen = () => {
    newWs.send(JSON.stringify({ type: "RECONNECT", payload: {} }));
  };
}
```

### 13.3 状态同步

进入游戏界面时先调用 `GET /user/status` 检查是否在游戏中，如果是则连接WebSocket并发送 `RECONNECT`。

---

## 14. 测试环境

**服务器地址:** `http://your-server:8081`

**测试账号:**
```
用户名: test1  密码: 123456
用户名: test2  密码: 123456
用户名: test3  密码: 123456
用户名: test4  密码: 123456
```

---

## 15. 更新日志

### v1.1.0 (2026-07-07)
- ✅ 新增历史战绩API (`GET /history`, `GET /games/:id`)
- ✅ 新增用户状态查询API (`GET /user/status`)
- ✅ 新增游戏统计数据API (`GET /user/stats`)
- ✅ 修复大厅队列长度显示为真实值

### v1.0.0 (2026-06-01)
- 初始版本发布
- 完整游戏逻辑实现
- WebSocket实时对战
- 匹配系统
- 房间系统
- 4个角色技能

---

## 联系方式

如有问题请联系服务端开发团队。
