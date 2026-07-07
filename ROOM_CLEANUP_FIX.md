# 游戏结束后房间清理问题修复

## 问题描述

**现象:**
- 游戏结束后，房间没有被销毁
- 房间继续显示在大厅列表中，状态为"游戏中"
- 玩家无法加入这些"僵尸房间"
- 房间数量不断累积，占用内存

## 原因分析

### 1. 正常游戏结束
- `finalizeGameIfOver()` 只广播 `GAME_OVER` 消息
- 但**没有调用** `Hub.DestroyRoom()` 清理房间
- 房间永久保留在 `Hub.Rooms` map中

### 2. 玩家中途退出
- `handlePlayerLeave()` 会检查是否还有人类玩家
- 只有当**所有人类玩家都退出**时才销毁房间
- 如果还有1个人类玩家在，房间就不会被销毁
- 导致单人或少数玩家的房间永久存在

## 解决方案

### 修改1: `finalizeGameIfOver()` - 正常结束自动清理

**位置:** `backend/internal/websocket/room.go:436`

**修改内容:**
```go
// 游戏结束后延迟5秒销毁房间，让玩家有时间看结果
go func() {
    time.Sleep(5 * time.Second)
    r.Hub.DestroyRoom(r.ID)
}()
```

**效果:**
- 游戏正常结束后，5秒后自动销毁房间
- 给玩家足够时间查看胜负结果
- 避免房间累积

### 修改2: `handlePlayerLeave()` - 中途退出清理

**位置:** `backend/internal/websocket/room.go:257`

**修改前逻辑:**
```go
// 只有当所有人类玩家都退出时才销毁
if !hasHuman {
    r.Hub.DestroyRoom(r.ID)
}
```

**修改后逻辑:**
```go
// 玩家中途退出导致游戏结束，延迟5秒后销毁房间
go func() {
    time.Sleep(5 * time.Second)
    r.Hub.DestroyRoom(r.ID)
}()
```

**效果:**
- 任何玩家中途退出导致游戏结束时，都会清理房间
- 不再判断是否还有人类玩家
- 统一使用延迟销毁，让剩余玩家看到结果

## 修改的文件

```
backend/internal/websocket/room.go
```

## 测试验证

### 测试场景1: 正常游戏结束
1. 4名玩家完整玩完一局游戏
2. 最后一名玩家淘汰，产生胜者
3. **预期:** 广播 `GAME_OVER` 后，5秒后房间从大厅消失

### 测试场景2: 玩家中途退出
1. 4名玩家开始游戏
2. 游戏进行中，某个玩家断开连接或主动退出
3. **预期:** 广播 `PLAYER_LEFT` (game_over=true)，5秒后房间从大厅消失

### 测试场景3: 多个房间同时结束
1. 同时有多个房间的游戏结束
2. **预期:** 所有房间都会在5秒后自动清理，大厅列表正确更新

### 验证命令

**查看活跃房间:**
```bash
curl http://localhost:8081/api/v1/lobby \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果:**
- 游戏结束前: `"active_rooms": N`
- 游戏结束5秒后: `"active_rooms": N-1`

## 性能影响

### 内存占用
- **修复前:** 每局游戏产生的房间永久保留，内存持续增长
- **修复后:** 游戏结束5秒后立即释放，内存稳定

### 资源计算
假设平均每局游戏15分钟：
- **修复前:** 1小时产生4个僵尸房间，1天96个，1周672个
- **修复后:** 最多同时存在的房间 = 正在进行的游戏数量

## 客户端影响

### 不需要客户端改动
- 客户端仍然接收 `GAME_OVER` 消息
- 5秒延迟对用户体验无影响（正常看结果的时间）
- 如果玩家停留在结算页面超过5秒，房间已销毁，但不影响显示

### 客户端最佳实践
收到 `GAME_OVER` 后：
1. 显示游戏结果页面
2. 用户点击"返回大厅"或等待自动跳转
3. 断开WebSocket连接（或保持连接但离开房间）
4. 重新调用 `GET /lobby` 获取最新大厅数据

## 后续优化建议

### 可配置的延迟时间
```go
const gameOverDelay = 5 * time.Second  // 可以在config中配置
```

### 主动通知房间销毁
可以在销毁前广播一个 `ROOM_CLOSING` 消息：
```go
r.broadcast(Message{
    Type: "ROOM_CLOSING",
    Payload: map[string]interface{}{
        "reason": "game_over",
        "delay_seconds": 5,
    },
})
```

### 房间回收统计
添加日志记录房间生命周期：
```go
log.Printf("Room %d lifecycle: created=%s, started=%s, ended=%s, duration=%s",
    r.ID, createTime, startTime, endTime, duration)
```

## 回归测试

确认修改不影响其他功能：
- ✅ 匹配系统正常工作
- ✅ 自建房间正常工作
- ✅ 游戏流程完整
- ✅ 断线重连功能正常
- ✅ 历史战绩正确记录
- ✅ 统计数据正确更新

## 部署建议

1. 先在测试环境验证
2. 观察内存使用是否稳定
3. 确认大厅房间列表正确更新
4. 验证多局游戏后系统稳定
5. 生产环境灰度发布

---

**修复时间:** 2026-07-07  
**影响范围:** 房间生命周期管理  
**优先级:** 高（影响用户体验和系统稳定性）
