package routers

import (
	"log/slog"
	"net/http"
	"server-go/managers"
	"server-go/models"
	"server-go/utils"

	"gorm.io/gorm"
)

const adminParty = "/admin"

func admin() {
	// 角色管理
	http.Handle(adminParty+"/roles", utils.CORS(verify(RequirePermission("manage_roles")(http.HandlerFunc(handleListRoles))), http.MethodGet))
	http.Handle(adminParty+"/role", utils.CORS(verify(RequirePermission("manage_roles")(http.HandlerFunc(handleCreateRole))), http.MethodPost))
	http.Handle(adminParty+"/role/update", utils.CORS(verify(RequirePermission("manage_roles")(http.HandlerFunc(handleUpdateRole))), http.MethodPut))
	http.Handle(adminParty+"/role/delete", utils.CORS(verify(RequirePermission("manage_roles")(http.HandlerFunc(handleDeleteRole))), http.MethodDelete))

	// 权限管理
	http.Handle(adminParty+"/permissions", utils.CORS(verify(RequirePermission("manage_permissions")(http.HandlerFunc(handleListPermissions))), http.MethodGet))
	http.Handle(adminParty+"/permission", utils.CORS(verify(RequirePermission("manage_permissions")(http.HandlerFunc(handleCreatePermission))), http.MethodPost))

	// 用户管理
	http.Handle(adminParty+"/users", utils.CORS(verify(RequirePermission("manage_users")(http.HandlerFunc(handleListUsers))), http.MethodGet))
	http.Handle(adminParty+"/user/roles", utils.CORS(verify(RequirePermission("manage_users")(http.HandlerFunc(handleAssignUserRoles))), http.MethodPost))
	http.Handle(adminParty+"/user/permissions", utils.CORS(verify(RequirePermission("manage_users")(http.HandlerFunc(handleAssignUserPermissions))), http.MethodPost))
}

// 获取所有角色
func handleListRoles(w http.ResponseWriter, r *http.Request) {
	var roles []models.Role
	if err := managers.DB.Preload("Permission").Find(&roles).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.SucessWithData(w, roles)
}

// 创建角色
func handleCreateRole(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("name")
	description := r.PostFormValue("description")

	if name == "" {
		http.Error(w, "Role name is required", http.StatusBadRequest)
		return
	}

	role := models.Role{
		Name:        name,
		Description: description,
	}

	if err := managers.DB.Create(&role).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.SucessWithData(w, role)
}

// 更新角色
func handleUpdateRole(w http.ResponseWriter, r *http.Request) {
	roleID := r.PostFormValue("id")
	description := r.PostFormValue("description")

	if roleID == "" {
		http.Error(w, "Role ID is required", http.StatusBadRequest)
		return
	}

	if err := managers.DB.Model(&models.Role{}).Where("id = ?", roleID).Update("description", description).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.Sucess(w)
}

// 删除角色
func handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	roleID := r.URL.Query().Get("id")

	if roleID == "" {
		http.Error(w, "Role ID is required", http.StatusBadRequest)
		return
	}

	if err := managers.DB.Delete(&models.Role{}, roleID).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.Sucess(w)
}

// 获取所有权限
func handleListPermissions(w http.ResponseWriter, r *http.Request) {
	var permissions []models.Permission
	if err := managers.DB.Find(&permissions).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.SucessWithData(w, permissions)
}

// 创建权限
func handleCreatePermission(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("name")
	description := r.PostFormValue("description")

	if name == "" {
		http.Error(w, "Permission name is required", http.StatusBadRequest)
		return
	}

	permission := models.Permission{
		Name:        name,
		Description: description,
	}

	if err := managers.DB.Create(&permission).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.SucessWithData(w, permission)
}

// 为用户分配角色
func handleAssignUserRoles(w http.ResponseWriter, r *http.Request) {
	userID := r.PostFormValue("userId")
	roleIDs := r.Form["roleIds[]"]

	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

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

	var roles []models.Role
	// 过滤空字符串
	var validRoleIDs []string
	for _, id := range roleIDs {
		if id != "" {
			validRoleIDs = append(validRoleIDs, id)
		}
	}

	// 只有在有有效的角色ID时才查询
	if len(validRoleIDs) > 0 {
		if err := managers.DB.Find(&roles, validRoleIDs).Error; err != nil {
			slog.Error(utils.DBErrorString, "err", err)
			http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
			return
		}
	}

	if err := managers.DB.Model(&user).Association("Role").Replace(&roles); err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.Sucess(w)
}

// 为用户分配权限
func handleAssignUserPermissions(w http.ResponseWriter, r *http.Request) {
	userID := r.PostFormValue("userId")
	permissionIDs := r.Form["permissionIds[]"]

	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

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

	var permissions []models.Permission
	// 过滤空字符串
	var validPermissionIDs []string
	for _, id := range permissionIDs {
		if id != "" {
			validPermissionIDs = append(validPermissionIDs, id)
		}
	}

	// 只有在有有效的权限ID时才查询
	if len(validPermissionIDs) > 0 {
		if err := managers.DB.Find(&permissions, validPermissionIDs).Error; err != nil {
			slog.Error(utils.DBErrorString, "err", err)
			http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
			return
		}
	}

	if err := managers.DB.Model(&user).Association("Permission").Replace(&permissions); err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.Sucess(w)
}

// 获取所有用户
func handleListUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	if err := managers.DB.Preload("Role").Preload("Permission").Find(&users).Error; err != nil {
		slog.Error(utils.DBErrorString, "err", err)
		http.Error(w, utils.DBErrorString, http.StatusInternalServerError)
		return
	}

	utils.SucessWithData(w, users)
}
