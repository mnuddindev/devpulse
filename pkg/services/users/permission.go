package users

import (
	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"github.com/mnuddindev/devpulse/pkg/models"
	"github.com/sirupsen/logrus"
)

// PermissionBy retrieves a permission from the database based on a condition
func (us *UserSystem) PermissionBy(condition string, args ...interface{}) (*models.Permission, error) {
	// Declare a variable to hold the permission struct that will be fetched
	var permission models.Permission
	// Execute a GORM query to find the first permission matching the condition and arguments
	err := us.Crud.DB.Where(condition, args...).First(&permission).Error
	// Check if an error occurred during the database query
	if err != nil {
		// Log an error with details to debug the failure to fetch the permission
		logger.Log.WithFields(logrus.Fields{
			"error":     err,
			"condition": condition,
			"args":      args,
		}).Error("Failed to fetch permission by condition")
		// Return nil and the error to indicate the fetch operation failed
		return nil, err
	}
	// Return the fetched permission and nil error to indicate success
	return &permission, nil
}

// CreatePermission creates a new permission in the database
func (us *UserSystem) CreatePermission(permission *models.Permission) error {
	// Attempt to create the permission in the database using the Crud system
	err := us.Crud.Create(permission)
	// Check if an error occurred during the creation process
	if err != nil {
		// Log an error with details to debug the failure to create the permission
		logger.Log.WithFields(logrus.Fields{
			"error": err,
			"name":  permission.Name,
		}).Error("Failed to create permission")
		// Return the error to indicate the creation operation failed
		return err
	}
	// Log a success message with details to confirm the permission was created
	logger.Log.WithFields(logrus.Fields{
		"permissionID": permission.ID,
		"name":         permission.Name,
	}).Debug("Permission created successfully")
	// Return nil to indicate the creation operation succeeded
	return nil
}

// UpdatePermission updates an existing permission in the database
func (us *UserSystem) UpdatePermission(condition string, permissionID uuid.UUID, updates map[string]interface{}) (*models.Permission, error) {
	// Fetch the existing permission from the database to ensure it exists before updating
	permission, err := us.PermissionBy(condition, permissionID)
	// Check if fetching the permission failed
	if err != nil {
		// Log an error with details to debug the failure to fetch the permission for update
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionID,
		}).Error("Failed to fetch permission for update")
		// Return nil and the error to indicate the fetch operation failed
		return nil, err
	}
	// Attempt to update the permission in the database with the provided updates using GORM
	err = us.Crud.DB.Model(permission).Updates(updates).Error
	// Check if an error occurred during the update process
	if err != nil {
		// Log an error with details to debug the failure to update the permission
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permissionID,
			"updates":      updates,
		}).Error("Failed to update permission")
		// Return nil and the error to indicate the update operation failed
		return nil, err
	}
	// Log a success message with details to confirm the permission was updated
	logger.Log.WithFields(logrus.Fields{
		"permissionID": permission.ID,
		"name":         permission.Name,
		"updates":      updates,
	}).Debug("Permission updated successfully")
	// Return the updated permission and nil error to indicate success
	return permission, nil
}

// DeletePermission deletes a permission from the database
func (us *UserSystem) DeletePermission(permission *models.Permission, condition string, pid uuid.UUID) error {
	// Attempt to delete the permission from the database using the Crud system (assumes soft delete if configured)
	err := us.Crud.Delete(permission, condition, []interface{}{pid})
	// Check if an error occurred during the deletion process
	if err != nil {
		// Log an error with details to debug the failure to delete the permission
		logger.Log.WithFields(logrus.Fields{
			"error":        err,
			"permissionID": permission.ID,
		}).Error("Failed to delete permission")
		// Return the error to indicate the deletion operation failed
		return err
	}
	// Log a success message with details to confirm the permission was deleted
	logger.Log.WithFields(logrus.Fields{
		"permissionID": permission.ID,
		"name":         permission.Name,
	}).Debug("Permission deleted successfully")
	// Return nil to indicate the deletion operation succeeded
	return nil
}

// ListPermissions retrieves all permissions from the database
func (us *UserSystem) ListPermissions() ([]models.Permission, error) {
	// Declare a slice to hold the list of permissions fetched from the database
	var permissions []models.Permission
	// Execute a GORM query to fetch all permissions (assumes no soft delete filtering unless configured)
	err := us.Crud.DB.Find(&permissions).Error
	// Check if an error occurred during the database query
	if err != nil {
		// Log an error with details to debug the failure to fetch permissions
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch permissions")
		// Return an empty slice and the error to indicate the fetch operation failed
		return nil, err
	}
	// Log a success message with details to confirm the permissions were retrieved
	logger.Log.WithFields(logrus.Fields{
		"count": len(permissions),
	}).Debug("Permissions listed successfully")
	// Return the list of permissions and nil error to indicate success
	return permissions, nil
}
