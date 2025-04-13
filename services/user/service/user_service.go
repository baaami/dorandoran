package service

import (
	"log"
	"solo/pkg/dto"
	"solo/pkg/models"
	"solo/services/user/repository"
)

type UserService struct {
	repo  *repository.UserRepository
	frepo *repository.FilterRepository
}

func NewUserService(repo *repository.UserRepository, frepo *repository.FilterRepository) *UserService {
	return &UserService{repo: repo, frepo: frepo}
}

// 유저 리스트 조회
func (s *UserService) GetUserList() ([]dto.UserDTO, error) {
	users, err := s.repo.GetUserList()
	if err != nil {
		return nil, err
	}

	var userDTOs []dto.UserDTO
	for _, user := range users {
		userDTOs = append(userDTOs, dto.UserDTO{
			ID:         user.ID,
			SnsType:    user.SnsType,
			SnsID:      user.SnsID,
			Name:       user.Name,
			Gender:     user.Gender,
			Birth:      user.Birth,
			Address:    dto.AddressDTO(user.Address),
			GameStatus: user.GameStatus,
			GameRoomID: user.GameRoomID,
			GamePoint:  user.GamePoint,
			Alert:      user.Alert,
		})
	}

	return userDTOs, nil
}

// 특정 유저 조회
func (s *UserService) GetUserByID(id int) (*dto.UserDTO, error) {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	return &dto.UserDTO{
		ID:         user.ID,
		SnsType:    user.SnsType,
		SnsID:      user.SnsID,
		Name:       user.Name,
		Gender:     user.Gender,
		Birth:      user.Birth,
		Address:    dto.AddressDTO(user.Address),
		GameStatus: user.GameStatus,
		GameRoomID: user.GameRoomID,
		GamePoint:  user.GamePoint,
		Alert:      user.Alert,
	}, nil
}

// 특정 유저 조회
func (s *UserService) GetUserBySNS(snsType int, snsID string) (*dto.UserDTO, error) {
	user, err := s.repo.GetUserBySNS(snsType, snsID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	return &dto.UserDTO{
		ID:         user.ID,
		SnsType:    user.SnsType,
		SnsID:      user.SnsID,
		Name:       user.Name,
		Gender:     user.Gender,
		Birth:      user.Birth,
		Address:    dto.AddressDTO(user.Address),
		GameStatus: user.GameStatus,
		GameRoomID: user.GameRoomID,
		GamePoint:  user.GamePoint,
		Alert:      user.Alert,
	}, nil
}

// 유저 알람 허용 여부 조회
func (s *UserService) GetUserAlert(id int) (bool, error) {
	alert, err := s.repo.GetUserAlert(id)
	if err != nil {
		log.Printf("❌ Failed to get user alert by ID %d: %v", id, err)
		return false, err
	}
	return alert, nil
}

// 유저 등록
func (s *UserService) RegisterUser(user dto.UserDTO) (*dto.UserDTO, error) {
	userModel := models.User{
		ID:         user.ID,
		SnsType:    user.SnsType,
		SnsID:      user.SnsID,
		Name:       user.Name,
		Gender:     user.Gender,
		Birth:      user.Birth,
		Address:    models.Address(user.Address),
		GameStatus: user.GameStatus,
		GameRoomID: user.GameRoomID,
		GamePoint:  user.GamePoint,
		Alert:      user.Alert,
	}
	id, err := s.repo.InsertUser(userModel)
	if err != nil {
		return nil, err
	}
	user.ID = id

	// TODO: 이후 사용자가 반드시 입렫하도록 수정 or User 필드에 합치기
	filter := models.MatchFilter{
		UserID:          user.ID,
		CoupleCount:     4,
		AddressRangeUse: false,
		AgeGroupUse:     false,
	}
	s.frepo.UpsertMatchFilter(filter)

	return &user, nil
}

// 유저 업데이트
func (s *UserService) UpdateUser(user dto.UserDTO) error {
	userModel := models.User{
		ID:      user.ID,
		Name:    user.Name,
		Gender:  user.Gender,
		Birth:   user.Birth,
		Address: models.Address(user.Address),
		Alert:   user.Alert,
	}
	return s.repo.UpdateUser(userModel)
}

// 유저 업데이트
func (s *UserService) UpdateUserGameInfo(userID int, newStatus int, gameRoomID string) error {
	return s.repo.UpdateUserGameInfo(userID, newStatus, gameRoomID)
}

// 유저 삭제
func (s *UserService) DeleteUser(id int) error {
	return s.repo.DeleteUser(id)
}
