"""
Liar's Bar AI Service - Simple Rule-Based Agent
"""
import json
import random
from typing import List, Optional
from fastapi import FastAPI
from pydantic import BaseModel
import uvicorn
import os

app = FastAPI(title="Liar's Bar AI Service", version="1.0.0")

CARDS = ["A", "K", "Q", "J"]


class GameStateRequest(BaseModel):
    player_id: int
    hand: List[str]
    target_card: str
    current_player_id: int
    last_play: Optional[dict] = None
    players: List[dict]
    legal_actions: List[str]


class InferenceResponse(BaseModel):
    action: str
    card_ids: Optional[List[int]] = None
    message: Optional[str] = ""
    confidence: float = 1.0


def count_card(hand: List[str], target: str) -> int:
    """Count how many target cards in hand."""
    return sum(1 for c in hand if c == target)


def make_decision(state: GameStateRequest) -> dict:
    """Simple rule-based AI decision logic."""

    hand = state.hand
    target = state.target_card
    legal = state.legal_actions
    last_play = state.last_play

    # Challenge logic
    if "CHALLENGE" in legal and last_play:
        last_count = last_play.get("count", 0)
        last_player_hand = last_play.get("player_hand_count", 6)

        # Challenge if suspicious
        if last_count >= 3 or (last_count >= 2 and last_player_hand <= 3):
            return {
                "action": "CHALLENGE",
                "confidence": 0.7,
                "message": random.choice(["I don't believe you!", "You're lying!", "Show me!"])
            }

    # Play card logic
    if "PLAY_CARD" in legal:
        # Check if hand is empty
        if len(hand) == 0:
            return {
                "action": "PASS",
                "confidence": 1.0,
                "message": ""
            }

        matching_cards = count_card(hand, target)

        if matching_cards > 0:
            # Play truthfully
            count = min(matching_cards, random.choice([1, 2, 3]))
            card_ids = [i for i, c in enumerate(hand) if c == target][:count]

            return {
                "action": "PLAY_CARD",
                "card_ids": card_ids,
                "confidence": 0.9,
                "message": random.choice(["Here you go", "Easy", "My turn"])
            }
        else:
            # Forced to lie (hand not empty but no matching cards)
            count = random.choice([1, 2]) if len(hand) >= 2 else 1
            card_ids = list(range(count))

            return {
                "action": "PLAY_CARD",
                "card_ids": card_ids,
                "confidence": 0.5,
                "message": random.choice(["Let's see...", "Here", "My cards"])
            }

    # Default: pass
    return {
        "action": "PASS",
        "confidence": 0.6,
        "message": ""
    }


@app.post("/ai/infer", response_model=InferenceResponse)
async def infer(req: GameStateRequest):
    """AI inference endpoint."""
    result = make_decision(req)

    return InferenceResponse(
        action=result["action"],
        card_ids=result.get("card_ids"),
        message=result.get("message", ""),
        confidence=result["confidence"]
    )


@app.get("/ai/status")
async def status():
    return {
        "model_name": "RuleBased-LiarsBar",
        "version": "1.0.0",
        "deployed": True
    }


@app.get("/health")
async def health():
    return {"status": "healthy"}


if __name__ == "__main__":
    port = int(os.environ.get("AI_PORT", 8000))
    uvicorn.run(app, host="0.0.0.0", port=port)
