package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/mnuddindev/devpulse/pkg/models"
)

// Follow used to follow a user
func (us *UserSystem) Follow(followerid, followingid uuid.UUID) (string, string, error) {
	var follower, following models.User

	// Check before following
	if err := us.crud.GetByID(&follower, followerid.String()); err != nil {
		return "", "", errors.New("follower not found")
	}
	if err := us.crud.GetByID(&following, followingid.String()); err != nil {
		return "", "", errors.New("following not found")
	}

	// check if users exist
	if follower.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return "", "", errors.New("follower not found")
	}
	if following.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return "", "", errors.New("user to unfollow not found")
	}

	// prevent a user from following themselves
	if followerid == followingid {
		return "", "", errors.New("you cannot follow yourself")
	}

	// add follow relationship in the join table
	if err := us.crud.AddManyToMany(&follower, "Following", &following); err != nil {
		return "", "", err
	}

	return (follower.FirstName + " " + follower.LastName), (following.FirstName + " " + following.LastName), nil
}

// Unfollow uses to unfollow a user
func (us *UserSystem) Unfollow(followerid, followingid uuid.UUID) (string, string, error) {
	var follower, following models.User

	// check if the users exist
	if err := us.crud.GetByID(&follower, followerid.String()); err != nil {
		return "", "", errors.New("follower not found")
	}
	if err := us.crud.GetByID(&following, followingid.String()); err != nil {
		return "", "", errors.New("user to unfollow not found")
	}

	if follower.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return "", "", errors.New("follower not found")
	}
	if following.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return "", "", errors.New("user to unfollow not found")
	}

	// add follow relationship in the join table
	if err := us.crud.DeleteManyToMany(&follower, "Following", &following); err != nil {
		return "", "", err
	}

	return (follower.FirstName + " " + follower.LastName), (following.FirstName + " " + following.LastName), nil
}

// GetFollowers find all followers of a user and returns it
func (us *UserSystem) GetFollowers(userid uuid.UUID) ([]models.User, error) {
	var user models.User
	var followers []models.User

	if err := us.crud.GetByCondition(&user, "id = ?", []interface{}{userid}, []string{}, "", 0, 0); err != nil {
		return nil, errors.New("user not found")
	}
	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, errors.New("user not found")
	}

	if err := us.crud.FindManyToMany(&user, "Followers", &followers); err != nil {
		return nil, err
	}

	return followers, nil
}

// GetFollowing find all following of a user and returns it
func (us *UserSystem) GetFollowing(userid uuid.UUID) ([]models.User, error) {
	var user models.User
	var following []models.User

	if err := us.crud.GetByCondition(&user, "id = ?", []interface{}{userid}, []string{}, "", 0, 0); err != nil {
		return nil, errors.New("user not found")
	}
	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, errors.New("user not found")
	}

	if err := us.crud.FindManyToMany(&user, "Following", &following); err != nil {
		return nil, err
	}

	return following, nil
}
