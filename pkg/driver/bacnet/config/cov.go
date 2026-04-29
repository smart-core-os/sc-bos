package config

type COVMethod string

const (
	COVMethodAuto  COVMethod = "auto"
	COVMethodPoll  COVMethod = "poll"
	COVMethodCov   COVMethod = "cov"
	COVMethodLocal COVMethod = "local"
	COVMethodNone  COVMethod = "none"
)

type COV struct {
	Method    COVMethod `json:"method,omitempty"`
	PollDelay Duration  `json:"pollDelay"`
}
