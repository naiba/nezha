package model

import (
	"errors"
	"math/big"
	"time"

	"github.com/nezhahq/nezha/pkg/utils"
	"gorm.io/gorm"
)

const (
	_ uint8 = iota
	WAFBlockReasonTypeLoginFail
	WAFBlockReasonTypeBruteForceToken
	WAFBlockReasonTypeAgentAuthFail
)

type WAFApiMock struct {
	IP                 string `json:"ip,omitempty"`
	Count              uint64 `json:"count,omitempty"`
	LastBlockReason    uint8  `json:"last_block_reason,omitempty"`
	LastBlockTimestamp uint64 `json:"last_block_timestamp,omitempty"`
}

type WAF struct {
	IP                 []byte `gorm:"type:binary(16);primaryKey" json:"ip,omitempty"`
	Count              uint64 `json:"count,omitempty"`
	LastBlockReason    uint8  `json:"last_block_reason,omitempty"`
	LastBlockTimestamp uint64 `json:"last_block_timestamp,omitempty"`
}

func (w *WAF) TableName() string {
	return "waf"
}

func CheckIP(db *gorm.DB, ip string) error {
	if ip == "" {
		return nil
	}
	ipBinary, err := utils.IPStringToBinary(ip)
	if err != nil {
		return err
	}
	var w WAF
	result := db.Limit(1).Find(&w, "ip = ?", ipBinary)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 { // 检查是否未找到记录
		return nil
	}
	now := time.Now().Unix()
	if powAdd(w.Count, 4, w.LastBlockTimestamp) > uint64(now) {
		return errors.New("you are blocked by nezha WAF")
	}
	return nil
}

func ClearIP(db *gorm.DB, ip string) error {
	if ip == "" {
		return nil
	}
	ipBinary, err := utils.IPStringToBinary(ip)
	if err != nil {
		return err
	}
	return db.Unscoped().Delete(&WAF{}, "ip = ?", ipBinary).Error
}

func BatchClearIP(db *gorm.DB, ip []string) error {
	if len(ip) < 1 {
		return nil
	}
	ips := make([][]byte, 0, len(ip))
	for _, s := range ip {
		ipBinary, err := utils.IPStringToBinary(s)
		if err != nil {
			continue
		}
		ips = append(ips, ipBinary)
	}
	return db.Unscoped().Delete(&WAF{}, "ip in (?)", ips).Error
}

func BlockIP(db *gorm.DB, ip string, reason uint8) error {
	if ip == "" {
		return nil
	}
	ipBinary, err := utils.IPStringToBinary(ip)
	if err != nil {
		return err
	}
	var w WAF
	w.IP = ipBinary
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&w).Attrs(WAF{
			LastBlockReason:    reason,
			LastBlockTimestamp: uint64(time.Now().Unix()),
		}).FirstOrCreate(&w).Error; err != nil {
			return err
		}
		return tx.Exec("UPDATE waf SET count = count + 1, last_block_reason = ?, last_block_timestamp = ? WHERE ip = ?", reason, uint64(time.Now().Unix()), ipBinary).Error
	})
}

func powAdd(x, y, z uint64) uint64 {
	base := big.NewInt(0).SetUint64(x)
	exp := big.NewInt(0).SetUint64(y)
	result := big.NewInt(1)
	result.Exp(base, exp, nil)
	result.Add(result, big.NewInt(0).SetUint64(z))
	if !result.IsUint64() {
		return ^uint64(0) // return max uint64 value on overflow
	}
	ret := result.Uint64()
	return utils.IfOr(ret < z+3, z+3, ret)
}
