// utils/avatar.go
package utils

import "backend/entity"

func BuildAvatarURL(user *entity.User) string {
    if user.AvatarSize > 0 {
        return "/auth/me/avatar"
    }
    return ""
}
