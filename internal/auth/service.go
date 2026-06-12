package auth

import (
	"database/sql"
	"errors"
	"time"

	"xsm/internal/encryption"
	"xsm/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db         *sql.DB
	jwtService *JWTService
	rsaService *encryption.RSAService
}

func NewAuthService(db *sql.DB, jwtService *JWTService, rsaService *encryption.RSAService) *AuthService {
	return &AuthService{
		db:         db,
		jwtService: jwtService,
		rsaService: rsaService,
	}
}

// RegisterResponse содержит и пользователя, и токены
type RegisterResponse struct {
	User         *models.UserResponse `json:"user"`
	AccessToken  string               `json:"access_token"`
	RefreshToken string               `json:"refresh_token"`
	ExpiresIn    int64                `json:"expires_in"`
	TokenType    string               `json:"token_type"`
}

type UpdateProfileRequest struct {
	Username   string `json:"username" binding:"required,min=3,max=50"`
	Email      string `json:"email" binding:"required,email"`
	StatusText string `json:"status_text"`
	Bio        string `json:"bio"`
}

func (s *AuthService) Register(req *models.RegisterRequest) (*RegisterResponse, error) {
	// Проверяем, существует ли пользователь
	var exists bool
	err := s.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE username=$1 OR email=$2)",
		req.Username, req.Email,
	).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("пользователь с таким именем или email уже существует")
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Генерируем RSA ключи
	privateKey, publicKey, err := s.rsaService.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	pubKeyStr, err := s.rsaService.ExportPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	privKeyStr, err := s.rsaService.ExportPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	// Шифруем приватный ключ
	encryptedPrivKey, err := s.encryptPrivateKey(privKeyStr, req.Password)
	if err != nil {
		return nil, err
	}

	// Вставляем пользователя
	var user models.User
	err = s.db.QueryRow(`
        INSERT INTO users (username, email, password_hash, public_key, private_key_encrypted)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, username, email, public_key, avatar_url, status, created_at, updated_at
    `, req.Username, req.Email, string(hashedPassword), pubKeyStr, encryptedPrivKey).
		Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PublicKey,
			&user.AvatarURL,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

	if err != nil {
		return nil, err
	}

	user.PasswordHash = string(hashedPassword)
	user.PrivateKeyEncrypted = encryptedPrivKey

	// Сразу генерируем токены для автоматического входа
	accessToken, err := s.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Сохраняем refresh токен
	_, err = s.db.Exec(`
        INSERT INTO refresh_tokens (user_id, token, expires_at)
        VALUES ($1, $2, $3)
    `, user.ID, refreshToken, time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, err
	}

	// Обновляем статус
	s.db.Exec("UPDATE users SET status='online', last_seen=$1 WHERE id=$2",
		time.Now(), user.ID)

	// Возвращаем пользователя и токены
	return &RegisterResponse{
		User:         user.ToResponsePtr(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtService.expiration.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) Login(req *models.LoginRequest) (*models.TokenResponse, error) {
	var user models.User

	err := s.db.QueryRow(`
        SELECT id, username, email, password_hash, avatar_url
        FROM users WHERE username=$1
    `, req.Username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.AvatarURL,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("неверное имя пользователя или пароль")
	}
	if err != nil {
		return nil, err
	}

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("неверное имя пользователя или пароль")
	}

	// Генерируем токены
	accessToken, err := s.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Сохраняем refresh токен
	_, err = s.db.Exec(`
        INSERT INTO refresh_tokens (user_id, token, expires_at)
        VALUES ($1, $2, $3)
    `, user.ID, refreshToken, time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, err
	}

	// Обновляем статус
	s.db.Exec("UPDATE users SET status='online', last_seen=$1 WHERE id=$2",
		time.Now(), user.ID)

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtService.expiration.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User

	err := s.db.QueryRow(`
        SELECT id, username, email, public_key, avatar_url, status, last_seen, created_at, updated_at
        FROM users WHERE id=$1
    `, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PublicKey,
		&user.AvatarURL,
		&user.Status,
		&user.LastSeen,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *AuthService) encryptPrivateKey(privateKey, password string) (string, error) {
	aesService := encryption.NewAESService(password)
	encrypted, _, err := aesService.Encrypt(privateKey)
	return encrypted, err
}

func (s *AuthService) UpdateProfile(userID uuid.UUID, req *models.UpdateProfileRequest) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(`
        UPDATE users 
        SET username = $2, email = $3, status_text = $4, bio = $5, updated_at = NOW()
        WHERE id = $1
        RETURNING id, username, email, public_key, avatar_url, status, status_text, bio, privacy_settings, last_seen, created_at, updated_at
    `, userID, req.Username, req.Email, req.StatusText, req.Bio).
		Scan(&user.ID, &user.Username, &user.Email, &user.PublicKey, &user.AvatarURL,
			&user.Status, &user.StatusText, &user.Bio, &user.PrivacySettings,
			&user.LastSeen, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
