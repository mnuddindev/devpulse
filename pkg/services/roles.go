package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/models"
	"gorm.io/gorm"
)

func (us *UserSystem) AssignRole(db *gorm.DB, userID uuid.UUID, roleName []string) error {
	var role []models.Role
	if err := us.crud.GetByCondition(&role, "name IN ?", []interface{}{roleName}, []string{}, "", 0, 0); err != nil {
		return err
	}

	if len(role) == 0 {
		return errors.New("no roles found")
	}

	var user models.User
	if err := us.crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Roles"}, "", 0, 0); err != nil {
		return err
	}

	return us.crud.AddManyToMany(&user, "Roles", &role)
}

func (us *UserSystem) GetUserRoles(db *gorm.DB, userID uuid.UUID) ([]string, error) {
	var user models.User
	if err := us.crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Roles"}, "", 0, 0); err != nil {
		return nil, err
	}

	var roleNames []string
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
	}

	return roleNames, nil
}

func (us *UserSystem) HasRole(userid uuid.UUID, roleName []string) (bool, error) {
	var count int64
	err := us.crud.DB.Model(&models.User{}).
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("users.id = ? AND roles.name IN ?", userid, roleName).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count == int64(len(roleName)), nil
}

func (us *UserSystem) UpdateRole(db *gorm.DB, userID uuid.UUID, roleName []string) error {
	var role []models.Role
	if err := us.crud.GetByCondition(&role, "name IN ?", []interface{}{roleName}, []string{}, "", 0, 0); err != nil {
		return err
	}

	if len(role) == 0 {
		return errors.New("no roles found")
	}

	var user models.User
	if err := us.crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Roles"}, "", 0, 0); err != nil {
		return err
	}

	return us.crud.UpdateManyToMany(&user, "Roles", &role)
}

func (us *UserSystem) RemoveRoles(userid uuid.UUID, roles []uuid.UUID) error {
	var user models.User
	if err := us.crud.GetByCondition(&user, "id = ?", []interface{}{userid}, []string{"Roles"}, "", 0, 0); err != nil {
		return err
	}

	var rolestoDelete []models.Role
	for _, roleId := range roles {
		for _, role := range user.Roles {
			if role.ID == roleId {
				rolestoDelete = append(rolestoDelete, role)
				break
			}
		}
	}

	return us.crud.DeleteManyToMany(&user, "Roles", &rolestoDelete)
}

func (us *UserSystem) RemoveAllRoles(userid uuid.UUID) error {
	return us.crud.ClearManyToMany(&models.User{}, "Roles", "id = ?", userid)
}

func (us *UserSystem) GetAllRoles() ([]models.Role, error) {
	var roles []models.Role
	if err := us.crud.GetAll(&roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (us *UserSystem) RolesBy(userID *uuid.UUID, roleIDs []uuid.UUID) ([]models.Role, error) {
	var roles []models.Role
	if userID != nil {
		var user models.User
		if err := us.crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Roles"}, "", 0, 0); err != nil {
			return nil, err
		}
		return user.Roles, nil
	} else if len(roleIDs) > 0 {
		if err := us.crud.GetByCondition(&roles, "id IN ?", []interface{}{roleIDs}, []string{}, "", 0, 0); err != nil {
			return nil, err
		}
		return roles, nil
	}
	return nil, nil
}
