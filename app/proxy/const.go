package proxy

const cacheResolveLongerThan = 10

// relaxedMethods are methods which are allowed to be called without wallet_id.
var relaxedMethods = []string{
	"blob_announce",
	"status",
	"resolve",
	"transaction_show",
	"stream_cost_estimate",
	"claim_search",
	"comment_list",
	"version",
	"routing_table_get",
}

// walletSpecificMethods are methods which require wallet_id.
// This list will inevitably turn stale sooner or later as new methods
// are added to the SDK so relaxedMethods should be used for strict validation
// whether wallet_id is required.
var walletSpecificMethods = []string{
	"publish",

	"address_unused",
	"address_list",
	"address_is_mine",

	"account_list",
	"account_balance",
	"account_send",
	"account_max_address_gap",

	"channel_abandon",
	"channel_create",
	"channel_list",
	"channel_update",
	"channel_export",
	"channel_import",

	"comment_abandon",
	"comment_create",
	"comment_hide",
	"comment_update",

	"claim_list",

	"stream_abandon",
	"stream_create",
	"stream_list",
	"stream_update",
	"stream_repost",

	"support_abandon",
	"support_create",
	"support_list",

	"sync_apply",
	"sync_hash",

	"preference_get",
	"preference_set",

	"transaction_list",

	"txo_list",
	"txo_sum",
	"txo_plot",

	"utxo_list",
	"utxo_release",

	"wallet_list",
	"wallet_send",
	"wallet_balance",
	"wallet_encrypt",
	"wallet_decrypt",
	"wallet_lock",
	"wallet_unlock",
	"wallet_status",
}

// forbiddenMethods are not allowed for remote calling.
// DEPRECATED: a sum of relaxedMethods and walletSpecificMethods should be used instead.
var forbiddenMethods = []string{
	"stop",

	"account_add",
	"account_create",
	"account_encrypt",
	"account_decrypt",
	"account_fund",
	"account_lock",
	"account_remove",
	"account_unlock",

	"file_delete",
	"file_list",
	"file_reflect",
	"file_save",
	"file_set_status",

	"peer_list",
	"peer_ping",

	"get",
	"sync_apply",

	"settings_get",
	"settings_set",

	"wallet_add",
	"wallet_create",
	"wallet_remove",

	"blob_get",
	"blob_reflect_all",
	"blob_list",
	"blob_delete",
	"blob_reflect",
}

const forbiddenParam = paramAccountID

const MethodGet = "get"
const MethodFileList = "file_list"
const MethodAccountList = "account_list"
const MethodAccountBalance = "account_balance"
const MethodStatus = "status"
const MethodResolve = "resolve"
const MethodClaimSearch = "claim_search"

const paramAccountID = "account_id"
const paramWalletID = "wallet_id"
const paramFundingAccountIDs = "funding_account_ids"
const paramUrls = "urls"

var ignoreLog = []string{
	MethodAccountBalance,
	MethodStatus,
}
