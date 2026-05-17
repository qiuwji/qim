package dal

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FriendStore interface {
	CreateRequest(req *FriendRequest) error
	ListIncomingRequests(uid uint64) ([]FriendRequest, error)
	ListOutgoingRequests(uid uint64) ([]FriendRequest, error)
	GetRequest(id uint64) (*FriendRequest, error)
	AcceptFriendRequest(reqID uint64, fromUID, toUID uint64) error
	RejectFriendRequest(reqID uint64) error

	CreateFriend(friend *Friend) error
	DeleteFriendBidirectional(uid, friendUID uint64) error
	ListFriends(uid uint64) ([]Friend, error)
	UpdateFriend(uid, friendUID uint64, updates map[string]any) error
	HasActiveFriend(uid, friendUID uint64) (bool, error)
	HasPendingRequest(uid, friendUID uint64) (bool, error)

	CreateGroup(group *FriendGroup) error
	ListGroups(uid uint64) ([]FriendGroup, error)
	UpdateGroup(id, uid uint64, updates map[string]any) error
	DeleteGroup(id, uid uint64) error
}

type gormFriendStore struct {
	db *gorm.DB
}

func NewFriendStore(db *gorm.DB) FriendStore {
	return &gormFriendStore{db: db}
}

func (s *gormFriendStore) CreateRequest(req *FriendRequest) error {
	return s.db.Create(req).Error
}

func (s *gormFriendStore) ListIncomingRequests(uid uint64) ([]FriendRequest, error) {
	var reqs []FriendRequest
	if err := s.db.Where("to_uid = ?", uid).Order("created_at DESC").Find(&reqs).Error; err != nil {
		return nil, err
	}
	return reqs, nil
}

func (s *gormFriendStore) ListOutgoingRequests(uid uint64) ([]FriendRequest, error) {
	var reqs []FriendRequest
	if err := s.db.Where("from_uid = ?", uid).Order("created_at DESC").Find(&reqs).Error; err != nil {
		return nil, err
	}
	return reqs, nil
}

func (s *gormFriendStore) GetRequest(id uint64) (*FriendRequest, error) {
	var req FriendRequest
	if err := s.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *gormFriendStore) AcceptFriendRequest(reqID uint64, fromUID, toUID uint64) error {
	now := time.Now().Unix()
	tx := s.db.Begin()
	if err := tx.Model(&FriendRequest{}).Where("id = ?", reqID).
		Updates(map[string]any{"status": int8(1), "updated_at": now}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := upsertFriendEdges(tx, fromUID, toUID, now).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func upsertFriendEdges(tx *gorm.DB, fromUID, toUID uint64, createdAt int64) *gorm.DB {
	friends := []Friend{
		{UserID: fromUID, FriendUID: toUID, Status: 0, CreatedAt: createdAt},
		{UserID: toUID, FriendUID: fromUID, Status: 0, CreatedAt: createdAt},
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "friend_uid"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "created_at"}),
	}).Create(&friends)
}

func (s *gormFriendStore) RejectFriendRequest(reqID uint64) error {
	now := time.Now().Unix()
	return s.db.Model(&FriendRequest{}).Where("id = ?", reqID).
		Updates(map[string]any{"status": int8(2), "updated_at": now}).Error // FriendRequestRejected
}

func (s *gormFriendStore) CreateFriend(friend *Friend) error {
	return s.db.Create(friend).Error
}

func (s *gormFriendStore) DeleteFriendBidirectional(uid, friendUID uint64) error {
	tx := s.db.Begin()
	if err := tx.Model(&Friend{}).Where("user_id = ? AND friend_uid = ?", uid, friendUID).
		Update("status", int8(1)).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&Friend{}).Where("user_id = ? AND friend_uid = ?", friendUID, uid).
		Update("status", int8(1)).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *gormFriendStore) ListFriends(uid uint64) ([]Friend, error) {
	var friends []Friend
	if err := s.db.Where("user_id = ? AND status = 0", uid).Find(&friends).Error; err != nil {
		return nil, err
	}
	return friends, nil
}

func (s *gormFriendStore) UpdateFriend(uid, friendUID uint64, updates map[string]any) error {
	return s.db.Model(&Friend{}).Where("user_id = ? AND friend_uid = ?", uid, friendUID).Updates(updates).Error
}

func (s *gormFriendStore) CreateGroup(group *FriendGroup) error {
	return s.db.Create(group).Error
}

func (s *gormFriendStore) ListGroups(uid uint64) ([]FriendGroup, error) {
	var groups []FriendGroup
	if err := s.db.Where("user_id = ? AND status = 0", uid).Order("sort_order ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *gormFriendStore) UpdateGroup(id, uid uint64, updates map[string]any) error {
	return s.db.Model(&FriendGroup{}).Where("id = ? AND user_id = ?", id, uid).Updates(updates).Error
}

func (s *gormFriendStore) DeleteGroup(id, uid uint64) error {
	return s.db.Model(&FriendGroup{}).Where("id = ? AND user_id = ?", id, uid).Update("status", int8(1)).Error
}

func (s *gormFriendStore) HasActiveFriend(uid, friendUID uint64) (bool, error) {
	var count int64
	if err := s.db.Model(&Friend{}).Where("user_id = ? AND friend_uid = ? AND status = 0", uid, friendUID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *gormFriendStore) HasPendingRequest(uid, friendUID uint64) (bool, error) {
	var count int64
	if err := s.db.Model(&FriendRequest{}).Where(
		"((from_uid = ? AND to_uid = ?) OR (from_uid = ? AND to_uid = ?)) AND status = 0",
		uid, friendUID, friendUID, uid,
	).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
