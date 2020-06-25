package agent

import "time"

type Agent struct {
	Id   string    `json:"id"`
	Ip   string    `json:"ip"`
	Time time.Time `json:"time"`
}
