package service

import (
	"solo/pkg/dto"
	"solo/services/user/repository"
)

type FilterService struct {
	repo *repository.FilterRepository
}

func NewFilterService(repo *repository.FilterRepository) *FilterService {
	return &FilterService{repo: repo}
}

// 유저의 매칭 필터 업데이트
func (s *FilterService) InsertDefaultMatchFilter(userID int, filter dto.MatchFilterDTO) error {
	filterModel := repository.MatchFilter{
		UserID:          userID,
		CoupleCount:     4,
		AddressRangeUse: false,
		AgeGroupUse:     false,
	}
	return s.repo.UpsertMatchFilter(filterModel)
}

// 유저 ID로 매칭 필터 조회
func (s *FilterService) GetMatchFilterByUserID(userID int) (*dto.MatchFilterDTO, error) {
	filter, err := s.repo.GetMatchFilterByUserID(userID)
	if err != nil {
		return nil, err
	}
	if filter == nil {
		return nil, nil
	}

	return &dto.MatchFilterDTO{
		UserID:          filter.UserID,
		CoupleCount:     filter.CoupleCount,
		AddressRangeUse: filter.AddressRangeUse,
		AgeGroupUse:     filter.AgeGroupUse,
	}, nil
}

// 유저의 매칭 필터 업데이트
func (s *FilterService) UpdateMatchFilter(userID int, filter dto.MatchFilterDTO) error {
	filterModel := repository.MatchFilter{
		UserID:          userID,
		CoupleCount:     filter.CoupleCount,
		AddressRangeUse: filter.AddressRangeUse,
		AgeGroupUse:     filter.AgeGroupUse,
	}
	return s.repo.UpsertMatchFilter(filterModel)
}
