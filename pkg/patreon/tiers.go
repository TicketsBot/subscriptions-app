package patreon

//go:generate stringer -type=Tier
type Tier int

const (
	Premium Tier = iota
	Whitelabel
)

func GetTierFromId(id uint64) (Tier, bool) {
	switch id {
	case 4071609:
		return Premium, true
	case 5259899:
		fallthrough
	case 7502618:
		return Whitelabel, true
	default:
		return -1, false
	}
}
