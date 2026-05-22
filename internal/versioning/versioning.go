package versioning

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseVersionCode(version string) (int64, error) {
	v := strings.TrimSpace(version)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")

	if idx := strings.IndexAny(v, "+-"); idx >= 0 {
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) > 3 {
		return 0, fmt.Errorf("invalid version: %s", version)
	}

	nums := []int64{0, 0, 0}
	for i, p := range parts {
		if p == "" {
			return 0, fmt.Errorf("invalid version: %s", version)
		}
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid version: %s", version)
		}
		nums[i] = n
	}

	return nums[0]*1_000_000 + nums[1]*1_000 + nums[2], nil
}
