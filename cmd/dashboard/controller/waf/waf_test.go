package waf

import (
	"math"
	"testing"
)

func TestPow(t *testing.T) {
	tests := []struct {
		x,
		y,
		expect uint64
	}{
		{2, 64, math.MaxUint64},                 // 2 的 64 次方，超过 uint64 最大值
		{uint64(1 << 63), 2, math.MaxUint64},    // 大数平方，可能溢出
		{uint64(^uint64(0)), 2, math.MaxUint64}, // uint64 最大值的平方，溢出
		{2, 3, 8},
		{5, 0, 1},
		{3, 1, 3},
		{0, 5, 0},
	}

	for _, tt := range tests {
		result := pow(tt.x, tt.y)
		if result != tt.expect {
			t.Errorf("pow(%d, %d) = %d; expect %d", tt.x, tt.y, result, tt.expect)
		}
	}
}
