package models

import (
	"github.com/goravel/framework/support/carbon"

	"panel/packages/helper"
)

type Monitor struct {
	ID        uint                  `gorm:"primaryKey" json:"id"`
	Info      helper.MonitoringInfo `gorm:"type:json;serializer:json" json:"info"`
	CreatedAt carbon.DateTime       `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt carbon.DateTime       `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`
}
