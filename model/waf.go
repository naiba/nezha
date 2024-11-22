package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	_ uint8 = iota
	WAFBlockReasonTypeLoginFail
	WAFBlockReasonTypeBruteForceToken
)

type WAF struct {
	IP                 string `gorm:"type:binary(16);primaryKey" json:"ip,omitempty"`
	Count              uint64 `json:"count,omitempty"`
	LastBlockReason    uint8  `json:"last_block_reason,omitempty"`
	LastBlockTimestamp uint64 `json:"last_block_timestamp,omitempty"`
}

func (w *WAF) TableName() string {
	return "waf"
}

func BlockIP(db *gorm.DB, ip string, reason uint8) error {
	if ip == "" {
		return errors.New("empty ip")
	}
	var w WAF
	w.LastBlockReason = reason
	w.LastBlockTimestamp = uint64(time.Now().Unix())
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.FirstOrCreate(&w, WAF{IP: ip}).Error; err != nil {
			return err
		}
		return tx.Exec("UPDATE waf SET count = count + 1 WHERE ip = ?", ip).Error
	})
}
