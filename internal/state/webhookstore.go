package state

// WebhookStore is the domain-scoped persistence surface for webhook
// definitions and delivery records. Consumers that only manage webhooks
// should depend on this interface rather than the aggregate StateStore.
type WebhookStore interface {
	CreateWebhook(webhook *Webhook) (int64, error)
	ListWebhooks() ([]*Webhook, error)
	GetWebhook(id int64) (*Webhook, error)
	UpdateWebhook(webhook *Webhook) error
	DeleteWebhook(id int64) error

	RecordWebhookDelivery(delivery *WebhookDelivery) error
	GetWebhookDeliveries(webhookID int64, limit int) ([]*WebhookDelivery, error)
}
