package models

import (
	"log/slog"
	"server-go/managers"
)

// SeedDatabase 初始化基础数据
func SeedDatabase() {
	// 创建基础权限
	permissions := []Permission{
		{Name: "manage_users", Description: "管理用户"},
		{Name: "manage_roles", Description: "管理角色"},
		{Name: "manage_permissions", Description: "管理权限"},
		{Name: "view_dashboard", Description: "查看仪表板"},
		{Name: "view_reports", Description: "查看报表"},
		{Name: "edit_content", Description: "编辑内容"},
		{Name: "delete_content", Description: "删除内容"},
		{Name: "system_settings", Description: "系统设置"},
	}

	for _, perm := range permissions {
		var existing Permission
		if err := managers.DB.Where("name = ?", perm.Name).First(&existing).Error; err != nil {
			// 权限不存在，创建新权限
			if err := managers.DB.Create(&perm).Error; err != nil {
				slog.Error("Failed to create permission", "name", perm.Name, "err", err)
			} else {
				slog.Info("Permission created", "name", perm.Name)
			}
		} else {
			slog.Info("Permission already exists", "name", perm.Name)
		}
	}

	// 创建管理员角色
	var adminRole Role
	if err := managers.DB.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		adminRole = Role{
			Name:        "admin",
			Description: "系统管理员，拥有所有权限",
		}
		if err := managers.DB.Create(&adminRole).Error; err != nil {
			slog.Error("Failed to create admin role", "err", err)
			return
		}
		slog.Info("Admin role created")
	} else {
		slog.Info("Admin role already exists")
	}

	// 为管理员角色分配所有权限
	var allPermissions []Permission
	managers.DB.Find(&allPermissions)

	if err := managers.DB.Model(&adminRole).Association("Permission").Replace(&allPermissions); err != nil {
		slog.Error("Failed to assign permissions to admin role", "err", err)
	} else {
		slog.Info("Assigned all permissions to admin role", "count", len(allPermissions))
	}

	// 查找 admin 用户
	var adminUser User
	if err := managers.DB.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		slog.Warn("Admin user not found. Please create an admin user first.")
		return
	}

	// 为 admin 用户分配管理员角色
	if err := managers.DB.Model(&adminUser).Association("Role").Append(&adminRole); err != nil {
		slog.Error("Failed to assign admin role to admin user", "err", err)
	} else {
		slog.Info("Assigned admin role to admin user")
	}

	// 验证用户权限
	managers.DB.Preload("Role.Permission").Preload("Permission").First(&adminUser, adminUser.ID)
	slog.Info("Admin user permissions loaded",
		"roles", len(adminUser.Role),
		"direct_permissions", len(adminUser.Permission))
}
