package routers

import (
	"bytes"
	"log/slog"
	"net/http"
	"server-go/managers"
	"server-go/models"
	"server-go/utils"
	"strconv"
	"time"

	"gorm.io/gorm"
)

const accountParty = "/account"

func account() {
	models.AccountInit()

	http.Handle(accountParty+"/renew", utils.CORS(verify(http.HandlerFunc(handleRenew)), http.MethodPut))

	http.Handle(accountParty+"/register", utils.CORS(http.HandlerFunc(handleRegister), http.MethodPost))
	http.Handle(accountParty+"/login", utils.CORS(http.HandlerFunc(handleLogin), http.MethodPost))
	http.Handle(accountParty+"/logout", utils.CORS(verify(http.HandlerFunc(handleLogout)), http.MethodPost))

	// 用户信息管理
	http.Handle(accountParty+"/info", utils.CORS(verify(http.HandlerFunc(handleGetUserInfo)), http.MethodGet))
	http.Handle(accountParty+"/permissions", utils.CORS(verify(http.HandlerFunc(handleGetUserPermissions)), http.MethodGet))
	http.Handle(accountParty+"/update", utils.CORS(verify(http.HandlerFunc(handleUpdateUserInfo)), http.MethodPut))
	http.Handle(accountParty+"/password", utils.CORS(verify(http.HandlerFunc(handleChangePassword)), http.MethodPut))

	// 头像管理
	http.Handle(accountParty+"/avatar", utils.CORS(verify(http.HandlerFunc(handleGetAvatar)), http.MethodGet))
	http.Handle(accountParty+"/avatar/upload-url", utils.CORS(verify(http.HandlerFunc(handleGetAvatarUploadURL)), http.MethodGet))
	http.Handle(accountParty+"/avatar/confirm", utils.CORS(verify(http.HandlerFunc(handleConfirmAvatar)), http.MethodPost))
}

func handleRenew(w http.ResponseWriter, r *http.Request) {
	if err := models.UpdateToken(w, r, r.Context().Value(UserID).(string), utils.ParseIP(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	// phoneNumber := r.PostFormValue("phoneNumber")
	// email := r.PostFormValue("email")
	sexStr := r.PostFormValue("sex")

	var sex uint64
	if sexStr != "" {
		var err error
		sex, err = strconv.ParseUint(sexStr, 10, 8)
		if err != nil {
			msg := "invalid value for sex"
			slog.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
	}

	if username == "" || password == "" {
		msg := "username or password is empty"
		slog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	user := models.User{
		Username: username,
		Sex:      uint8(sex),
	}

	user.SetPassword(password)

	if err := managers.DB.Create(&user).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.SucessWithData(w, user)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	if username == "" || password == "" {
		msg := "username or password is empty"
		slog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	ip := utils.ParseIP(r)
	ipKey := managers.IPLIMIT + ip

	reply, err := utils.IncreaseAndExpireNonatomic(managers.Redis, r.Context(), ipKey, time.Hour)
	if err != nil {
		msg := "Failed to determine if the ip hasbeen frozen"
		slog.Error(msg, "err", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if reply > 5 {
		msg := "You are logged in top often. Please try again later."
		slog.Error(msg, "ip", ip)
		http.Error(w, msg, http.StatusTooManyRequests)
		return
	}

	user := models.User{Username: username}
	if err := managers.DB.
		Select("id", "salt", "password").
		Where(&user).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			msg := "username or password is wrong"
			slog.Error(msg, "username", username)
			http.Error(w, msg, http.StatusBadRequest)
		} else {
			slog.Error(utils.DBErrorString, "err", err)
			http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		}
		return
	}

	if !bytes.Equal(user.Password, models.PasswordMaker(password, user.Salt)) {
		msg := "username or password is wrong"
		slog.Error(msg, "username", username)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if err := managers.DB.First(&user).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	if err := managers.Redis.Del(r.Context(), ipKey).Err(); err != nil {
		msg := "Failed to delete ipKey"
		slog.Error(msg, "err", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if err := models.UpdateToken(w, r, managers.IDToString(user.ID), ip); err != nil {
		slog.Error(utils.CacheErrorString, "err", err)
		http.Error(w, utils.CacheErrorString, http.StatusInternalServerError)
		return
	}

	if err := utils.SucessWithData(w, user); err != nil {
		slog.Error(utils.ReturnFailedString, "err", err)
		http.Error(w, utils.ReturnFailedString, http.StatusInternalServerError)
		return
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	token, _ := r.Cookie("token")

	if err := managers.Redis.Expire(r.Context(), managers.TOKEN+token.Value, -1).Err(); err != nil {
		msg := "Expire token filed"
		slog.Error(msg, "err", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	models.SetCookie(w, r, &http.Cookie{Name: "token", Value: "", Path: "/", Expires: time.Now()})
	models.SetCookie(w, r, &http.Cookie{Name: "auth_status", Value: "0", Path: "/", Expires: time.Now()})

	utils.Sucess(w)
}

// 获取当前用户信息
func handleGetUserInfo(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserID).(string)

	var user models.User
	if err := managers.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			slog.Error(utils.DBErrorString, "err", err)
			http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		}
		return
	}

	utils.SucessWithData(w, user)
}

// 获取当前用户权限
func handleGetUserPermissions(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserID).(string)

	var user models.User
	if err := managers.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			slog.Error(utils.DBErrorString, "err", err)
			http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		}
		return
	}

	// 加载完整权限信息
	if err := user.LoadPermissions(); err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	// 收集所有权限
	permissions := make(map[string]bool)
	for _, perm := range user.Permission {
		permissions[perm.Name] = true
	}
	for _, role := range user.Role {
		for _, perm := range role.Permission {
			permissions[perm.Name] = true
		}
	}

	result := map[string]interface{}{
		"roles":       user.Role,
		"permissions": permissions,
	}

	utils.SucessWithData(w, result)
}

// 更新用户信息
func handleUpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserID).(string)

	name := r.PostFormValue("name")
	email := r.PostFormValue("email")
	phoneNumber := r.PostFormValue("phoneNumber")

	// 构建更新数据
	updateData := make(map[string]interface{})
	if name != "" {
		updateData["name"] = name
	}
	if email != "" {
		updateData["email"] = email
	}
	if phoneNumber != "" {
		updateData["phone_number"] = phoneNumber
	}

	if len(updateData) == 0 {
		http.Error(w, "No data to update", http.StatusBadRequest)
		return
	}

	// 更新数据库
	if err := managers.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updateData).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	// 获取更新后的用户信息
	var user models.User
	if err := managers.DB.First(&user, userID).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.SucessWithData(w, user)
}

// 修改密码
func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserID).(string)

	oldPassword := r.PostFormValue("oldPassword")
	newPassword := r.PostFormValue("newPassword")

	if oldPassword == "" || newPassword == "" {
		http.Error(w, "Old password and new password are required", http.StatusBadRequest)
		return
	}

	// 验证新密码长度
	if len(newPassword) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// 获取用户信息
	var user models.User
	if err := managers.DB.Select("id", "salt", "password").First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			slog.Error(utils.DBErrorString, "err", err)
			http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		}
		return
	}

	// 验证旧密码
	if !bytes.Equal(user.Password, models.PasswordMaker(oldPassword, user.Salt)) {
		http.Error(w, "Old password is incorrect", http.StatusBadRequest)
		return
	}

	// 设置新密码
	user.SetPassword(newPassword)

	// 更新数据库
	if err := managers.DB.Model(&user).Updates(map[string]interface{}{
		"password": user.Password,
		"salt":     user.Salt,
	}).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.Sucess(w)
}

