package entities

type Currency string

const (
	EUR Currency = "EUR"
	USD Currency = "USD"
	RUB Currency = "RUB"
	CNY Currency = "CNY"
)

func (c Currency) IsValid() bool {
	return c == EUR || c == USD || c == RUB || c == CNY
}

func (c Currency) String() string {
	switch c {
	case EUR:
		return "EUR"
	case USD:
		return "USD"
	case RUB:
		return "RUB"
	case CNY:
		return "CNY"
	}
	return ""
}

func (c Currency) OKVCode() string {
	switch c {
	case RUB:
		return "643"
	case USD:
		return "840"
	case EUR:
		return "978"
	case CNY:
		return "156"
	default:
		return ""
	}
}
