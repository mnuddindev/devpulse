package users

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/models"
)

func (us *UserSystem) AssignRole(userID, roleid uuid.UUID) error {
	// Fetch the user from the database with their current roles
	user, err := us.UserBy("id = ?", userID)
	if err != nil {
		// Return a 404 Not Found if the user doesn’t exist
		return err
	}
	// Fetch the role from the database
	var role models.Role
	if err := us.Crud.DB.Where("id = ?", roleid).First(&role).Error; err != nil {
		// Return a 404 Not Found if the role doesn’t exist
		return err
	}

	// Check if the user already has this role
	for _, existingRole := range user.Roles {
		if existingRole.ID == role.ID {
			// Return a 409 Conflict if the role is already assigned
			return errors.New("User already has this role")
		}
	}

	return us.Crud.AddManyToMany(&user, "Roles", &role)
}

func (us *UserSystem) GetUserRoles(userID uuid.UUID) ([]string, error) {
	var user models.User
	if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Roles"}, "", 0, 0); err != nil {
		return nil, err
	}

	var roleNames []string
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
	}

	return roleNames, nil
}

func (us *UserSystem) GetUserPermsissions(userID uuid.UUID) ([]string, error) {
	// Fetch the user from the database with their roles and permissions
	user, err := us.UserBy("id = ?", userID)
	if err != nil {
		// Return a 404 Not Found if the user doesn’t exist
		return []string{}, errors.New("User not found")
	}

	// Build a map of unique permissions
	permissions := make(map[string]bool)
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			permissions[perm.Name] = true
		}
	}

	// Convert map to slice for JSON response
	var permList []string
	for perm := range permissions {
		permList = append(permList, perm)
	}

	return permList, nil
}

func (us *UserSystem) UpdateUserRolePermission(userID, roleid uuid.UUID, permissions []string) error {
	// Fetch the user from the database with their roles and permissions
	user, err := us.UserBy("id = ?", userID)
	if err != nil {
		// Return a 404 Not Found if the user doesn’t exist
		return errors.New("User not found")
	}

	// Find the role in the user’s current roles
	var targetRole *models.Role
	for i, role := range user.Roles {
		if role.ID == roleid {
			targetRole = &user.Roles[i]
			break
		}
	}
	// Check if the role is assigned to the user
	if targetRole == nil {
		// Return a 404 Not Found if the role isn’t assigned to the user
		return errors.New("Role not assigned to user")
	}

	// Fetch existing permissions from the database
	var perms []models.Permission
	err = us.Crud.GetByCondition(&perms, "name IN ?", []interface{}{permissions}, []string{}, "", 0, 0)
	if err != nil || len(perms) != len(permissions) {
		// Return a 400 Bad Request if any permission names are invalid
		return errors.New("Invalid permissions")
	}

	return us.Crud.UpdateManyToMany(&targetRole, "Permissions", perms)
}

func (us *UserSystem) RemoveRoleFromUser(userid, roleid uuid.UUID) error {
	// Fetch the user from the database with their roles and permissions
	user, err := us.UserBy("id = ?", userid)
	if err != nil {
		// Return a 404 Not Found if the user doesn’t exist
		return errors.New("User not found")
	}

	// Find the role in the user’s current roles
	var targetRole *models.Role
	for i, role := range user.Roles {
		if role.ID == roleid {
			targetRole = &user.Roles[i]
			break
		}
	}
	// Check if the role is assigned to the user
	if targetRole == nil {
		// Return a 404 Not Found if the role isn’t assigned
		return errors.New("Role not assigned to user")
	}

	return us.Crud.DeleteManyToMany(&user, "Roles", targetRole)
}

func (us *UserSystem) HasRole(userid uuid.UUID, roleName string) (bool, error) {
	var user models.User
	if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userid}, []string{"Roles"}, "", 0, 0); err != nil {
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
	if err := us.Crud.GetByCondition(&role, "name IN ?", []interface{}{roleName}, []string{}, "", 0, 0); err != nil {
		return err
	}

	if len(role) == 0 {
		return errors.New("no roles found")
	}

	var user models.User
	if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Roles"}, "", 0, 0); err != nil {
		return err
	}

	return us.Crud.UpdateManyToMany(&user, "Roles", &role)
}

func (us *UserSystem) RemoveAllRoles(userid uuid.UUID) error {
	return us.Crud.ClearManyToMany(&models.User{}, "Roles", "id = ?", userid)
}

func (us *UserSystem) GetAllRoles() ([]models.Role, error) {
	var roles []models.Role
	if err := us.Crud.GetAll(&roles, []string{}); err != nil {
		return nil, err
	}
	return roles, nil
}

func (us *UserSystem) RolesBy(userID *uuid.UUID, roleIDs []uuid.UUID) ([]models.Role, error) {
	var roles []models.Role
	if userID != nil {
		var user models.User
		if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Roles"}, "", 0, 0); err != nil {
			return nil, err
		}
		return user.Roles, nil
	} else if len(roleIDs) > 0 {
		if err := us.Crud.GetByCondition(&roles, "id IN ?", []interface{}{roleIDs}, []string{}, "", 0, 0); err != nil {
			return nil, err
		}
		return roles, nil
	}
	return nil, nil
}
