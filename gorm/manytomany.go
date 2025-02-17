package gorm

import "github.com/mnuddindev/devpulse/pkg/logger"

// Many2Many add many2many connections
func (g *GormDB) AddManyToMany(model interface{}, assoc string, userdata interface{}) error {
	if err := g.DB.Model(model).Association(assoc).Append(userdata); err != nil {
		logger.Log.WithError(err).Error("Failed to Add ManyToMany")
		return err
	}
	return nil
}

// Many2Many Find many2many connections
func (g *GormDB) FindManyToMany(model interface{}, assoc string, savemodel interface{}) error {
	if err := g.DB.Model(model).Association(assoc).Find(savemodel); err != nil {
		logger.Log.WithError(err).Error("Failed to Find ManyToMany")
		return err
	}
	return nil
}

// Many2Many update many2many connections
func (g *GormDB) UpdateManyToMany(model interface{}, assoc string, userdata interface{}) error {
	if err := g.DB.Model(model).Association(assoc).Replace(userdata); err != nil {
		logger.Log.WithError(err).Error("Failed to update ManyToMany")
		return err
	}
	return nil
}

// Many2Many update many2many connections
func (g *GormDB) DeleteManyToMany(model interface{}, assoc string, userdata interface{}) error {
	if err := g.DB.Model(model).Association(assoc).Delete(userdata); err != nil {
		logger.Log.WithError(err).Error("Failed to delete ManyToMany")
		return err
	}
	return nil
}

// Many2Many update many2many connections
func (g *GormDB) ClearManyToMany(model interface{}, assoc, condition string, args ...interface{}) error {
	if err := g.DB.Model(model).Where(condition, args...).Association(assoc).Clear(); err != nil {
		logger.Log.WithError(err).Error("Failed to Clear ManyToMany")
		return err
	}
	return nil
}
