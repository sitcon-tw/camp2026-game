package apimodel

type ActivityListResponse struct {
	Activities []ActivitySummary `json:"activities"`
}

type ActivityDetailResponse struct {
	Activity ActivitySummary `json:"activity"`
}

type ActivitySummary struct {
	ActivityID  string `json:"activityId" example:"booth-linux-101"`
	Name        string `json:"name" example:"Linux 101 Booth"`
	Description string `json:"description" example:"Complete the booth challenge to receive a sitone."`
	Status      string `json:"status" example:"claimable"`
	Reward      Reward `json:"reward"`
}

type ActivityClaimResponse struct {
	Activity ActivitySummary `json:"activity"`
	Reward   Reward          `json:"reward"`
}
