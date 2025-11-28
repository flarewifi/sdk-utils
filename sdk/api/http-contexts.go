/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package sdkapi

type sysacctctx string
type clientctx string
type httpsctx string
type webhookDeviceIDCtx string
type webhookPurchaseUIDCtx string

var (
	ClientCtxKey          clientctx             = "clnt"
	SysAcctCtxKey         sysacctctx            = "adminacct"
	WebhookDeviceIDKey    webhookDeviceIDCtx    = "webhook_device_id"
	WebhookPurchaseUIDKey webhookPurchaseUIDCtx = "webhook_purchase_uid"
	HttpsCtxKey           httpsctx              = "http_ctx"
)
