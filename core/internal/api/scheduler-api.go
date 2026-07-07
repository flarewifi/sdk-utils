package api

import (
	"context"
	"time"
)

func NewSchedulerApi(api *PluginApi) {
	schedulerApi := &SchedulerApi{api: api}
	api.SchedulerAPI = schedulerApi
}

type SchedulerApi struct {
	api *PluginApi
}

func (s *SchedulerApi) Go(name string, fn func(ctx context.Context)) error {
	return s.api.SchedulerMgr.Go(s.api.info.Package, name, fn)
}

func (s *SchedulerApi) Every(name string, interval time.Duration, fn func(ctx context.Context)) error {
	return s.api.SchedulerMgr.Every(s.api.info.Package, name, interval, fn)
}

func (s *SchedulerApi) Cron(name string, expr string, fn func(ctx context.Context)) error {
	return s.api.SchedulerMgr.Cron(s.api.info.Package, name, expr, fn)
}

func (s *SchedulerApi) Cancel(name string) {
	s.api.SchedulerMgr.Cancel(s.api.info.Package, name)
}
