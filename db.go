package main

import (
	"time"
)

// Computer is a GORM model for a computer
type Computer struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	Name      string
	VcErrors  []VcError `gorm:"foreignkey:Computer;association_foreignkey:Name"`
}

// VcError is a GORM model that represents a DateTime when vcUpdater had to be started
type VcError struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	Computer  string
	Status    string
}