// 获取头像上传URL
func handleGetAvatarUploadURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserID).(string)

	// 从查询参数获取文件扩展名，默认为png
	ext := r.URL.Query().Get("ext")
	if ext == "" {
		ext = "png"
	}

	// 验证文件扩展名
	allowedExts := map[string]bool{
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"gif":  true,
		"webp": true,
	}

	if !allowedExts[ext] {
		http.Error(w, "Unsupported file type", http.StatusBadRequest)
		return
	}

	url, err := managers.RustFSClient.Client.GetPreSignedUploadURL(
		ctx,
		managers.RustFSClient.Bucket, "avatar/"+userID+"."+ext, time.Hour)
	if err != nil {
		slog.Error("Failed to get pre signed upload url", "err", err)
		http.Error(w, "Failed to get upload URL", http.StatusInternalServerError)
		return
	}

	utils.SucessWithData(w, map[string]string{"uploadUrl": url})
}

// 获取头像
func handleGetAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserID).(string)

	var user models.User
	if err := managers.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			slog.Error(utils.DBErrorString, "err", err)
			http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		}
		return
	}

	avatarURL := user.GetAvatarURL(ctx)
	utils.SucessWithData(w, map[string]string{"avatarUrl": avatarURL})
}

// 确认头像上传完成
func handleConfirmAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserID).(string)

	// 从请求体获取文件扩展名
	ext := r.PostFormValue("ext")
	if ext == "" {
		ext = "png"
	}

	// 验证文件扩展名
	allowedExts := map[string]bool{
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"gif":  true,
		"webp": true,
	}

	if !allowedExts[ext] {
		http.Error(w, "Unsupported file type", http.StatusBadRequest)
		return
	}

	// 存储头像路径而非预签名URL
	avatarPath := "avatar/" + userID + "." + ext

	// 更新数据库中的头像路径
	if err := managers.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar", avatarPath).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	// 获取更新后的用户信息
	var user models.User
	if err := managers.DB.First(&user, userID).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	// 返回头像访问URL
	avatarURL := user.GetAvatarURL(ctx)
	utils.SucessWithData(w, map[string]string{"avatarUrl": avatarURL})
}
