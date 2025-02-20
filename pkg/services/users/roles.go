package users

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/models"
)

func (us *UserSystem) AssignRole(userID uuid.UUID, roleName []string) error {
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

func (us *UserSystem) GetUserRoles(userID uuid.UUID) ([]string, error) {
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

func (us *UserSystem) HasRole(userid uuid.UUID, roleName string) (bool, error) {
	var user models.User
	if err := us.crud.GetByCondition(&user, "id = ?", []interface{}{userid}, []string{"Roles"}, "", 0, 0); err != nil {
		return false, err
	}

	for _, role := range user.Roles {
		if role.Name == roleName {
			return true, nil
		}
	}
	return false, nil
}

func (us *UserSystem) UpdateRole(userID uuid.UUID, roleName []string) error {
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
	if err := us.crud.GetAll(&roles, []string{}); err != nil {
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
