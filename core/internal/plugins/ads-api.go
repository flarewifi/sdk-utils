package plugins

type AdsApi struct {
	plugin *PluginApi
}

func (ads *AdsApi) Init(appId string) {

}

func NewAdsApi(plugin *PluginApi) {
	adsApi := &AdsApi{plugin}
	plugin.AdsAPI = adsApi
}
