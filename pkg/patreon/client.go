package patreon

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TicketsBot/subscriptions-app/internal/config"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient  *http.Client
	config      config.Config
	logger      *zap.Logger
	ratelimiter *rate.Limiter

	Tokens Tokens
}

const UserAgent = "ticketsbot.net/subscriptions-app (https://github.com/TicketsBot/subscriptions-app)"

func NewClient(config config.Config, logger *zap.Logger) *Client {
	return &Client{
		httpClient: http.DefaultClient,
		config:     config,
		logger:     logger,
		ratelimiter: rate.NewLimiter(
			rate.Every(time.Minute/time.Duration(config.Patreon.RequestsPerMinute)),
			config.Patreon.RequestsPerMinute,
		),
	}
}

func (c *Client) FetchPledges(ctx context.Context) (map[string]Patron, error) {
	url := fmt.Sprintf(
		"https://www.patreon.com/api/oauth2/v2/campaigns/%d/members?include=currently_entitled_tiers,user&fields%%5Bmember%%5D=last_charge_date,last_charge_status,patron_status,email,pledge_relationship_start&fields%%5Buser%%5D=social_connections",
		c.config.Patreon.CampaignId,
	)

	// Email -> Data
	data := make(map[string]Patron)
	for {
		res, err := c.FetchPageWithTimeout(ctx, 10*time.Minute, url)
		if err != nil {
			return nil, err
		}

		for _, member := range res.Data {
			id := member.Relationships.User.Data.Id

			if member.Attributes.Email == "" {
				c.logger.Debug("member has no email", zap.Uint64("patron_id", id))
				continue
			}

			// Parse tiers
			var tiers []uint64
			for _, tier := range member.Relationships.CurrentlyEntitledTiers.Data {
				// Check if tier is known
				if _, ok := c.config.Tiers[tier.TierId]; !ok {
					c.logger.Warn("unknown tier", zap.Uint64("tier_id", tier.TierId))
					continue
				}

				tiers = append(tiers, tier.TierId)
			}

			// Parse "included" metadata
			var discordId *uint64
			for _, included := range res.Included {
				if id == included.Id {
					if tmp := included.Attributes.SocialConnections.Discord.Id; tmp != nil {
						discordId = tmp
					}

					break
				}
			}

			data[member.Attributes.Email] = Patron{
				Attributes: member.Attributes,
				Id:         id,
				Tiers:      tiers,
				DiscordId:  discordId,
			}
		}

		if res.Links == nil || res.Links.Next == nil {
			break
		}

		url = *res.Links.Next
	}

	return data, nil
}

func (c *Client) FetchPageWithTimeout(ctx context.Context, timeout time.Duration, url string) (PledgeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.FetchPage(ctx, url)
}

func (c *Client) FetchPage(ctx context.Context, url string) (PledgeResponse, error) {
	c.logger.Debug("Fetching page", zap.String("url", url))

	if c.Tokens.ExpiresAt.Before(time.Now()) {
		return PledgeResponse{}, fmt.Errorf("Can't refresh: refresh token has already expired (expired at %s)", c.Tokens.ExpiresAt.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PledgeResponse{}, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Tokens.AccessToken)
	req.Header.Set("User-Agent", UserAgent)

	if err := c.ratelimiter.Wait(ctx); err != nil {
		return PledgeResponse{}, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return PledgeResponse{}, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			c.logger.Error(
				"error reading body of pledge response",
				zap.Int("status_code", res.StatusCode),
				zap.Error(err),
			)
			return PledgeResponse{}, err
		}

		c.logger.Error(
			"pledge response returned non-OK status code",
			zap.Int("status_code", res.StatusCode),
			zap.String("body", string(body)),
		)

		return PledgeResponse{}, fmt.Errorf("pledge response returned %d status code", res.StatusCode)
	}

	var body PledgeResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return PledgeResponse{}, err
	}

	c.logger.Debug("Page fetched successfully", zap.String("url", url))

	return body, nil
}

func (c *Client) GrantCredentials(ctx context.Context) (Tokens, error) {
	c.logger.Info("Doing client_credentials grant")

	uri := "https://www.patreon.com/api/oauth2/token"

	form := &url.Values{}
	form.Add("grant_type", "client_credentials")
	form.Add("client_id", c.config.Patreon.ClientId)
	form.Add("client_secret", c.config.Patreon.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", UserAgent)

	if err := c.ratelimiter.Wait(ctx); err != nil {
		return Tokens{}, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return Tokens{}, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			c.logger.Error(
				"error reading body during token refresh",
				zap.Int("status_code", res.StatusCode),
				zap.Error(err),
			)
			return Tokens{}, err
		}

		c.logger.Error(
			"Token grant returned non-OK status code",
			zap.Int("status_code", res.StatusCode),
			zap.String("body", string(body)),
		)

		return Tokens{}, fmt.Errorf("token grant returned %d status code", res.StatusCode)
	}

	var body RefreshResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Tokens{}, err
	}

	tokens := Tokens{
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(body.ExpiresIn) * time.Second),
	}

	c.logger.Info("Token grant successful", zap.Time("expires_at", tokens.ExpiresAt))

	c.Tokens = tokens
	return tokens, nil
}

func (c *Client) DoRefresh(ctx context.Context) (Tokens, error) {
	c.logger.Info("Doing token refresh")

	if c.Tokens.ExpiresAt.Before(time.Now()) {
		return Tokens{}, fmt.Errorf("Can't refresh: refresh token has already expired (expired at %s)", c.Tokens.ExpiresAt.String())
	}

	url := fmt.Sprintf(
		"https://www.patreon.com/api/oauth2/token?grant_type=refresh_token&refresh_token=%s&client_id=%s&client_secret=%s",
		c.Tokens.RefreshToken,
		c.config.Patreon.ClientId,
		c.config.Patreon.ClientSecret,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return Tokens{}, err
	}

	req.Header.Add("User-Agent", UserAgent)

	if err := c.ratelimiter.Wait(ctx); err != nil {
		return Tokens{}, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return Tokens{}, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			c.logger.Error(
				"error reading body during token refresh",
				zap.Int("status_code", res.StatusCode),
				zap.Error(err),
			)
			return Tokens{}, err
		}

		c.logger.Error(
			"token refresh returned non-OK status code",
			zap.Int("status_code", res.StatusCode),
			zap.String("body", string(body)),
		)

		return Tokens{}, fmt.Errorf("token refresh returned %d status code", res.StatusCode)
	}

	var body RefreshResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Tokens{}, err
	}

	tokens := Tokens{
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(body.ExpiresIn) * time.Second),
	}

	c.logger.Info("Token refresh successful", zap.Time("expires_at", tokens.ExpiresAt))

	c.Tokens = tokens
	return tokens, nil
}
