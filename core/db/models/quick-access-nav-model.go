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

// UpsertQuickAccessNavParams holds parameters for upserting a quick access nav
type UpsertQuickAccessNavParams struct {
	PluginPkg   string
	RouteName   string
	RouteParams string
}

// FindQuickAccessNavParams holds parameters for finding a quick access nav
type FindQuickAccessNavParams struct {
	PluginPkg   string
	RouteName   string
	RouteParams string
}

func NewQuickAccessNavModel(database *db.Database, mdls *Models) *QuickAccessNavModel {
	return &QuickAccessNavModel{
		db:     database,
		models: mdls,
	}
}

func (self *QuickAccessNavModel) Upsert(ctx context.Context, params UpsertQuickAccessNavParams) error {
	err := self.db.Queries.UpsertQuickAccessNav(ctx, queries.UpsertQuickAccessNavParams{
		PluginPkg:   params.PluginPkg,
		RouteName:   params.RouteName,
		RouteParams: params.RouteParams,
	})
	return err
}

func (self *QuickAccessNavModel) GetTop5(ctx context.Context) ([]*QuickAccessNav, error) {
	result, err := self.db.Queries.GetTop5QuickAccessNavs(ctx)
	if err != nil {
		return nil, err
	}

	navs := make([]*QuickAccessNav, len(result))
	for i, row := range result {
		navs[i] = NewQuickAccessNav(row)
	}

	return navs, nil
}

func (self *QuickAccessNavModel) Find(ctx context.Context, params FindQuickAccessNavParams) (*QuickAccessNav, error) {
	result, err := self.db.Queries.FindQuickAccessNav(ctx, queries.FindQuickAccessNavParams{
		PluginPkg:   params.PluginPkg,
		RouteName:   params.RouteName,
		RouteParams: params.RouteParams,
	})
	if err != nil {
		return nil, err
	}

	return NewQuickAccessNav(result), nil
}
