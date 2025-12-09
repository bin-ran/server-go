package models

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"net/http"
	"server-go/managers"
	"server-go/utils"
	"time"

	"golang.org/x/crypto/pbkdf2"
	"gorm.io/gorm"
)

type User struct {
	ID          uint           `gorm:"primary_key" json:"id"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Username    string         `gorm:"size:30;unique;not null" json:"username"`
	Password    []byte         `json:"-"`
	Salt        []byte         `json:"-"`
	Name        string         `gorm:"size:50" json:"name,omitempty"`
	AvatarPath  string         `gorm:"size:255;column:avatar" json:"-"`
	PhoneNumber string         `gorm:"size:20" json:"phoneNumber,omitempty"`
	Email       string         `gorm:"size:100" json:"email,omitempty"`
	Sex         uint8          `gorm:"default:0;not null" json:"sex,omitempty"`
	Role        []Role         `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	Permission  []Permission   `gorm:"many2many:user_permissions;" json:"permissions,omitempty"`
}

type Role struct {
	ID          uint           `gorm:"primary_key" json:"id"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:30;unique;not null" json:"name"`
	Description string         `gorm:"size:255;" json:"description"`
	User        []User         `gorm:"many2many:user_roles;"`
	Permission  []Permission   `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

type Permission struct {
	ID          uint           `gorm:"primary_key" json:"id"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:50;unique;not null" json:"name"`
	Description string         `gorm:"size:255;" json:"description"`
	User        []User         `gorm:"many2many:user_permissions;" json:"users,omitempty"`
	Role        []Role         `gorm:"many2many:role_permissions;" json:"roles,omitempty"`
}

func AccountInit() {
	managers.DB.AutoMigrate(&User{}, &Role{}, &Permission{})
}

// HasPermission 检查用户是否有指定权限
func (user *User) HasPermission(permissionName string) bool {
	// 检查直接权限
	for _, perm := range user.Permission {
		if perm.Name == permissionName {
			return true
		}
	}

	// 检查角色权限
	for _, role := range user.Role {
		for _, perm := range role.Permission {
			if perm.Name == permissionName {
				return true
			}
		}
	}

	return false
}

// HasRole 检查用户是否有指定角色
func (user *User) HasRole(roleName string) bool {
	for _, role := range user.Role {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// LoadPermissions 加载用户的完整权限信息
func (user *User) LoadPermissions() error {
	return managers.DB.Preload("Role.Permission").Preload("Permission").First(user, user.ID).Error
}

// GetAvatarURL 获取用户头像URL
func (user *User) GetAvatarURL(ctx context.Context) string {
	if user.AvatarPath == "" {
		return ""
	}

	url, err := managers.RustFSClient.Client.GetPreSignedDownloadURL(
		ctx,
		managers.RustFSClient.Bucket, user.AvatarPath, 7*24*time.Hour)
	if err != nil {
		return ""
	}

	return url
}

func (user *User) SetPassword(password string) {
	user.Salt = utils.RandomBytes(32)
	user.Password = PasswordMaker(password, user.Salt)
}

func PasswordMaker(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, 4096, 128, sha256.New)
}

func UpdateToken(w http.ResponseWriter, r *http.Request, userID string, ip string) error {
	token := TokenMaker()

	if err := utils.HSetAndExpireNonatomic(managers.Redis, r.Context(), managers.TOKEN+token, map[string]interface{}{"id": userID}, managers.UserTokenLife); err != nil {
		slog.Error("Set token cache failed.", "err", err)
		return err
	}

	SetCookie(w, r, &http.Cookie{Name: "token", Value: token, Path: "/", HttpOnly: true, MaxAge: int(managers.UserTokenLife.Seconds())})
	SetCookie(w, r, &http.Cookie{Name: "auth_status", Value: "1", Path: "/", HttpOnly: false, MaxAge: int(managers.UserTokenLife.Seconds())})

	return nil
}

func TokenMaker() string {
	return utils.RandomURLBase64(24)
}

func SetCookie(w http.ResponseWriter, r *http.Request, cookie *http.Cookie) {
	cookie.SameSite = http.SameSiteStrictMode
	cookie.Secure = true

	if managers.Config.Domain != "" {
		cookie.Domain = managers.Config.Domain
	}

	http.SetCookie(w, cookie)
}

func Renew(ctx context.Context, token string) error {
	return managers.Redis.Expire(ctx, managers.TOKEN+token, managers.UserTokenLife).Err()
}
