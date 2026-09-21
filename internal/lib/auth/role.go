package auth

import "photo-viewer-server/internal/storage/entity"

func IsLessThanAdmin(role string) bool {
	return roleToNumber(role) < roleAdmin
}

func IsLessThanModerator(role string) bool {
	return roleToNumber(role) < roleModerator
}

func roleToNumber(role string) int {
	switch role {
	case string(entity.UserDefaultRole): return roleUser
	case string(entity.UserModeratorRole): return roleModerator
	case string(entity.UserAdminRole): return roleAdmin
	default: return roleUser
	}
}
