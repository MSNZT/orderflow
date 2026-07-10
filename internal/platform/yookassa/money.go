package yookassa

import (
	"fmt"
	"strconv"
	"strings"
)

func parseAmountCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return 0, fmt.Errorf("invalid value: empty amount")
	}

	if strings.HasPrefix(value, "-") {
		return 0, fmt.Errorf("invalid value: negative number not allowed")
	}

	parts := strings.Split(value, ".")

	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid value: too many dots")
	}

	intPart := parts[0]
	if intPart == "" {
		return 0, fmt.Errorf("invalid value: missing integer part")
	}

	fracPart := ""

	if len(parts) == 2 {
		fracPart = parts[1]

		if len(fracPart) == 0 {
			return 0, fmt.Errorf("invalid value: missing fractional part")
		}

		if len(fracPart) >= 3 {
			return 0, fmt.Errorf("invalid value: too many decimal places, maximum 2 allowed")
		}

		if len(fracPart) == 1 {
			fracPart += "0"
		}
	} else {
		fracPart = "00"
	}

	for _, ch := range intPart + fracPart {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid value: contains non-digit characters")
		}
	}

	amount, err := strconv.ParseInt(intPart+fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value: failed to parse amount: %w", err)
	}

	return amount, nil
}

func formatAmount(amount int64) string {
	return fmt.Sprintf("%d.%02d", amount/100, amount%100)
}
