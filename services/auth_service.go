package services

import (
	"backend/entity"
	"backend/repository"
	"backend/utils"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthService จัดการ business logic ของการ login/register
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	jwtTTL    time.Duration
}

func NewAuthService(repo *repository.UserRepository, secret string, ttl time.Duration) *AuthService {
	return &AuthService{
		userRepo:  repo,
		jwtSecret: secret,
		jwtTTL:    ttl,
	}
}

// Register สร้าง user ใหม่ ถ้า email ซ้ำจะ error
func (s *AuthService) Register(email, password, firstName, lastName, phone string) (*entity.User, error) {
	// trim และ normalize email
	email = strings.ToLower(strings.TrimSpace(email))

	// ตรวจซ้ำ email
	count, err := s.userRepo.CountByEmail(email)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("email already registered")
	}

	// hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("hash password failed")
	}

	user := &entity.User{
		Email:       email,
		Password:    string(hashed),
		FirstName:   strings.TrimSpace(firstName),
		LastName:    strings.TrimSpace(lastName),
		PhoneNumber: strings.TrimSpace(phone),
		Role:        "customer",
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login ตรวจสอบ user + สร้าง JWT
func (s *AuthService) Login(email, password string) (string, *entity.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// เทียบรหัสผ่าน
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// ออก token
	token, err := utils.GenerateToken(user.ID, user.Role, s.jwtSecret, s.jwtTTL)
	if err != nil {
		return "", nil, errors.New("cannot generate token")
	}

	return token, user, nil
}

// GetProfile
func (s *AuthService) GetProfile(userID uint) (*entity.User, error) {
	return s.userRepo.FindByID(userID)
}

// UpdateProfile อัปเดตข้อมูลผู้ใช้
func (s *AuthService) UpdateProfile(userID uint, updates map[string]any) (*entity.User, error) {
	if err := s.userRepo.Update(userID, updates); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(userID)
}

// ✅ Upload avatar (Base64)
func (s *AuthService) UploadAvatarBase64(userID uint, b64 string) error {
	if len(b64) > 10*1024*1024 { // limit 10MB
		return errors.New("file too large")
	}
	if !strings.HasPrefix(b64, "data:image/") {
		return errors.New("invalid image format")
	}
	return s.userRepo.SaveAvatarBase64(userID, b64)
}

// ✅ Get avatar (Base64)
func (s *AuthService) GetAvatarBase64(userID uint) (string, error) {
	return s.userRepo.FindAvatarBase64(userID)
}


func (s *AuthService) GetProfileWithRestaurant(userID uint) (*entity.User, *entity.Restaurant, error) {
    return s.userRepo.FindWithRestaurant(userID)
}

func (s *AuthService) GetRestaurantByUserID(userID uint) (*entity.Restaurant, error) {
    return s.userRepo.FindRestaurantByUserID(userID)
}