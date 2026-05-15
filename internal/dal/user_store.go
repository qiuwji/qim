package dal

import "gorm.io/gorm"

type UserStore interface {
	GetUser(id uint64) (*User, error)
	GetUserByUsername(username string) (*User, error)
	CreateUser(user *User) error
	UpdateUser(id uint64, updates map[string]any) error
	SearchUsers(keyword string, limit int) ([]User, error)
	Authenticate(username, password string) (*User, error)
}

type gormUserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) UserStore {
	return &gormUserStore{db: db}
}

func (s *gormUserStore) GetUser(id uint64) (*User, error) {
	var u User
	if err := s.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *gormUserStore) GetUserByUsername(username string) (*User, error) {
	var u User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *gormUserStore) CreateUser(user *User) error {
	return s.db.Create(user).Error
}

func (s *gormUserStore) UpdateUser(id uint64, updates map[string]any) error {
	return s.db.Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (s *gormUserStore) SearchUsers(keyword string, limit int) ([]User, error) {
	var users []User
	if err := s.db.Where("username LIKE ? OR nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *gormUserStore) Authenticate(username, password string) (*User, error) {
	var u User
	if err := s.db.Where("username = ? AND password = ?", username, password).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
