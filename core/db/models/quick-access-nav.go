package models

import (
	"core/db/queries"
	"time"
)

type QuickAccessNav struct {
	id          int64
	pluginPkg   string
	routeName   string
	routeParams string
	visitCount  int64
	createdAt   time.Time
	updatedAt   time.Time
}

func NewQuickAccessNav(qan queries.QuickAccessNav) *QuickAccessNav {
	return &QuickAccessNav{
		id:          qan.ID,
		pluginPkg:   qan.PluginPkg,
		routeName:   qan.RouteName,
		routeParams: qan.RouteParams,
		visitCount:  qan.VisitCount,
		createdAt:   qan.CreatedAt,
		updatedAt:   qan.UpdatedAt,
	}
}

func (self *QuickAccessNav) ID() int64 {
	return self.id
}

func (self *QuickAccessNav) PluginPkg() string {
	return self.pluginPkg
}

func (self *QuickAccessNav) RouteName() string {
	return self.routeName
}

func (self *QuickAccessNav) RouteParams() string {
	return self.routeParams
}

func (self *QuickAccessNav) VisitCount() int64 {
	return self.visitCount
}

func (self *QuickAccessNav) CreatedAt() time.Time {
	return self.createdAt
}

func (self *QuickAccessNav) UpdatedAt() time.Time {
	return self.updatedAt
}
