package users

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/models"
)

func (us *UserSystem) AssignBadge(userID uuid.UUID, badges []string) error {
	var badge []models.Badge
	if err := us.Crud.GetByCondition(&badge, "name IN ?", []interface{}{badges}, []string{}, "", 0, 0); err != nil {
		return err
	}

	if len(badge) == 0 {
		return errors.New("no badges found")
	}

	var user models.User
	if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Badges"}, "", 0, 0); err != nil {
		return err
	}

	return us.Crud.AddManyToMany(&user, "Badges", &badge)
}

func (us *UserSystem) GetUserBadges(userID uuid.UUID) ([]string, error) {
	var user models.User
	if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Badges"}, "", 0, 0); err != nil {
		return nil, err
	}

	var BadgeNames []string
	for _, badge := range user.Badges {
		BadgeNames = append(BadgeNames, badge.Name)
	}

	return BadgeNames, nil
}

func (us *UserSystem) HasBadge(userid uuid.UUID, badgeName string) (bool, error) {
	var user models.User
	if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userid}, []string{"Badges"}, "", 0, 0); err != nil {
		return false, err
	}

	for _, badge := range user.Badges {
		if badge.Name == badgeName {
			return true, nil
		}
	}
	return false, nil
}

func (us *UserSystem) UpdateBadge(userID uuid.UUID, badgeName []string) error {
	var badge []models.Badge
	if err := us.Crud.GetByCondition(&badge, "name IN ?", []interface{}{badgeName}, []string{}, "", 0, 0); err != nil {
		return err
	}

	if len(badge) == 0 {
		return errors.New("no badge found")
	}

	var user models.User
	if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Badges"}, "", 0, 0); err != nil {
		return err
	}

	return us.Crud.UpdateManyToMany(&user, "Badges", &badge)
}

func (us *UserSystem) RemoveBadge(userid uuid.UUID, badges []uuid.UUID) error {
	var user models.User
	if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userid}, []string{"Badges"}, "", 0, 0); err != nil {
		return err
	}

	var badgesToDelete []models.Role
	for _, badgeId := range badges {
		for _, badge := range user.Roles {
			if badge.ID == badgeId {
				badgesToDelete = append(badgesToDelete, badge)
				break
			}
		}
	}

	return us.Crud.DeleteManyToMany(&user, "Badges", &badgesToDelete)
}

func (us *UserSystem) RemoveAllBadges(userid uuid.UUID) error {
	return us.Crud.ClearManyToMany(&models.User{}, "Badges", "id = ?", userid)
}

func (us *UserSystem) GetAllBadges() ([]models.Badge, error) {
	var badges []models.Badge
	if err := us.Crud.GetAll(&badges, []string{}); err != nil {
		return nil, err
	}
	return badges, nil
}

func (us *UserSystem) BadgeBy(userID *uuid.UUID, badgeIDs []uuid.UUID) ([]models.Badge, error) {
	var badges []models.Badge
	if userID != nil {
		var user models.User
		if err := us.Crud.GetByCondition(&user, "id = ?", []interface{}{userID}, []string{"Badge"}, "", 0, 0); err != nil {
			return nil, err
		}
		return user.Badges, nil
	} else if len(badgeIDs) > 0 {
		if err := us.Crud.GetByCondition(&badges, "id IN ?", []interface{}{badgeIDs}, []string{}, "", 0, 0); err != nil {
			return nil, err
		}
		return badges, nil
	}
	return nil, nil
}
