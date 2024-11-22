package model

import (
	"errors"
	"log"
	"math/big"
	"net/netip"
	"time"

	"gorm.io/gorm"
)

const (
	_ uint8 = iota
	WAFBlockReasonTypeLoginFail
	WAFBlockReasonTypeBruteForceToken
)

type WAF struct {
	IP                 []byte `gorm:"type:binary(16);primaryKey" json:"ip,omitempty"`
	Count              uint64 `json:"count,omitempty"`
	LastBlockReason    uint8  `json:"last_block_reason,omitempty"`
	LastBlockTimestamp uint64 `json:"last_block_timestamp,omitempty"`
}

func (w *WAF) TableName() string {
	return "waf"
}

func ipStringToBinary(ip string) ([]byte, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}
	b := addr.As16()
	return b[:], nil
}

func binaryToIPString(b []byte) string {
	var addr16 [16]byte
	copy(addr16[:], b)
	addr := netip.AddrFrom16(addr16)
	return addr.Unmap().String()
}

func CheckIP(db *gorm.DB, ip string) error {
	ipBinary, err := ipStringToBinary(ip)
	if err != nil {
		return err
	}
	var w WAF
	if err := db.First(&w, "ip = ?", ipBinary).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}
	now := time.Now().Unix()
	if w.LastBlockTimestamp+pow(w.Count, 4) > uint64(now) {
		log.Println(w.Count, w.LastBlockTimestamp+pow(w.Count, 4)-uint64(now))
		return errors.New("you are blocked by nezha WAF")
	}
	return nil
}

func BlockIP(db *gorm.DB, ip string, reason uint8) error {
	if ip == "" {
		return errors.New("empty ip")
	}
	ipBinary, err := ipStringToBinary(ip)
	if err != nil {
		return err
	}
	var w WAF
	w.LastBlockReason = reason
	w.LastBlockTimestamp = uint64(time.Now().Unix())
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.FirstOrCreate(&w, WAF{IP: ipBinary}).Error; err != nil {
			return err
		}
		return tx.Exec("UPDATE waf SET count = count + 1 WHERE ip = ?", ipBinary).Error
	})
}

func pow(x, y uint64) uint64 {
	base := big.NewInt(0).SetUint64(x)
	exp := big.NewInt(0).SetUint64(y)
	result := big.NewInt(1)
	result.Exp(base, exp, nil)
	if !result.IsUint64() {
		return ^uint64(0) // return max uint64 value on overflow
	}
	return result.Uint64()
}
