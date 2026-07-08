<template>
  <div class="game-room" :class="{ 'screen-shake': shakeScreen }">
    <!-- Bar background layers -->
    <div class="bar-bg">
      <div class="bar-bg__wall"></div>
      <div class="bar-bg__vignette"></div>
      <div class="bar-bg__lamp"></div>
    </div>

    <!-- Effect overlays -->
    <Transition name="effect-pop">
      <div v-if="challengeFx.show" class="fx-overlay fx-challenge" :class="challengeFx.success ? 'fx-success' : 'fx-fail'">
        <div class="fx-burst"></div>
        <div class="fx-title">{{ challengeFx.success ? '质疑成功!' : '质疑失败' }}</div>
        <div class="fx-sub">{{ challengeFx.success ? '对方在撒谎' : '对方说真话' }}</div>
      </div>
    </Transition>

    <Transition name="effect-pop">
      <div v-if="eliminateFx.show" class="fx-overlay fx-eliminate">
        <div class="fx-skull">💀</div>
        <div class="fx-title">玩家淘汰</div>
        <div class="fx-sub">{{ eliminateFx.nickname }} 被击杀</div>
      </div>
    </Transition>

    <!-- Top bar -->
    <div class="top-bar">
      <button class="back-btn" @click="leaveRoom">← 返回大厅</button>
      <div class="room-info">
        <span class="round-badge">第 {{ gameState?.current_round || 1 }} 轮</span>
        <span class="turn-badge">回合 {{ gameState?.current_turn || 0 }}</span>
        <span class="target-badge">目标牌: {{ gameState?.target_card || '-' }}</span>
      </div>
      <div class="alive-count">{{ gameState?.alive_count || 4 }} 人存活</div>
      <button class="rules-btn" @click="showRules = true">📖 规则</button>
    </div>

    <!-- Game Rules modal -->
    <div v-if="showRules" class="rules-overlay" @click.self="showRules = false">
      <div class="rules-modal">
        <button class="rules-close" @click="showRules = false">×</button>
        <h2>📖 骗子酒馆 规则</h2>
        <div class="rules-body">
          <section>
            <h3>目标</h3>
            <p>4 人对战，通过出牌、撒谎、质疑和心理博弈，成为最后存活的玩家。</p>
          </section>
          <section>
            <h3>牌组</h3>
            <p>A / K / Q / J 各 6 张，共 24 张。每人发 6 张。</p>
          </section>
          <section>
            <h3>目标牌</h3>
            <p>每一轮有指定目标牌（A→K→Q→J 循环）。出牌时必须声明打出的全部是当前目标牌。</p>
          </section>
          <section>
            <h3>出牌</h3>
            <p>当前回合玩家选 1~3 张手牌打出。可以真出（全是目标牌），也可以虚张声势（含非目标牌）。其他玩家只能看到声明，看不到真实牌面。</p>
          </section>
          <section>
            <h3>质疑</h3>
            <p>下一名玩家可选择质疑上家。系统翻开上家刚打出的牌：</p>
            <ul>
              <li>含非目标牌（撒谎）→ 质疑成功，撒谎者受罚</li>
              <li>全是目标牌（说真话）→ 质疑失败，质疑者受罚</li>
            </ul>
          </section>
          <section>
            <h3>俄罗斯轮盘惩罚</h3>
            <p>每次受罚累计子弹数：第 1 次 1 发，第 2 次 2 发……第 6 次必死。被击中即淘汰出局，存活则继续。</p>
          </section>
          <section>
            <h3>手牌耗尽</h3>
            <p>所有存活玩家手牌为空时，重新洗牌发牌，目标牌切换到下一种。</p>
          </section>
          <section>
            <h3>操作</h3>
            <ul>
              <li><b>出牌</b>：选 1~3 张牌，点击「出牌」</li>
              <li><b>质疑上家</b>：仅当上家有出牌时可点</li>
              <li><b>过</b>：不质疑，交由下家</li>
            </ul>
          </section>
        </div>
      </div>
    </div>

    <!-- Game Over overlay -->
    <div v-if="gameState?.phase === 'GAME_OVER'" class="game-over-overlay">
      <div class="game-over-card">
        <h1 v-if="leaveReason">🚪 {{ leaveReason }}</h1>
        <h1 v-else>🏆 游戏结束</h1>
        <p v-if="leaveReason" class="lose-text">{{ leaveDetail }}</p>
        <p v-else-if="gameState?.winner_id === authStore.user?.id" class="win-text">你赢了！</p>
        <p v-else class="lose-text">
          玩家 {{ winnerName }} 获得了胜利
        </p>
        <button @click="leaveRoom">返回大厅</button>
      </div>
    </div>

    <!-- Game area -->
    <div v-if="gameState && gameState.phase !== 'GAME_OVER'" class="game-area">
      <!-- Opponents -->
      <div class="opponents-row">
        <div
          v-for="player in opponents"
          :key="player.id"
          class="player-card"
          :class="{ active: gameState.current_player === player.seat_index, eliminated: !player.is_alive }"
        >
          <div class="player-avatar">
            <span v-if="player.is_ai">🤖</span>
            <span v-else>👤</span>
          </div>
          <div class="player-name">{{ player.nickname }}</div>
          <div class="player-hp">
            <span v-for="i in 6" :key="i" class="hp-dot" :class="{ filled: i <= (player.punishment_count || 0) }"></span>
          </div>
          <div class="card-count">{{ player.hand_count }} 张牌</div>
          <div v-if="player.is_ai" class="ai-tag">AI</div>
          <div v-if="!player.is_alive" class="dead-tag">💀</div>
        </div>
      </div>

      <!-- Center area - last play -->
      <div class="center-area">
        <div v-if="gameState.last_play" class="last-play">
          <div class="last-play-label">上家出牌</div>
          <div class="last-play-cards">
            <div v-for="i in gameState.last_play.count" :key="i" class="mini-card">{{ gameState.last_play.claim }}</div>
          </div>
          <div class="last-play-player">玩家 {{ lastPlayPlayerName }}</div>
        </div>
        <div v-else class="no-play">等待出牌...</div>
      </div>

      <!-- My hand -->
      <div class="my-hand">
        <div class="hand-label">我的手牌</div>
        <div class="cards-row" v-if="myHand && myHand.length > 0">
          <div
            v-for="(card, idx) in myHand"
            :key="idx"
            class="card"
            :class="{ selected: selectedCards.includes(idx) }"
            @click="toggleCard(idx)"
          >
            <div class="card-value">{{ card }}</div>
            <div class="card-suit">♠</div>
          </div>
        </div>
        <div v-else class="no-cards">无手牌</div>
      </div>

      <!-- Action buttons -->
      <div class="actions" v-if="isMyTurn">
        <p class="action-hint">
          已选择 {{ selectedCards.length }} 张牌（1-3张）
        </p>
        <div class="action-row">
          <button
            class="action-btn play-btn"
            :disabled="selectedCards.length === 0"
            @click="playCards"
          >
            出牌 (声称 {{ gameState?.target_card || '-' }})
          </button>
          <button
            class="action-btn challenge-btn"
            v-if="gameState?.last_play && gameState.last_play.player_id !== myPlayerId"
            @click="challenge"
          >
            质疑上家
          </button>
          <button class="action-btn pass-btn" @click="passTurn">
            过
          </button>
        </div>
      </div>
      <div v-else class="waiting-turn">等待其他玩家操作...</div>
    </div>

    <!-- Chat area -->
    <div class="chat-panel">
      <h3>聊天</h3>
      <div class="chat-messages" ref="chatRef">
        <div v-for="(msg, idx) in chatMessages" :key="idx" class="chat-msg">
          <span class="chat-sender" :class="{ ai: msg.is_ai }">
            {{ msg.sender_name || '玩家' + msg.sender_id }}
          </span>
          : {{ msg.content }}
        </div>
        <div v-if="chatMessages.length === 0" class="chat-empty">暂无消息</div>
      </div>
      <div class="chat-input-row">
        <input
          v-model="chatText"
          @keyup.enter="sendChat"
          placeholder="输入消息..."
          maxlength="200"
        />
        <button @click="sendChat">发送</button>
      </div>
    </div>

    <!-- Waiting room state -->
    <div v-if="!gameState && !connecting" class="waiting-room">
      <div class="wait-card">
        <div class="wait-title">{{ roomState?.name || '游戏房间' }}</div>
        <div class="wait-count">
          {{ roomPlayers.length }}/{{ roomState?.max_players || 4 }} 玩家 · {{ roomState?.ready_count || 0 }} 已准备
        </div>

        <div class="waiting-players">
          <div
            v-for="seat in waitingSeats"
            :key="seat.index"
            class="waiting-player"
            :class="{ empty: !seat.player, ready: seat.player?.is_ready }"
          >
            <div class="waiting-avatar">{{ seat.player ? (seat.player.is_ai ? '🤖' : '👤') : '-' }}</div>
            <div class="waiting-name">{{ seat.player?.nickname || '等待加入' }}</div>
            <div class="waiting-status">
              {{ seat.player ? (seat.player.is_ready ? '已准备' : '未准备') : '空位' }}
            </div>
          </div>
        </div>

        <button class="ready-btn" :disabled="myRoomPlayer?.is_ready" @click="setReady">
          {{ myRoomPlayer?.is_ready ? '已准备' : '准备' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { roomAPI } from '../api'
import wsClient from '../ws/client'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const gameState = ref(null)
const roomState = ref(null)
const chatMessages = ref([])
const chatText = ref('')
const selectedCards = ref([])
const connecting = ref(true)
const chatRef = ref(null)
const showRules = ref(false)
const leaveReason = ref('')
const leaveDetail = ref('')

// Effect overlays
const challengeFx = ref({ show: false, success: false })
const eliminateFx = ref({ show: false, nickname: '' })
const shakeScreen = ref(false)
let challengeFxTimer = null
let eliminateFxTimer = null
let shakeTimer = null

function triggerChallengeFx(success) {
  challengeFx.value = { show: true, success }
  if (challengeFxTimer) clearTimeout(challengeFxTimer)
  challengeFxTimer = setTimeout(() => { challengeFx.value.show = false }, 1600)
}

function triggerEliminateFx(playerId) {
  const p = gameState.value?.players?.find(x => x.id === playerId)
  eliminateFx.value = { show: true, nickname: p?.nickname || `玩家${playerId}` }
  if (eliminateFxTimer) clearTimeout(eliminateFxTimer)
  eliminateFxTimer = setTimeout(() => { eliminateFx.value.show = false }, 1800)
  // screen shake
  shakeScreen.value = true
  if (shakeTimer) clearTimeout(shakeTimer)
  shakeTimer = setTimeout(() => { shakeScreen.value = false }, 500)
}

const myPlayerId = computed(() => authStore.user?.id)
const myPlayer = computed(() => {
  if (!gameState.value?.players) return null
  return gameState.value.players.find(p => p.id === myPlayerId.value)
})
const myHand = computed(() => myPlayer.value?.hand || [])

const roomPlayers = computed(() => roomState.value?.players || [])
const myRoomPlayer = computed(() => roomPlayers.value.find(p => p.id === myPlayerId.value))
const waitingSeats = computed(() => {
  const playersBySeat = new Map(roomPlayers.value.map(p => [p.seat_index, p]))
  return Array.from({ length: roomState.value?.max_players || 4 }, (_, index) => ({
    index,
    player: playersBySeat.get(index),
  }))
})

const opponents = computed(() => {
  if (!gameState.value?.players) return []
  return gameState.value.players.filter(p => p.id !== myPlayerId.value)
})

const isMyTurn = computed(() => {
  if (!gameState.value || !myPlayer.value) return false
  const currentIdx = gameState.value.current_player
  const myIdx = myPlayer.value.seat_index
  return currentIdx === myIdx && gameState.value.phase === 'PLAYING'
})

const lastPlayPlayerName = computed(() => {
  if (!gameState.value?.last_play || !gameState.value?.players) return '?'
  const player = gameState.value.players.find(p => p.id === gameState.value.last_play.player_id)
  return player?.nickname || '?'
})

const winnerName = computed(() => {
  if (!gameState.value?.winner_id || !gameState.value?.players) return '?'
  const player = gameState.value.players.find(p => p.id === gameState.value.winner_id)
  return player?.nickname || '?'
})

onMounted(async () => {
  setupListeners()
  const roomId = Number(route.params.id)
  try {
    await roomAPI.join(roomId)
    wsClient.connect()
    wsClient.send('PLAYER_JOIN', { room_id: roomId })
  } catch (e) {
    addSystemMsg(e.response?.data?.msg || '加入房间失败')
    router.push('/')
  } finally {
    setTimeout(() => { connecting.value = false }, 2000)
  }
})

onUnmounted(() => {
  wsClient.off('GAME_STATE', onGameState)
  wsClient.off('GAME_STARTED', onGameStarted)
  wsClient.off('ROOM_STATE', onRoomState)
  wsClient.off('ERROR', onError)
  wsClient.off('CHALLENGE_RESULT', onChallengeResult)
  wsClient.off('PLAYER_ELIMINATED', onEliminated)
  wsClient.off('GAME_OVER', onGameOver)
  wsClient.off('PLAYER_LEFT', onPlayerLeft)
  wsClient.off('CHAT', onChat)
  wsClient.off('RUSSIAN_ROULETTE', onRoulette)
})

function setupListeners() {
  wsClient.on('GAME_STATE', onGameState)
  wsClient.on('GAME_STARTED', onGameStarted)
  wsClient.on('ROOM_STATE', onRoomState)
  wsClient.on('ERROR', onError)
  wsClient.on('CHALLENGE_RESULT', onChallengeResult)
  wsClient.on('PLAYER_ELIMINATED', onEliminated)
  wsClient.on('GAME_OVER', onGameOver)
  wsClient.on('PLAYER_LEFT', onPlayerLeft)
  wsClient.on('CHAT', onChat)
  wsClient.on('RUSSIAN_ROULETTE', onRoulette)
}

function onGameState(payload) {
  gameState.value = payload
  resetSelection()
}

function onGameStarted(payload) {
  gameState.value = payload
  connecting.value = false
}

function onRoomState(payload) {
  roomState.value = payload
  connecting.value = false
}

function onError(payload) {
  addSystemMsg(payload?.msg || '操作失败')
}

function onChallengeResult(payload) {
  const verb = payload.success ? '成功！' : '失败...'
  const cards = payload.challenged_cards || []
  const cardStr = cards.length > 0 ? ` 实际打出的牌: ${cards.join(', ')}` : ''

  if (payload.success) {
    addSystemMsg(`质疑${verb} 玩家 ${payload.liar_id} 说谎了！${cardStr}`)
  } else {
    addSystemMsg(`质疑${verb} 玩家 ${payload.liar_id} 说的是真话。${cardStr}`)
  }
  triggerChallengeFx(payload.success)
}

function onRoulette(payload) {
  if (payload.survived) {
    addSystemMsg(`玩家 ${payload.player_id} 俄罗斯轮盘存活！`)
  } else {
    addSystemMsg(`玩家 ${payload.player_id} 在俄罗斯轮盘中被淘汰！`)
  }
}

function onEliminated(payload) {
  addSystemMsg(`玩家 ${payload.player_id} 被淘汰`)
  triggerEliminateFx(payload.player_id)
}

function onGameOver(payload) {
  gameState.value = {
    ...(gameState.value || {}),
    phase: 'GAME_OVER',
    winner_id: payload.winner_id,
  }
  addSystemMsg(`游戏结束！胜利者: ${payload.winner_id}`)
}

function onPlayerLeft(payload) {
  if (payload?.game_over) {
    // A player left mid-game → game abandoned.
    leaveReason.value = payload.reason || '玩家退出，游戏结束'
    const name = payload.nickname || `玩家${payload.player_id}`
    leaveDetail.value = `${name} 退出了对局，本局已结束`
    gameState.value = {
      ...(gameState.value || {}),
      phase: 'GAME_OVER',
      winner_id: payload.winner_id,
    }
    addSystemMsg(leaveDetail.value)
  } else {
    addSystemMsg(`玩家 ${payload.nickname || payload.player_id} 离开了房间`)
  }
}

function onChat(payload) {
  chatMessages.value.push({
    sender_id: payload.sender_id,
    sender_name: payload.sender_name || `玩家${payload.sender_id}`,
    content: payload.content,
    is_ai: payload.is_ai,
  })
  nextTick(() => {
    if (chatRef.value) {
      chatRef.value.scrollTop = chatRef.value.scrollHeight
    }
  })
}

function addSystemMsg(content) {
  chatMessages.value.push({
    sender_id: 0,
    sender_name: '[系统]',
    content,
    is_ai: false,
  })
}

function toggleCard(idx) {
  if (!isMyTurn.value) return
  const pos = selectedCards.value.indexOf(idx)
  if (pos >= 0) {
    selectedCards.value.splice(pos, 1)
  } else if (selectedCards.value.length < 3) {
    selectedCards.value.push(idx)
  }
}

function resetSelection() {
  selectedCards.value = []
}

function playCards() {
  if (selectedCards.value.length === 0) return
  wsClient.send('PLAY_CARD', {
    card_ids: selectedCards.value,
    claim: gameState.value.target_card,
  })
  resetSelection()
}

function challenge() {
  const targetId = gameState.value?.last_play?.player_id
  wsClient.send('CHALLENGE', { target_player_id: targetId })
}

function passTurn() {
  wsClient.send('PASS')
  resetSelection()
}

function setReady() {
  wsClient.send('PLAYER_READY')
}

function sendChat() {
  if (!chatText.value.trim()) return
  wsClient.send('CHAT', { content: chatText.value.trim() })
  chatText.value = ''
}

async function leaveRoom() {
  try {
    await roomAPI.leave(Number(route.params.id))
  } catch (e) {
    console.error('Leave room failed:', e)
  }
  router.push('/')
}
</script>

<style scoped>
.game-room {
  min-height: 100vh;
  position: relative;
  display: flex;
  flex-direction: column;
  background: #0d0a08;
  overflow: hidden;
}

/* ---- Bar background layers ---- */
.bar-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  overflow: hidden;
}
.bar-bg__wall {
  position: absolute;
  inset: 0;
  background:
    /* warm wall paneling stripes */
    repeating-linear-gradient(90deg,
      rgba(60, 36, 18, 0.55) 0px,
      rgba(45, 27, 14, 0.55) 2px,
      rgba(45, 27, 14, 0.55) 80px,
      rgba(60, 36, 18, 0.55) 82px,
      rgba(60, 36, 18, 0.55) 160px),
    /* vertical light gradient (top dim, mid warm) */
    radial-gradient(ellipse 120% 70% at 50% 40%,
      rgba(120, 70, 30, 0.45) 0%,
      rgba(60, 32, 14, 0.6) 35%,
      rgba(20, 12, 8, 0.95) 80%),
    linear-gradient(180deg, #1a1108 0%, #0d0805 100%);
}
.bar-bg__vignette {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse 80% 60% at 50% 55%,
    transparent 0%, transparent 40%, rgba(0,0,0,0.55) 100%);
}
/* hanging pendant lamp glow over the table */
.bar-bg__lamp {
  position: absolute;
  top: -10%;
  left: 50%;
  width: 60vw;
  height: 60vh;
  transform: translateX(-50%);
  background: radial-gradient(ellipse 50% 60% at 50% 0%,
    rgba(255, 180, 90, 0.32) 0%,
    rgba(220, 130, 50, 0.14) 30%,
    transparent 70%);
  filter: blur(8px);
  animation: lamp-flicker 6s ease-in-out infinite;
}
@keyframes lamp-flicker {
  0%, 100% { opacity: 1; }
  47% { opacity: 0.92; }
  50% { opacity: 0.78; }
  53% { opacity: 0.95; }
}

/* table surface felt, centered */
.game-room::before {
  content: '';
  position: fixed;
  left: 50%;
  top: 52%;
  width: min(1100px, 96vw);
  height: 60vh;
  transform: translate(-50%, -50%);
  background:
    radial-gradient(ellipse at center,
      #2e5d3a 0%, #24522f 45%, #173a23 75%, #0c2417 100%);
  border-radius: 50%;
  box-shadow:
    0 0 0 12px #3a2412,
    0 0 0 14px #1d1208,
    0 30px 80px rgba(0,0,0,0.7),
    inset 0 0 120px rgba(0,0,0,0.55);
  z-index: 0;
  pointer-events: none;
}

/* keep content above background */
.top-bar, .game-area, .chat-panel, .waiting-room, .game-over-overlay {
  position: relative;
  z-index: 2;
}

/* ---- Screen shake ---- */
.screen-shake {
  animation: shake 0.5s cubic-bezier(.36,.07,.19,.97) both;
}
@keyframes shake {
  10%, 90% { transform: translate3d(-2px, 0, 0); }
  20%, 80% { transform: translate3d(4px, 0, 0); }
  30%, 50%, 70% { transform: translate3d(-7px, 0, 0); }
  40%, 60% { transform: translate3d(7px, 0, 0); }
}

/* ---- Effect overlays ---- */
.fx-overlay {
  position: fixed;
  inset: 0;
  z-index: 90;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  text-align: center;
}
.fx-title {
  font-size: 52px;
  font-weight: 800;
  letter-spacing: 4px;
  text-shadow: 0 4px 24px rgba(0,0,0,0.8);
  margin-top: 12px;
}
.fx-sub {
  font-size: 20px;
  margin-top: 8px;
  opacity: 0.9;
}

/* challenge success: red flash + burst */
.fx-challenge.fx-success {
  background: radial-gradient(ellipse at center, rgba(233,69,96,0.45) 0%, transparent 60%);
  animation: fx-flash 0.6s ease-out;
}
.fx-challenge.fx-success .fx-title { color: #ff5577; }
.fx-challenge.fx-success .fx-burst {
  position: absolute;
  width: 320px; height: 320px;
  border-radius: 50%;
  border: 6px solid rgba(255, 90, 120, 0.85);
  animation: fx-ring 0.9s ease-out;
}
/* challenge fail: blueish */
.fx-challenge.fx-fail .fx-title { color: #7aa2ff; }
.fx-challenge.fx-fail {
  background: radial-gradient(ellipse at center, rgba(80,110,200,0.35) 0%, transparent 60%);
  animation: fx-flash 0.6s ease-out;
}

/* elimination: skull drop + dark pulse */
.fx-eliminate {
  background: radial-gradient(ellipse at center, rgba(0,0,0,0.7) 0%, rgba(0,0,0,0.4) 60%, transparent 100%);
  animation: fx-flash 0.6s ease-out;
}
.fx-eliminate .fx-title { color: #e94560; }
.fx-skull {
  font-size: 110px;
  animation: skull-drop 0.7s cubic-bezier(.18,.89,.32,1.28) both;
  filter: drop-shadow(0 0 20px rgba(233,69,96,0.7));
}

@keyframes fx-flash {
  0% { opacity: 0; }
  20% { opacity: 1; }
  100% { opacity: 1; }
}
@keyframes fx-ring {
  0% { transform: scale(0.2); opacity: 1; border-width: 12px; }
  100% { transform: scale(2.2); opacity: 0; border-width: 2px; }
}
@keyframes skull-drop {
  0% { transform: translateY(-120px) scale(0.4) rotate(-30deg); opacity: 0; }
  60% { transform: translateY(10px) scale(1.15) rotate(8deg); opacity: 1; }
  100% { transform: translateY(0) scale(1) rotate(0); opacity: 1; }
}

/* Transition for the overlays */
.effect-pop-enter-active, .effect-pop-leave-active {
  transition: opacity 0.25s ease;
}
.effect-pop-enter-from, .effect-pop-leave-to {
  opacity: 0;
}

.top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: rgba(0,0,0,0.3);
  border-bottom: 1px solid #333;
}

.back-btn {
  background: none;
  color: #888;
  padding: 6px 12px;
}

.room-info {
  display: flex;
  gap: 12px;
}

.round-badge, .turn-badge, .target-badge {
  background: rgba(233, 69, 96, 0.2);
  color: #e94560;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 13px;
}

.alive-count {
  color: #4ade80;
  font-size: 14px;
}

.rules-btn {
  background: rgba(233, 69, 96, 0.15);
  border: 1px solid #e94560;
  color: #e94560;
  padding: 6px 14px;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.rules-btn:hover {
  background: rgba(233, 69, 96, 0.3);
}

/* ---- Rules modal ---- */
.rules-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}
.rules-modal {
  position: relative;
  background: linear-gradient(160deg, #1f1610 0%, #2a1c12 100%);
  border: 1px solid #5a3a1e;
  border-radius: 14px;
  max-width: 520px;
  width: 100%;
  max-height: 85vh;
  overflow-y: auto;
  padding: 28px 28px 24px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.6), 0 0 0 1px rgba(233,69,96,0.2);
}
.rules-modal h2 {
  color: #e94560;
  font-size: 22px;
  margin-bottom: 16px;
  text-align: center;
}
.rules-close {
  position: absolute;
  top: 10px;
  right: 14px;
  background: none;
  border: none;
  color: #888;
  font-size: 28px;
  line-height: 1;
  cursor: pointer;
}
.rules-close:hover { color: #e94560; }
.rules-body section {
  margin-bottom: 14px;
  padding-bottom: 12px;
  border-bottom: 1px solid #3a2616;
}
.rules-body section:last-child {
  border-bottom: none;
  margin-bottom: 0;
}
.rules-body h3 {
  color: #f5a623;
  font-size: 15px;
  margin-bottom: 6px;
}
.rules-body p, .rules-body ul {
  color: #d0c8c0;
  font-size: 13px;
  line-height: 1.6;
  margin: 0;
}
.rules-body ul {
  padding-left: 18px;
}
.rules-body li {
  margin-bottom: 3px;
}
.rules-body b {
  color: #e94560;
}

.game-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 16px;
  max-width: 900px;
  width: 100%;
  margin: 0 auto;
}

.opponents-row {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.player-card {
  background: rgba(22, 33, 62, 0.8);
  border: 1px solid #333;
  border-radius: 12px;
  padding: 16px;
  text-align: center;
  min-width: 140px;
  transition: all 0.2s;
}

.player-card.active {
  border-color: #e94560;
  box-shadow: 0 0 12px rgba(233, 69, 96, 0.3);
}

.player-card.eliminated {
  opacity: 0.4;
}

.player-avatar { font-size: 32px; margin-bottom: 4px; }
.player-name { color: #e0e0e0; font-weight: 600; margin-bottom: 4px; }
.ai-tag { color: #e94560; font-size: 12px; }
.dead-tag { font-size: 24px; }

.player-hp { display: flex; gap: 4px; justify-content: center; margin: 8px 0; }
.hp-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  background: #333;
}
.hp-dot.filled { background: #e94560; }

.card-count { color: #888; font-size: 13px; }

.center-area {
  background: rgba(22, 33, 62, 0.6);
  border: 1px solid #333;
  border-radius: 12px;
  padding: 20px;
  text-align: center;
  margin-bottom: 16px;
  min-height: 100px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.last-play-label { color: #888; font-size: 13px; margin-bottom: 8px; }
.last-play-cards { display: flex; gap: 8px; justify-content: center; margin-bottom: 8px; }

.mini-card {
  background: #e94560;
  color: white;
  width: 40px; height: 56px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 18px;
}

.last-play-player { color: #888; font-size: 13px; }

.my-hand { margin-bottom: 16px; }
.hand-label { color: #888; margin-bottom: 8px; }
.cards-row { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }

.card {
  width: 64px; height: 90px;
  background: white;
  color: #1a1a2e;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
  border: 3px solid transparent;
}

.card:hover { transform: translateY(-4px); }
.card.selected {
  border-color: #e94560;
  transform: translateY(-8px);
  box-shadow: 0 4px 12px rgba(233, 69, 96, 0.3);
}

.card-value { font-size: 24px; font-weight: 700; }
.card-suit { font-size: 16px; color: #e94560; }

.actions { text-align: center; margin-bottom: 16px; }
.action-hint { color: #4ade80; margin-bottom: 8px; }
.action-row { display: flex; gap: 8px; justify-content: center; flex-wrap: wrap; }

.action-btn {
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 15px;
}

.play-btn { background: #4ade80; color: #1a1a2e; }
.play-btn:disabled { background: #333; color: #666; cursor: not-allowed; }
.challenge-btn { background: #e94560; color: white; }
.pass-btn { background: #333; color: #e0e0e0; }

.waiting-turn {
  text-align: center;
  color: #888;
  padding: 20px;
}

.chat-panel {
  background: rgba(0,0,0,0.3);
  border-top: 1px solid #333;
  padding: 12px 20px;
  max-height: 200px;
  display: flex;
  flex-direction: column;
}

.chat-panel h3 {
  color: #888;
  font-size: 14px;
  margin-bottom: 8px;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  margin-bottom: 8px;
  max-height: 100px;
}

.chat-msg {
  color: #aaa;
  font-size: 13px;
  padding: 2px 0;
}

.chat-sender { color: #e94560; font-weight: 600; }
.chat-sender.ai { color: #f59e0b; }

.chat-empty { color: #555; font-size: 13px; }

.chat-input-row {
  display: flex;
  gap: 8px;
}

.chat-input-row input {
  flex: 1;
  padding: 8px 12px;
  font-size: 14px;
}

.chat-input-row button {
  background: #e94560;
  color: white;
  padding: 8px 16px;
  font-size: 13px;
}

.waiting-room {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.wait-card {
  text-align: center;
  color: #888;
  width: min(760px, calc(100vw - 32px));
}

.wait-title {
  color: #e0e0e0;
  font-size: 22px;
  font-weight: 700;
  margin-bottom: 8px;
}

.wait-count {
  color: #888;
  margin-bottom: 18px;
}

.waiting-players {
  display: grid;
  grid-template-columns: repeat(4, minmax(120px, 1fr));
  gap: 12px;
}

.waiting-player {
  background: rgba(22, 33, 62, 0.8);
  border: 1px solid #333;
  border-radius: 8px;
  min-height: 126px;
  padding: 16px 10px;
}

.waiting-player.ready {
  border-color: #4ade80;
}

.waiting-player.empty {
  opacity: 0.55;
}

.waiting-avatar {
  font-size: 28px;
  margin-bottom: 8px;
}

.waiting-name {
  color: #e0e0e0;
  font-weight: 700;
  min-height: 22px;
  overflow-wrap: anywhere;
}

.waiting-status {
  color: #888;
  font-size: 13px;
  margin-top: 6px;
}

.ready-btn {
  margin-top: 18px;
  background: #4ade80;
  color: #1a1a2e;
}

.ready-btn:disabled {
  background: #333;
  color: #777;
  cursor: not-allowed;
}

@media (max-width: 700px) {
  .waiting-players {
    grid-template-columns: repeat(2, minmax(120px, 1fr));
  }
}

.game-over-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.game-over-card {
  text-align: center;
  padding: 40px;
}

.game-over-card h1 { font-size: 36px; margin-bottom: 16px; }
.win-text { color: #4ade80; font-size: 24px; margin-bottom: 24px; }
.lose-text { color: #f59e0b; font-size: 20px; margin-bottom: 24px; }

.game-over-card button {
  background: #e94560;
  color: white;
  padding: 14px 32px;
}

.no-play, .no-cards { color: #555; }
</style>
