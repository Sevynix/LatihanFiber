package service

import (
	"tugas2/app/model"
	"tugas2/helper"
)

func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	if ownerID > 0 && current.UserID == ownerID {
		return true
	}
	return perms.Can(current.Role, anyPermission)
}

func ownerIDOf(s model.Student) int {
	if s.OwnerID == nil {
		return 0
	}
	return *s.OwnerID
}