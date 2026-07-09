package service

import (
	"fmt"

	"liars-bar/internal/model"
	"liars-bar/internal/repository"

	"github.com/google/uuid"
)

type RoomService struct {
	repo *repository.RoomRepo
}

func NewRoomService() *RoomService {
	return &RoomService{repo: repository.NewRoomRepo()}
}

func (s *RoomService) CreateRoom(hostID uint, name string) (*model.Room, error) {
	room := &model.Room{
		RoomUUID:       uuid.New().String(),
		HostUserID:     hostID,
		RoomName:       name,
		MaxPlayers:     4,
		CurrentPlayers: 1,
		RoomStatus:     model.RoomStatusWaiting,
	}
	if err := s.repo.Create(room); err != nil {
		return nil, err
	}
	s.repo.AddPlayer(room.ID, hostID, 0, false)
	return room, nil
}

func (s *RoomService) GetRoom(id uint) (*model.Room, error) {
	return s.repo.FindByID(id)
}

func (s *RoomService) GetRoomByUUID(uuid string) (*model.Room, error) {
	return s.repo.FindByUUID(uuid)
}

func (s *RoomService) ListActiveRooms() ([]model.Room, error) {
	return s.repo.FindActive()
}

func (s *RoomService) JoinRoom(roomID, userID uint) error {
	room, err := s.repo.FindByID(roomID)
	if err != nil {
		return err
	}
	if room.RoomStatus == model.RoomStatusPlaying || room.RoomStatus == model.RoomStatusFinished {
		return fmt.Errorf("room is not joinable")
	}

	players, _ := s.repo.GetPlayers(roomID)
	usedSeats := make(map[int]bool, len(players))
	for _, p := range players {
		if p.UserID == userID {
			room.CurrentPlayers = len(players)
			return s.repo.Update(room)
		}
		usedSeats[p.SeatIndex] = true
	}
	if len(players) >= room.MaxPlayers {
		return fmt.Errorf("room is full")
	}

	seat := 0
	for i := 0; i < room.MaxPlayers; i++ {
		if !usedSeats[i] {
			seat = i
			break
		}
	}
	if err := s.repo.AddPlayer(roomID, userID, seat, false); err != nil {
		return err
	}
	room.CurrentPlayers = len(players) + 1
	return s.repo.Update(room)
}

func (s *RoomService) LeaveRoom(roomID, userID uint) error {
	if err := s.repo.RemovePlayer(roomID, userID); err != nil {
		return err
	}
	room, _ := s.repo.FindByID(roomID)
	if room == nil || room.ID == 0 {
		return nil
	}
	players, _ := s.repo.GetPlayers(roomID)
	if len(players) == 0 {
		return s.repo.Delete(roomID)
	}
	room.CurrentPlayers = len(players)
	return s.repo.Update(room)
}

func (s *RoomService) GetPlayers(roomID uint) ([]model.RoomPlayer, error) {
	return s.repo.GetPlayers(roomID)
}

func (s *RoomService) UpdateStatus(roomID uint, status model.RoomStatus) error {
	return s.repo.UpdateStatus(roomID, status)
}

func (s *RoomService) DeleteRoom(roomID uint) error {
	return s.repo.Delete(roomID)
}

func (s *RoomService) CleanupStaleRooms(maxAgeMinutes int) (int, error) {
	return s.repo.DeleteStaleRooms(maxAgeMinutes)
}
