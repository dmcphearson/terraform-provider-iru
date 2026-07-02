package client

// CustomScript models the /api/v1/library/custom-scripts contract. Note the
// self-service fields are accepted on write but NOT returned by GET (see
// API_REFERENCE.md), so omitempty keeps them out of read decoding.
type CustomScript struct {
	ID                     string `json:"id,omitempty"`
	Name                   string `json:"name"`
	ExecutionFrequency     string `json:"execution_frequency,omitempty"`
	Script                 string `json:"script,omitempty"`
	RemediationScript      string `json:"remediation_script,omitempty"`
	ShowInSelfService      bool   `json:"show_in_self_service"`
	SelfServiceCategoryID  string `json:"self_service_category_id,omitempty"`
	SelfServiceRecommended bool   `json:"self_service_recommended,omitempty"`
	Active                 bool   `json:"active"`
	Restart                bool   `json:"restart"`
}

// Tag models /api/v1/tags. There is no GET-by-id endpoint; Read lists and filters.
type Tag struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

// SelfServiceCategory models an item of the flat array returned by
// /api/v1/self-service/categories.
type SelfServiceCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CustomProfile models /api/v1/library/custom-profiles GET responses. The input
// .mobileconfig is uploaded as multipart `file`; the API returns it as `profile`.
type CustomProfile struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name"`
	MDMIdentifier string `json:"mdm_identifier,omitempty"`
	Profile       string `json:"profile,omitempty"`
	Active        bool   `json:"active"`
	RunsOnMac     bool   `json:"runs_on_mac"`
	RunsOnIPhone  bool   `json:"runs_on_iphone"`
	RunsOnIPad    bool   `json:"runs_on_ipad"`
	RunsOnTV      bool   `json:"runs_on_tv"`
	RunsOnVision  bool   `json:"runs_on_vision"`
}
