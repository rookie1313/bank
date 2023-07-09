package util

const (
	USD = "USD"
	EUR = "EUR"
	CAD = "CAD"
)

func IsSupportCurrency(currency string) bool {
	switch currency {
	case USD, CAD, EUR:
		return true
	}

	return false
}
