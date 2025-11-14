package models

import (
	"context"
	"core/db"
	"core/db/queries"
)

type QuickAccessNavModel struct {
	db     *db.Database
	models *Models
}

func NewQuickAccessNavModel(database *db.Database, mdls *Models) *QuickAccessNavModel {
	return &QuickAccessNavModel{
		db:     database,
		models: mdls,
	}
}

func (self *QuickAccessNavModel) Upsert(ctx context.Context, pluginPkg string, routeName string, routeParams string) error {
	err := self.db.Queries.UpsertQuickAccessNav(ctx, queries.UpsertQuickAccessNavParams{
		PluginPkg:   pluginPkg,
		RouteName:   routeName,
		RouteParams: routeParams,
	})
	return err
}

func (self *QuickAccessNavModel) GetTop3(ctx context.Context) ([]*QuickAccessNav, error) {
	result, err := self.db.Queries.GetTop3QuickAccessNavs(ctx)
	if err != nil {
		return nil, err
	}

	navs := make([]*QuickAccessNav, len(result))
	for i, row := range result {
		navs[i] = NewQuickAccessNav(row)
	}

	return navs, nil
}

func (self *QuickAccessNavModel) Find(ctx context.Context, pluginPkg string, routeName string, routeParams string) (*QuickAccessNav, error) {
	result, err := self.db.Queries.FindQuickAccessNav(ctx, queries.FindQuickAccessNavParams{
		PluginPkg:   pluginPkg,
		RouteName:   routeName,
		RouteParams: routeParams,
	})
	if err != nil {
		return nil, err
	}

	return NewQuickAccessNav(result), nil
}
