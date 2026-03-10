package ads

type Campaign struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	State        string `json:"state"`
}

type Keyword struct {
	KeywordID   string `json:"keyword_id"`
	CampaignID  string `json:"campaign_id"`
	KeywordText string `json:"keyword_text"`
	State       string `json:"state"`
}

type SearchTerm struct {
	CampaignID string  `json:"campaign_id"`
	KeywordID  string  `json:"keyword_id"`
	Term       string  `json:"term"`
	Clicks     int64   `json:"clicks"`
	Spend      float64 `json:"spend"`
}

type PreviewData struct {
	Campaigns   []Campaign   `json:"campaigns"`
	Keywords    []Keyword    `json:"keywords"`
	SearchTerms []SearchTerm `json:"search_terms"`
	Source      string       `json:"source"`
}
