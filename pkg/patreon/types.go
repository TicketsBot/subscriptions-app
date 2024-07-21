package patreon

import "time"

type (
	Patron struct {
		Attributes
		Id        uint64
		Tiers     []uint64
		DiscordId *uint64
	}

	PledgeResponse struct {
		Data     []Member         `json:"data"`
		Included []PatronMetadata `json:"included"`
		Links    *struct {
			First string  `json:"first"`
			Next  *string `json:"next"`
		} `json:"links"`
	}

	Member struct {
		Attributes    Attributes `json:"attributes"`
		Relationships struct {
			User struct {
				Data struct {
					Id uint64 `json:"id,string"`
				} `json:"data"`
			} `json:"user"`
			CurrentlyEntitledTiers struct {
				Data []struct {
					TierId uint64 `json:"id,string"`
				} `json:"data"`
			} `json:"currently_entitled_tiers"`
		} `json:"relationships"`
	}

	Attributes struct {
		Email                   string    `json:"email"`
		LastChargeDate          time.Time `json:"last_charge_date"`
		LastChargeStatus        string    `json:"last_charge_status"`
		PatronStatus            string    `json:"patron_status"`
		PledgeRelationshipStart time.Time `json:"pledge_relationship_start"`
	}

	PatronMetadata struct {
		Id         uint64 `json:"id,string"`
		Attributes struct {
			SocialConnections struct {
				Discord struct {
					Id *uint64 `json:"user_id,string"`
				} `json:"discord"`
			} `json:"social_connections"`
		} `json:"attributes"`
	}

	RefreshResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"` // Seconds
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}

	Tokens struct {
		AccessToken  string    `json:"access_token"`
		RefreshToken string    `json:"refresh_token"`
		ExpiresAt    time.Time `json:"expires_at"`
	}
)
