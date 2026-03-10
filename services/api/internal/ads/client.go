package ads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fenco/trademate/services/api/internal/config"
)

type Client struct {
	httpClient *http.Client
	apiBase    string
	tokenURL   string
	clientID   string
	secret     string

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		apiBase:    strings.TrimRight(cfg.AmazonAdsAPIBase, "/"),
		tokenURL:   cfg.AmazonAdsTokenURL,
		clientID:   cfg.AmazonAdsClientID,
		secret:     cfg.AmazonAdsClientSecret,
	}
}

func (c *Client) FetchPreviewData(ctx context.Context, storeID string) (PreviewData, error) {
	if !c.isConfigured() {
		return mockPreviewData(storeID), nil
	}

	campaigns, campaignErr := c.ListCampaigns(ctx)
	keywords, keywordErr := c.ListKeywords(ctx)
	searchTerms, termErr := c.ListSearchTerms(ctx)
	if campaignErr != nil || keywordErr != nil || termErr != nil {
		return PreviewData{}, fmt.Errorf("fetch ads preview failed: %w", joinErrors(campaignErr, keywordErr, termErr))
	}

	return PreviewData{
		Campaigns:   campaigns,
		Keywords:    keywords,
		SearchTerms: searchTerms,
		Source:      "amazon_ads_api",
	}, nil
}

func (c *Client) ListCampaigns(ctx context.Context) ([]Campaign, error) {
	payload := []Campaign{}
	err := c.doJSON(ctx, http.MethodPost, "/sp/campaigns/list", map[string]any{"maxResults": 10}, &payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Client) ListKeywords(ctx context.Context) ([]Keyword, error) {
	payload := []Keyword{}
	err := c.doJSON(ctx, http.MethodPost, "/sp/keywords/list", map[string]any{"maxResults": 20}, &payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Client) ListSearchTerms(ctx context.Context) ([]SearchTerm, error) {
	payload := []SearchTerm{}
	err := c.doJSON(ctx, http.MethodPost, "/reporting/searchTerms", map[string]any{"lookbackDays": 7}, &payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, out any) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	var rawBody []byte
	if reqBody != nil {
		raw, marshalErr := json.Marshal(reqBody)
		if marshalErr != nil {
			return marshalErr
		}
		rawBody = raw
	}

	url := c.apiBase + path
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		var bodyReader io.Reader
		if len(rawBody) > 0 {
			bodyReader = bytes.NewReader(rawBody)
		}
		req, reqErr := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if reqErr != nil {
			return reqErr
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, doErr := c.httpClient.Do(req)
		if doErr != nil {
			lastErr = doErr
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}

		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}

		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("upstream status: %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt+1) * 250 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("upstream status: %d body: %s", resp.StatusCode, string(data))
		}
		if len(data) == 0 || string(data) == "null" {
			return nil
		}

		if err := json.Unmarshal(data, out); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("request failed after retries: %w", lastErr)
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.accessToken != "" && time.Now().UTC().Before(c.expiresAt.Add(-1*time.Minute)) {
		token := c.accessToken
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	if !c.isConfigured() {
		return "", errors.New("amazon ads credentials are not configured")
	}

	body := strings.NewReader("grant_type=client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, body)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.clientID, c.secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fetch token failed: %d %s", resp.StatusCode, string(data))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", errors.New("empty access token")
	}

	c.mu.Lock()
	c.accessToken = result.AccessToken
	if result.ExpiresIn <= 0 {
		result.ExpiresIn = 3600
	}
	c.expiresAt = time.Now().UTC().Add(time.Duration(result.ExpiresIn) * time.Second)
	token := c.accessToken
	c.mu.Unlock()

	return token, nil
}

func (c *Client) isConfigured() bool {
	return strings.TrimSpace(c.clientID) != "" && strings.TrimSpace(c.secret) != ""
}

func mockPreviewData(storeID string) PreviewData {
	return PreviewData{
		Campaigns: []Campaign{
			{CampaignID: "cmp_001", CampaignName: "Store " + storeID + " - Main Campaign", State: "enabled"},
			{CampaignID: "cmp_002", CampaignName: "Store " + storeID + " - Brand Defense", State: "enabled"},
		},
		Keywords: []Keyword{
			{KeywordID: "kw_001", CampaignID: "cmp_001", KeywordText: "wireless mouse", State: "enabled"},
			{KeywordID: "kw_002", CampaignID: "cmp_001", KeywordText: "bluetooth mouse", State: "enabled"},
		},
		SearchTerms: []SearchTerm{
			{CampaignID: "cmp_001", KeywordID: "kw_001", Term: "ergonomic wireless mouse", Clicks: 53, Spend: 31.22},
			{CampaignID: "cmp_001", KeywordID: "kw_002", Term: "small bluetooth mouse", Clicks: 21, Spend: 12.18},
		},
		Source: "mock",
	}
}

func joinErrors(errs ...error) error {
	parts := make([]string, 0)
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return errors.New(strings.Join(parts, "; "))
}
