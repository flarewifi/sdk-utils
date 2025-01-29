package models

import "time"

type Log struct {
	Package   string
	Level     string
	Message   string
	Filepath  string
	Line      int
	CreatedAt time.Time
}
